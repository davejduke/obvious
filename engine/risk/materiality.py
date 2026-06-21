"""Risk Engine — Per-domain materiality thresholds.

Extends the base materiality check (risk_weight >= 3.5) with per-domain
thresholds that reflect each NIS 2 Article 21 sub-clause's inherent risk
profile.  Supports per-organisation overrides via OrgMaterialityConfig.

Built-in domain thresholds
--------------------------
  21b  Incident handling                  3.0  (high-risk → lower threshold)
  21c  Business continuity                3.0
  21i  HR security & access control       3.0
  21j  MFA & secure communications        3.0
  21a  Risk analysis                      3.5  (matches global default)
  21d  Supply chain                       3.5
  21e  Development security               3.5
  21h  Cryptography                       3.5
  21f  Effectiveness assessment           4.0  (lower-risk → higher threshold)
  21g  Cyber hygiene                      4.0
  DEFAULT (unknown domain)               3.5
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Optional

from engine.risk.scorer import MATERIALITY_THRESHOLD, TDR_MATERIAL, TDR_NON_MATERIAL


# ---------------------------------------------------------------------------
# Domain → threshold mapping
# ---------------------------------------------------------------------------

DOMAIN_THRESHOLDS: dict[str, float] = {
    # High-risk domains — lower threshold → more controls classified material
    "21b": 3.0,  # Incident handling
    "21c": 3.0,  # Business continuity
    "21i": 3.0,  # HR security & access control
    "21j": 3.0,  # MFA & secure communications
    # Medium-risk domains — global default
    "21a": 3.5,  # Risk analysis
    "21d": 3.5,  # Supply chain
    "21e": 3.5,  # Development security
    "21h": 3.5,  # Cryptography
    # Lower-risk domains — higher threshold → fewer controls classified material
    "21f": 4.0,  # Effectiveness assessment
    "21g": 4.0,  # Cyber hygiene
}

# Stricter TDR for material controls in the highest-risk domains
DOMAIN_TDR_MATERIAL: dict[str, float] = {
    "21b": 0.03,
    "21c": 0.03,
    "21i": 0.03,
    "21j": 0.03,
}


# ---------------------------------------------------------------------------
# Organisation-level materiality configuration
# ---------------------------------------------------------------------------

@dataclass
class OrgMaterialityConfig:
    """Customisable materiality configuration per organisation.

    Allows tenants to tune domain thresholds and TDR values to match
    their risk appetite without changing global module constants.
    """

    org_id: str
    global_threshold: float = MATERIALITY_THRESHOLD             # default 3.5
    domain_thresholds: dict[str, float] = field(default_factory=dict)
    tdr_material: float = TDR_MATERIAL                          # default 0.05
    tdr_non_material: float = TDR_NON_MATERIAL                  # default 0.10
    domain_tdr_overrides: dict[str, float] = field(default_factory=dict)

    def get_threshold(self, domain: Optional[str]) -> float:  # noqa: UP007
        """Effective materiality threshold for a given domain."""
        if domain:
            if domain in self.domain_thresholds:
                return self.domain_thresholds[domain]
            if domain in DOMAIN_THRESHOLDS:
                return DOMAIN_THRESHOLDS[domain]
        return self.global_threshold

    def get_tdr(self, domain: Optional[str], is_material: bool) -> float:  # noqa: UP007
        """Effective TDR for a given domain + materiality flag."""
        if domain:
            if is_material and domain in self.domain_tdr_overrides:
                return self.domain_tdr_overrides[domain]
            if is_material and domain in DOMAIN_TDR_MATERIAL:
                return DOMAIN_TDR_MATERIAL[domain]
        return self.tdr_material if is_material else self.tdr_non_material


# Singleton default config (pure system defaults, no org customisation)
DEFAULT_ORG_CONFIG: OrgMaterialityConfig = OrgMaterialityConfig(org_id="__default__")


# ---------------------------------------------------------------------------
# Result type
# ---------------------------------------------------------------------------

@dataclass
class MaterialityResult:
    """Output of a materiality assessment for one control."""

    control_id: str
    risk_weight: float
    domain: Optional[str]       # noqa: UP007
    effective_threshold: float
    is_material: bool
    tdr_threshold: float
    basis: str                  # human-readable explanation


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def compute_domain_materiality(
    control_id: str,
    risk_weight: float,
    domain: Optional[str] = None,        # noqa: UP007
    org_config: OrgMaterialityConfig = DEFAULT_ORG_CONFIG,
) -> MaterialityResult:
    """Compute materiality for a control using per-domain thresholds.

    Args:
        control_id:  Control identifier string.
        risk_weight: Control risk weight (0.0 – 5.0).
        domain:      NIS 2 article reference (e.g. "21b") or None.
        org_config:  Organisation-specific materiality configuration.

    Returns:
        MaterialityResult with threshold, flag, TDR, and plain-language basis.
    """
    threshold = org_config.get_threshold(domain)
    is_material = risk_weight >= threshold
    tdr = org_config.get_tdr(domain, is_material)

    is_org_override = domain is not None and domain in org_config.domain_thresholds
    is_system_domain = domain is not None and domain in DOMAIN_THRESHOLDS

    if is_org_override:
        basis = f"Domain '{domain}' threshold {threshold:.1f} (organisation override)."
    elif is_system_domain:
        basis = f"Domain '{domain}' threshold {threshold:.1f} (system default)."
    else:
        basis = f"Global threshold {threshold:.1f} applied (domain unknown or unregistered)."

    return MaterialityResult(
        control_id=control_id,
        risk_weight=risk_weight,
        domain=domain,
        effective_threshold=threshold,
        is_material=is_material,
        tdr_threshold=tdr,
        basis=basis,
    )

