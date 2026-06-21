"""Quality Engine — Quality gate enforcement.

The quality gate is the single decision point that determines whether a control
has sufficient reliable evidence to proceed to conclusion generation.

Gate enforcement is deliberately conservative:
- Blocks by default when the hard floor is not met.
- Blocks by default when evidence sufficiency is INSUFFICIENT.
- Blocking on unresolved conflicts is optional (default: off) because the
  conflict resolver already produces a cleaned accepted set.

All computations are deterministic.  No LLM is consulted.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum

from engine.quality.floors import (
    FloorCheckResult,
    QualityFloorConfig,
    check_quality_floor,
)
from engine.quality.sampler import (
    EvidenceQualityInput,
    SufficiencyResult,
    SufficiencyVerdict,
    assess_sufficiency,
)

# ---------------------------------------------------------------------------
# Enums & data models
# ---------------------------------------------------------------------------


class GateBlockReason(StrEnum):
    FLOOR_NOT_MET = "floor_not_met"
    INSUFFICIENT_EVIDENCE = "insufficient_evidence"
    UNRESOLVED_CONFLICTS = "unresolved_conflicts"


@dataclass
class QualityGateConfig:
    """Configuration for quality gate enforcement."""

    block_on_floor_failure: bool = True
    block_on_insufficient: bool = True
    block_on_conflicts: bool = False  # off by default; conflict resolver handles this


@dataclass
class QualityGateResult:
    """Result of quality gate enforcement for one control."""

    control_id: str
    passed: bool
    block_reasons: list[GateBlockReason] = field(default_factory=list)
    floor_result: FloorCheckResult | None = None
    sufficiency_result: SufficiencyResult | None = None
    unresolved_conflict_count: int = 0


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def enforce_quality_gate(
    control_id: str,
    evidence_items: list[EvidenceQualityInput],
    required_sample_size: int,
    floor_config: QualityFloorConfig | None = None,
    gate_config: QualityGateConfig | None = None,
    unresolved_conflict_count: int = 0,
) -> QualityGateResult:
    """Evaluate whether *evidence_items* passes the quality gate for *control_id*.

    Parameters
    ----------
    control_id:
        Identifier of the control being assessed.
    evidence_items:
        Evidence items after conflict resolution (accepted set).
    required_sample_size:
        Required sample size from Cochran calculation.
    floor_config:
        Hard floor configuration.  Defaults to ``QualityFloorConfig()``.
    gate_config:
        Gate enforcement configuration.  Defaults to ``QualityGateConfig()``.
    unresolved_conflict_count:
        Number of conflicts not resolved (used only when
        ``gate_config.block_on_conflicts`` is True).

    Returns
    -------
    QualityGateResult
        ``passed=True`` only when all enabled gate checks pass.
    """
    if floor_config is None:
        floor_config = QualityFloorConfig()
    if gate_config is None:
        gate_config = QualityGateConfig()

    block_reasons: list[GateBlockReason] = []

    # 1. Hard floor check
    floor_result = check_quality_floor(control_id, evidence_items, floor_config)
    if gate_config.block_on_floor_failure and not floor_result.passed:
        block_reasons.append(GateBlockReason.FLOOR_NOT_MET)

    # 2. Evidence sufficiency check
    sufficiency_result = assess_sufficiency(control_id, evidence_items, required_sample_size)
    if (
        gate_config.block_on_insufficient
        and sufficiency_result.verdict == SufficiencyVerdict.INSUFFICIENT
    ):
        block_reasons.append(GateBlockReason.INSUFFICIENT_EVIDENCE)

    # 3. Unresolved conflicts check (optional)
    if gate_config.block_on_conflicts and unresolved_conflict_count > 0:
        block_reasons.append(GateBlockReason.UNRESOLVED_CONFLICTS)

    return QualityGateResult(
        control_id=control_id,
        passed=len(block_reasons) == 0,
        block_reasons=block_reasons,
        floor_result=floor_result,
        sufficiency_result=sufficiency_result,
        unresolved_conflict_count=unresolved_conflict_count,
    )
