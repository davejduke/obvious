"""Risk Engine — Bayesian scoring, exception detection, materiality, heat maps, aggregation."""
from engine.risk.scorer import (
    AuditConclusion,
    EvidenceItem,
    RiskScoreResult,
    bayesian_update,
    compute_deviation_rate,
    score_control,
    PENALTY_RATIO,
    MATERIALITY_THRESHOLD,
    TDR_MATERIAL,
    TDR_NON_MATERIAL,
)
from engine.risk.exceptions import (
    AnomalyType,
    ExceptionDetector,
    ExceptionFlag,
    detect_exceptions,
)
from engine.risk.materiality import (
    DOMAIN_THRESHOLDS,
    DEFAULT_ORG_CONFIG,
    MaterialityResult,
    OrgMaterialityConfig,
    compute_domain_materiality,
)
from engine.risk.heat_map import (
    HEAT_MAP_SIZE,
    HeatMapCell,
    HeatMapInput,
    HeatMapResult,
    compute_residual_risk,
    generate_heat_map,
)
from engine.risk.aggregator import (
    ArticleRiskScore,
    ArticleTrend,
    ControlMeta,
    OrgRiskThresholds,
    PeriodScore,
    RiskTrend,
    TrendDirection,
    aggregate_by_article,
    compute_risk_trend,
    score_control_with_thresholds,
)

__all__ = [
    # scorer
    "AuditConclusion",
    "EvidenceItem",
    "RiskScoreResult",
    "bayesian_update",
    "compute_deviation_rate",
    "score_control",
    "PENALTY_RATIO",
    "MATERIALITY_THRESHOLD",
    "TDR_MATERIAL",
    "TDR_NON_MATERIAL",
    # exceptions
    "AnomalyType",
    "ExceptionDetector",
    "ExceptionFlag",
    "detect_exceptions",
    # materiality
    "DEFAULT_ORG_CONFIG",
    "DOMAIN_THRESHOLDS",
    "MaterialityResult",
    "OrgMaterialityConfig",
    "compute_domain_materiality",
    # heat_map
    "HEAT_MAP_SIZE",
    "HeatMapCell",
    "HeatMapInput",
    "HeatMapResult",
    "compute_residual_risk",
    "generate_heat_map",
    # aggregator
    "ArticleRiskScore",
    "ArticleTrend",
    "ControlMeta",
    "OrgRiskThresholds",
    "PeriodScore",
    "RiskTrend",
    "TrendDirection",
    "aggregate_by_article",
    "compute_risk_trend",
    "score_control_with_thresholds",
]

