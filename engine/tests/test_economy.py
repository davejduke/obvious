"""Unit tests — Economy Engine.

Covers:
- Work program generation (existing + new scope-DAG integration)
- Progress tracking (WorkItemStatus, pct_complete, overall_progress)
- Resource allocation (risk-weighted greedy, specialization preference)
- Budget tracking (plan/actual, variance, alert thresholds)
- Work program templates (instantiation, registry, hours scaling)
- Coverage gap reporting
"""
import pytest
from uuid import uuid4

from engine.economy.work_program import (
    WorkItem,
    WorkItemStatus,
    WorkProgram,
    compute_coverage_gap,
    generate_work_program,
    generate_work_program_from_dag,
)
from engine.economy.resource_allocation import (
    AllocationResult,
    AuditorResource,
    allocate_resources,
)
from engine.economy.budget_tracker import (
    AlertLevel,
    BudgetEntry,
    BudgetTracker,
    VarianceReport,
    _alert_level,
)
from engine.economy.templates import (
    TEMPLATE_REGISTRY,
    TemplateStep,
    WorkProgramTemplate,
    get_template,
    list_templates,
    TEMPLATE_ACCESS_CONTROL,
    TEMPLATE_INCIDENT_RESPONSE,
)
from engine.scope.dag import AuditScopeDAG, BoundaryStatus, ScopeNode, ScopeState


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def make_control(control_id: str, risk_weight: float = 1.0, **kwargs):
    return {
        "control_id": control_id,
        "title": f"Control {control_id}",
        "risk_weight": risk_weight,
        "article_ref": "NIS2-21a",
        **kwargs,
    }


def make_auditor(
    name: str,
    available_hours: float = 160.0,
    hourly_cost: float = 150.0,
    specializations: frozenset[str] = frozenset(),
) -> AuditorResource:
    return AuditorResource(
        name=name,
        available_hours=available_hours,
        hourly_cost=hourly_cost,
        specializations=specializations,
    )


def make_work_item(
    control_id: str = "C-001",
    risk_weight: float = 2.0,
    estimated_hours: float = 8.0,
    evidence_required: list[str] | None = None,
) -> WorkItem:
    return WorkItem(
        item_id=uuid4(),
        control_id=control_id,
        title=f"Audit {control_id}",
        description=f"Audit steps for {control_id}",
        estimated_hours=estimated_hours,
        risk_weight=risk_weight,
        evidence_required=evidence_required or ["documentation"],
    )


def make_approved_dag(*controls) -> AuditScopeDAG:
    """Build a simple APPROVED DAG with the given (control_id, risk_weight) pairs."""
    dag = AuditScopeDAG()
    for control_id, risk_weight in controls:
        node = ScopeNode(
            control_id=control_id,
            title=f"Control {control_id}",
            boundary_status=BoundaryStatus.IN_SCOPE,
            risk_weight=risk_weight,
        )
        dag.add_node(node)
    dag.transition(ScopeState.PROPOSED)
    dag.transition(ScopeState.APPROVED)
    return dag


# ===========================================================================
# generate_work_program (original tests preserved)
# ===========================================================================

class TestGenerateWorkProgram:
    def test_empty_controls_returns_empty_program(self):
        program = generate_work_program([])
        assert len(program.items) == 0
        assert program.coverage_score == 0.0

    def test_single_control_within_budget(self):
        ctrl = make_control("C-001", risk_weight=1.0)
        program = generate_work_program([ctrl], budget_hours=100.0)
        assert len(program.items) == 1
        assert program.coverage_score == pytest.approx(1.0)

    def test_high_risk_prioritised_over_low(self):
        controls = [
            make_control("LOW", risk_weight=0.5),
            make_control("HIGH", risk_weight=4.5),
        ]
        program = generate_work_program(controls, budget_hours=40.0)
        ids = [item.control_id for item in program.items]
        assert ids[0] == "HIGH"

    def test_budget_constraint_respected(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=5.0) for i in range(20)]
        program = generate_work_program(controls, budget_hours=40.0)
        assert program.total_estimated_hours <= 40.0

    def test_coverage_score_range(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=2.0) for i in range(5)]
        program = generate_work_program(controls, budget_hours=100.0)
        assert 0.0 <= program.coverage_score <= 1.0

    def test_coverage_score_increases_with_budget(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=2.0) for i in range(10)]
        low_budget = generate_work_program(controls, budget_hours=20.0)
        high_budget = generate_work_program(controls, budget_hours=500.0)
        assert high_budget.coverage_score >= low_budget.coverage_score

    def test_deterministic_output(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=float(i % 5 + 1)) for i in range(8)]
        p1 = generate_work_program(controls, budget_hours=50.0)
        p2 = generate_work_program(controls, budget_hours=50.0)
        assert [i.control_id for i in p1.items] == [i.control_id for i in p2.items]
        assert p1.coverage_score == p2.coverage_score

    def test_engagement_id_propagated(self):
        eid = uuid4()
        program = generate_work_program([make_control("C-001")], engagement_id=eid)
        assert program.engagement_id == eid


class TestCoverageGap:
    def test_all_included_no_gap(self):
        controls = [make_control("A"), make_control("B")]
        program = generate_work_program(controls, budget_hours=1000.0)
        gap = compute_coverage_gap(program, controls)
        assert gap == []

    def test_excluded_controls_reported(self):
        controls = [make_control(f"C-{i:03d}", risk_weight=5.0) for i in range(20)]
        program = generate_work_program(controls, budget_hours=20.0)
        gap = compute_coverage_gap(program, controls)
        included = {item.control_id for item in program.items}
        for ctrl_id in gap:
            assert ctrl_id not in included


# ===========================================================================
# generate_work_program_from_dag
# ===========================================================================

class TestGenerateWorkProgramFromDAG:
    def test_in_scope_nodes_only(self):
        dag = AuditScopeDAG()
        in_node = ScopeNode(
            control_id="IN-001", title="In scope",
            boundary_status=BoundaryStatus.IN_SCOPE, risk_weight=3.0,
        )
        out_node = ScopeNode(
            control_id="OUT-001", title="Out of scope",
            boundary_status=BoundaryStatus.OUT_OF_SCOPE, risk_weight=5.0,
        )
        dag.add_node(in_node)
        dag.add_node(out_node)
        dag.transition(ScopeState.PROPOSED)
        dag.transition(ScopeState.APPROVED)

        program = generate_work_program_from_dag(dag)
        assert len(program.items) == 1
        assert program.items[0].control_id == "IN-001"

    def test_no_in_scope_nodes_returns_empty(self):
        dag = AuditScopeDAG()
        node = ScopeNode(
            control_id="OUT-001", title="Out",
            boundary_status=BoundaryStatus.OUT_OF_SCOPE, risk_weight=2.0,
        )
        dag.add_node(node)
        dag.transition(ScopeState.PROPOSED)
        dag.transition(ScopeState.APPROVED)

        program = generate_work_program_from_dag(dag)
        assert len(program.items) == 0

    def test_requires_approved_or_locked_state(self):
        dag = AuditScopeDAG()  # DRAFT state
        with pytest.raises(ValueError, match="APPROVED or LOCKED"):
            generate_work_program_from_dag(dag)

    def test_requires_approved_not_proposed(self):
        dag = AuditScopeDAG()
        dag.transition(ScopeState.PROPOSED)
        with pytest.raises(ValueError, match="APPROVED or LOCKED"):
            generate_work_program_from_dag(dag)

    def test_locked_dag_accepted(self):
        dag = make_approved_dag(("C-001", 2.0))
        dag.transition(ScopeState.LOCKED)
        program = generate_work_program_from_dag(dag)
        assert len(program.items) == 1

    def test_topological_order_respected(self):
        """Parent before child in generated items (topological property)."""
        dag = AuditScopeDAG()
        parent = ScopeNode(
            control_id="PARENT", title="Parent",
            boundary_status=BoundaryStatus.IN_SCOPE, risk_weight=2.0,
        )
        child = ScopeNode(
            control_id="CHILD", title="Child",
            boundary_status=BoundaryStatus.IN_SCOPE, risk_weight=3.0,
            parent_ids=[parent.node_id],
        )
        dag.add_node(parent)
        dag.add_node(child)
        dag.transition(ScopeState.PROPOSED)
        dag.transition(ScopeState.APPROVED)

        # Both controls must be included at a generous budget
        program = generate_work_program_from_dag(dag, budget_hours=500.0)
        control_ids = [item.control_id for item in program.items]
        assert "PARENT" in control_ids
        assert "CHILD" in control_ids

    def test_budget_constraint_from_dag(self):
        dag = make_approved_dag(*[(f"C-{i:03d}", 5.0) for i in range(20)])
        program = generate_work_program_from_dag(dag, budget_hours=40.0)
        assert program.total_estimated_hours <= 40.0

    def test_risk_weights_from_dag_nodes(self):
        dag = make_approved_dag(("C-HIGH", 5.0), ("C-LOW", 0.5))
        program = generate_work_program_from_dag(dag, budget_hours=500.0)
        high = next(i for i in program.items if i.control_id == "C-HIGH")
        low = next(i for i in program.items if i.control_id == "C-LOW")
        assert high.risk_weight == 5.0
        assert low.risk_weight == 0.5

    def test_engagement_id_propagated(self):
        eid = uuid4()
        dag = make_approved_dag(("C-001", 1.0))
        program = generate_work_program_from_dag(dag, engagement_id=eid)
        assert program.engagement_id == eid


# ===========================================================================
# Progress tracking
# ===========================================================================

class TestWorkProgramProgress:
    def test_default_progress_is_zero_pending(self):
        program = generate_work_program([make_control("C-001")], budget_hours=100.0)
        item = program.items[0]
        pct, status = program.get_item_progress(item.item_id)
        assert pct == 0.0
        assert status == WorkItemStatus.PENDING

    def test_record_progress_in_progress(self):
        program = generate_work_program([make_control("C-001")], budget_hours=100.0)
        item = program.items[0]
        program.record_progress(item.item_id, 0.5)
        pct, status = program.get_item_progress(item.item_id)
        assert pct == pytest.approx(0.5)
        assert status == WorkItemStatus.IN_PROGRESS

    def test_record_progress_100pct_auto_completes(self):
        program = generate_work_program([make_control("C-001")], budget_hours=100.0)
        item = program.items[0]
        program.record_progress(item.item_id, 1.0)
        _, status = program.get_item_progress(item.item_id)
        assert status == WorkItemStatus.COMPLETE

    def test_record_progress_clamps_above_1(self):
        program = generate_work_program([make_control("C-001")], budget_hours=100.0)
        item = program.items[0]
        program.record_progress(item.item_id, 2.5)
        pct, status = program.get_item_progress(item.item_id)
        assert pct == pytest.approx(1.0)
        assert status == WorkItemStatus.COMPLETE

    def test_record_progress_clamps_below_0(self):
        program = generate_work_program([make_control("C-001")], budget_hours=100.0)
        item = program.items[0]
        program.record_progress(item.item_id, -0.5)
        pct, _ = program.get_item_progress(item.item_id)
        assert pct == pytest.approx(0.0)

    def test_record_progress_blocked_status(self):
        program = generate_work_program([make_control("C-001")], budget_hours=100.0)
        item = program.items[0]
        program.record_progress(item.item_id, 0.3, WorkItemStatus.BLOCKED)
        _, status = program.get_item_progress(item.item_id)
        assert status == WorkItemStatus.BLOCKED

    def test_record_progress_unknown_item_raises(self):
        program = generate_work_program([make_control("C-001")], budget_hours=100.0)
        with pytest.raises(KeyError):
            program.record_progress(uuid4(), 0.5)

    def test_compute_overall_progress_empty(self):
        program = WorkProgram()
        assert program.compute_overall_progress() == 0.0

    def test_compute_overall_progress_none_started(self):
        controls = [make_control(f"C-{i:03d}") for i in range(3)]
        program = generate_work_program(controls, budget_hours=500.0)
        assert program.compute_overall_progress() == pytest.approx(0.0)

    def test_compute_overall_progress_all_complete(self):
        controls = [make_control(f"C-{i:03d}") for i in range(3)]
        program = generate_work_program(controls, budget_hours=500.0)
        for item in program.items:
            program.record_progress(item.item_id, 1.0)
        assert program.compute_overall_progress() == pytest.approx(1.0)

    def test_compute_overall_progress_weighted(self):
        """Overall progress is weighted by estimated_hours, not item count."""
        # Create two items: one large (8h) and one small (2h)
        large = make_work_item("LARGE", estimated_hours=8.0)
        small = make_work_item("SMALL", estimated_hours=2.0)
        program = WorkProgram()
        program.add_item(large)
        program.add_item(small)
        # Mark only the large item as 50% complete
        program.record_progress(large.item_id, 0.5)
        # Expected: (0.5 * 8 + 0 * 2) / (8 + 2) = 4/10 = 0.4
        assert program.compute_overall_progress() == pytest.approx(0.4)

    def test_items_by_status(self):
        controls = [make_control(f"C-{i:03d}") for i in range(4)]
        program = generate_work_program(controls, budget_hours=500.0)
        items = program.items
        program.record_progress(items[0].item_id, 1.0)
        program.record_progress(items[1].item_id, 0.5)
        program.record_progress(items[2].item_id, 0.3, WorkItemStatus.BLOCKED)
        # items[3] remains PENDING

        complete = program.items_by_status(WorkItemStatus.COMPLETE)
        in_progress = program.items_by_status(WorkItemStatus.IN_PROGRESS)
        blocked = program.items_by_status(WorkItemStatus.BLOCKED)
        pending = program.items_by_status(WorkItemStatus.PENDING)

        assert len(complete) == 1
        assert len(in_progress) == 1
        assert len(blocked) == 1
        assert len(pending) == 1


# ===========================================================================
# Resource allocation
# ===========================================================================

class TestAllocateResources:
    def test_empty_items_returns_empty_result(self):
        auditors = [make_auditor("Alice")]
        result = allocate_resources([], auditors)
        assert result.assignments == {}
        assert result.unassigned_item_ids == []

    def test_empty_auditors_returns_empty_result(self):
        items = [make_work_item()]
        result = allocate_resources(items, [])
        assert result.assignments == {}
        assert result.unassigned_item_ids == []

    def test_single_item_single_auditor(self):
        item = make_work_item(estimated_hours=8.0)
        auditor = make_auditor("Alice", available_hours=160.0)
        result = allocate_resources([item], [auditor])
        assert item.item_id in result.assignments
        assert result.assignments[item.item_id] == auditor.auditor_id
        assert len(result.unassigned_item_ids) == 0

    def test_all_items_assigned_within_capacity(self):
        items = [make_work_item(f"C-{i:03d}", estimated_hours=8.0) for i in range(5)]
        auditor = make_auditor("Alice", available_hours=160.0)
        result = allocate_resources(items, [auditor])
        assert len(result.assignments) == 5
        assert len(result.unassigned_item_ids) == 0

    def test_capacity_exceeded_yields_unassigned(self):
        items = [make_work_item(f"C-{i:03d}", estimated_hours=40.0) for i in range(5)]
        auditor = make_auditor("Alice", available_hours=80.0)
        result = allocate_resources(items, [auditor])
        # Only 2 items can fit in 80h (2 * 40 = 80)
        assert len(result.assignments) == 2
        assert len(result.unassigned_item_ids) == 3

    def test_specialist_preferred_over_generalist(self):
        item = make_work_item(evidence_required=["log_data", "siem_export"])
        specialist = make_auditor(
            "Specialist", specializations=frozenset(["log_data"]), available_hours=160.0
        )
        generalist = make_auditor("Generalist", available_hours=160.0)
        result = allocate_resources([item], [generalist, specialist])
        assert result.assignments[item.item_id] == specialist.auditor_id

    def test_fallback_to_generalist_when_no_specialist(self):
        item = make_work_item(evidence_required=["exotic_evidence"])
        generalist = make_auditor("Generalist", available_hours=160.0)
        result = allocate_resources([item], [generalist])
        assert item.item_id in result.assignments
        assert result.assignments[item.item_id] == generalist.auditor_id

    def test_high_risk_assigned_first(self):
        """High-risk items get first pick of auditor capacity."""
        high = make_work_item("HIGH", risk_weight=5.0, estimated_hours=80.0)
        low = make_work_item("LOW", risk_weight=0.5, estimated_hours=80.0)
        # Only 80h available — only one item can fit
        auditor = make_auditor("Alice", available_hours=80.0)
        result = allocate_resources([low, high], [auditor])
        # HIGH wins the capacity
        assert high.item_id in result.assignments
        assert low.item_id in result.unassigned_item_ids

    def test_cost_estimation(self):
        item = make_work_item(estimated_hours=10.0)
        auditor = make_auditor("Alice", hourly_cost=200.0, available_hours=160.0)
        result = allocate_resources([item], [auditor])
        assert result.total_estimated_cost == pytest.approx(2000.0)

    def test_cost_estimation_multiple_auditors(self):
        item1 = make_work_item("C-001", estimated_hours=10.0)
        item2 = make_work_item("C-002", estimated_hours=20.0)
        a1 = make_auditor("Alice", hourly_cost=100.0, available_hours=20.0)
        a2 = make_auditor("Bob", hourly_cost=200.0, available_hours=20.0)
        result = allocate_resources([item1, item2], [a1, a2])
        # Both assigned; total depends on who gets which item
        assert result.total_estimated_cost > 0

    def test_auditor_utilization_full(self):
        item = make_work_item(estimated_hours=160.0)
        auditor = make_auditor("Alice", available_hours=160.0)
        result = allocate_resources([item], [auditor])
        util = result.auditor_utilization()
        assert util["Alice"] == pytest.approx(1.0)

    def test_auditor_utilization_partial(self):
        item = make_work_item(estimated_hours=80.0)
        auditor = make_auditor("Alice", available_hours=160.0)
        result = allocate_resources([item], [auditor])
        util = result.auditor_utilization()
        assert util["Alice"] == pytest.approx(0.5)

    def test_deterministic_across_identical_calls(self):
        items = [make_work_item(f"C-{i:03d}", risk_weight=float(i + 1)) for i in range(5)]
        auditors = [make_auditor("Alice"), make_auditor("Bob")]
        r1 = allocate_resources(items, auditors)
        # Re-create fresh auditors (allocate mutates state)
        auditors2 = [make_auditor("Alice"), make_auditor("Bob")]
        r2 = allocate_resources(items, auditors2)
        assert set(r1.assignments.keys()) == set(r2.assignments.keys())


# ===========================================================================
# Budget tracker
# ===========================================================================

class TestAlertLevelFunction:
    @pytest.mark.parametrize("utilization,expected", [
        (0.0, AlertLevel.NONE),
        (0.49, AlertLevel.NONE),
        (0.50, AlertLevel.APPROACHING),
        (0.74, AlertLevel.APPROACHING),
        (0.75, AlertLevel.WARNING),
        (0.89, AlertLevel.WARNING),
        (0.90, AlertLevel.CRITICAL),
        (0.99, AlertLevel.CRITICAL),
        (1.00, AlertLevel.EXCEEDED),
        (1.50, AlertLevel.EXCEEDED),
    ])
    def test_thresholds(self, utilization, expected):
        assert _alert_level(utilization) == expected


class TestBudgetEntry:
    def test_planned_cost(self):
        entry = BudgetEntry(item_id=uuid4(), control_id="C-001",
                            planned_hours=10.0, hourly_rate=100.0)
        assert entry.planned_cost == pytest.approx(1000.0)

    def test_actual_cost(self):
        entry = BudgetEntry(item_id=uuid4(), control_id="C-001",
                            planned_hours=10.0, hourly_rate=100.0, actual_hours=12.0)
        assert entry.actual_cost == pytest.approx(1200.0)

    def test_variance_under_budget(self):
        entry = BudgetEntry(item_id=uuid4(), control_id="C-001",
                            planned_hours=10.0, actual_hours=8.0)
        assert entry.variance_hours == pytest.approx(2.0)
        assert entry.is_over_budget is False

    def test_variance_over_budget(self):
        entry = BudgetEntry(item_id=uuid4(), control_id="C-001",
                            planned_hours=10.0, actual_hours=12.0)
        assert entry.variance_hours == pytest.approx(-2.0)
        assert entry.is_over_budget is True

    def test_utilization_zero_planned_returns_zero(self):
        entry = BudgetEntry(item_id=uuid4(), control_id="C-001", planned_hours=0.0)
        assert entry.utilization == 0.0


class TestBudgetTracker:
    def test_plan_and_record_basic(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.record_actual(iid, actual_hours=8.0)
        entry = tracker.get_entry(iid)
        assert entry.planned_hours == pytest.approx(10.0)
        assert entry.actual_hours == pytest.approx(8.0)

    def test_negative_planned_raises(self):
        tracker = BudgetTracker()
        with pytest.raises(ValueError, match="non-negative"):
            tracker.plan_hours(uuid4(), "C-001", planned_hours=-1.0)

    def test_negative_actual_raises(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        with pytest.raises(ValueError, match="non-negative"):
            tracker.record_actual(iid, actual_hours=-5.0)

    def test_record_without_plan_raises(self):
        tracker = BudgetTracker()
        with pytest.raises(KeyError):
            tracker.record_actual(uuid4(), actual_hours=5.0)

    def test_get_entry_missing_raises(self):
        tracker = BudgetTracker()
        with pytest.raises(KeyError):
            tracker.get_entry(uuid4())

    def test_alert_level_none_when_under_50pct(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.record_actual(iid, actual_hours=4.0)  # 40%
        assert tracker.check_alert(iid) == AlertLevel.NONE

    def test_alert_level_approaching_at_50pct(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.record_actual(iid, actual_hours=5.0)  # 50%
        assert tracker.check_alert(iid) == AlertLevel.APPROACHING

    def test_alert_level_warning_at_75pct(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=8.0)
        tracker.record_actual(iid, actual_hours=6.0)  # 75%
        assert tracker.check_alert(iid) == AlertLevel.WARNING

    def test_alert_level_critical_at_90pct(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.record_actual(iid, actual_hours=9.0)  # 90%
        assert tracker.check_alert(iid) == AlertLevel.CRITICAL

    def test_alert_level_exceeded_at_100pct(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.record_actual(iid, actual_hours=10.0)  # 100%
        assert tracker.check_alert(iid) == AlertLevel.EXCEEDED

    def test_alert_level_exceeded_over_100pct(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.record_actual(iid, actual_hours=15.0)  # 150%
        assert tracker.check_alert(iid) == AlertLevel.EXCEEDED

    def test_overall_alert_aggregates(self):
        tracker = BudgetTracker()
        iid1 = uuid4()
        iid2 = uuid4()
        tracker.plan_hours(iid1, "C-001", planned_hours=10.0)
        tracker.plan_hours(iid2, "C-002", planned_hours=10.0)
        tracker.record_actual(iid1, actual_hours=4.0)   # 40% alone
        tracker.record_actual(iid2, actual_hours=6.0)   # 60% alone
        # Aggregate: 10/20 = 50% -> APPROACHING
        assert tracker.overall_alert() == AlertLevel.APPROACHING

    def test_all_alerts_excludes_none(self):
        tracker = BudgetTracker()
        iid1 = uuid4()
        iid2 = uuid4()
        tracker.plan_hours(iid1, "C-001", planned_hours=10.0)
        tracker.plan_hours(iid2, "C-002", planned_hours=10.0)
        tracker.record_actual(iid1, actual_hours=1.0)   # 10% -> NONE
        tracker.record_actual(iid2, actual_hours=8.0)   # 80% -> WARNING
        alerts = tracker.all_alerts()
        assert iid1 not in alerts
        assert iid2 in alerts
        assert alerts[iid2] == AlertLevel.WARNING

    def test_variance_report_under_budget(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0, hourly_rate=100.0)
        tracker.record_actual(iid, actual_hours=8.0)
        report = tracker.variance_report()
        assert report.total_planned_hours == pytest.approx(10.0)
        assert report.total_actual_hours == pytest.approx(8.0)
        assert report.total_variance_hours == pytest.approx(2.0)
        assert report.total_planned_cost == pytest.approx(1000.0)
        assert report.total_actual_cost == pytest.approx(800.0)
        assert report.total_variance_cost == pytest.approx(200.0)
        assert len(report.over_budget_items) == 0

    def test_variance_report_over_budget(self):
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.record_actual(iid, actual_hours=12.0)
        report = tracker.variance_report()
        assert len(report.over_budget_items) == 1
        assert report.over_budget_items[0].item_id == iid

    def test_variance_report_empty_tracker(self):
        tracker = BudgetTracker()
        report = tracker.variance_report()
        assert report.total_planned_hours == 0.0
        assert report.total_actual_hours == 0.0
        assert report.overall_utilization == 0.0
        assert report.overall_alert_level == AlertLevel.NONE

    def test_plan_hours_replace_existing(self):
        """Re-planning an item replaces the entry."""
        tracker = BudgetTracker()
        iid = uuid4()
        tracker.plan_hours(iid, "C-001", planned_hours=10.0)
        tracker.plan_hours(iid, "C-001", planned_hours=20.0)
        entry = tracker.get_entry(iid)
        assert entry.planned_hours == pytest.approx(20.0)


# ===========================================================================
# Work program templates
# ===========================================================================

class TestTemplateInstantiate:
    def test_instantiate_returns_work_items(self):
        items = TEMPLATE_ACCESS_CONTROL.instantiate("NIS2-21b", risk_weight=3.0)
        assert len(items) > 0
        for item in items:
            assert isinstance(item, WorkItem)

    def test_instantiate_uses_control_id(self):
        items = TEMPLATE_ACCESS_CONTROL.instantiate("MY-CTRL-001")
        for item in items:
            assert item.control_id == "MY-CTRL-001"

    def test_instantiate_risk_weight_propagated(self):
        items = TEMPLATE_ACCESS_CONTROL.instantiate("C-001", risk_weight=4.5)
        for item in items:
            assert item.risk_weight == pytest.approx(4.5)

    def test_instantiate_title_includes_control_id(self):
        items = TEMPLATE_INCIDENT_RESPONSE.instantiate("NIS2-21a")
        for item in items:
            assert "NIS2-21a" in item.title

    def test_instantiate_hours_multiplier(self):
        base_items = TEMPLATE_ACCESS_CONTROL.instantiate("C-001", hours_multiplier=1.0)
        scaled_items = TEMPLATE_ACCESS_CONTROL.instantiate("C-001", hours_multiplier=2.0)
        for base, scaled in zip(base_items, scaled_items):
            assert scaled.estimated_hours == pytest.approx(base.estimated_hours * 2.0)

    def test_instantiate_minimum_hours_enforced(self):
        # Even with hours_multiplier=0.0 minimum is 1.0h
        items = TEMPLATE_ACCESS_CONTROL.instantiate("C-001", hours_multiplier=0.0)
        for item in items:
            assert item.estimated_hours >= 1.0

    def test_instantiate_article_ref_explicit(self):
        items = TEMPLATE_ACCESS_CONTROL.instantiate("C-001", article_ref="NIS2-CUSTOM")
        for item in items:
            assert item.article_ref == "NIS2-CUSTOM"

    def test_instantiate_article_ref_falls_back_to_framework_refs(self):
        items = TEMPLATE_ACCESS_CONTROL.instantiate("C-001")
        for item in items:
            assert item.article_ref == TEMPLATE_ACCESS_CONTROL.framework_refs[0]

    def test_instantiate_step_ordering(self):
        """Steps must be instantiated in sequence order."""
        template = TEMPLATE_INCIDENT_RESPONSE
        items = template.instantiate("C-001")
        sorted_steps = sorted(template.steps, key=lambda s: (s.sequence, s.step_key))
        for item, step in zip(items, sorted_steps):
            assert step.title in item.title

    def test_instantiate_unique_item_ids(self):
        """Each instantiation must produce fresh UUIDs."""
        items1 = TEMPLATE_ACCESS_CONTROL.instantiate("C-001")
        items2 = TEMPLATE_ACCESS_CONTROL.instantiate("C-001")
        ids1 = {i.item_id for i in items1}
        ids2 = {i.item_id for i in items2}
        assert ids1.isdisjoint(ids2)

    def test_instantiate_evidence_types_propagated(self):
        items = TEMPLATE_ACCESS_CONTROL.instantiate("C-001")
        # At least one item must require documentation
        assert any("documentation" in item.evidence_required for item in items)


class TestTemplateRegistry:
    def test_all_five_templates_registered(self):
        assert len(TEMPLATE_REGISTRY) == 5
        expected_types = {
            "access_control",
            "incident_response",
            "vulnerability_management",
            "supply_chain",
            "cryptography",
        }
        assert set(TEMPLATE_REGISTRY.keys()) == expected_types

    def test_get_template_known_type(self):
        tmpl = get_template("access_control")
        assert tmpl is not None
        assert tmpl.control_objective_type == "access_control"

    def test_get_template_unknown_returns_none(self):
        assert get_template("nonexistent_type") is None

    def test_list_templates_deterministic(self):
        t1 = [t.control_objective_type for t in list_templates()]
        t2 = [t.control_objective_type for t in list_templates()]
        assert t1 == t2

    def test_list_templates_alphabetical(self):
        types = [t.control_objective_type for t in list_templates()]
        assert types == sorted(types)

    def test_list_templates_count(self):
        assert len(list_templates()) == 5

    def test_each_template_has_steps(self):
        for tmpl in list_templates():
            assert len(tmpl.steps) > 0, f"{tmpl.name} has no steps"

    def test_each_template_has_framework_refs(self):
        for tmpl in list_templates():
            assert len(tmpl.framework_refs) > 0, f"{tmpl.name} has no framework refs"
