"""Risk Engine — Bayesian scoring, materiality, TDR thresholds, 5x asymmetric penalty."""
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

__all__ = [
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
]
