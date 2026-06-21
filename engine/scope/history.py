"""Scope Engine — version history and diff capabilities.

Records immutable ``ScopeSnapshot`` objects each time a scope DAG is approved,
and provides ``ScopeDiff`` to show exactly what changed between two versions.

Usage::

    history = ScopeVersionHistory(engagement_id="eng-001")

    # Capture version 1
    snap1 = history.record_version(dag, version=1, changed_by="alice")

    # Mutate the DAG … then capture version 2
    dag.set_boundary(node_id, BoundaryStatus.IN_SCOPE)
    snap2 = history.record_version(dag, version=2, changed_by="bob")

    diff = history.diff(1, 2)
    print(diff.added, diff.removed, diff.modified)
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any
from uuid import UUID

from engine.scope.dag import AuditScopeDAG, BoundaryStatus, ScopeNode


@dataclass
class NodeSnapshot:
    """Immutable snapshot of a single scope node."""

    node_id: UUID
    control_id: str
    title: str
    boundary_status: BoundaryStatus
    risk_weight: float
    parent_ids: list[UUID]

    @classmethod
    def from_node(cls, node: ScopeNode) -> NodeSnapshot:
        return cls(
            node_id=node.node_id,
            control_id=node.control_id,
            title=node.title,
            boundary_status=node.boundary_status,
            risk_weight=node.risk_weight,
            parent_ids=list(node.parent_ids),
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "node_id": str(self.node_id),
            "control_id": self.control_id,
            "title": self.title,
            "boundary_status": self.boundary_status.value,
            "risk_weight": self.risk_weight,
            "parent_ids": [str(p) for p in self.parent_ids],
        }


@dataclass
class ScopeSnapshot:
    """Immutable snapshot of an entire scope DAG at a specific version."""

    version: int
    engagement_id: str
    nodes: dict[UUID, NodeSnapshot] = field(default_factory=dict)
    captured_at: datetime = field(default_factory=lambda: datetime.now(UTC))
    changed_by: str = ""
    note: str = ""

    @property
    def in_scope_controls(self) -> set[str]:
        """Set of control IDs currently IN_SCOPE in this snapshot."""
        return {
            n.control_id
            for n in self.nodes.values()
            if n.boundary_status == BoundaryStatus.IN_SCOPE
        }

    def to_dict(self) -> dict[str, Any]:
        return {
            "version": self.version,
            "engagement_id": self.engagement_id,
            "nodes": {str(k): v.to_dict() for k, v in self.nodes.items()},
            "captured_at": self.captured_at.isoformat(),
            "changed_by": self.changed_by,
            "note": self.note,
        }


@dataclass
class NodeDiff:
    """Change to a single node between two scope versions."""

    control_id: str
    node_id: UUID
    kind: str  # "added" | "removed" | "modified"
    old_boundary: BoundaryStatus | None = None
    new_boundary: BoundaryStatus | None = None
    old_risk_weight: float | None = None
    new_risk_weight: float | None = None

    def to_dict(self) -> dict[str, Any]:
        return {
            "control_id": self.control_id,
            "node_id": str(self.node_id),
            "kind": self.kind,
            "old_boundary": self.old_boundary.value if self.old_boundary else None,
            "new_boundary": self.new_boundary.value if self.new_boundary else None,
            "old_risk_weight": self.old_risk_weight,
            "new_risk_weight": self.new_risk_weight,
        }


@dataclass
class ScopeDiff:
    """Structural diff between two scope versions."""

    from_version: int
    to_version: int
    changes: list[NodeDiff] = field(default_factory=list)

    @property
    def added(self) -> list[NodeDiff]:
        return [c for c in self.changes if c.kind == "added"]

    @property
    def removed(self) -> list[NodeDiff]:
        return [c for c in self.changes if c.kind == "removed"]

    @property
    def modified(self) -> list[NodeDiff]:
        return [c for c in self.changes if c.kind == "modified"]

    def to_dict(self) -> dict[str, Any]:
        return {
            "from_version": self.from_version,
            "to_version": self.to_version,
            "added": [c.to_dict() for c in self.added],
            "removed": [c.to_dict() for c in self.removed],
            "modified": [c.to_dict() for c in self.modified],
        }


def diff_snapshots(snap_a: ScopeSnapshot, snap_b: ScopeSnapshot) -> ScopeDiff:
    """Compute a structural diff between two scope snapshots.

    Nodes are matched by ``node_id``.  Changes in ``control_id`` (rename) are
    represented as a remove + add pair.
    """
    result = ScopeDiff(from_version=snap_a.version, to_version=snap_b.version)

    ids_a = set(snap_a.nodes)
    ids_b = set(snap_b.nodes)

    # Nodes present only in snap_b → added
    for node_id in ids_b - ids_a:
        n = snap_b.nodes[node_id]
        result.changes.append(
            NodeDiff(
                control_id=n.control_id,
                node_id=node_id,
                kind="added",
                new_boundary=n.boundary_status,
                new_risk_weight=n.risk_weight,
            )
        )

    # Nodes present only in snap_a → removed
    for node_id in ids_a - ids_b:
        n = snap_a.nodes[node_id]
        result.changes.append(
            NodeDiff(
                control_id=n.control_id,
                node_id=node_id,
                kind="removed",
                old_boundary=n.boundary_status,
                old_risk_weight=n.risk_weight,
            )
        )

    # Nodes present in both → check for modifications
    for node_id in ids_a & ids_b:
        na = snap_a.nodes[node_id]
        nb = snap_b.nodes[node_id]
        if na.boundary_status != nb.boundary_status or na.risk_weight != nb.risk_weight:
            result.changes.append(
                NodeDiff(
                    control_id=na.control_id,
                    node_id=node_id,
                    kind="modified",
                    old_boundary=na.boundary_status,
                    new_boundary=nb.boundary_status,
                    old_risk_weight=na.risk_weight,
                    new_risk_weight=nb.risk_weight,
                )
            )

    return result


class ScopeVersionHistory:
    """Maintains a versioned history of scope DAG snapshots for one engagement.

    Versions are immutable once recorded.  Version numbers are caller-controlled
    and should match the ``scope_version`` field in the engagement database row.
    """

    def __init__(self, engagement_id: str) -> None:
        self._engagement_id = engagement_id
        self._versions: dict[int, ScopeSnapshot] = {}

    @property
    def engagement_id(self) -> str:
        return self._engagement_id

    def record_version(
        self,
        dag: AuditScopeDAG,
        version: int,
        changed_by: str = "",
        note: str = "",
    ) -> ScopeSnapshot:
        """Capture the current state of a DAG as an immutable snapshot.

        Raises:
            ValueError: if ``version`` has already been recorded.
        """
        if version in self._versions:
            raise ValueError(
                f"Version {version} already recorded for engagement "
                f"{self._engagement_id!r}"
            )
        nodes = {
            node.node_id: NodeSnapshot.from_node(node)
            for node in dag.all_nodes()
        }
        snapshot = ScopeSnapshot(
            version=version,
            engagement_id=self._engagement_id,
            nodes=nodes,
            changed_by=changed_by,
            note=note,
        )
        self._versions[version] = snapshot
        return snapshot

    def get_version(self, version: int) -> ScopeSnapshot:
        """Retrieve a specific version snapshot.

        Raises:
            KeyError: if the version has not been recorded.
        """
        if version not in self._versions:
            raise KeyError(
                f"Version {version} not found for engagement "
                f"{self._engagement_id!r}"
            )
        return self._versions[version]

    def latest_version(self) -> ScopeSnapshot | None:
        """Return the most recently recorded snapshot, or None."""
        if not self._versions:
            return None
        return self._versions[max(self._versions)]

    def all_versions(self) -> list[ScopeSnapshot]:
        """Return all snapshots in ascending version order."""
        return [self._versions[v] for v in sorted(self._versions)]

    def diff(self, from_version: int, to_version: int) -> ScopeDiff:
        """Compute the structural diff between two recorded versions.

        Raises:
            KeyError: if either version has not been recorded.
        """
        snap_a = self.get_version(from_version)
        snap_b = self.get_version(to_version)
        return diff_snapshots(snap_a, snap_b)

