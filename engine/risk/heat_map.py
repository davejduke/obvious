"""Risk Engine — Impact × Likelihood risk heat map data generator.

Produces a 5×5 matrix of risk scores for dashboard visualisation.

Heat Map Design
---------------
  Impact     axis: 1 (minimal) → 5 (catastrophic)
  Likelihood axis: 1 (rare)    → 5 (almost certain)
  Cell score = impact × likelihood  (1–25)

Risk Zones
----------
  Critical  (score 15–25) — immediate action required
  High      (score 10–14) — priority remediation
  Medium    (score  5– 9) — planned remediation
  Low       (score  1– 4) — accept / monitor

Control Placement
-----------------
  impact     = ceil(risk_weight)                  (risk weight  → impact axis)
  likelihood = 5 − floor(posterior × 4)          (posterior    → likelihood axis)
  residual   = impact × likelihood × (1 − posterior)  (remaining exposure)

All computation is deterministic.  No LLM involvement.
"""
from __future__ import annotations

import math
from dataclasses import dataclass, field
from typing import Any, Optional


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

HEAT_MAP_SIZE: int = 5  # 5×5 grid


# ---------------------------------------------------------------------------
# Data types
# ---------------------------------------------------------------------------

@dataclass
class HeatMapInput:
    """Minimal control data needed to place a control on the heat map."""

    control_id: str
    risk_weight: float       # 0.0 – 5.0 → maps to impact dimension
    posterior: float         # 0.0 – 1.0 (from Bayesian update)
    article_ref: Optional[str] = None  # noqa: UP007


@dataclass
class HeatMapCell:
    """A single cell on the 5×5 impact × likelihood matrix."""

    impact: int          # 1 – 5
    likelihood: int      # 1 – 5
    raw_score: int       # impact × likelihood (1 – 25)
    zone: str            # "critical" | "high" | "medium" | "low"
    control_ids: list[str] = field(default_factory=list)


@dataclass
class HeatMapResult:
    """Full 5×5 heat map with control placements."""

    cells: list[list[HeatMapCell]]           # cells[impact-1][likelihood-1]
    control_placements: dict[str, tuple[int, int]]  # control_id → (impact, likelihood)
    zone_summary: dict[str, int]             # zone → count of controls in that zone
    metadata: dict[str, Any] = field(default_factory=dict)


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------

def _raw_score_to_zone(score: int) -> str:
    """Map a raw impact × likelihood score (1–25) to a risk zone."""
    if score >= 15:
        return "critical"
    if score >= 10:
        return "high"
    if score >= 5:
        return "medium"
    return "low"


def _risk_weight_to_impact(risk_weight: float) -> int:
    """Map risk_weight (0.0–5.0) to impact axis (1–5)."""
    clamped = max(0.0, min(5.0, risk_weight))
    return max(1, math.ceil(clamped))


def _posterior_to_likelihood(posterior: float) -> int:
    """Map posterior (0.0–1.0) to likelihood axis (1–5).

    Higher posterior (control likely effective)   → lower likelihood of failure.
    Lower  posterior (control likely ineffective) → higher likelihood of failure.
    """
    clamped = max(0.0, min(1.0, posterior))
    likelihood = 5 - math.floor(clamped * 4)
    return max(1, min(5, likelihood))


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def generate_heat_map(controls: list[HeatMapInput]) -> HeatMapResult:
    """Generate a 5×5 impact × likelihood risk heat map.

    Each control is placed in a cell based on its risk_weight (impact
    dimension) and posterior probability (likelihood-of-failure dimension).
    Controls that share the same cell are listed in control_ids.

    Args:
        controls: List of HeatMapInput instances to plot.

    Returns:
        HeatMapResult with full cell matrix, placement index, and zone summary.
    """
    # Initialise the 5×5 grid
    cells: list[list[HeatMapCell]] = []
    for impact in range(1, HEAT_MAP_SIZE + 1):
        row: list[HeatMapCell] = []
        for likelihood in range(1, HEAT_MAP_SIZE + 1):
            score = impact * likelihood
            row.append(HeatMapCell(
                impact=impact,
                likelihood=likelihood,
                raw_score=score,
                zone=_raw_score_to_zone(score),
            ))
        cells.append(row)

    placements: dict[str, tuple[int, int]] = {}
    zone_counts: dict[str, int] = {"critical": 0, "high": 0, "medium": 0, "low": 0}

    for ctrl in controls:
        impact = _risk_weight_to_impact(ctrl.risk_weight)
        likelihood = _posterior_to_likelihood(ctrl.posterior)
        cell = cells[impact - 1][likelihood - 1]
        cell.control_ids.append(ctrl.control_id)
        placements[ctrl.control_id] = (impact, likelihood)
        zone_counts[cell.zone] += 1

    return HeatMapResult(
        cells=cells,
        control_placements=placements,
        zone_summary=zone_counts,
        metadata={
            "grid_size": HEAT_MAP_SIZE,
            "total_controls": len(controls),
            "axes": {
                "x": "likelihood_of_failure (1=rare, 5=almost_certain)",
                "y": "impact (1=minimal, 5=catastrophic)",
            },
        },
    )


def compute_residual_risk(
    inherent_impact: int,
    inherent_likelihood: int,
    control_posterior: float,
) -> float:
    """Compute residual risk score after applying control effectiveness.

    Formula::
        inherent_score  = inherent_impact × inherent_likelihood
        residual_risk   = inherent_score × (1 − control_posterior)

    Args:
        inherent_impact:     Impact score (1–5) without controls.
        inherent_likelihood: Likelihood score (1–5) without controls.
        control_posterior:   P(control_effective | evidence) from Bayesian update.

    Returns:
        Residual risk score in [0.0, 25.0] — lower is better.
    """
    inherent_score = inherent_impact * inherent_likelihood
    posterior_clamped = max(0.0, min(1.0, control_posterior))
    return round(inherent_score * (1.0 - posterior_clamped), 4)

