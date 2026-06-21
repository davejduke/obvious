"""Scope Engine — DAG boundary enforcement triggers.

Wire ``ScopeEnforcer`` to engagement creation and modification events to
guarantee scope boundaries are validated before audit work begins.

Usage::

    enforcer = ScopeEnforcer()

    # On engagement creation
    result = enforcer.enforce_on_engagement_created(
        engagement_id="eng-001",
        dag=dag,
        required_control_ids={"NIS2-21a", "NIS2-21b"},
        control_evidence_map={"NIS2-21a": ["ev-001"], "NIS2-21b": ["ev-002"]},
    )
    if not result.passed:
        raise ValueError(result.validation.violations)

    # On engagement modification
    result = enforcer.enforce_on_engagement_modified("eng-001", updated_dag)
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC, datetime
from enum import Enum
from typing import Any
from uuid import UUID, uuid4

from engine.scope.dag import AuditScopeDAG
from engine.scope.validation import ScopeValidationResult, validate_dag_boundaries


class EngagementEventType(str, Enum):  # noqa: UP042
    """Lifecycle events that trigger scope boundary enforcement."""

    CREATED = "created"
    MODIFIED = "modified"


@dataclass
class EnforcementTrigger:
    """Metadata recorded every time enforcement runs."""

    trigger_id: UUID = field(default_factory=uuid4)
    engagement_id: str = ""
    event_type: EngagementEventType = EngagementEventType.CREATED
    triggered_at: datetime = field(default_factory=lambda: datetime.now(UTC))
    passed: bool = False
    violation_count: int = 0


@dataclass
class EnforcementResult:
    """Result of a single enforcement pass."""

    trigger: EnforcementTrigger
    validation: ScopeValidationResult

    @property
    def passed(self) -> bool:
        return self.validation.is_valid

    def to_dict(self) -> dict[str, Any]:
        return {
            "trigger_id": str(self.trigger.trigger_id),
            "engagement_id": self.trigger.engagement_id,
            "event_type": self.trigger.event_type.value,
            "triggered_at": self.trigger.triggered_at.isoformat(),
            "passed": self.passed,
            "violations": [
                {
                    "code": v.code.value,
                    "node_id": str(v.node_id) if v.node_id else None,
                    "control_id": v.control_id,
                    "message": v.message,
                }
                for v in self.validation.violations
            ],
        }


class ScopeEnforcer:
    """Runs DAG boundary enforcement on engagement lifecycle events.

    The enforcer is stateless: each ``enforce_*`` call returns a fresh
    ``EnforcementResult`` without side-effects.  Wire cache invalidation
    separately via ``engine.shared.cache.InvalidationHandler`` after confirming
    a result has passed.
    """

    def enforce_on_engagement_created(
        self,
        engagement_id: str,
        dag: AuditScopeDAG,
        required_control_ids: set[str] | None = None,
        control_evidence_map: dict[str, list[str]] | None = None,
    ) -> EnforcementResult:
        """Validate scope DAG boundaries when an engagement is first created."""
        return self._enforce(
            engagement_id=engagement_id,
            dag=dag,
            event_type=EngagementEventType.CREATED,
            required_control_ids=required_control_ids,
            control_evidence_map=control_evidence_map,
        )

    def enforce_on_engagement_modified(
        self,
        engagement_id: str,
        dag: AuditScopeDAG,
        required_control_ids: set[str] | None = None,
        control_evidence_map: dict[str, list[str]] | None = None,
    ) -> EnforcementResult:
        """Validate scope DAG boundaries after an engagement scope change."""
        return self._enforce(
            engagement_id=engagement_id,
            dag=dag,
            event_type=EngagementEventType.MODIFIED,
            required_control_ids=required_control_ids,
            control_evidence_map=control_evidence_map,
        )

    def _enforce(
        self,
        engagement_id: str,
        dag: AuditScopeDAG,
        event_type: EngagementEventType,
        required_control_ids: set[str] | None,
        control_evidence_map: dict[str, list[str]] | None,
    ) -> EnforcementResult:
        validation = validate_dag_boundaries(
            dag=dag,
            required_control_ids=required_control_ids,
            control_evidence_map=control_evidence_map,
        )
        trigger = EnforcementTrigger(
            engagement_id=engagement_id,
            event_type=event_type,
            passed=validation.is_valid,
            violation_count=len(validation.violations),
        )
        return EnforcementResult(trigger=trigger, validation=validation)

