"""Quality Engine — Conflicting evidence detection and resolution.

When two evidence items for the same control contradict each other the
resolver flags the conflict and resolves it deterministically by preferring
the item with the higher tier weight (lower tier number), breaking ties by
higher quality score.

Conflict types
--------------
PASS_FAIL_DISAGREEMENT
    One item has ``passes=True``, the other ``passes=False``.
QUALITY_VARIANCE
    Both items agree on pass/fail but their quality scores differ by more
    than ``quality_variance_threshold`` (default 0.30).

All computations are deterministic.  No LLM is consulted.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum
from uuid import UUID

from engine.quality.sampler import TIER_WEIGHTS, EvidenceQualityInput

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

QUALITY_VARIANCE_THRESHOLD: float = 0.30


# ---------------------------------------------------------------------------
# Enums & data models
# ---------------------------------------------------------------------------


class ConflictType(StrEnum):
    PASS_FAIL_DISAGREEMENT = "pass_fail_disagreement"
    QUALITY_VARIANCE = "quality_variance"


@dataclass
class EvidenceConflict:
    """A detected conflict between two evidence items."""

    evidence_a_id: UUID
    evidence_b_id: UUID
    conflict_type: ConflictType
    tier_a: int
    tier_b: int
    quality_a: float
    quality_b: float
    preferred_id: UUID   # deterministic winner (higher tier-weight, tie-break quality)
    flagged_id: UUID     # item that was overridden


@dataclass
class ConflictResolutionResult:
    """Outcome of conflict detection and resolution for one control."""

    control_id: str
    total_items: int
    conflicts: list[EvidenceConflict] = field(default_factory=list)
    flagged_item_ids: list[UUID] = field(default_factory=list)
    accepted_items: list[EvidenceQualityInput] = field(default_factory=list)

    @property
    def conflict_count(self) -> int:
        """Number of detected conflicts."""
        return len(self.conflicts)

    @property
    def has_conflicts(self) -> bool:
        """True when at least one conflict was detected."""
        return bool(self.conflicts)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _prefer(
    a: EvidenceQualityInput,
    b: EvidenceQualityInput,
) -> EvidenceQualityInput:
    """Return the stronger item (lower tier number -> higher weight; tie-break by quality)."""
    weight_a = TIER_WEIGHTS.get(a.tier, 0.0)
    weight_b = TIER_WEIGHTS.get(b.tier, 0.0)
    if weight_a > weight_b:
        return a
    if weight_b > weight_a:
        return b
    # Same tier — prefer higher quality score (a wins on exact tie)
    return a if a.quality_score >= b.quality_score else b


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def detect_conflicts(
    items: list[EvidenceQualityInput],
    quality_variance_threshold: float = QUALITY_VARIANCE_THRESHOLD,
) -> list[EvidenceConflict]:
    """Detect conflicts between all pairs of evidence items for one control.

    Parameters
    ----------
    items:
        Evidence items for a single control.
    quality_variance_threshold:
        Minimum absolute quality-score difference required to flag a
        ``QUALITY_VARIANCE`` conflict.  Default: 0.30.

    Returns
    -------
    list[EvidenceConflict]
        Detected conflicts (may be empty).  Pairs are checked in index
        order so output is deterministic.
    """
    conflicts: list[EvidenceConflict] = []
    n = len(items)
    for i in range(n):
        for j in range(i + 1, n):
            a, b = items[i], items[j]
            if a.passes != b.passes:
                winner = _prefer(a, b)
                loser = b if winner.evidence_id == a.evidence_id else a
                conflicts.append(
                    EvidenceConflict(
                        evidence_a_id=a.evidence_id,
                        evidence_b_id=b.evidence_id,
                        conflict_type=ConflictType.PASS_FAIL_DISAGREEMENT,
                        tier_a=a.tier,
                        tier_b=b.tier,
                        quality_a=a.quality_score,
                        quality_b=b.quality_score,
                        preferred_id=winner.evidence_id,
                        flagged_id=loser.evidence_id,
                    )
                )
            elif abs(a.quality_score - b.quality_score) >= quality_variance_threshold:
                winner = _prefer(a, b)
                loser = b if winner.evidence_id == a.evidence_id else a
                conflicts.append(
                    EvidenceConflict(
                        evidence_a_id=a.evidence_id,
                        evidence_b_id=b.evidence_id,
                        conflict_type=ConflictType.QUALITY_VARIANCE,
                        tier_a=a.tier,
                        tier_b=b.tier,
                        quality_a=a.quality_score,
                        quality_b=b.quality_score,
                        preferred_id=winner.evidence_id,
                        flagged_id=loser.evidence_id,
                    )
                )
    return conflicts


def resolve_conflicts(
    control_id: str,
    items: list[EvidenceQualityInput],
    quality_variance_threshold: float = QUALITY_VARIANCE_THRESHOLD,
) -> ConflictResolutionResult:
    """Detect and resolve conflicts in *items* for *control_id*.

    Resolution strategy
    -------------------
    For each conflict pair, the weaker item is flagged.  An item is excluded
    from the accepted set if it is flagged in *any* conflict.  The accepted
    set is used for downstream sufficiency and gate calculations.

    Parameters
    ----------
    control_id:
        Identifier of the control being assessed.
    items:
        Evidence items for the control.
    quality_variance_threshold:
        Quality variance threshold forwarded to ``detect_conflicts``.

    Returns
    -------
    ConflictResolutionResult
    """
    conflicts = detect_conflicts(items, quality_variance_threshold)
    flagged_ids: set[UUID] = {c.flagged_id for c in conflicts}
    accepted = [it for it in items if it.evidence_id not in flagged_ids]
    return ConflictResolutionResult(
        control_id=control_id,
        total_items=len(items),
        conflicts=conflicts,
        flagged_item_ids=list(flagged_ids),
        accepted_items=accepted,
    )
