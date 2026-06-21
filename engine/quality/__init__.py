"""Quality Engine — Cochran sampling, 6-tier evidence scoring, sufficiency,
hard floors, cross-validation, conflict resolution, gate enforcement, trending.
"""
from engine.quality.conflict_resolver import (
    QUALITY_VARIANCE_THRESHOLD,
    ConflictResolutionResult,
    ConflictType,
    EvidenceConflict,
    detect_conflicts,
    resolve_conflicts,
)
from engine.quality.cross_validator import (
    AnnotatedEvidenceItem,
    CrossValidationResult,
    SourceAggregation,
    cross_validate_evidence,
)
from engine.quality.floors import (
    DEFAULT_MINIMUM_EVIDENCE,
    FloorCheckResult,
    QualityFloorConfig,
    check_quality_floor,
    get_floor_for_control,
    recalculate_floor,
)
from engine.quality.gate import (
    GateBlockReason,
    QualityGateConfig,
    QualityGateResult,
    enforce_quality_gate,
)
from engine.quality.sampler import (
    TIER_WEIGHTS,
    Z_SCORES,
    EvidenceQualityInput,
    SampleSizeResult,
    SamplingParameters,
    SufficiencyResult,
    SufficiencyVerdict,
    assess_sufficiency,
    cochran_sample_size,
    score_evidence_tier,
)
from engine.quality.trending import (
    TREND_STABLE_THRESHOLD,
    PeriodScore,
    QualityTrend,
    TrendDirection,
    add_period_score,
    compute_trend,
)

__all__ = [
    # sampler (Cochran + sufficiency)
    "cochran_sample_size",
    "assess_sufficiency",
    "score_evidence_tier",
    "SamplingParameters",
    "SampleSizeResult",
    "EvidenceQualityInput",
    "SufficiencyResult",
    "SufficiencyVerdict",
    "TIER_WEIGHTS",
    "Z_SCORES",
    # floors
    "QualityFloorConfig",
    "FloorCheckResult",
    "DEFAULT_MINIMUM_EVIDENCE",
    "get_floor_for_control",
    "check_quality_floor",
    "recalculate_floor",
    # cross_validator
    "AnnotatedEvidenceItem",
    "SourceAggregation",
    "CrossValidationResult",
    "cross_validate_evidence",
    # conflict_resolver
    "ConflictType",
    "EvidenceConflict",
    "ConflictResolutionResult",
    "QUALITY_VARIANCE_THRESHOLD",
    "detect_conflicts",
    "resolve_conflicts",
    # gate
    "GateBlockReason",
    "QualityGateConfig",
    "QualityGateResult",
    "enforce_quality_gate",
    # trending
    "TrendDirection",
    "PeriodScore",
    "QualityTrend",
    "TREND_STABLE_THRESHOLD",
    "compute_trend",
    "add_period_score",
]
