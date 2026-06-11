"""Risk Engine — Bayesian scoring with 5x asymmetric false-negative penalty.

All scoring is deterministic.  No LLM is consulted.

Bayesian update
---------------
  posterior = (likelihood * prior) / normaliser
  likelihood_pass  = P(evidence_supports | control_effective)
  likelihood_fail  = P(evidence_supports | control_not_effective)

5x Asymmetric Penalty
---------------------
False negatives (concluding "Effective" when actually not) are 5x more costly
than false positives (concluding "Not Effective" when actually effective).
Applied as a loss-weighted threshold shift:
  adjusted_threshold_effective = 1 - (1 / (1 + PENALTY_RATIO))   = 0.833…
  adjusted_threshold_partial   = 1 - (1 / (1 + PENALTY_RATIO/2)) = 0.714…

Materiality
-----------
Controls with risk_weight >= MATERIALITY_THRESHOLD (default 3.5) are material.
Material controls trigger lower TDR (Tolerable Deviation Rate) thresholds.

TDR Thresholds
--------------
  Material controls:     TDR_MATERIAL   = 0.05  (5%)
  Non-material controls: TDR_NON_MATERIAL = 0.10 (10%)
"""
from __future__ import annotations

import math
from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Optional
from uuid import UUID


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

PENALTY_RATIO: float = 5.0  # false negatives cost 5x more than false positives
MATERIALITY_THRESHOLD: float = 3.5
TDR_MATERIAL: float = 0.05
TDR_NON_MATERIAL: float = 0.10

# Adjusted thresholds accounting for 5x penalty
# P(effective | evidence) must exceed these to conclude Effective / Partial
_THRESHOLD_EFFECTIVE = 1.0 - (1.0 / (1.0 + PENALTY_RATIO))        # ≈ 0.8333
_THRESHOLD_PARTIAL   = 1.0 - (1.0 / (1.0 + PENALTY_RATIO / 2.0))  # ≈ 0.7143

# Likelihoods: given the control IS effective, how likely is a positive evidence item?
_LIKELIHOOD_PASS_GIVEN_EFFECTIVE: float = 0.92
_LIKELIHOOD_PASS_GIVEN_INEFFECTIVE: float = 0.15


# ---------------------------------------------------------------------------
# Enumerations & data models
# ---------------------------------------------------------------------------

class AuditConclusion(str, Enum):
    EFFECTIVE = "Effective"
    PARTIALLY_EFFECTIVE = "Partially Effective"
    NOT_EFFECTIVE = "Not Effective"


@dataclass
class EvidenceItem:
    """A single piece of evidence submitted for scoring."""
    evidence_id: UUID
    passes: bool          # True = supports control effectiveness
    quality_score: float  # 0.0 – 1.0 from Quality Engine
    tier: int             # 1 (strongest) – 6 (weakest)


@dataclass
class RiskScoreResult:
    """Full output from the Risk Engine for one control."""
    control_id: str
    prior: float
    posterior: float
    confidence: float          # 0.0 – 1.0
    conclusion: AuditConclusion
    is_material: bool
    tdr_threshold: float
    observed_deviation_rate: float
    tdr_exceeded: bool
    evidence_count: int
    penalty_applied: bool
    traceability: dict[str, Any] = field(default_factory=dict)


# ---------------------------------------------------------------------------
# Core algorithms
# ---------------------------------------------------------------------------

def bayesian_update(
    prior: float,
    evidence_items: list[EvidenceItem],
) -> float:
    """Compute posterior P(control_effective | evidence) via Bayesian update.

    Each evidence item independently updates the belief using:
        P(E|H)   = LIKELIHOOD_PASS_GIVEN_EFFECTIVE   if item.passes
        P(E|¬H)  = LIKELIHOOD_PASS_GIVEN_INEFFECTIVE  if item.passes
    (inverse likelihoods for failing items)

    Quality score acts as a weight: items with quality < 0.5 contribute
    reduced update strength (linearly interpolated).
    """
    if not 0.0 <= prior <= 1.0:
        raise ValueError(f"Prior must be in [0, 1]; got {prior}")

    posterior = prior
    for item in evidence_items:
        # Scale likelihoods by quality score to down-weight low-quality evidence
        q = max(0.0, min(1.0, item.quality_score))
        # Tier weight: tier 1 = 1.0, tier 6 = 0.5
        tier_weight = 1.0 - (item.tier - 1) * 0.1
        weight = q * tier_weight

        if item.passes:
            l_h  = _LIKELIHOOD_PASS_GIVEN_EFFECTIVE * weight + (1 - weight) * 0.5
            l_nh = _LIKELIHOOD_PASS_GIVEN_INEFFECTIVE * weight + (1 - weight) * 0.5
        else:
            # Failing evidence: flip likelihoods
            l_h  = (1 - _LIKELIHOOD_PASS_GIVEN_EFFECTIVE) * weight + (1 - weight) * 0.5
            l_nh = (1 - _LIKELIHOOD_PASS_GIVEN_INEFFECTIVE) * weight + (1 - weight) * 0.5

        # Bayes rule
        numerator = l_h * posterior
        denominator = l_h * posterior + l_nh * (1 - posterior)
        if denominator == 0.0:
            continue
        posterior = numerator / denominator

    return max(0.0, min(1.0, posterior))


def compute_deviation_rate(evidence_items: list[EvidenceItem]) -> float:
    """Observed deviation rate = failed_items / total_items."""
    if not evidence_items:
        return 0.0
    failures = sum(1 for e in evidence_items if not e.passes)
    return failures / len(evidence_items)


def score_control(
    control_id: str,
    risk_weight: float,
    prior: float,
    evidence_items: list[EvidenceItem],
) -> RiskScoreResult:
    """Run the full Risk Engine scoring pipeline for one control.

    Steps:
    1. Bayesian posterior update over all evidence
    2. Materiality determination
    3. TDR threshold selection and check
    4. 5x asymmetric threshold application for conclusion
    5. Confidence derived from posterior distance from nearest threshold

    Returns deterministic RiskScoreResult.
    """
    if not evidence_items:
        # No evidence → treat as Not Effective with low confidence
        return RiskScoreResult(
            control_id=control_id,
            prior=prior,
            posterior=prior,
            confidence=0.0,
            conclusion=AuditConclusion.NOT_EFFECTIVE,
            is_material=risk_weight >= MATERIALITY_THRESHOLD,
            tdr_threshold=TDR_MATERIAL if risk_weight >= MATERIALITY_THRESHOLD else TDR_NON_MATERIAL,
            observed_deviation_rate=0.0,
            tdr_exceeded=False,
            evidence_count=0,
            penalty_applied=True,
            traceability={"reason": "no_evidence", "penalty_ratio": PENALTY_RATIO},
        )

    posterior = bayesian_update(prior, evidence_items)
    is_material = risk_weight >= MATERIALITY_THRESHOLD
    tdr = TDR_MATERIAL if is_material else TDR_NON_MATERIAL
    dev_rate = compute_deviation_rate(evidence_items)
    tdr_exceeded = dev_rate > tdr

    # Determine conclusion using 5x asymmetric thresholds
    if posterior >= _THRESHOLD_EFFECTIVE and not tdr_exceeded:
        conclusion = AuditConclusion.EFFECTIVE
    elif posterior >= _THRESHOLD_PARTIAL and not tdr_exceeded:
        conclusion = AuditConclusion.PARTIALLY_EFFECTIVE
    else:
        conclusion = AuditConclusion.NOT_EFFECTIVE

    # Confidence: distance from the active threshold boundary, normalised
    if conclusion == AuditConclusion.EFFECTIVE:
        confidence = min(1.0, (posterior - _THRESHOLD_EFFECTIVE) / (1.0 - _THRESHOLD_EFFECTIVE))
    elif conclusion == AuditConclusion.PARTIALLY_EFFECTIVE:
        confidence = min(1.0, (posterior - _THRESHOLD_PARTIAL) / (_THRESHOLD_EFFECTIVE - _THRESHOLD_PARTIAL))
    else:
        confidence = min(1.0, (1.0 - posterior) / (1.0 - 0.0))

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
            "threshold_effective": _THRESHOLD_EFFECTIVE,
            "threshold_partial": _THRESHOLD_PARTIAL,
            "penalty_ratio": PENALTY_RATIO,
            "evidence_ids": [str(e.evidence_id) for e in evidence_items],
        },
    )
