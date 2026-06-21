"""Quality Engine — Quality score trending across engagement periods.

Trend analysis tracks how a control's sufficiency ratio changes across
successive audit engagement periods.  Direction is determined by comparing the
most recent period against the earliest period.

Trend direction thresholds
--------------------------
  delta > +0.05  -> IMPROVING
  delta < -0.05  -> DECLINING
  else           -> STABLE

All computations are deterministic.  No LLM is consulted.
"""
from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

from engine.quality.sampler import SufficiencyVerdict

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

TREND_STABLE_THRESHOLD: float = 0.05  # ±5% change is considered stable


# ---------------------------------------------------------------------------
# Enums & data models
# ---------------------------------------------------------------------------


class TrendDirection(StrEnum):
    IMPROVING = "improving"
    DECLINING = "declining"
    STABLE = "stable"
    INSUFFICIENT_DATA = "insufficient_data"  # fewer than 2 periods


@dataclass
class PeriodScore:
    """Quality score snapshot for one engagement period."""

    period_id: str              # e.g. "Q1-2024", "FY2024", "2024-01"
    sufficiency_ratio: float    # 0.0-1.0
    evidence_count: int
    verdict: SufficiencyVerdict


@dataclass
class QualityTrend:
    """Quality score trend for one control across multiple engagement periods."""

    control_id: str
    periods: list[PeriodScore]  # ordered oldest -> newest
    direction: TrendDirection
    delta: float                # latest.sufficiency_ratio - earliest.sufficiency_ratio


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def compute_trend(
    control_id: str,
    period_scores: list[PeriodScore],
) -> QualityTrend:
    """Compute trend direction from an ordered list of period scores.

    Parameters
    ----------
    control_id:
        Identifier of the control being assessed.
    period_scores:
        Ordered list of period scores (oldest first).  Must not be empty.

    Returns
    -------
    QualityTrend
        Trend with direction and delta.  When fewer than 2 periods are
        provided, direction is ``INSUFFICIENT_DATA`` and delta is ``0.0``.

    Raises
    ------
    ValueError
        If *period_scores* is empty.
    """
    if not period_scores:
        raise ValueError("period_scores must not be empty")

    if len(period_scores) < 2:
        return QualityTrend(
            control_id=control_id,
            periods=list(period_scores),
            direction=TrendDirection.INSUFFICIENT_DATA,
            delta=0.0,
        )

    earliest = period_scores[0].sufficiency_ratio
    latest = period_scores[-1].sufficiency_ratio
    delta = round(latest - earliest, 4)

    if delta > TREND_STABLE_THRESHOLD:
        direction = TrendDirection.IMPROVING
    elif delta < -TREND_STABLE_THRESHOLD:
        direction = TrendDirection.DECLINING
    else:
        direction = TrendDirection.STABLE

    return QualityTrend(
        control_id=control_id,
        periods=list(period_scores),
        direction=direction,
        delta=delta,
    )


def add_period_score(
    trend: QualityTrend,
    new_period: PeriodScore,
) -> QualityTrend:
    """Return a new QualityTrend with *new_period* appended and direction recomputed.

    Parameters
    ----------
    trend:
        Existing trend.  Unchanged — a new object is returned.
    new_period:
        Period score to append (assumed to be newer than all existing periods).

    Returns
    -------
    QualityTrend
        Updated trend.
    """
    updated_periods = list(trend.periods) + [new_period]
    return compute_trend(trend.control_id, updated_periods)
