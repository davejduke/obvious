"""Quality Engine — Cochran attribute sampling, 6-tier evidence hierarchy, sufficiency.

All computations are deterministic. No LLM is consulted.

Cochran Formula (attribute sampling)
-------------------------------------
For large populations (N → ∞):
    n₀ = (Z² * p * q) / e²

Finite population correction:
    n = n₀ / (1 + (n₀ - 1) / N)

Where:
    Z  = Z-score for confidence level (1.645 @ 90%, 1.960 @ 95%, 2.576 @ 99%)
    p  = expected deviation rate (proportion of defective items)
    q  = 1 - p
    e  = tolerable deviation rate (precision / margin of error)
    N  = population size

Evidence Tier Hierarchy (1 = strongest, 6 = weakest)
-----------------------------------------------------
Tier 1: Automated system-generated log (e.g. SIEM export)          weight = 1.00
Tier 2: Third-party audit report / penetration test                 weight = 0.85
Tier 3: Configuration export / screenshot with metadata             weight = 0.70
Tier 4: Policy / procedure document                                 weight = 0.55
Tier 5: Interview transcript / walkthrough notes                    weight = 0.40
Tier 6: Management assertion / self-certification                   weight = 0.25

Sufficiency Formula
--------------------
    sufficiency = (Σ(tier_weight_i * quality_score_i)) / required_sample_size
    capped at 1.0; < 0.6 → insufficient; 0.6–0.8 → marginal; > 0.8 → sufficient
"""
from __future__ import annotations

import math
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional
from uuid import UUID


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

# Z-scores for common confidence levels
Z_SCORES: dict[float, float] = {
    0.90: 1.6449,
    0.95: 1.9600,
    0.99: 2.5758,
}

# Evidence tier weights (1 = strongest)
TIER_WEIGHTS: dict[int, float] = {
    1: 1.00,
    2: 0.85,
    3: 0.70,
    4: 0.55,
    5: 0.40,
    6: 0.25,
}

# Sufficiency thresholds
SUFFICIENCY_INSUFFICIENT = 0.60
SUFFICIENCY_MARGINAL = 0.80


class SufficiencyVerdict(str, Enum):
    SUFFICIENT = "sufficient"
    MARGINAL = "marginal"
    INSUFFICIENT = "insufficient"


# ---------------------------------------------------------------------------
# Data models
# ---------------------------------------------------------------------------

@dataclass
class SamplingParameters:
    """Input parameters for Cochran sample size calculation."""
    confidence_level: float   # 0.90, 0.95, or 0.99
    expected_deviation_rate: float  # p — expected error rate in population
    tolerable_deviation_rate: float  # e — maximum acceptable error rate (TDR)
    population_size: Optional[int] = None  # N; None = treat as infinite


@dataclass
class SampleSizeResult:
    """Result of Cochran sample size calculation."""
    n_infinite: int           # n₀ before finite population correction
    n_final: int              # n after finite population correction (if applicable)
    confidence_level: float
    expected_deviation_rate: float
    tolerable_deviation_rate: float
    population_size: Optional[int]
    z_score: float
    finite_correction_applied: bool


@dataclass
class EvidenceQualityInput:
    """Evidence item for quality scoring."""
    evidence_id: UUID
    tier: int                 # 1–6
    quality_score: float      # 0.0–1.0 from upstream quality assessment
    passes: bool = True


@dataclass
class SufficiencyResult:
    """Evidence sufficiency assessment for one control."""
    control_id: str
    raw_score: float
    required_sample_size: int
    evidence_count: int
    weighted_score_sum: float
    sufficiency_ratio: float  # weighted_score_sum / required_sample_size
    verdict: SufficiencyVerdict
    tier_distribution: dict[int, int] = field(default_factory=dict)


# ---------------------------------------------------------------------------
# Cochran formula implementation
# ---------------------------------------------------------------------------

def cochran_sample_size(params: SamplingParameters) -> SampleSizeResult:
    """Compute required sample size using the Cochran (1977) formula.

    Raises
    ------
    ValueError
        If confidence level is not supported or parameters are invalid.
    """
    cl = params.confidence_level
    if cl not in Z_SCORES:
        supported = sorted(Z_SCORES.keys())
        raise ValueError(
            f"Confidence level {cl} not supported. Use one of: {supported}"
        )
    p = params.expected_deviation_rate
    e = params.tolerable_deviation_rate
    if not (0.0 < p < 1.0):
        raise ValueError(f"expected_deviation_rate must be in (0, 1); got {p}")
    if not (0.0 < e < 1.0):
        raise ValueError(f"tolerable_deviation_rate must be in (0, 1); got {e}")
    if e <= p:
        raise ValueError(
            f"tolerable_deviation_rate ({e}) must be > expected_deviation_rate ({p})"
        )

    z = Z_SCORES[cl]
    q = 1.0 - p

    # Cochran infinite population formula
    n0_raw = (z ** 2 * p * q) / (e ** 2)
    n0 = math.ceil(n0_raw)

    n_final = n0
    finite_applied = False

    if params.population_size is not None:
        n_pop = params.population_size
        if n_pop <= 0:
            raise ValueError(f"population_size must be positive; got {n_pop}")
        # Finite population correction
        n_corrected = n0 / (1 + (n0 - 1) / n_pop)
        n_final = max(1, math.ceil(n_corrected))
        finite_applied = True

    return SampleSizeResult(
        n_infinite=n0,
        n_final=n_final,
        confidence_level=cl,
        expected_deviation_rate=p,
        tolerable_deviation_rate=e,
        population_size=params.population_size,
        z_score=z,
        finite_correction_applied=finite_applied,
    )


# ---------------------------------------------------------------------------
# Evidence sufficiency
# ---------------------------------------------------------------------------

def score_evidence_tier(tier: int, quality_score: float) -> float:
    """Compute weighted score for a single evidence item.

    score = TIER_WEIGHTS[tier] * quality_score
    """
    if tier not in TIER_WEIGHTS:
        raise ValueError(f"Evidence tier must be 1–6; got {tier}")
    if not (0.0 <= quality_score <= 1.0):
        raise ValueError(f"quality_score must be in [0, 1]; got {quality_score}")
    return TIER_WEIGHTS[tier] * quality_score


def assess_sufficiency(
    control_id: str,
    evidence_items: list[EvidenceQualityInput],
    required_sample_size: int,
) -> SufficiencyResult:
    """Compute evidence sufficiency for a control.

    sufficiency_ratio = sum(tier_weight * quality_score) / required_sample_size
    Verdict: >= 0.8 → sufficient, 0.6–0.8 → marginal, < 0.6 → insufficient
    """
    if required_sample_size <= 0:
        raise ValueError(f"required_sample_size must be positive; got {required_sample_size}")

    tier_dist: dict[int, int] = {}
    weighted_sum = 0.0

    for item in evidence_items:
        weighted_sum += score_evidence_tier(item.tier, item.quality_score)
        tier_dist[item.tier] = tier_dist.get(item.tier, 0) + 1

    ratio = weighted_sum / required_sample_size

    if ratio >= SUFFICIENCY_MARGINAL:
        verdict = SufficiencyVerdict.SUFFICIENT
    elif ratio >= SUFFICIENCY_INSUFFICIENT:
        verdict = SufficiencyVerdict.MARGINAL
    else:
        verdict = SufficiencyVerdict.INSUFFICIENT

    return SufficiencyResult(
        control_id=control_id,
        raw_score=round(weighted_sum, 4),
        required_sample_size=required_sample_size,
        evidence_count=len(evidence_items),
        weighted_score_sum=round(weighted_sum, 4),
        sufficiency_ratio=round(min(1.0, ratio), 4),
        verdict=verdict,
        tier_distribution=tier_dist,
    )
