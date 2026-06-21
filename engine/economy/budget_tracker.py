"""Economy Engine — Budget tracking: planned vs actual hours, cost estimation,
variance reporting, and alert thresholds.

All tracking is deterministic: no LLM is consulted.

Alert thresholds
----------------
Four escalating levels are triggered by the ratio (actual / planned) hours:
  - APPROACHING : >= 50 % consumed
  - WARNING     : >= 75 % consumed
  - CRITICAL    : >= 90 % consumed
  - EXCEEDED    : >= 100 % consumed (over budget)

These thresholds apply both per work-item (BudgetEntry.alert_level) and in
aggregate across the entire engagement (BudgetTracker.overall_alert).
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Optional
from uuid import UUID


class AlertLevel(str, Enum):
    """Budget consumption alert threshold."""
    NONE = "none"            # < 50 % consumed
    APPROACHING = "approaching"  # >= 50 %
    WARNING = "warning"          # >= 75 %
    CRITICAL = "critical"        # >= 90 %
    EXCEEDED = "exceeded"        # >= 100 %


# Ordered from highest to lowest for threshold matching
_THRESHOLDS: list[tuple[float, AlertLevel]] = [
    (1.00, AlertLevel.EXCEEDED),
    (0.90, AlertLevel.CRITICAL),
    (0.75, AlertLevel.WARNING),
    (0.50, AlertLevel.APPROACHING),
]


def _alert_level(utilization: float) -> AlertLevel:
    """Return the AlertLevel for a given utilization ratio."""
    for threshold, level in _THRESHOLDS:
        if utilization >= threshold:
            return level
    return AlertLevel.NONE


@dataclass
class BudgetEntry:
    """A single budget line-item for one work item."""
    item_id: UUID
    control_id: str
    planned_hours: float
    hourly_rate: float = 150.0   # USD per hour
    actual_hours: float = 0.0

    # ------------------------------------------------------------------
    # Derived metrics
    # ------------------------------------------------------------------

    @property
    def planned_cost(self) -> float:
        """Planned cost in USD."""
        return self.planned_hours * self.hourly_rate

    @property
    def actual_cost(self) -> float:
        """Actual cost incurred in USD."""
        return self.actual_hours * self.hourly_rate

    @property
    def variance_hours(self) -> float:
        """Hours variance: positive = under budget, negative = over budget."""
        return self.planned_hours - self.actual_hours

    @property
    def variance_cost(self) -> float:
        """Cost variance in USD: positive = under budget."""
        return self.planned_cost - self.actual_cost

    @property
    def utilization(self) -> float:
        """Fraction of planned hours consumed (can exceed 1.0 when over-budget)."""
        if self.planned_hours <= 0:
            return 0.0
        return self.actual_hours / self.planned_hours

    @property
    def alert_level(self) -> AlertLevel:
        """Current alert level based on hours utilization."""
        return _alert_level(self.utilization)

    @property
    def is_over_budget(self) -> bool:
        return self.actual_hours > self.planned_hours


@dataclass
class VarianceReport:
    """Aggregate variance report across all budget entries."""
    total_planned_hours: float
    total_actual_hours: float
    total_planned_cost: float
    total_actual_cost: float
    entries: list[BudgetEntry] = field(default_factory=list)
    over_budget_items: list[BudgetEntry] = field(default_factory=list)

    @property
    def total_variance_hours(self) -> float:
        """Aggregate hours variance: positive = under budget."""
        return self.total_planned_hours - self.total_actual_hours

    @property
    def total_variance_cost(self) -> float:
        """Aggregate cost variance in USD."""
        return self.total_planned_cost - self.total_actual_cost

    @property
    def overall_utilization(self) -> float:
        """Overall fraction of planned hours consumed."""
        if self.total_planned_hours <= 0:
            return 0.0
        return self.total_actual_hours / self.total_planned_hours

    @property
    def overall_alert_level(self) -> AlertLevel:
        """Aggregate alert level across the engagement."""
        return _alert_level(self.overall_utilization)


class BudgetTracker:
    """Tracks planned vs actual hours and costs for a work program.

    Usage
    -----
    1. Call ``plan_hours`` once per WorkItem to register expected effort.
    2. Call ``record_actual`` as work progresses to log real hours.
    3. Call ``variance_report`` for a full picture, or ``overall_alert`` for
       a quick status check.

    Thread-safety: not thread-safe; callers own synchronization.
    """

    def __init__(self, engagement_id: Optional[UUID] = None) -> None:
        self.engagement_id = engagement_id
        self._entries: dict[UUID, BudgetEntry] = {}

    # ------------------------------------------------------------------
    # Mutation
    # ------------------------------------------------------------------

    def plan_hours(
        self,
        item_id: UUID,
        control_id: str,
        planned_hours: float,
        hourly_rate: float = 150.0,
    ) -> BudgetEntry:
        """Register planned hours for a work item.

        Calling plan_hours a second time for the same item_id replaces the
        entry (useful for re-scoping).
        """
        if planned_hours < 0:
            raise ValueError(f"planned_hours must be non-negative, got {planned_hours}")
        entry = BudgetEntry(
            item_id=item_id,
            control_id=control_id,
            planned_hours=planned_hours,
            hourly_rate=hourly_rate,
        )
        self._entries[item_id] = entry
        return entry

    def record_actual(self, item_id: UUID, actual_hours: float) -> BudgetEntry:
        """Record actual hours spent on a work item.

        Raises KeyError if plan_hours was not called first.
        """
        if item_id not in self._entries:
            raise KeyError(
                f"No budget entry for item {item_id}; call plan_hours first"
            )
        if actual_hours < 0:
            raise ValueError(f"actual_hours must be non-negative, got {actual_hours}")
        self._entries[item_id].actual_hours = actual_hours
        return self._entries[item_id]

    # ------------------------------------------------------------------
    # Queries
    # ------------------------------------------------------------------

    def get_entry(self, item_id: UUID) -> BudgetEntry:
        """Retrieve the budget entry for a work item."""
        if item_id not in self._entries:
            raise KeyError(f"No budget entry for item {item_id}")
        return self._entries[item_id]

    def variance_report(self) -> VarianceReport:
        """Compute aggregate variance report across all budget entries."""
        entries = list(self._entries.values())
        total_planned = sum(e.planned_hours for e in entries)
        total_actual = sum(e.actual_hours for e in entries)
        total_planned_cost = sum(e.planned_cost for e in entries)
        total_actual_cost = sum(e.actual_cost for e in entries)
        over_budget = [e for e in entries if e.is_over_budget]

        return VarianceReport(
            total_planned_hours=total_planned,
            total_actual_hours=total_actual,
            total_planned_cost=total_planned_cost,
            total_actual_cost=total_actual_cost,
            entries=entries,
            over_budget_items=over_budget,
        )

    def check_alert(self, item_id: UUID) -> AlertLevel:
        """Return the alert level for a specific work item."""
        return self.get_entry(item_id).alert_level

    def overall_alert(self) -> AlertLevel:
        """Return the aggregate alert level across all tracked items."""
        return self.variance_report().overall_alert_level

    def all_alerts(self) -> dict[UUID, AlertLevel]:
        """Return {item_id: AlertLevel} for all items not at NONE level."""
        return {
            entry.item_id: entry.alert_level
            for entry in self._entries.values()
            if entry.alert_level != AlertLevel.NONE
        }
