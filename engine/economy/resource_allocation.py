"""Economy Engine — Risk-weighted resource allocation across engagement teams.

All allocation decisions are deterministic: no LLM is consulted.

Algorithm
---------
Greedy risk-weighted allocation:
  1. Sort work items by risk_weight descending (highest-risk first).
  2. For each item, prefer auditors whose specializations overlap with the
     item's evidence_required set and have sufficient remaining capacity.
  3. If no specialist is available, fall back to any auditor with capacity
     (picks the one with the most remaining hours).
  4. Assign estimated_hours to the chosen auditor and record the assignment.
  5. Items that cannot be absorbed by any auditor are reported as unassigned.

Cost estimation
---------------
Total estimated cost = sum(item.estimated_hours * auditor.hourly_cost) for each
assigned (item, auditor) pair.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Optional
from uuid import UUID, uuid4

from engine.economy.work_program import WorkItem


@dataclass
class AuditorResource:
    """A single auditor with capacity, cost rate, and specializations."""
    auditor_id: UUID = field(default_factory=uuid4)
    name: str = ""
    available_hours: float = 160.0   # standard engagement capacity (hours)
    hourly_cost: float = 150.0       # USD per billable hour
    specializations: frozenset[str] = field(default_factory=frozenset)

    # Mutable tracking fields
    allocated_hours: float = 0.0
    assigned_item_ids: list[UUID] = field(default_factory=list)

    @property
    def remaining_hours(self) -> float:
        """Hours still available for assignment."""
        return max(0.0, self.available_hours - self.allocated_hours)

    def can_absorb(self, hours: float) -> bool:
        """True if this auditor has enough remaining capacity."""
        return self.remaining_hours >= hours

    def assign(self, item: WorkItem) -> None:
        """Assign a work item to this auditor (mutates state)."""
        self.allocated_hours += item.estimated_hours
        self.assigned_item_ids.append(item.item_id)

    def utilization(self) -> float:
        """Fraction of available hours that are allocated (0.0-1.0, capped)."""
        if self.available_hours <= 0:
            return 0.0
        return min(1.0, self.allocated_hours / self.available_hours)


@dataclass
class AllocationResult:
    """Result of resource allocation across a work program."""
    assignments: dict[UUID, UUID] = field(default_factory=dict)  # item_id -> auditor_id
    unassigned_item_ids: list[UUID] = field(default_factory=list)
    auditors: list[AuditorResource] = field(default_factory=list)
    total_estimated_cost: float = 0.0

    def auditor_utilization(self) -> dict[str, float]:
        """Map auditor name (or id str) -> utilization ratio."""
        return {
            (a.name or str(a.auditor_id)): a.utilization()
            for a in self.auditors
        }


def allocate_resources(
    items: list[WorkItem],
    auditors: list[AuditorResource],
) -> AllocationResult:
    """Risk-weighted greedy allocation of work items to auditors.

    Items with higher risk_weight are assigned first.  Within each item,
    auditors whose specializations overlap with evidence_required are
    preferred over generalists.  Among equally eligible auditors, the one
    with the most remaining capacity wins (deterministic tie-breaking).

    Parameters
    ----------
    items:
        Work items to assign (not mutated).
    auditors:
        Available auditor resources.  ``AuditorResource`` state is mutated
        in place; clone inputs if the caller needs the original state.

    Returns
    -------
    AllocationResult with per-item auditor assignments, cost estimate, and
    any items that could not be assigned due to insufficient capacity.
    """
    if not items or not auditors:
        return AllocationResult(auditors=list(auditors))

    # Sort items: highest risk_weight first, then lexicographic for ties
    sorted_items = sorted(items, key=lambda i: (-i.risk_weight, i.control_id))

    result = AllocationResult(auditors=list(auditors))

    for item in sorted_items:
        evidence_needs = set(item.evidence_required)
        best: Optional[AuditorResource] = None

        # Pass 1: prefer specialists with capacity
        for aud in result.auditors:
            if not aud.can_absorb(item.estimated_hours):
                continue
            if aud.specializations & evidence_needs:
                if best is None or aud.remaining_hours > best.remaining_hours:
                    best = aud

        # Pass 2: fall back to any auditor with capacity
        if best is None:
            for aud in result.auditors:
                if aud.can_absorb(item.estimated_hours):
                    if best is None or aud.remaining_hours > best.remaining_hours:
                        best = aud

        if best is not None:
            best.assign(item)
            result.assignments[item.item_id] = best.auditor_id
            result.total_estimated_cost += item.estimated_hours * best.hourly_cost
        else:
            result.unassigned_item_ids.append(item.item_id)

    return result
