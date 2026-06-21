"""Risk Engine — Article-level aggregation, residual risk, trending, org thresholds.

Aggregation
-----------
Roll up per-control RiskScoreResults to NIS 2 Article 21 sub-clause level:
  article_score       = risk-weight-weighted average of control posteriors
  article_conclusion  = worst-case conclusion among all controls in the article

Risk Trending
-------------
Compare aggregated article scores across two engagement periods:
  IMPROVING     — current posterior > previous  (control posture strengthened)
  DETERIORATING — current posterior < previous  (control posture weakened)
  STABLE        — |Δ| ≤ STABLE_DELTA (0.05)

Org Thresholds
--------------
OrgRiskThresholds packages all configurable decision constants for a tenant.
score_control_with_thresholds() is a drop-in replacement for score_control()
that uses org-specific values.  The 5x asymmetric penalty *structure* is
preserved; orgs only change the penalty_ratio *value*.

All computation is deterministic.  No LLM involvement.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Optional

from engine.risk.scorer import (
    AuditConclusion,
    EvidenceItem,
    MATERIALITY_THRESHOLD,
    PENALTY_RATIO,
    RiskScoreResult,
    TDR_MATERIAL,
    TDR_NON_MATERIAL,
    bayesian_update,
    compute_deviation_rate,
)


# ---------------------------------------------------------------------------
# Article-level aggregation
# ---------------------------------------------------------------------------

@dataclass
class ControlMeta:
    """Metadata required to aggregate a control to article level."""

    control_id: str
    article_ref: str     # e.g. "NIS2-21a" or "21a"
    risk_weight: float   # 0.0 – 5.0


@dataclass
class ArticleRiskScore:
    """Aggregated risk score for one NIS 2 Article 21 sub-clause."""

    article_ref: str
    weighted_posterior: float           # risk-weight-weighted avg of posteriors
    worst_conclusion: AuditConclusion   # worst-case conclusion among controls
    control_count: int
    material_control_count: int
    traceability: dict[str, Any] = field(default_factory=dict)


_CONCLUSION_SEVERITY: dict[AuditConclusion, int] = {
    AuditConclusion.EFFECTIVE: 0,
    AuditConclusion.PARTIALLY_EFFECTIVE: 1,
    AuditConclusion.NOT_EFFECTIVE: 2,
}


def _extract_article_key(article_ref: str) -> str:
    """Normalise article reference to bare sub-clause key (e.g. '21a')."""
    return (
        article_ref.strip()
        .upper()
        .replace("NIS2-", "")
        .replace("NIS-2-", "")
        .lower()
    )


def aggregate_by_article(
    results: list[RiskScoreResult],
    metadata: list[ControlMeta],
) -> dict[str, ArticleRiskScore]:
    """Roll up per-control scores to NIS 2 Article 21 sub-clause level.

    Args:
        results:  Per-control RiskScoreResult instances (one per control).
        metadata: ControlMeta for each control (same length as results,
                  same positional order).

    Returns:
        Dict mapping normalised article key (e.g. "21a") to ArticleRiskScore.

    Raises:
        ValueError: if results and metadata have different lengths.
    """
    if len(results) != len(metadata):
        raise ValueError(
            f"results and metadata must have equal length; "
            f"got {len(results)} result(s) vs {len(metadata)} metadata item(s)"
        )

    # Group by article
    groups: dict[str, list[tuple[RiskScoreResult, ControlMeta]]] = {}
    for result, meta in zip(results, metadata):
        key = _extract_article_key(meta.article_ref)
        groups.setdefault(key, []).append((result, meta))

    article_scores: dict[str, ArticleRiskScore] = {}
    for article_key, items in groups.items():
        total_weight = sum(m.risk_weight for _, m in items) or 1.0  # avoid /0
        weighted_posterior = sum(
            r.posterior * m.risk_weight for r, m in items
        ) / total_weight

        worst_severity = max(_CONCLUSION_SEVERITY[r.conclusion] for r, _ in items)
        worst_conclusion = next(
            c for c, s in _CONCLUSION_SEVERITY.items() if s == worst_severity
        )
        material_count = sum(1 for r, _ in items if r.is_material)

        article_scores[article_key] = ArticleRiskScore(
            article_ref=article_key,
            weighted_posterior=round(weighted_posterior, 4),
            worst_conclusion=worst_conclusion,
            control_count=len(items),
            material_control_count=material_count,
            traceability={
                "control_ids": [r.control_id for r, _ in items],
                "weights": {r.control_id: m.risk_weight for r, m in items},
                "individual_posteriors": {
                    r.control_id: round(r.posterior, 4) for r, _ in items
                },
            },
        )

    return article_scores


# ---------------------------------------------------------------------------
# Risk Trending
# ---------------------------------------------------------------------------

class TrendDirection(str, Enum):
    IMPROVING = "improving"
    DETERIORATING = "deteriorating"
    STABLE = "stable"
    INSUFFICIENT_DATA = "insufficient_data"


_STABLE_DELTA: float = 0.05  # |delta posterior| ≤ this = no material change


@dataclass
class PeriodScore:
    """Aggregated risk scores for one engagement period."""

    period_id: str
    article_scores: dict[str, float]   # article_key → weighted_posterior
    period_label: Optional[str] = None  # noqa: UP007


@dataclass
class ArticleTrend:
    """Trend for one NIS 2 article across two consecutive periods."""

    article_ref: str
    previous_score: float
    current_score: float
    delta: float                # current − previous (positive = improving)
    direction: TrendDirection
    percent_change: float       # (delta / previous) × 100; 0.0 if previous=0


@dataclass
class RiskTrend:
    """Full trend report comparing two engagement periods."""

    period_current: str
    period_previous: str
    article_trends: dict[str, ArticleTrend]
    overall_direction: TrendDirection
    improved_count: int
    deteriorated_count: int
    stable_count: int


def compute_risk_trend(current: PeriodScore, previous: PeriodScore) -> RiskTrend:
    """Compute risk trend between two consecutive engagement periods.

    Higher posterior = more effective controls = improving risk posture.

    Args:
        current:  Latest period’s aggregated scores.
        previous: Prior period’s aggregated scores (the baseline).

    Returns:
        RiskTrend with per-article direction and an overall direction summary.
    """
    article_trends: dict[str, ArticleTrend] = {}
    improved = deteriorated = stable = 0

    all_articles = sorted(set(current.article_scores) | set(previous.article_scores))
    if not all_articles:
        return RiskTrend(
            period_current=current.period_id,
            period_previous=previous.period_id,
            article_trends={},
            overall_direction=TrendDirection.INSUFFICIENT_DATA,
            improved_count=0,
            deteriorated_count=0,
            stable_count=0,
        )

    for article in all_articles:
        cur_score = current.article_scores.get(article, 0.0)
        prev_score = previous.article_scores.get(article, 0.0)
        delta = cur_score - prev_score

        if abs(delta) <= _STABLE_DELTA:
            direction = TrendDirection.STABLE
            stable += 1
        elif delta > 0:
            direction = TrendDirection.IMPROVING
            improved += 1
        else:
            direction = TrendDirection.DETERIORATING
            deteriorated += 1

        pct_change = (delta / prev_score * 100.0) if prev_score != 0.0 else 0.0

        article_trends[article] = ArticleTrend(
            article_ref=article,
            previous_score=round(prev_score, 4),
            current_score=round(cur_score, 4),
            delta=round(delta, 4),
            direction=direction,
            percent_change=round(pct_change, 2),
        )

    # Overall: majority wins; ties resolve to STABLE
    if improved > deteriorated and improved > stable:
        overall = TrendDirection.IMPROVING
    elif deteriorated > improved and deteriorated > stable:
        overall = TrendDirection.DETERIORATING
    else:
        overall = TrendDirection.STABLE

    return RiskTrend(
        period_current=current.period_id,
        period_previous=previous.period_id,
        article_trends=article_trends,
        overall_direction=overall,
        improved_count=improved,
        deteriorated_count=deteriorated,
        stable_count=stable,
    )


# ---------------------------------------------------------------------------
# Per-organisation risk thresholds
# ---------------------------------------------------------------------------

@dataclass
class OrgRiskThresholds:
    """Customisable risk thresholds per organisation.

    Encapsulates all decision constants so tenants can express their risk
    appetite without modifying global module state.

    All values default to the production constants in scorer.py.
    The 5x asymmetric penalty *structure* is preserved; penalty_ratio only
    changes the *value*, not the mathematical form.
    """

    org_id: str
    penalty_ratio: float = PENALTY_RATIO              # default 5.0
    materiality_threshold: float = MATERIALITY_THRESHOLD  # default 3.5
    tdr_material: float = TDR_MATERIAL                # default 0.05
    tdr_non_material: float = TDR_NON_MATERIAL        # default 0.10

    @property
    def threshold_effective(self) -> float:
        """Asymmetric effective threshold derived from penalty_ratio."""
        return 1.0 - (1.0 / (1.0 + self.penalty_ratio))

    @property
    def threshold_partial(self) -> float:
        """Asymmetric partial threshold derived from penalty_ratio."""
        return 1.0 - (1.0 / (1.0 + self.penalty_ratio / 2.0))


def score_control_with_thresholds(
    control_id: str,
    risk_weight: float,
    prior: float,
    evidence_items: list[EvidenceItem],
    org_thresholds: OrgRiskThresholds,
) -> RiskScoreResult:
    """Score a control using organisation-specific risk thresholds.

    Drop-in replacement for score_control() that accepts an OrgRiskThresholds
    instance instead of relying on module-level constants.

    The Bayesian update algorithm and evidence weighting are identical to
    score_control(); only the decision thresholds and TDR change.

    Args:
        control_id:     Control identifier string.
        risk_weight:    Control risk weight (0.0 – 5.0).
        prior:          Prior probability of effectiveness (0.0 – 1.0).
        evidence_items: Evidence list.
        org_thresholds: Organisation-specific threshold configuration.

    Returns:
        Deterministic RiskScoreResult using org-specific constants.
    """
    if not evidence_items:
        is_mat = risk_weight >= org_thresholds.materiality_threshold
        return RiskScoreResult(
            control_id=control_id,
            prior=prior,
            posterior=prior,
            confidence=0.0,
            conclusion=AuditConclusion.NOT_EFFECTIVE,
            is_material=is_mat,
            tdr_threshold=(
                org_thresholds.tdr_material if is_mat else org_thresholds.tdr_non_material
            ),
            observed_deviation_rate=0.0,
            tdr_exceeded=False,
            evidence_count=0,
            penalty_applied=True,
            traceability={
                "reason": "no_evidence",
                "penalty_ratio": org_thresholds.penalty_ratio,
                "org_id": org_thresholds.org_id,
            },
        )

    posterior = bayesian_update(prior, evidence_items)
    is_material = risk_weight >= org_thresholds.materiality_threshold
    tdr = org_thresholds.tdr_material if is_material else org_thresholds.tdr_non_material
    dev_rate = compute_deviation_rate(evidence_items)
    tdr_exceeded = dev_rate > tdr

    t_eff = org_thresholds.threshold_effective
    t_par = org_thresholds.threshold_partial

    if posterior >= t_eff and not tdr_exceeded:
        conclusion = AuditConclusion.EFFECTIVE
    elif posterior >= t_par and not tdr_exceeded:
        conclusion = AuditConclusion.PARTIALLY_EFFECTIVE
    else:
        conclusion = AuditConclusion.NOT_EFFECTIVE

    if conclusion == AuditConclusion.EFFECTIVE:
        confidence = min(1.0, (posterior - t_eff) / (1.0 - t_eff))
    elif conclusion == AuditConclusion.PARTIALLY_EFFECTIVE:
        confidence = min(1.0, (posterior - t_par) / (t_eff - t_par))
    else:
        confidence = min(1.0, 1.0 - posterior)

    return RiskScoreResult(
        control_id=control_id,
        prior=prior,
        posterior=posterior,
        confidence=round(confidence, 4),
        conclusion=conclusion,
        is_material=is_material,
        tdr_threshold=tdr,
        observed_deviation_rate=round(dev_rate, 4),
        tdr_exceeded=tdr_exceeded,
        evidence_count=len(evidence_items),
        penalty_applied=True,
        traceability={
            "threshold_effective": t_eff,
            "threshold_partial": t_par,
            "penalty_ratio": org_thresholds.penalty_ratio,
            "org_id": org_thresholds.org_id,
            "evidence_ids": [str(e.evidence_id) for e in evidence_items],
        },
    )

