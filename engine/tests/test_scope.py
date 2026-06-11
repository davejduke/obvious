"""Unit tests — Scope Engine (DAG model, boundary management, state machine)."""
import pytest
from uuid import uuid4
from engine.scope.dag import (
    AuditScopeDAG,
    BoundaryStatus,
    ScopeNode,
    ScopeState,
    SCOPE_STATE_TRANSITIONS,
)


def make_node(control_id: str = "C-001", risk_weight: float = 1.0, parent_ids=None):
    return ScopeNode(
        control_id=control_id,
        title=f"Control {control_id}",
        article_ref="NIS2-21a",
        risk_weight=risk_weight,
        parent_ids=parent_ids or [],
    )


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
        with pytest.raises(Exception):
            make_node(risk_weight=6.0)
        with pytest.raises(Exception):
            make_node(risk_weight=-0.1)


class TestAuditScopeDAG:
    def test_add_single_node(self):
        dag = AuditScopeDAG()
        node = make_node("C-001")
        dag.add_node(node)
        assert dag.node_count() == 1
        assert dag.get_node(node.node_id).control_id == "C-001"

    def test_topological_order_linear_chain(self):
        dag = AuditScopeDAG()
        root = make_node("ROOT")
        dag.add_node(root)
        child = make_node("CHILD", parent_ids=[root.node_id])
        dag.add_node(child)
        grandchild = make_node("GRANDCHILD", parent_ids=[child.node_id])
        dag.add_node(grandchild)

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
