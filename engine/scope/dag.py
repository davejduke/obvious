"""Scope Engine — DAG model for NIS 2 audit scope and boundary management.

The DAG models control dependencies: a parent control must be in scope
before its children can be audited. Each node tracks boundary status
and participates in topological resolution.
"""
from __future__ import annotations

from collections import deque
from enum import Enum
from typing import Any, Optional
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class BoundaryStatus(str, Enum):
    """Whether a control node is inside or outside audit scope."""
    IN_SCOPE = "in_scope"
    OUT_OF_SCOPE = "out_of_scope"
    CONDITIONAL = "conditional"


class ScopeState(str, Enum):
    """Lifecycle states for an audit scope document: Draft→Proposed→Approved→Locked."""
    DRAFT = "draft"
    PROPOSED = "proposed"
    APPROVED = "approved"
    LOCKED = "locked"


# Valid state transitions (state machine)
SCOPE_STATE_TRANSITIONS: dict[ScopeState, set[ScopeState]] = {
    ScopeState.DRAFT: {ScopeState.PROPOSED},
    ScopeState.PROPOSED: {ScopeState.APPROVED, ScopeState.DRAFT},
    ScopeState.APPROVED: {ScopeState.LOCKED, ScopeState.PROPOSED},
    ScopeState.LOCKED: set(),  # Terminal state
}


class ScopeNode(BaseModel):
    """A single control node in the audit scope DAG."""
    node_id: UUID = Field(default_factory=uuid4)
    control_id: str
    title: str
    article_ref: Optional[str] = None  # e.g. "NIS2-21a"
    boundary_status: BoundaryStatus = BoundaryStatus.CONDITIONAL
    risk_weight: float = Field(default=1.0, ge=0.0, le=5.0)
    parent_ids: list[UUID] = Field(default_factory=list)
    metadata: dict[str, Any] = Field(default_factory=dict)

    def is_in_scope(self) -> bool:
        return self.boundary_status == BoundaryStatus.IN_SCOPE

    def is_out_of_scope(self) -> bool:
        return self.boundary_status == BoundaryStatus.OUT_OF_SCOPE


class AuditScopeDAG:
    """Directed Acyclic Graph for audit scope management.

    - Adding nodes with dependency edges
    - Topological ordering via Kahn's algorithm O(V+E)
    - Reachability analysis from a root node
    - In-scope subset extraction
    - Cycle detection (raises ValueError on cycle introduction)
    """

    def __init__(self) -> None:
        self._nodes: dict[UUID, ScopeNode] = {}
        self._children: dict[UUID, set[UUID]] = {}  # parent_id → child_ids
        self._state: ScopeState = ScopeState.DRAFT

    @property
    def state(self) -> ScopeState:
        return self._state

    def transition(self, new_state: ScopeState) -> None:
        """Advance the scope state machine."""
        allowed = SCOPE_STATE_TRANSITIONS.get(self._state, set())
        if new_state not in allowed:
            raise ValueError(
                f"Invalid transition {self._state} → {new_state}. "
                f"Allowed: {allowed}"
            )
        if self._state == ScopeState.LOCKED:
            raise ValueError("Scope is LOCKED — no further modifications allowed")
        self._state = new_state

    def _check_not_locked(self) -> None:
        if self._state == ScopeState.LOCKED:
            raise ValueError("Scope is LOCKED — mutations are not permitted")

    def add_node(self, node: ScopeNode) -> None:
        """Add a node to the DAG, validating parents exist and no cycle is introduced."""
        self._check_not_locked()
        if node.node_id in self._nodes:
            raise ValueError(f"Node {node.node_id} already exists in DAG")

        for pid in node.parent_ids:
            if pid not in self._nodes:
                raise ValueError(
                    f"Parent node {pid} does not exist; add parents before children"
                )

        self._nodes[node.node_id] = node
        self._children.setdefault(node.node_id, set())
        for pid in node.parent_ids:
            self._children[pid].add(node.node_id)

        if not self._is_dag():
            del self._nodes[node.node_id]
            for pid in node.parent_ids:
                self._children[pid].discard(node.node_id)
            raise ValueError(f"Adding node {node.control_id!r} would create a cycle")

    def set_boundary(self, node_id: UUID, status: BoundaryStatus) -> None:
        self._check_not_locked()
        if node_id not in self._nodes:
            raise KeyError(f"Node {node_id} not found")
        self._nodes[node_id].boundary_status = status

    def get_node(self, node_id: UUID) -> ScopeNode:
        if node_id not in self._nodes:
            raise KeyError(f"Node {node_id} not found")
        return self._nodes[node_id]

    def topological_order(self) -> list[ScopeNode]:
        """Return nodes in topological order using Kahn's BFS algorithm."""
        in_degree: dict[UUID, int] = {nid: 0 for nid in self._nodes}
        for children in self._children.values():
            for cid in children:
                in_degree[cid] += 1

        queue: deque[UUID] = deque(
            nid for nid, deg in in_degree.items() if deg == 0
        )
        result: list[ScopeNode] = []

        while queue:
            nid = queue.popleft()
            result.append(self._nodes[nid])
            for cid in sorted(self._children.get(nid, set()), key=str):
                in_degree[cid] -= 1
                if in_degree[cid] == 0:
                    queue.append(cid)

        if len(result) != len(self._nodes):
            raise RuntimeError("Cycle detected during topological sort")
        return result

    def reachable_from(self, root_id: UUID) -> set[UUID]:
        """Return all node IDs reachable from root_id (inclusive, BFS)."""
        if root_id not in self._nodes:
            raise KeyError(f"Root node {root_id} not found")
        visited: set[UUID] = set()
        stack = [root_id]
        while stack:
            current = stack.pop()
            if current in visited:
                continue
            visited.add(current)
            stack.extend(self._children.get(current, set()))
        return visited

    def in_scope_nodes(self) -> list[ScopeNode]:
        return [n for n in self._nodes.values() if n.is_in_scope()]

    def node_count(self) -> int:
        return len(self._nodes)

    def all_nodes(self) -> list[ScopeNode]:
        return list(self._nodes.values())

    def _is_dag(self) -> bool:
        """DFS coloring cycle detection."""
        WHITE, GRAY, BLACK = 0, 1, 2
        color: dict[UUID, int] = {nid: WHITE for nid in self._nodes}

        def dfs(node: UUID) -> bool:
            color[node] = GRAY
            for child in self._children.get(node, set()):
                if color[child] == GRAY:
                    return False
                if color[child] == WHITE and not dfs(child):
                    return False
            color[node] = BLACK
            return True

        return all(dfs(n) for n in self._nodes if color[n] == WHITE)
