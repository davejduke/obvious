"""Scope Engine — validation rules for DAG boundary integrity.

Four rules are enforced:

1. ``validate_no_orphan_in_scope_nodes`` — an in-scope node whose ALL parents
   are out-of-scope is an orphan; it cannot be audited in isolation.
2. ``validate_parent_scope_consistency`` — any in-scope node with even one
   out-of-scope parent is a dependency-ordering violation.
3. ``validate_coverage_completeness`` — every required control ID must be
   present in the DAG and marked IN_SCOPE.
4. ``validate_control_evidence_mapping`` — every in-scope control must map to
   at least one evidence source.

Use ``validate_dag_boundaries`` to run all applicable rules in one call.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from uuid import UUID

from engine.scope.dag import AuditScopeDAG, BoundaryStatus


class ValidationCode(str, Enum):  # noqa: UP042
    """Machine-readable error codes for validation violations."""

    ORPHAN_NODE = "ORPHAN_NODE"
    CIRCULAR_DEPENDENCY = "CIRCULAR_DEPENDENCY"
    MISSING_COVERAGE = "MISSING_COVERAGE"
    NO_EVIDENCE_SOURCE = "NO_EVIDENCE_SOURCE"
    PARENT_OUT_OF_SCOPE = "PARENT_OUT_OF_SCOPE"


@dataclass
class ValidationViolation:
    """A single validation rule failure."""

    code: ValidationCode
    message: str
    node_id: UUID | None = None
    control_id: str | None = None


@dataclass
class ScopeValidationResult:
    """Aggregated result of one or more validation passes."""

    violations: list[ValidationViolation] = field(default_factory=list)

    @property
    def is_valid(self) -> bool:
        """True iff no violations were recorded."""
        return len(self.violations) == 0

    def add(self, violation: ValidationViolation) -> None:
        self.violations.append(violation)

    def merge(self, other: ScopeValidationResult) -> None:
        """Merge all violations from *other* into this result."""
        self.violations.extend(other.violations)


# ─── Individual rule validators ───────────────────────────────────────────────


def validate_no_orphan_in_scope_nodes(dag: AuditScopeDAG) -> ScopeValidationResult:
    """Flag in-scope nodes whose ALL direct parents are OUT_OF_SCOPE.

    An orphan control is meaningless to audit because its prerequisite
    controls were excluded from scope.
    """
    result = ScopeValidationResult()
    for node in dag.in_scope_nodes():
        if not node.parent_ids:
            continue  # root node — always valid
        parent_statuses = [
            dag.get_node(pid).boundary_status for pid in node.parent_ids
        ]
        if all(s == BoundaryStatus.OUT_OF_SCOPE for s in parent_statuses):
            result.add(
                ValidationViolation(
                    code=ValidationCode.ORPHAN_NODE,
                    node_id=node.node_id,
                    control_id=node.control_id,
                    message=(
                        f"Node {node.control_id!r} is IN_SCOPE but all its parents "
                        f"are OUT_OF_SCOPE (orphan dependency)."
                    ),
                )
            )
    return result


def validate_parent_scope_consistency(dag: AuditScopeDAG) -> ScopeValidationResult:
    """Flag any in-scope node that has at least one OUT_OF_SCOPE parent.

    Stricter than the orphan check: even a single out-of-scope parent is a
    dependency-ordering violation.
    """
    result = ScopeValidationResult()
    for node in dag.in_scope_nodes():
        for pid in node.parent_ids:
            parent = dag.get_node(pid)
            if parent.is_out_of_scope():
                result.add(
                    ValidationViolation(
                        code=ValidationCode.PARENT_OUT_OF_SCOPE,
                        node_id=node.node_id,
                        control_id=node.control_id,
                        message=(
                            f"In-scope control {node.control_id!r} depends on parent "
                            f"{parent.control_id!r} which is OUT_OF_SCOPE. "
                            f"Parent controls must be in scope before their children."
                        ),
                    )
                )
    return result


def validate_coverage_completeness(
    dag: AuditScopeDAG,
    required_control_ids: set[str],
) -> ScopeValidationResult:
    """Verify every required control is present in the DAG and marked IN_SCOPE."""
    result = ScopeValidationResult()
    in_scope_ids = {n.control_id for n in dag.in_scope_nodes()}
    all_ids = {n.control_id for n in dag.all_nodes()}

    for control_id in required_control_ids:
        if control_id not in all_ids:
            result.add(
                ValidationViolation(
                    code=ValidationCode.MISSING_COVERAGE,
                    control_id=control_id,
                    message=(
                        f"Required control {control_id!r} is not present in the scope DAG."
                    ),
                )
            )
        elif control_id not in in_scope_ids:
            result.add(
                ValidationViolation(
                    code=ValidationCode.MISSING_COVERAGE,
                    control_id=control_id,
                    message=(
                        f"Required control {control_id!r} exists in the DAG "
                        f"but is not marked IN_SCOPE."
                    ),
                )
            )
    return result


def validate_control_evidence_mapping(
    dag: AuditScopeDAG,
    control_evidence_map: dict[str, list[str]],
) -> ScopeValidationResult:
    """Verify every in-scope control has at least one mapped evidence source.

    Args:
        dag: The scope DAG to validate.
        control_evidence_map: ``{control_id: [evidence_source_id, ...]}``.  An
            empty list or a missing key means no evidence is mapped.
    """
    result = ScopeValidationResult()
    for node in dag.in_scope_nodes():
        sources = control_evidence_map.get(node.control_id, [])
        if not sources:
            result.add(
                ValidationViolation(
                    code=ValidationCode.NO_EVIDENCE_SOURCE,
                    node_id=node.node_id,
                    control_id=node.control_id,
                    message=(
                        f"In-scope control {node.control_id!r} has no mapped evidence source. "
                        f"All in-scope controls must map to at least one evidence source."
                    ),
                )
            )
    return result


# ─── Composite validator ──────────────────────────────────────────────────────


def validate_dag_boundaries(
    dag: AuditScopeDAG,
    required_control_ids: set[str] | None = None,
    control_evidence_map: dict[str, list[str]] | None = None,
) -> ScopeValidationResult:
    """Run all applicable validation rules and return the combined result.

    Always runs orphan and parent-consistency checks.  Pass the optional
    arguments to enable coverage-completeness and evidence-mapping checks.

    Args:
        dag: The scope DAG to validate.
        required_control_ids: Control IDs that *must* be in scope.  Pass
            ``None`` to skip the coverage completeness check.
        control_evidence_map: Mapping of control_id → evidence source IDs.  Pass
            ``None`` to skip the evidence mapping check.
    """
    combined = ScopeValidationResult()
    combined.merge(validate_no_orphan_in_scope_nodes(dag))
    combined.merge(validate_parent_scope_consistency(dag))
    if required_control_ids is not None:
        combined.merge(validate_coverage_completeness(dag, required_control_ids))
    if control_evidence_map is not None:
        combined.merge(validate_control_evidence_mapping(dag, control_evidence_map))
    return combined

