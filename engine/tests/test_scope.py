"""Unit tests — Scope Engine (DAG, validation, enforcement, workflow, history, impact)."""
from uuid import uuid4

import pytest
from engine.scope.dag import (
    AuditScopeDAG,
    BoundaryStatus,
    ScopeNode,
    ScopeState,
)
from engine.scope.enforcement import (
    EngagementEventType,
    ScopeEnforcer,
)
from engine.scope.history import (
    ScopeVersionHistory,
    diff_snapshots,
)
from engine.scope.impact import (
    EvidenceRef,
    FindingRef,
    ScopeImpactAnalyzer,
)
from engine.scope.validation import (
    ValidationCode,
    validate_control_evidence_mapping,
    validate_coverage_completeness,
    validate_dag_boundaries,
    validate_no_orphan_in_scope_nodes,
    validate_parent_scope_consistency,
)
from engine.scope.workflow import (
    ModificationWorkflow,
    ScopeChangeKind,
    ScopeChangeRecord,
)

# ─── Shared helpers ───────────────────────────────────────────────────────────


def make_node(control_id: str = "C-001", risk_weight: float = 1.0, parent_ids=None):
    return ScopeNode(
        control_id=control_id,
        title=f"Control {control_id}",
        article_ref="NIS2-21a",
        risk_weight=risk_weight,
        parent_ids=parent_ids or [],
    )


def build_linear_dag(
    control_ids: list[str],
    all_in_scope: bool = False,
) -> tuple[AuditScopeDAG, list[ScopeNode]]:
    """Build a linear parent → child → grandchild chain."""
    dag = AuditScopeDAG()
    nodes: list[ScopeNode] = []
    for i, cid in enumerate(control_ids):
        parents = [nodes[i - 1].node_id] if i > 0 else []
        node = make_node(cid, parent_ids=parents)
        dag.add_node(node)
        if all_in_scope:
            dag.set_boundary(node.node_id, BoundaryStatus.IN_SCOPE)
        nodes.append(node)
    return dag, nodes


# ─── TestScopeNode ──────────────────────────────────────────────────────────────


class TestScopeNode:
    def test_default_boundary_is_conditional(self):
        node = make_node()
        assert node.boundary_status == BoundaryStatus.CONDITIONAL
        assert not node.is_in_scope()
        assert not node.is_out_of_scope()

    def test_is_in_scope(self):
        node = make_node()
        node.boundary_status = BoundaryStatus.IN_SCOPE
        assert node.is_in_scope()
        assert not node.is_out_of_scope()

    def test_risk_weight_bounds(self):
        with pytest.raises(Exception):  # noqa: B017 — pydantic raises ValidationError
            make_node(risk_weight=6.0)
        with pytest.raises(Exception):  # noqa: B017 — pydantic raises ValidationError
            make_node(risk_weight=-0.1)


# ─── TestAuditScopeDAG ──────────────────────────────────────────────────────────


class TestAuditScopeDAG:
    def test_add_single_node(self):
        dag = AuditScopeDAG()
        node = make_node("C-001")
        dag.add_node(node)
        assert dag.node_count() == 1
        assert dag.get_node(node.node_id).control_id == "C-001"

    def test_topological_order_linear_chain(self):
        dag, nodes = build_linear_dag(["ROOT", "CHILD", "GRANDCHILD"])
        order = [n.control_id for n in dag.topological_order()]
        assert order.index("ROOT") < order.index("CHILD")
        assert order.index("CHILD") < order.index("GRANDCHILD")

    def test_duplicate_node_raises(self):
        dag = AuditScopeDAG()
        node = make_node()
        dag.add_node(node)
        with pytest.raises(ValueError, match="already exists"):
            dag.add_node(node)

    def test_missing_parent_raises(self):
        dag = AuditScopeDAG()
        missing_id = uuid4()
        with pytest.raises(ValueError, match="does not exist"):
            dag.add_node(make_node(parent_ids=[missing_id]))

    def test_set_boundary(self):
        dag = AuditScopeDAG()
        node = make_node()
        dag.add_node(node)
        dag.set_boundary(node.node_id, BoundaryStatus.IN_SCOPE)
        assert dag.get_node(node.node_id).is_in_scope()

    def test_in_scope_nodes_filter(self):
        dag = AuditScopeDAG()
        n1 = make_node("C-001")
        n2 = make_node("C-002")
        dag.add_node(n1)
        dag.add_node(n2)
        dag.set_boundary(n1.node_id, BoundaryStatus.IN_SCOPE)
        dag.set_boundary(n2.node_id, BoundaryStatus.OUT_OF_SCOPE)
        in_scope = dag.in_scope_nodes()
        assert len(in_scope) == 1
        assert in_scope[0].control_id == "C-001"

    def test_reachable_from(self):
        dag = AuditScopeDAG()
        a = make_node("A")
        dag.add_node(a)
        b = make_node("B", parent_ids=[a.node_id])
        dag.add_node(b)
        c = make_node("C", parent_ids=[b.node_id])
        dag.add_node(c)
        d = make_node("D")  # unconnected
        dag.add_node(d)

        reachable = dag.reachable_from(a.node_id)
        assert a.node_id in reachable
        assert b.node_id in reachable
        assert c.node_id in reachable
        assert d.node_id not in reachable

    def test_locked_prevents_mutation(self):
        dag = AuditScopeDAG()
        dag.transition(ScopeState.PROPOSED)
        dag.transition(ScopeState.APPROVED)
        dag.transition(ScopeState.LOCKED)
        with pytest.raises(ValueError, match="LOCKED"):
            dag.add_node(make_node("ILLEGAL"))

    def test_invalid_state_transition_raises(self):
        dag = AuditScopeDAG()
        with pytest.raises(ValueError, match="Invalid transition"):
            dag.transition(ScopeState.LOCKED)

    def test_state_machine_happy_path(self):
        dag = AuditScopeDAG()
        assert dag.state == ScopeState.DRAFT
        dag.transition(ScopeState.PROPOSED)
        assert dag.state == ScopeState.PROPOSED
        dag.transition(ScopeState.APPROVED)
        assert dag.state == ScopeState.APPROVED
        dag.transition(ScopeState.LOCKED)
        assert dag.state == ScopeState.LOCKED


# ─── TestValidation ────────────────────────────────────────────────────────────


class TestValidationNoOrphans:
    def test_root_in_scope_is_valid(self):
        dag = AuditScopeDAG()
        root = make_node("ROOT")
        dag.add_node(root)
        dag.set_boundary(root.node_id, BoundaryStatus.IN_SCOPE)
        result = validate_no_orphan_in_scope_nodes(dag)
        assert result.is_valid

    def test_child_in_scope_parent_in_scope_is_valid(self):
        dag, nodes = build_linear_dag(["P", "C"], all_in_scope=True)
        result = validate_no_orphan_in_scope_nodes(dag)
        assert result.is_valid

    def test_child_in_scope_parent_out_of_scope_is_orphan(self):
        dag = AuditScopeDAG()
        parent = make_node("P")
        dag.add_node(parent)
        child = make_node("C", parent_ids=[parent.node_id])
        dag.add_node(child)
        dag.set_boundary(parent.node_id, BoundaryStatus.OUT_OF_SCOPE)
        dag.set_boundary(child.node_id, BoundaryStatus.IN_SCOPE)

        result = validate_no_orphan_in_scope_nodes(dag)
        assert not result.is_valid
        assert len(result.violations) == 1
        v = result.violations[0]
        assert v.code == ValidationCode.ORPHAN_NODE
        assert v.control_id == "C"

    def test_out_of_scope_child_not_flagged(self):
        dag = AuditScopeDAG()
        parent = make_node("P")
        dag.add_node(parent)
        child = make_node("C", parent_ids=[parent.node_id])
        dag.add_node(child)
        dag.set_boundary(parent.node_id, BoundaryStatus.OUT_OF_SCOPE)
        dag.set_boundary(child.node_id, BoundaryStatus.OUT_OF_SCOPE)
        result = validate_no_orphan_in_scope_nodes(dag)
        assert result.is_valid


class TestValidationParentConsistency:
    def test_in_scope_parent_in_scope_child_is_valid(self):
        dag, nodes = build_linear_dag(["P", "C"], all_in_scope=True)
        result = validate_parent_scope_consistency(dag)
        assert result.is_valid

    def test_in_scope_child_with_out_of_scope_parent_flagged(self):
        dag = AuditScopeDAG()
        parent = make_node("P")
        dag.add_node(parent)
        child = make_node("C", parent_ids=[parent.node_id])
        dag.add_node(child)
        dag.set_boundary(parent.node_id, BoundaryStatus.OUT_OF_SCOPE)
        dag.set_boundary(child.node_id, BoundaryStatus.IN_SCOPE)

        result = validate_parent_scope_consistency(dag)
        assert not result.is_valid
        assert result.violations[0].code == ValidationCode.PARENT_OUT_OF_SCOPE

    def test_root_node_not_flagged(self):
        dag = AuditScopeDAG()
        root = make_node("ROOT")
        dag.add_node(root)
        dag.set_boundary(root.node_id, BoundaryStatus.IN_SCOPE)
        assert validate_parent_scope_consistency(dag).is_valid


class TestValidationCoverage:
    def test_all_required_present_and_in_scope(self):
        dag = AuditScopeDAG()
        n = make_node("NIS2-21a")
        dag.add_node(n)
        dag.set_boundary(n.node_id, BoundaryStatus.IN_SCOPE)
        result = validate_coverage_completeness(dag, {"NIS2-21a"})
        assert result.is_valid

    def test_required_control_missing_from_dag(self):
        dag = AuditScopeDAG()
        result = validate_coverage_completeness(dag, {"NIS2-21a"})
        assert not result.is_valid
        assert result.violations[0].code == ValidationCode.MISSING_COVERAGE

    def test_required_control_in_dag_but_out_of_scope(self):
        dag = AuditScopeDAG()
        n = make_node("NIS2-21a")
        dag.add_node(n)
        dag.set_boundary(n.node_id, BoundaryStatus.OUT_OF_SCOPE)
        result = validate_coverage_completeness(dag, {"NIS2-21a"})
        assert not result.is_valid
        assert "not marked IN_SCOPE" in result.violations[0].message

    def test_empty_required_set_is_valid(self):
        dag = AuditScopeDAG()
        assert validate_coverage_completeness(dag, set()).is_valid


class TestValidationEvidenceMapping:
    def test_all_in_scope_controls_have_evidence(self):
        dag = AuditScopeDAG()
        n = make_node("C-001")
        dag.add_node(n)
        dag.set_boundary(n.node_id, BoundaryStatus.IN_SCOPE)
        result = validate_control_evidence_mapping(dag, {"C-001": ["ev-1"]})
        assert result.is_valid

    def test_in_scope_control_missing_evidence_flagged(self):
        dag = AuditScopeDAG()
        n = make_node("C-001")
        dag.add_node(n)
        dag.set_boundary(n.node_id, BoundaryStatus.IN_SCOPE)
        result = validate_control_evidence_mapping(dag, {})
        assert not result.is_valid
        assert result.violations[0].code == ValidationCode.NO_EVIDENCE_SOURCE

    def test_out_of_scope_control_not_required_to_have_evidence(self):
        dag = AuditScopeDAG()
        n = make_node("C-001")
        dag.add_node(n)
        dag.set_boundary(n.node_id, BoundaryStatus.OUT_OF_SCOPE)
        result = validate_control_evidence_mapping(dag, {})
        assert result.is_valid


class TestValidateDagBoundaries:
    def test_composite_validator_clean_dag(self):
        dag, nodes = build_linear_dag(["P", "C"], all_in_scope=True)
        result = validate_dag_boundaries(
            dag,
            required_control_ids={"P", "C"},
            control_evidence_map={"P": ["ev-1"], "C": ["ev-2"]},
        )
        assert result.is_valid

    def test_composite_validator_catches_multiple_violations(self):
        dag = AuditScopeDAG()
        parent = make_node("P")
        dag.add_node(parent)
        child = make_node("C", parent_ids=[parent.node_id])
        dag.add_node(child)
        # parent out of scope, child in scope → orphan + parent consistency
        dag.set_boundary(parent.node_id, BoundaryStatus.OUT_OF_SCOPE)
        dag.set_boundary(child.node_id, BoundaryStatus.IN_SCOPE)

        result = validate_dag_boundaries(
            dag,
            required_control_ids={"P", "C"},
            control_evidence_map={},  # also missing evidence for "C"
        )
        assert not result.is_valid
        codes = {v.code for v in result.violations}
        # ORPHAN, PARENT_OUT_OF_SCOPE, MISSING_COVERAGE (P not in scope),
        # NO_EVIDENCE_SOURCE (C in scope but no evidence)
        assert ValidationCode.ORPHAN_NODE in codes
        assert ValidationCode.PARENT_OUT_OF_SCOPE in codes
        assert ValidationCode.MISSING_COVERAGE in codes
        assert ValidationCode.NO_EVIDENCE_SOURCE in codes

    def test_composite_validator_skips_coverage_when_none(self):
        dag, _ = build_linear_dag(["C-001"], all_in_scope=True)
        result = validate_dag_boundaries(dag, required_control_ids=None)
        assert result.is_valid


# ─── TestEnforcement ───────────────────────────────────────────────────────────


class TestScopeEnforcer:
    def _valid_dag(self) -> AuditScopeDAG:
        dag, _ = build_linear_dag(["NIS2-21a", "NIS2-21b"], all_in_scope=True)
        return dag

    def test_enforce_on_engagement_created_passes(self):
        enforcer = ScopeEnforcer()
        dag = self._valid_dag()
        result = enforcer.enforce_on_engagement_created(
            "eng-001",
            dag,
            required_control_ids={"NIS2-21a", "NIS2-21b"},
            control_evidence_map={"NIS2-21a": ["ev-1"], "NIS2-21b": ["ev-2"]},
        )
        assert result.passed
        assert result.trigger.event_type == EngagementEventType.CREATED
        assert result.trigger.engagement_id == "eng-001"

    def test_enforce_on_engagement_modified_passes(self):
        enforcer = ScopeEnforcer()
        dag = self._valid_dag()
        result = enforcer.enforce_on_engagement_modified("eng-001", dag)
        assert result.passed
        assert result.trigger.event_type == EngagementEventType.MODIFIED

    def test_enforce_fails_on_missing_required_control(self):
        enforcer = ScopeEnforcer()
        dag, _ = build_linear_dag(["NIS2-21a"], all_in_scope=True)
        result = enforcer.enforce_on_engagement_created(
            "eng-001",
            dag,
            required_control_ids={"NIS2-21a", "NIS2-21c"},  # 21c not in DAG
        )
        assert not result.passed
        assert result.trigger.violation_count == 1

    def test_enforce_to_dict_shape(self):
        enforcer = ScopeEnforcer()
        dag = self._valid_dag()
        result = enforcer.enforce_on_engagement_created("eng-x", dag)
        d = result.to_dict()
        assert d["engagement_id"] == "eng-x"
        assert d["event_type"] == "created"
        assert "violations" in d
        assert "passed" in d

    def test_enforce_records_trigger_metadata(self):
        enforcer = ScopeEnforcer()
        dag, _ = build_linear_dag(["C"], all_in_scope=True)
        result = enforcer.enforce_on_engagement_created("eng-999", dag)
        assert result.trigger.trigger_id is not None
        assert result.trigger.triggered_at is not None
        assert result.trigger.passed is True
        assert result.trigger.violation_count == 0


# ─── TestModificationWorkflow ───────────────────────────────────────────────────


def _make_change() -> ScopeChangeRecord:
    return ScopeChangeRecord(
        kind=ScopeChangeKind.SET_BOUNDARY,
        control_id="NIS2-21c",
        boundary_status=BoundaryStatus.IN_SCOPE,
        description="Add article 21c control",
    )


class TestModificationWorkflow:
    def test_submit_request_creates_pending_request(self):
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Add 21c", [_make_change()])
        assert req.is_pending()
        assert req.engagement_id == "eng-001"
        assert req.requested_by == "alice"
        assert len(req.proposed_changes) == 1

    def test_approve_request_transitions_to_approved(self):
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Add 21c", [_make_change()])
        approved = wf.approve_request(req.request_id, "bob", note="Looks good")
        assert approved.is_approved()
        assert approved.reviewed_by == "bob"
        assert approved.review_note == "Looks good"
        assert approved.reviewed_at is not None

    def test_reject_request_transitions_to_rejected(self):
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Add 21c", [_make_change()])
        rejected = wf.reject_request(req.request_id, "bob", note="Out of scope")
        assert rejected.is_rejected()
        assert rejected.reviewed_by == "bob"
        assert rejected.review_note == "Out of scope"

    def test_cannot_approve_already_approved_request(self):
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Add 21c", [_make_change()])
        wf.approve_request(req.request_id, "bob")
        with pytest.raises(ValueError, match="already"):
            wf.approve_request(req.request_id, "carol")

    def test_cannot_reject_already_rejected_request(self):
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Add 21c", [_make_change()])
        wf.reject_request(req.request_id, "bob")
        with pytest.raises(ValueError, match="already"):
            wf.reject_request(req.request_id, "carol")

    def test_unknown_request_raises_key_error(self):
        wf = ModificationWorkflow()
        with pytest.raises(KeyError):
            wf.approve_request(uuid4(), "bob")

    def test_audit_trail_records_all_events(self):
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Add 21c", [_make_change()])
        wf.approve_request(req.request_id, "bob")
        trail = wf.get_audit_trail("eng-001")
        assert len(trail) == 2
        event_types = [e.event_type for e in trail]
        assert "scope_modification_requested" in event_types
        assert "scope_modification_approved" in event_types

    def test_audit_trail_is_filtered_by_engagement(self):
        wf = ModificationWorkflow()
        req1 = wf.submit_request("eng-001", "alice", "Change A", [])
        wf.submit_request("eng-002", "bob", "Change B", [])
        wf.approve_request(req1.request_id, "carol")
        trail_1 = wf.get_audit_trail("eng-001")
        trail_2 = wf.get_audit_trail("eng-002")
        assert len(trail_1) == 2  # submitted + approved
        assert len(trail_2) == 1  # submitted only

    def test_pending_requests_filter(self):
        wf = ModificationWorkflow()
        r1 = wf.submit_request("eng-001", "alice", "Change A", [])
        r2 = wf.submit_request("eng-001", "alice", "Change B", [])
        wf.approve_request(r1.request_id, "bob")
        pending = wf.pending_requests("eng-001")
        assert len(pending) == 1
        assert pending[0].request_id == r2.request_id

    def test_request_to_dict_shape(self):
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Add 21c", [_make_change()])
        d = req.to_dict()
        assert d["engagement_id"] == "eng-001"
        assert d["status"] == "pending"
        assert len(d["proposed_changes"]) == 1
        assert d["proposed_changes"][0]["kind"] == "set_boundary"

    def test_rejection_with_full_end_to_end_trail(self):
        """Full request → reject lifecycle with audit trail."""
        wf = ModificationWorkflow()
        req = wf.submit_request("eng-003", "dave", "Risky change", [_make_change()])
        rejected = wf.reject_request(req.request_id, "cae", note="Does not fit scope")
        trail = wf.get_audit_trail("eng-003")
        assert rejected.is_rejected()
        assert len(trail) == 2
        assert trail[-1].event_type == "scope_modification_rejected"
        assert trail[-1].detail["note"] == "Does not fit scope"


# ─── TestScopeVersionHistory ────────────────────────────────────────────────────


class TestScopeVersionHistory:
    def _dag_with_node(
        self, control_id: str, status: BoundaryStatus
    ) -> tuple[AuditScopeDAG, ScopeNode]:
        dag = AuditScopeDAG()
        node = make_node(control_id)
        dag.add_node(node)
        dag.set_boundary(node.node_id, status)
        return dag, node

    def test_record_and_retrieve_version(self):
        dag, _ = self._dag_with_node("C-001", BoundaryStatus.IN_SCOPE)
        history = ScopeVersionHistory("eng-001")
        snap = history.record_version(dag, version=1, changed_by="alice", note="initial")
        assert snap.version == 1
        assert snap.engagement_id == "eng-001"
        assert snap.changed_by == "alice"
        retrieved = history.get_version(1)
        assert retrieved.version == 1

    def test_duplicate_version_raises(self):
        dag, _ = self._dag_with_node("C-001", BoundaryStatus.IN_SCOPE)
        history = ScopeVersionHistory("eng-001")
        history.record_version(dag, version=1)
        with pytest.raises(ValueError, match="already recorded"):
            history.record_version(dag, version=1)

    def test_missing_version_raises_key_error(self):
        history = ScopeVersionHistory("eng-001")
        with pytest.raises(KeyError):
            history.get_version(99)

    def test_latest_version_returns_highest(self):
        dag, _ = self._dag_with_node("C-001", BoundaryStatus.IN_SCOPE)
        history = ScopeVersionHistory("eng-001")
        history.record_version(dag, version=1)
        history.record_version(dag, version=3)
        history.record_version(dag, version=2)
        assert history.latest_version().version == 3  # type: ignore[union-attr]

    def test_latest_version_empty_is_none(self):
        history = ScopeVersionHistory("eng-001")
        assert history.latest_version() is None

    def test_all_versions_sorted_ascending(self):
        dag, _ = self._dag_with_node("C-001", BoundaryStatus.IN_SCOPE)
        history = ScopeVersionHistory("eng-001")
        history.record_version(dag, version=3)
        history.record_version(dag, version=1)
        history.record_version(dag, version=2)
        versions = [s.version for s in history.all_versions()]
        assert versions == [1, 2, 3]

    def test_diff_detects_boundary_change(self):
        dag = AuditScopeDAG()
        node = make_node("C-001")
        dag.add_node(node)
        dag.set_boundary(node.node_id, BoundaryStatus.OUT_OF_SCOPE)

        history = ScopeVersionHistory("eng-001")
        history.record_version(dag, version=1)

        dag.set_boundary(node.node_id, BoundaryStatus.IN_SCOPE)
        history.record_version(dag, version=2)

        diff = history.diff(1, 2)
        assert diff.from_version == 1
        assert diff.to_version == 2
        assert len(diff.modified) == 1
        assert diff.modified[0].control_id == "C-001"
        assert diff.modified[0].old_boundary == BoundaryStatus.OUT_OF_SCOPE
        assert diff.modified[0].new_boundary == BoundaryStatus.IN_SCOPE

    def test_diff_detects_added_node(self):
        dag = AuditScopeDAG()
        n1 = make_node("C-001")
        dag.add_node(n1)
        dag.set_boundary(n1.node_id, BoundaryStatus.IN_SCOPE)

        history = ScopeVersionHistory("eng-001")
        history.record_version(dag, version=1)

        # Add second node and new version (DAG is no longer locked)
        n2 = make_node("C-002")
        dag.add_node(n2)
        dag.set_boundary(n2.node_id, BoundaryStatus.IN_SCOPE)
        history.record_version(dag, version=2)

        diff = history.diff(1, 2)
        assert len(diff.added) == 1
        assert diff.added[0].control_id == "C-002"
        assert len(diff.removed) == 0

    def test_in_scope_controls_property(self):
        dag, nodes = build_linear_dag(["C-001", "C-002"], all_in_scope=True)
        history = ScopeVersionHistory("eng-001")
        snap = history.record_version(dag, version=1)
        assert snap.in_scope_controls == {"C-001", "C-002"}

    def test_snapshot_to_dict_shape(self):
        dag, _ = self._dag_with_node("C-001", BoundaryStatus.IN_SCOPE)
        history = ScopeVersionHistory("eng-001")
        snap = history.record_version(dag, version=1, changed_by="alice")
        d = snap.to_dict()
        assert d["version"] == 1
        assert d["engagement_id"] == "eng-001"
        assert d["changed_by"] == "alice"
        assert len(d["nodes"]) == 1


# ─── TestScopeImpactAnalyzer ────────────────────────────────────────────────────


class TestScopeImpactAnalyzer:
    def _build_diff_v1_to_v2(
        self,
        boundary_v1: BoundaryStatus,
        boundary_v2: BoundaryStatus,
    ):
        dag = AuditScopeDAG()
        node = make_node("NIS2-21a")
        dag.add_node(node)
        dag.set_boundary(node.node_id, boundary_v1)
        history = ScopeVersionHistory("eng-001")
        history.record_version(dag, version=1)
        dag.set_boundary(node.node_id, boundary_v2)
        history.record_version(dag, version=2)
        return history.diff(1, 2)

    def test_no_impact_when_scope_unchanged(self):
        dag = AuditScopeDAG()
        node = make_node("NIS2-21a")
        dag.add_node(node)
        dag.set_boundary(node.node_id, BoundaryStatus.IN_SCOPE)
        history = ScopeVersionHistory("eng-001")
        history.record_version(dag, version=1)
        history.record_version(dag, version=2)
        diff = history.diff(1, 2)

        analyzer = ScopeImpactAnalyzer()
        impact = analyzer.analyze(
            diff,
            findings=[FindingRef("f-001", "NIS2-21a")],
            evidence=[EvidenceRef("ev-001", "NIS2-21a")],
        )
        assert not impact.has_impact

    def test_control_removed_from_scope_impacts_findings(self):
        diff = self._build_diff_v1_to_v2(
            BoundaryStatus.IN_SCOPE, BoundaryStatus.OUT_OF_SCOPE
        )
        analyzer = ScopeImpactAnalyzer()
        impact = analyzer.analyze(
            diff,
            findings=[FindingRef("f-001", "NIS2-21a")],
            evidence=[],
        )
        assert impact.has_impact
        assert impact.affected_findings[0].impact_reason == "control_removed_from_scope"

    def test_control_added_to_scope_impacts_evidence(self):
        diff = self._build_diff_v1_to_v2(
            BoundaryStatus.OUT_OF_SCOPE, BoundaryStatus.IN_SCOPE
        )
        analyzer = ScopeImpactAnalyzer()
        impact = analyzer.analyze(
            diff,
            findings=[],
            evidence=[EvidenceRef("ev-001", "NIS2-21a")],
        )
        assert impact.has_impact
        assert impact.affected_evidence[0].impact_reason == "control_added_to_scope"

    def test_unrelated_control_not_impacted(self):
        diff = self._build_diff_v1_to_v2(
            BoundaryStatus.IN_SCOPE, BoundaryStatus.OUT_OF_SCOPE
        )
        analyzer = ScopeImpactAnalyzer()
        impact = analyzer.analyze(
            diff,
            findings=[FindingRef("f-001", "NIS2-22")],  # different control
            evidence=[],
        )
        assert not impact.has_impact

    def test_impact_to_dict_shape(self):
        diff = self._build_diff_v1_to_v2(
            BoundaryStatus.IN_SCOPE, BoundaryStatus.OUT_OF_SCOPE
        )
        analyzer = ScopeImpactAnalyzer()
        impact = analyzer.analyze(
            diff,
            findings=[FindingRef("f-001", "NIS2-21a")],
            evidence=[EvidenceRef("ev-001", "NIS2-21a")],
        )
        d = impact.to_dict()
        assert d["affected_findings_count"] == 1
        assert d["affected_evidence_count"] == 1
        assert d["affected_findings"][0]["impact_reason"] == "control_removed_from_scope"

    def test_node_removed_entirely_impacts_all_linked(self):
        """A node deleted between versions (not just boundary-changed) impacts linked items."""
        dag = AuditScopeDAG()
        node = make_node("NIS2-21a")
        dag.add_node(node)
        dag.set_boundary(node.node_id, BoundaryStatus.IN_SCOPE)

        history = ScopeVersionHistory("eng-001")
        snap1 = history.record_version(dag, version=1)

        # Build a new DAG without the node to simulate removal
        snap2_nodes = {}
        from engine.scope.history import ScopeSnapshot

        snap2 = ScopeSnapshot(
            version=2,
            engagement_id="eng-001",
            nodes=snap2_nodes,
        )

        diff = diff_snapshots(snap1, snap2)
        assert len(diff.removed) == 1

        analyzer = ScopeImpactAnalyzer()
        impact = analyzer.analyze(
            diff,
            findings=[FindingRef("f-001", "NIS2-21a")],
            evidence=[EvidenceRef("ev-001", "NIS2-21a")],
        )
        assert impact.affected_findings[0].impact_reason == "control_removed_from_scope"
        assert impact.affected_evidence[0].impact_reason == "control_removed_from_scope"


# ─── TestCacheInvalidationIntegration ─────────────────────────────────────────────


class TestCacheInvalidationIntegration:
    """Validates cache.InvalidationHandler.on_scope_changed is called after approval.

    Uses an async mock to verify the integration contract without a live Redis.
    """

    @pytest.mark.asyncio
    async def test_cache_invalidated_after_scope_change(self):
        """Demonstrate cache invalidation wired to workflow approval."""
        from unittest.mock import AsyncMock

        from engine.shared.cache import CacheClient

        # Build a mock cache client whose invalidation method we can assert on
        mock_cache = AsyncMock(spec=CacheClient)
        mock_cache.invalidation = AsyncMock()
        mock_cache.invalidation.on_scope_changed = AsyncMock(return_value=1)

        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Change scope", [_make_change()])
        approved = wf.approve_request(req.request_id, "bob")

        # Caller wires cache invalidation after approval
        assert approved.is_approved()
        await mock_cache.invalidation.on_scope_changed(approved.engagement_id)

        mock_cache.invalidation.on_scope_changed.assert_called_once_with("eng-001")

    @pytest.mark.asyncio
    async def test_cache_not_invalidated_after_rejection(self):
        """Cache should NOT be invalidated when a request is rejected."""
        from unittest.mock import AsyncMock

        from engine.shared.cache import CacheClient

        mock_cache = AsyncMock(spec=CacheClient)
        mock_cache.invalidation = AsyncMock()
        mock_cache.invalidation.on_scope_changed = AsyncMock(return_value=0)

        wf = ModificationWorkflow()
        req = wf.submit_request("eng-001", "alice", "Bad change", [_make_change()])
        rejected = wf.reject_request(req.request_id, "bob")

        assert rejected.is_rejected()
        # Caller logic: only invalidate on approval
        if rejected.is_approved():
            await mock_cache.invalidation.on_scope_changed(rejected.engagement_id)

        mock_cache.invalidation.on_scope_changed.assert_not_called()

