"""Quality Engine — Cross-validation of evidence from independent sources.

Cross-validation compares evidence items that test the same assertion but
originate from different independent sources.  Inconsistency between sources
is flagged so auditors can investigate.

All computations are deterministic.  No LLM is consulted.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from uuid import UUID

# ---------------------------------------------------------------------------
# Data models
# ---------------------------------------------------------------------------


@dataclass
class AnnotatedEvidenceItem:
    """Evidence item extended with source and assertion metadata."""

    evidence_id: UUID
    tier: int                 # 1-6
    quality_score: float      # 0.0-1.0
    passes: bool              # does this evidence indicate compliance?
    source_id: str            # independent source identifier (e.g. "SIEM-01")
    assertion_key: str        # what this evidence tests (e.g. "MFA_enabled")


@dataclass
class SourceAggregation:
    """Aggregated evidence view for one (assertion_key, source_id) pair."""

    source_id: str
    assertion_key: str
    passes_votes: int         # items with passes=True
    fails_votes: int          # items with passes=False
    avg_quality: float        # mean quality_score across items
    majority_passes: bool     # True if passes_votes >= fails_votes (tie -> passes)


@dataclass
class CrossValidationResult:
    """Cross-validation outcome for one assertion key."""

    assertion_key: str
    source_count: int
    consistent: bool          # all sources agree with majority verdict
    consistency_score: float  # fraction of sources that agree with majority (0.0-1.0)
    majority_passes: bool     # overall majority verdict
    sources: list[SourceAggregation] = field(default_factory=list)
    inconsistent_source_ids: list[str] = field(default_factory=list)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _aggregate_source(
    source_id: str,
    assertion_key: str,
    items: list[AnnotatedEvidenceItem],
) -> SourceAggregation:
    passes_count = sum(1 for it in items if it.passes)
    fails_count = len(items) - passes_count
    avg_quality = sum(it.quality_score for it in items) / len(items)
    majority_passes = passes_count >= fails_count  # tie -> passes
    return SourceAggregation(
        source_id=source_id,
        assertion_key=assertion_key,
        passes_votes=passes_count,
        fails_votes=fails_count,
        avg_quality=round(avg_quality, 4),
        majority_passes=majority_passes,
    )


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def cross_validate_evidence(
    items: list[AnnotatedEvidenceItem],
) -> list[CrossValidationResult]:
    """Perform cross-validation across all assertion keys found in *items*.

    For each unique ``assertion_key``:
    1. Group items by ``source_id``.
    2. Determine the majority verdict for each source.
    3. Compare source verdicts — flag sources that disagree with the overall
       majority as inconsistent.

    Parameters
    ----------
    items:
        Annotated evidence items.  Items with the same ``assertion_key`` but
        different ``source_id`` values are compared.

    Returns
    -------
    list[CrossValidationResult]
        One result per unique ``assertion_key``.  Returns an empty list when
        *items* is empty.
    """
    if not items:
        return []

    # Group by assertion_key -> source_id -> [items]
    by_assertion: dict[str, dict[str, list[AnnotatedEvidenceItem]]] = {}
    for item in items:
        by_assertion.setdefault(item.assertion_key, {}).setdefault(
            item.source_id, []
        ).append(item)

    results: list[CrossValidationResult] = []

    for assertion_key, by_source in by_assertion.items():
        aggregations = [
            _aggregate_source(src, assertion_key, src_items)
            for src, src_items in by_source.items()
        ]

        # Overall majority verdict across sources
        majority_passes_count = sum(1 for a in aggregations if a.majority_passes)
        majority_fails_count = len(aggregations) - majority_passes_count
        overall_majority_passes = majority_passes_count >= majority_fails_count

        # Sources that disagree with the overall majority
        inconsistent = [
            a.source_id
            for a in aggregations
            if a.majority_passes != overall_majority_passes
        ]

        agreeing = len(aggregations) - len(inconsistent)
        consistency_score = round(agreeing / len(aggregations), 4) if aggregations else 1.0

        results.append(
            CrossValidationResult(
                assertion_key=assertion_key,
                source_count=len(aggregations),
                consistent=len(inconsistent) == 0,
                consistency_score=consistency_score,
                majority_passes=overall_majority_passes,
                sources=aggregations,
                inconsistent_source_ids=inconsistent,
            )
        )

    return results
