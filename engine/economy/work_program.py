"""Economy Engine — Work program generation and coverage maximisation.

Generates an optimised audit work program given a set of in-scope controls,
available auditor hours, and evidence collection costs.  All decisions are
deterministic: no LLM is consulted.

Algorithm
---------
Coverage maximisation is a variant of the 0/1 fractional knapsack:
  - Each in-scope control has a risk_weight (priority) and a cost (hours).
  - We maximise sum(risk_weight) subject to sum(cost) <= budget.
  - Tie-breaking: lexicographic on control_id ensures reproducibility.

Work steps are generated per control with estimated hours derived from
risk_weight and evidence tier requirements.

Scope DAG integration
---------------------
``generate_work_program_from_dag`` accepts an ``AuditScopeDAG`` (APPROVED or
LOCKED) and extracts in-scope nodes in topological order, then runs the same
greedy optimiser.  This guarantees parent controls are always processed before
their children.

Progress tracking
-----------------
``WorkProgram.record_progress`` records completion percentage (0.0-1.0) and
a ``WorkItemStatus`` for each step.  ``compute_overall_progress`` weights
completion by estimated hours so larger steps contribute proportionally.
"""
from __future__ import annotations

import math
from dataclasses import dataclass, field
from enum import Enum
from typing import TYPE_CHECKING, Any, Optional
from uuid import UUID, uuid4

if TYPE_CHECKING:
    from engine.scope.dag import AuditScopeDAG


class WorkItemStatus(str, Enum):
    """Lifecycle status of a single work program step."""
    PENDING = "pending"
    IN_PROGRESS = "in_progress"
    COMPLETE = "complete"
    BLOCKED = "blocked"


@dataclass(frozen=True)
class WorkItem:
    """A single auditable work step."""
    item_id: UUID
    control_id: str
    title: str
    description: str
    estimated_hours: float
    risk_weight: float
    evidence_required: list[str]
    article_ref: Optional[str] = None

    def __lt__(self, other: "WorkItem") -> bool:
        # Stable sort: higher risk_weight first, then lexicographic control_id
        if self.risk_weight != other.risk_weight:
            return self.risk_weight > other.risk_weight
        return self.control_id < other.control_id


@dataclass
class WorkProgram:
    """An optimised audit work program with integrated progress tracking."""
    program_id: UUID = field(default_factory=uuid4)
    engagement_id: Optional[UUID] = None
    items: list[WorkItem] = field(default_factory=list)
    total_estimated_hours: float = 0.0
    coverage_score: float = 0.0  # 0.0 - 1.0: fraction of risk weight covered
    metadata: dict[str, Any] = field(default_factory=dict)
    # Progress tracking: item_id -> (pct_complete: 0.0-1.0, WorkItemStatus)
    _progress: dict[UUID, tuple[float, WorkItemStatus]] = field(
        default_factory=dict, repr=False
    )

    def add_item(self, item: WorkItem) -> None:
        self.items.append(item)
        self.total_estimated_hours += item.estimated_hours

    # ------------------------------------------------------------------
    # Progress tracking
    # ------------------------------------------------------------------

    def record_progress(
        self,
        item_id: UUID,
        pct_complete: float,
        status: WorkItemStatus = WorkItemStatus.IN_PROGRESS,
    ) -> None:
        """Record progress on a work item (0.0 = not started, 1.0 = done).

        Setting pct_complete = 1.0 automatically promotes status to COMPLETE.
        Setting pct_complete = 0.0 with no explicit status keeps PENDING.
        """
        if item_id not in {i.item_id for i in self.items}:
            raise KeyError(f"WorkItem {item_id} not found in this WorkProgram")
        pct = max(0.0, min(1.0, pct_complete))
        if pct >= 1.0:
            status = WorkItemStatus.COMPLETE
        elif pct == 0.0 and status == WorkItemStatus.IN_PROGRESS:
            status = WorkItemStatus.PENDING
        self._progress[item_id] = (pct, status)

    def get_item_progress(self, item_id: UUID) -> tuple[float, WorkItemStatus]:
        """Return (pct_complete, status) for a work item (defaults: 0.0, PENDING)."""
        return self._progress.get(item_id, (0.0, WorkItemStatus.PENDING))

    def compute_overall_progress(self) -> float:
        """Compute overall program completion as a fraction (0.0-1.0).

        Weighted by estimated_hours so larger items contribute proportionally.
        """
        if not self.items:
            return 0.0
        total_weight = sum(i.estimated_hours for i in self.items)
        if total_weight <= 0:
            return 0.0
        completed_weight = sum(
            self.get_item_progress(item.item_id)[0] * item.estimated_hours
            for item in self.items
        )
        return completed_weight / total_weight

    def items_by_status(self, status: WorkItemStatus) -> list[WorkItem]:
        """Return all items with the given status."""
        return [
            item
            for item in self.items
            if self.get_item_progress(item.item_id)[1] == status
        ]


# ---------------------------------------------------------------------------
# Hour estimation constants
# ---------------------------------------------------------------------------

# Hours-per-risk-weight-unit base rate (tunable constant)
_BASE_HOURS_PER_UNIT: float = 4.0
# Minimum hours for any work item regardless of weight
_MIN_ITEM_HOURS: float = 2.0


def _estimate_hours(risk_weight: float) -> float:
    """Deterministic hours estimate from risk weight."""
    return max(_MIN_ITEM_HOURS, math.ceil(risk_weight * _BASE_HOURS_PER_UNIT))


# ---------------------------------------------------------------------------
# Core generation functions
# ---------------------------------------------------------------------------

def generate_work_program(
    controls: list[dict[str, Any]],
    budget_hours: float = 160.0,
    engagement_id: Optional[UUID] = None,
) -> WorkProgram:
    """Generate an optimised work program via greedy coverage maximisation.

    Parameters
    ----------
    controls:
        List of dicts with keys: control_id, title, risk_weight (float 0-5),
        article_ref (optional), description (optional).
    budget_hours:
        Total available auditor hours.
    engagement_id:
        Optional engagement linkage.

    Returns
    -------
    WorkProgram with items sorted highest-risk-weight first, total coverage
    score computed as sum(included risk_weight) / sum(all risk_weight).
    """
    if not controls:
        return WorkProgram(engagement_id=engagement_id)

    # Build candidate work items
    candidates: list[tuple[float, WorkItem]] = []
    total_risk = 0.0
    for ctrl in controls:
        rw = float(ctrl.get("risk_weight", 1.0))
        hrs = _estimate_hours(rw)
        item = WorkItem(
            item_id=uuid4(),
            control_id=ctrl["control_id"],
            title=ctrl.get("title", ctrl["control_id"]),
            description=ctrl.get("description", f"Audit {ctrl['control_id']}"),
            estimated_hours=hrs,
            risk_weight=rw,
            evidence_required=ctrl.get("evidence_required", ["documentation"]),
            article_ref=ctrl.get("article_ref"),
        )
        # efficiency ratio = risk_weight / hours (greedy key)
        efficiency = rw / hrs if hrs > 0 else 0.0
        candidates.append((efficiency, item))
        total_risk += rw

    # Sort by efficiency descending, then by control_id ascending for ties
    candidates.sort(key=lambda x: (-x[0], x[1].control_id))

    program = WorkProgram(engagement_id=engagement_id)
    covered_risk = 0.0

    for _efficiency, item in candidates:
        if program.total_estimated_hours + item.estimated_hours <= budget_hours:
            program.add_item(item)
            covered_risk += item.risk_weight

    program.coverage_score = covered_risk / total_risk if total_risk > 0 else 0.0
    # Sort final items for consistent ordering
    program.items.sort()
    return program


def generate_work_program_from_dag(
    dag: "AuditScopeDAG",
    budget_hours: float = 160.0,
    engagement_id: Optional[UUID] = None,
) -> WorkProgram:
    """Auto-generate a work program from an AuditScopeDAG.

    Only IN_SCOPE nodes contribute.  Nodes are visited in topological order
    so parent controls are always processed before their dependents.

    Parameters
    ----------
    dag:
        The audit scope DAG.  Must be in APPROVED or LOCKED state.
    budget_hours:
        Total available auditor hours.
    engagement_id:
        Optional engagement linkage.

    Returns
    -------
    WorkProgram generated from in-scope nodes, respecting topological order.

    Raises
    ------
    ValueError
        If the DAG is not in APPROVED or LOCKED state.
    """
    from engine.scope.dag import ScopeState

    if dag.state not in (ScopeState.APPROVED, ScopeState.LOCKED):
        raise ValueError(
            f"Scope DAG must be APPROVED or LOCKED to generate a work program "
            f"(current state: {dag.state!r})"
        )

    # Extract in-scope nodes in topological order
    topo_nodes = dag.topological_order()
    in_scope = [n for n in topo_nodes if n.is_in_scope()]

    if not in_scope:
        return WorkProgram(engagement_id=engagement_id)

    controls = [
        {
            "control_id": node.control_id,
            "title": node.title,
            "risk_weight": node.risk_weight,
            "article_ref": node.article_ref,
            "description": node.metadata.get("description", f"Audit {node.control_id}"),
            "evidence_required": node.metadata.get("evidence_required", ["documentation"]),
        }
        for node in in_scope
    ]

    return generate_work_program(
        controls, budget_hours=budget_hours, engagement_id=engagement_id
    )


def compute_coverage_gap(
    work_program: WorkProgram,
    all_controls: list[dict[str, Any]],
) -> list[str]:
    """Return control_ids excluded from the program due to budget constraints."""
    included_ids = {item.control_id for item in work_program.items}
    return [
        ctrl["control_id"]
        for ctrl in all_controls
        if ctrl["control_id"] not in included_ids
    ]
