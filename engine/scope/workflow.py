"""Scope Engine — scope modification workflow with audit trail.

Flow: submit_request → (review) → approve_request | reject_request

The ``ModificationWorkflow`` is in-memory and intentionally persistence-agnostic.
Callers should persist the returned ``AuditEvent`` objects to the platform
append-only event store (services/audit_trail).

Usage::

    wf = ModificationWorkflow()

    # Auditor requests a scope change
    req = wf.submit_request(
        engagement_id="eng-001",
        requested_by="alice@example.com",
        description="Add NIS2 Article 21(c) controls",
        proposed_changes=[
            ScopeChangeRecord(
                kind=ScopeChangeKind.SET_BOUNDARY,
                control_id="NIS2-21c",
                boundary_status=BoundaryStatus.IN_SCOPE,
            )
        ],
    )

    # CAE reviews and approves
    approved = wf.approve_request(req.request_id, reviewed_by="bob@example.com")

    # Retrieve audit trail for the engagement
    trail = wf.get_audit_trail("eng-001")
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC, datetime
from enum import Enum
from typing import Any
from uuid import UUID, uuid4

from engine.scope.dag import BoundaryStatus


class ModificationRequestStatus(str, Enum):  # noqa: UP042
    """Lifecycle states for a scope modification request."""

    PENDING = "pending"
    APPROVED = "approved"
    REJECTED = "rejected"


class ScopeChangeKind(str, Enum):  # noqa: UP042
    """The nature of a single proposed scope change."""

    ADD_NODE = "add_node"
    REMOVE_NODE = "remove_node"
    SET_BOUNDARY = "set_boundary"


@dataclass
class ScopeChangeRecord:
    """Describes a single intended change to the scope DAG."""

    kind: ScopeChangeKind
    control_id: str
    node_id: UUID | None = None
    boundary_status: BoundaryStatus | None = None
    description: str = ""

    def to_dict(self) -> dict[str, Any]:
        return {
            "kind": self.kind.value,
            "control_id": self.control_id,
            "node_id": str(self.node_id) if self.node_id else None,
            "boundary_status": self.boundary_status.value if self.boundary_status else None,
            "description": self.description,
        }


@dataclass
class ScopeModificationRequest:
    """A scope change request that is pending review."""

    request_id: UUID = field(default_factory=uuid4)
    engagement_id: str = ""
    requested_by: str = ""
    requested_at: datetime = field(default_factory=lambda: datetime.now(UTC))
    description: str = ""
    proposed_changes: list[ScopeChangeRecord] = field(default_factory=list)
    status: ModificationRequestStatus = ModificationRequestStatus.PENDING
    reviewed_by: str | None = None
    reviewed_at: datetime | None = None
    review_note: str | None = None

    def is_pending(self) -> bool:
        return self.status == ModificationRequestStatus.PENDING

    def is_approved(self) -> bool:
        return self.status == ModificationRequestStatus.APPROVED

    def is_rejected(self) -> bool:
        return self.status == ModificationRequestStatus.REJECTED

    def to_dict(self) -> dict[str, Any]:
        return {
            "request_id": str(self.request_id),
            "engagement_id": self.engagement_id,
            "requested_by": self.requested_by,
            "requested_at": self.requested_at.isoformat(),
            "description": self.description,
            "proposed_changes": [c.to_dict() for c in self.proposed_changes],
            "status": self.status.value,
            "reviewed_by": self.reviewed_by,
            "reviewed_at": self.reviewed_at.isoformat() if self.reviewed_at else None,
            "review_note": self.review_note,
        }


@dataclass
class AuditEvent:
    """An immutable audit-trail record for scope workflow transitions."""

    event_id: UUID = field(default_factory=uuid4)
    engagement_id: str = ""
    event_type: str = ""
    actor: str = ""
    occurred_at: datetime = field(default_factory=lambda: datetime.now(UTC))
    request_id: UUID | None = None
    detail: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "event_id": str(self.event_id),
            "engagement_id": self.engagement_id,
            "event_type": self.event_type,
            "actor": self.actor,
            "occurred_at": self.occurred_at.isoformat(),
            "request_id": str(self.request_id) if self.request_id else None,
            "detail": self.detail,
        }


class ModificationWorkflow:
    """Manages scope modification requests with a full audit trail.

    Thread-safety: not thread-safe.  Use one instance per engagement if
    concurrent access is required (or add external locking).
    """

    def __init__(self) -> None:
        self._requests: dict[UUID, ScopeModificationRequest] = {}
        self._audit_trail: list[AuditEvent] = []

    # ─── Public API ─────────────────────────────────────────────────────────────

    def submit_request(
        self,
        engagement_id: str,
        requested_by: str,
        description: str,
        proposed_changes: list[ScopeChangeRecord],
    ) -> ScopeModificationRequest:
        """Submit a new scope modification request for review.

        Returns the newly created ``ScopeModificationRequest`` in PENDING state.
        Appends a ``scope_modification_requested`` event to the audit trail.
        """
        req = ScopeModificationRequest(
            engagement_id=engagement_id,
            requested_by=requested_by,
            description=description,
            proposed_changes=proposed_changes,
        )
        self._requests[req.request_id] = req
        self._record_event(
            engagement_id=engagement_id,
            event_type="scope_modification_requested",
            actor=requested_by,
            request_id=req.request_id,
            detail={
                "description": description,
                "change_count": len(proposed_changes),
            },
        )
        return req

    def approve_request(
        self,
        request_id: UUID,
        reviewed_by: str,
        note: str = "",
    ) -> ScopeModificationRequest:
        """Approve a PENDING modification request.

        Transitions the request to APPROVED and records a
        ``scope_modification_approved`` audit event.

        Raises:
            KeyError: if ``request_id`` is unknown.
            ValueError: if the request is not in PENDING state.
        """
        req = self._get_pending_request(request_id)
        req.status = ModificationRequestStatus.APPROVED
        req.reviewed_by = reviewed_by
        req.reviewed_at = datetime.now(UTC)
        req.review_note = note
        self._record_event(
            engagement_id=req.engagement_id,
            event_type="scope_modification_approved",
            actor=reviewed_by,
            request_id=request_id,
            detail={"note": note},
        )
        return req

    def reject_request(
        self,
        request_id: UUID,
        reviewed_by: str,
        note: str = "",
    ) -> ScopeModificationRequest:
        """Reject a PENDING modification request.

        Transitions the request to REJECTED and records a
        ``scope_modification_rejected`` audit event.

        Raises:
            KeyError: if ``request_id`` is unknown.
            ValueError: if the request is not in PENDING state.
        """
        req = self._get_pending_request(request_id)
        req.status = ModificationRequestStatus.REJECTED
        req.reviewed_by = reviewed_by
        req.reviewed_at = datetime.now(UTC)
        req.review_note = note
        self._record_event(
            engagement_id=req.engagement_id,
            event_type="scope_modification_rejected",
            actor=reviewed_by,
            request_id=request_id,
            detail={"note": note},
        )
        return req

    def get_request(self, request_id: UUID) -> ScopeModificationRequest:
        """Retrieve a request by ID.

        Raises:
            KeyError: if ``request_id`` is unknown.
        """
        if request_id not in self._requests:
            raise KeyError(f"Request {request_id} not found")
        return self._requests[request_id]

    def pending_requests(self, engagement_id: str) -> list[ScopeModificationRequest]:
        """Return all PENDING requests for an engagement, in submission order."""
        return [
            r
            for r in self._requests.values()
            if r.engagement_id == engagement_id and r.is_pending()
        ]

    def get_audit_trail(self, engagement_id: str) -> list[AuditEvent]:
        """Return all audit events for an engagement in chronological order."""
        return [
            e for e in self._audit_trail if e.engagement_id == engagement_id
        ]

    # ─── Private helpers ──────────────────────────────────────────────────

    def _get_pending_request(
        self, request_id: UUID
    ) -> ScopeModificationRequest:
        if request_id not in self._requests:
            raise KeyError(f"Request {request_id} not found")
        req = self._requests[request_id]
        if not req.is_pending():
            raise ValueError(
                f"Request {request_id} is already "
                f"{req.status.value!r} and cannot be reviewed again"
            )
        return req

    def _record_event(
        self,
        engagement_id: str,
        event_type: str,
        actor: str,
        request_id: UUID | None,
        detail: dict[str, Any],
    ) -> AuditEvent:
        event = AuditEvent(
            engagement_id=engagement_id,
            event_type=event_type,
            actor=actor,
            request_id=request_id,
            detail=detail,
        )
        self._audit_trail.append(event)
        return event

