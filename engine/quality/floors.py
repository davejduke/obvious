"""Quality Engine — Hard quality floor enforcement.

A hard quality floor specifies the minimum number of evidence items required
before a control can proceed to sufficiency assessment and conclusion
generation.  Floors are configurable per-control; the default minimum is 3.

All computations are deterministic.  No LLM is consulted.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from uuid import UUID

from engine.quality.sampler import EvidenceQualityInput

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

DEFAULT_MINIMUM_EVIDENCE: int = 3


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------


@dataclass
class QualityFloorConfig:
    """Configuration for hard quality floors."""

    default_minimum: int = DEFAULT_MINIMUM_EVIDENCE
    per_control_overrides: dict[str, int] = field(default_factory=dict)

    def __post_init__(self) -> None:
        if self.default_minimum < 1:
            raise ValueError(
                f"default_minimum must be >= 1; got {self.default_minimum}"
            )
        for ctrl, min_val in self.per_control_overrides.items():
            if min_val < 1:
                raise ValueError(
                    f"per_control_override for '{ctrl}' must be >= 1; got {min_val}"
                )


# ---------------------------------------------------------------------------
# Result type
# ---------------------------------------------------------------------------


@dataclass
class FloorCheckResult:
    """Result of a hard quality floor check for one control."""

    control_id: str
    evidence_count: int
    required_minimum: int
    passed: bool
    deficit: int  # max(0, required_minimum - evidence_count)


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def get_floor_for_control(config: QualityFloorConfig, control_id: str) -> int:
    """Return the minimum evidence count required for *control_id*."""
    return config.per_control_overrides.get(control_id, config.default_minimum)


def check_quality_floor(
    control_id: str,
    evidence_items: list[EvidenceQualityInput],
    config: QualityFloorConfig | None = None,
) -> FloorCheckResult:
    """Evaluate whether *evidence_items* meets the hard floor for *control_id*.

    Parameters
    ----------
    control_id:
        Identifier of the control being assessed.
    evidence_items:
        Current evidence items for the control.
    config:
        Floor configuration.  Uses DEFAULT_MINIMUM_EVIDENCE (3) when None.

    Returns
    -------
    FloorCheckResult
        Passed when len(evidence_items) >= required_minimum.
    """
    if config is None:
        config = QualityFloorConfig()

    required = get_floor_for_control(config, control_id)
    count = len(evidence_items)
    deficit = max(0, required - count)
    return FloorCheckResult(
        control_id=control_id,
        evidence_count=count,
        required_minimum=required,
        passed=count >= required,
        deficit=deficit,
    )


def recalculate_floor(
    current_items: list[EvidenceQualityInput],
    control_id: str,
    config: QualityFloorConfig | None = None,
    added_items: list[EvidenceQualityInput] | None = None,
    removed_ids: set[UUID] | None = None,
) -> FloorCheckResult:
    """Recompute the floor check after adding or removing evidence items.

    Applies removals before additions so that a remove-then-add of the same
    UUID produces a net-zero change.

    Parameters
    ----------
    current_items:
        Evidence items before the mutation.
    control_id:
        Identifier of the control being assessed.
    config:
        Floor configuration.
    added_items:
        Evidence items to add (appended after removals).
    removed_ids:
        Evidence item UUIDs to remove.

    Returns
    -------
    FloorCheckResult
        Updated floor check reflecting the mutated evidence set.
    """
    items: list[EvidenceQualityInput] = list(current_items)
    if removed_ids:
        items = [it for it in items if it.evidence_id not in removed_ids]
    if added_items:
        items.extend(added_items)
    return check_quality_floor(control_id, items, config)
