"""Risk Engine — Real-time exception detection for evidence anomalies.

Anomaly types detected
----------------------
  LOW_QUALITY_PASS      — evidence passes but quality_score < QUALITY_FLOOR (0.40)
  HIGH_QUALITY_FAIL     — evidence fails but quality_score > QUALITY_CEILING (0.80)
  TIER_QUALITY_MISMATCH — tier-1 evidence with quality below tier-1 minimum (0.70)
  QUALITY_DROP          — abrupt quality decline vs. running average (Δ > 0.30)
  DEVIATION_SPIKE       — running deviation rate breaches TDR mid-stream (first breach)
  INCONSISTENT_BATCH    — all evidence in a batch shares the same pass/fail verdict

All detection is deterministic — no LLM involved.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Any
from uuid import UUID

from engine.risk.scorer import EvidenceItem, TDR_MATERIAL, TDR_NON_MATERIAL


# ---------------------------------------------------------------------------
# Anomaly type & flag
# ---------------------------------------------------------------------------

class AnomalyType(str, Enum):
    LOW_QUALITY_PASS = "low_quality_pass"
    HIGH_QUALITY_FAIL = "high_quality_fail"
    TIER_QUALITY_MISMATCH = "tier_quality_mismatch"
    QUALITY_DROP = "quality_drop"
    DEVIATION_SPIKE = "deviation_spike"
    INCONSISTENT_BATCH = "inconsistent_batch"


@dataclass
class ExceptionFlag:
    """A flagged anomaly in one or more evidence items."""

    anomaly_type: AnomalyType
    evidence_id: UUID | None           # None = batch-level finding
    severity: str                      # "high" | "medium" | "low"
    description: str
    context: dict[str, Any] = field(default_factory=dict)


# ---------------------------------------------------------------------------
# Detection thresholds
# ---------------------------------------------------------------------------

_QUALITY_FLOOR: float = 0.40          # below this + passes → suspicious
_QUALITY_CEILING: float = 0.80        # above this + fails  → suspicious
_TIER1_QUALITY_MINIMUM: float = 0.70  # tier-1 evidence minimum quality
_QUALITY_DROP_DELTA: float = 0.30     # drop from running avg that triggers flag


# ---------------------------------------------------------------------------
# Streaming detector
# ---------------------------------------------------------------------------

class ExceptionDetector:
    """Stateful exception detector — processes evidence items one at a time.

    Usage::

        detector = ExceptionDetector(is_material=True)
        for item in stream:
            flags = detector.update(item)   # per-item flags
            handle_anomalies(flags)
        batch_flags = detector.finalize()   # batch-level flags
    """

    def __init__(self, is_material: bool = False) -> None:
        self._is_material = is_material
        self._tdr = TDR_MATERIAL if is_material else TDR_NON_MATERIAL
        self._items: list[EvidenceItem] = []
        self._running_quality_sum: float = 0.0
        self._failure_count: int = 0
        self._deviation_spike_active: bool = False  # first-breach tracking

    # ------------------------------------------------------------------
    # Stream API
    # ------------------------------------------------------------------

    def update(self, item: EvidenceItem) -> list[ExceptionFlag]:
        """Process one incoming evidence item; return any flags raised."""
        flags: list[ExceptionFlag] = []

        q = item.quality_score
        n_before = len(self._items)  # item count BEFORE appending
        running_avg = (self._running_quality_sum / n_before) if n_before > 0 else q

        self._items.append(item)
        if not item.passes:
            self._failure_count += 1

        # --- LOW_QUALITY_PASS ------------------------------------------
        if item.passes and q < _QUALITY_FLOOR:
            flags.append(ExceptionFlag(
                anomaly_type=AnomalyType.LOW_QUALITY_PASS,
                evidence_id=item.evidence_id,
                severity="high",
                description=(
                    f"Evidence passes but quality score {q:.2f} is below floor "
                    f"{_QUALITY_FLOOR:.2f}. Result reliability is questionable."
                ),
                context={"quality_score": q, "floor": _QUALITY_FLOOR},
            ))

        # --- HIGH_QUALITY_FAIL -----------------------------------------
        if not item.passes and q > _QUALITY_CEILING:
            flags.append(ExceptionFlag(
                anomaly_type=AnomalyType.HIGH_QUALITY_FAIL,
                evidence_id=item.evidence_id,
                severity="medium",
                description=(
                    f"High-quality evidence (score {q:.2f}) indicates control failure. "
                    f"Warrants immediate investigation."
                ),
                context={"quality_score": q, "ceiling": _QUALITY_CEILING},
            ))

        # --- TIER_QUALITY_MISMATCH ------------------------------------
        if item.tier == 1 and q < _TIER1_QUALITY_MINIMUM:
            flags.append(ExceptionFlag(
                anomaly_type=AnomalyType.TIER_QUALITY_MISMATCH,
                evidence_id=item.evidence_id,
                severity="medium",
                description=(
                    f"Tier-1 evidence quality {q:.2f} is below tier-1 minimum "
                    f"{_TIER1_QUALITY_MINIMUM:.2f}. Evidence may be misclassified."
                ),
                context={
                    "tier": item.tier,
                    "quality_score": q,
                    "minimum": _TIER1_QUALITY_MINIMUM,
                },
            ))

        # --- QUALITY_DROP (requires ≥3 prior items for stable running avg) ---
        if n_before >= 3 and (running_avg - q) > _QUALITY_DROP_DELTA:
            flags.append(ExceptionFlag(
                anomaly_type=AnomalyType.QUALITY_DROP,
                evidence_id=item.evidence_id,
                severity="low",
                description=(
                    f"Quality dropped from running average {running_avg:.2f} to {q:.2f} "
                    f"(Δ {running_avg - q:.2f} > threshold {_QUALITY_DROP_DELTA:.2f})."
                ),
                context={
                    "quality_score": q,
                    "running_average": round(running_avg, 4),
                    "delta": round(running_avg - q, 4),
                },
            ))

        # --- DEVIATION_SPIKE (flag once on first breach; reset on recovery) ---
        n = len(self._items)
        running_dev_rate = self._failure_count / n
        if n >= 5 and running_dev_rate > self._tdr and not self._deviation_spike_active:
            self._deviation_spike_active = True
            flags.append(ExceptionFlag(
                anomaly_type=AnomalyType.DEVIATION_SPIKE,
                evidence_id=None,
                severity="high",
                description=(
                    f"Running deviation rate {running_dev_rate:.1%} breached TDR "
                    f"{self._tdr:.1%} after {n} items. Consider expanding sample."
                ),
                context={
                    "running_deviation_rate": round(running_dev_rate, 4),
                    "tdr": self._tdr,
                    "items_processed": n,
                    "failure_count": self._failure_count,
                },
            ))
        elif running_dev_rate <= self._tdr:
            self._deviation_spike_active = False

        self._running_quality_sum += q
        return flags

    def finalize(self) -> list[ExceptionFlag]:
        """Batch-level checks run once all evidence has been processed."""
        flags: list[ExceptionFlag] = []
        n = len(self._items)
        if n < 2:
            return flags

        all_pass = all(e.passes for e in self._items)
        all_fail = not any(e.passes for e in self._items)
        if all_pass or all_fail:
            verdict = "pass" if all_pass else "fail"
            flags.append(ExceptionFlag(
                anomaly_type=AnomalyType.INCONSISTENT_BATCH,
                evidence_id=None,
                severity="low",
                description=(
                    f"All {n} evidence items have identical '{verdict}' verdict. "
                    f"Zero variance may indicate sampling bias or pre-selection."
                ),
                context={"verdict": verdict, "item_count": n},
            ))

        return flags

    # ------------------------------------------------------------------
    # Properties
    # ------------------------------------------------------------------

    @property
    def item_count(self) -> int:
        return len(self._items)

    @property
    def running_deviation_rate(self) -> float:
        if not self._items:
            return 0.0
        return self._failure_count / len(self._items)


# ---------------------------------------------------------------------------
# Batch convenience function
# ---------------------------------------------------------------------------

def detect_exceptions(
    evidence_items: list[EvidenceItem],
    is_material: bool = False,
) -> list[ExceptionFlag]:
    """Run all exception checks over a complete evidence list.

    Equivalent to feeding items one-by-one into ExceptionDetector then
    calling finalize().  Returns all anomaly flags (per-item + batch).
    """
    detector = ExceptionDetector(is_material=is_material)
    flags: list[ExceptionFlag] = []
    for item in evidence_items:
        flags.extend(detector.update(item))
    flags.extend(detector.finalize())
    return flags

