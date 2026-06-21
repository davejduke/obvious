"""Scope Engine — DAG-based audit scope management with enforcement and workflow."""
from engine.scope.dag import AuditScopeDAG, BoundaryStatus, ScopeNode, ScopeState
from engine.scope.enforcement import (
    EnforcementResult,
    EnforcementTrigger,
    EngagementEventType,
    ScopeEnforcer,
)
from engine.scope.history import (
    NodeDiff,
    NodeSnapshot,
    ScopeDiff,
    ScopeSnapshot,
    ScopeVersionHistory,
    diff_snapshots,
)
from engine.scope.impact import (
    EvidenceRef,
    FindingRef,
    ImpactedEvidence,
    ImpactedFinding,
    ScopeChangeImpact,
    ScopeImpactAnalyzer,
)
from engine.scope.validation import (
    ScopeValidationResult,
    ValidationCode,
    ValidationViolation,
    validate_control_evidence_mapping,
    validate_coverage_completeness,
    validate_dag_boundaries,
    validate_no_orphan_in_scope_nodes,
    validate_parent_scope_consistency,
)
from engine.scope.workflow import (
    AuditEvent,
    ModificationRequestStatus,
    ModificationWorkflow,
    ScopeChangeKind,
    ScopeChangeRecord,
    ScopeModificationRequest,
)

__all__ = [
    # Core DAG
    "AuditScopeDAG",
    "BoundaryStatus",
    "ScopeNode",
    "ScopeState",
    # Enforcement
    "EngagementEventType",
    "EnforcementResult",
    "EnforcementTrigger",
    "ScopeEnforcer",
    # Version history & diff
    "NodeDiff",
    "NodeSnapshot",
    "ScopeDiff",
    "ScopeSnapshot",
    "ScopeVersionHistory",
    "diff_snapshots",
    # Impact analysis
    "EvidenceRef",
    "FindingRef",
    "ImpactedEvidence",
    "ImpactedFinding",
    "ScopeChangeImpact",
    "ScopeImpactAnalyzer",
    # Validation
    "ScopeValidationResult",
    "ValidationCode",
    "ValidationViolation",
    "validate_control_evidence_mapping",
    "validate_coverage_completeness",
    "validate_dag_boundaries",
    "validate_no_orphan_in_scope_nodes",
    "validate_parent_scope_consistency",
    # Workflow
    "AuditEvent",
    "ModificationRequestStatus",
    "ModificationWorkflow",
    "ScopeChangeKind",
    "ScopeChangeRecord",
    "ScopeModificationRequest",
]

