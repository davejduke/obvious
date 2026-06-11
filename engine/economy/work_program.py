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
"""
from __future__ import annotations

import math
from dataclasses import dataclass, field
from typing import Any, Optional
from uuid import UUID, uuid4


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
    """An optimised audit work program."""
    program_id: UUID = field(default_factory=uuid4)
    engagement_id: Optional[UUID] = None
    items: list[WorkItem] = field(default_factory=list)
    total_estimated_hours: float = 0.0
    coverage_score: float = 0.0  # 0.0 – 1.0: fraction of risk weight covered
    metadata: dict[str, Any] = field(default_factory=dict)

    def add_item(self, item: WorkItem) -> None:
        self.items.append(item)
        self.total_estimated_hours += item.estimated_hours


# Hours-per-risk-weight-unit base rate (tunable constant)
_BASE_HOURS_PER_UNIT: float = 4.0
# Minimum hours for any work item regardless of weight
_MIN_ITEM_HOURS: float = 2.0


def _estimate_hours(risk_weight: float) -> float:
    """Deterministic hours estimate from risk weight."""
    return max(_MIN_ITEM_HOURS, math.ceil(risk_weight * _BASE_HOURS_PER_UNIT))


def generate_work_program(
    controls: list[dict[str, Any]],
    budget_hours: float = 160.0,
    engagement_id: Optional[UUID] = None,
) -> WorkProgram:
    """Generate an optimised work program via greedy coverage maximisation.

    Parameters
    ----------
    controls:
        List of dicts with keys: control_id, title, risk_weight (float 0–5),
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
