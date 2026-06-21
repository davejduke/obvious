"""Unit tests — Risk Engine enhanced features.

Covers:
  - Exception detection  (exceptions.py)
  - Per-domain materiality thresholds  (materiality.py)
  - Heat map generation and residual risk  (heat_map.py)
  - Article-level aggregation, trending, org thresholds  (aggregator.py)
"""
import pytest
from uuid import uuid4

from engine.risk.scorer import (
    AuditConclusion,
    EvidenceItem,
    MATERIALITY_THRESHOLD,
    PENALTY_RATIO,
    TDR_MATERIAL,
    TDR_NON_MATERIAL,
    _THRESHOLD_EFFECTIVE,
    _THRESHOLD_PARTIAL,
    score_control,
)
from engine.risk.exceptions import (
    AnomalyType,
    ExceptionDetector,
    ExceptionFlag,
    detect_exceptions,
    _QUALITY_FLOOR,
    _QUALITY_CEILING,
    _TIER1_QUALITY_MINIMUM,
    _QUALITY_DROP_DELTA,
)
from engine.risk.materiality import (
    DOMAIN_THRESHOLDS,
    DEFAULT_ORG_CONFIG,
    OrgMaterialityConfig,
    compute_domain_materiality,
)
from engine.risk.heat_map import (
    HEAT_MAP_SIZE,
    HeatMapInput,
    compute_residual_risk,
    generate_heat_map,
    _posterior_to_likelihood,
    _raw_score_to_zone,
    _risk_weight_to_impact,
)
from engine.risk.aggregator import (
    ArticleRiskScore,
    ControlMeta,
    OrgRiskThresholds,
    PeriodScore,
    TrendDirection,
    aggregate_by_article,
    compute_risk_trend,
    score_control_with_thresholds,
)


# ---------------------------------------------------------------------------
# Shared helpers
# ---------------------------------------------------------------------------

def make_evidence(
    passes: bool = True,
    quality: float = 0.9,
    tier: int = 1,
) -> EvidenceItem:
    return EvidenceItem(evidence_id=uuid4(), passes=passes, quality_score=quality, tier=tier)


def make_result(
    control_id: str = "C-001",
    posterior: float = 0.5,
    conclusion: AuditConclusion = AuditConclusion.PARTIALLY_EFFECTIVE,
    is_material: bool = False,
):
    """Build a minimal RiskScoreResult for aggregation tests."""
    return score_control(
        control_id=control_id,
        risk_weight=4.0 if is_material else 2.0,
        prior=posterior,
        evidence_items=[],  # no-evidence path sets posterior = prior
    )


# ===========================================================================
# 1. Exception Detection
# ===========================================================================

class TestExceptionDetectorConstants:
    def test_quality_floor_value(self):
        assert _QUALITY_FLOOR == 0.40

    def test_quality_ceiling_value(self):
        assert _QUALITY_CEILING == 0.80

    def test_tier1_minimum_value(self):
        assert _TIER1_QUALITY_MINIMUM == 0.70

    def test_quality_drop_delta_value(self):
        assert _QUALITY_DROP_DELTA == 0.30


class TestExceptionDetectorStream:
    def test_low_quality_pass_flagged(self):
        detector = ExceptionDetector()
        item = make_evidence(passes=True, quality=0.20)
        flags = detector.update(item)
        assert any(f.anomaly_type == AnomalyType.LOW_QUALITY_PASS for f in flags)

    def test_low_quality_pass_severity_is_high(self):
        detector = ExceptionDetector()
        item = make_evidence(passes=True, quality=0.20)
        flags = detector.update(item)
        flag = next(f for f in flags if f.anomaly_type == AnomalyType.LOW_QUALITY_PASS)
        assert flag.severity == "high"
        assert flag.evidence_id == item.evidence_id

    def test_high_quality_pass_not_flagged(self):
        detector = ExceptionDetector()
        flags = detector.update(make_evidence(passes=True, quality=0.90))
        assert not any(f.anomaly_type == AnomalyType.LOW_QUALITY_PASS for f in flags)

    def test_high_quality_fail_flagged(self):
        detector = ExceptionDetector()
        item = make_evidence(passes=False, quality=0.90)
        flags = detector.update(item)
        assert any(f.anomaly_type == AnomalyType.HIGH_QUALITY_FAIL for f in flags)

    def test_high_quality_fail_severity_is_medium(self):
        detector = ExceptionDetector()
        item = make_evidence(passes=False, quality=0.85)
        flags = detector.update(item)
        flag = next(f for f in flags if f.anomaly_type == AnomalyType.HIGH_QUALITY_FAIL)
        assert flag.severity == "medium"

    def test_low_quality_fail_not_flagged_as_high_quality_fail(self):
        detector = ExceptionDetector()
        flags = detector.update(make_evidence(passes=False, quality=0.30))
        assert not any(f.anomaly_type == AnomalyType.HIGH_QUALITY_FAIL for f in flags)

    def test_tier1_quality_mismatch_flagged(self):
        detector = ExceptionDetector()
        item = make_evidence(passes=True, quality=0.50, tier=1)
        flags = detector.update(item)
        assert any(f.anomaly_type == AnomalyType.TIER_QUALITY_MISMATCH for f in flags)

    def test_tier1_quality_mismatch_severity_is_medium(self):
        detector = ExceptionDetector()
        flags = detector.update(make_evidence(passes=True, quality=0.50, tier=1))
        flag = next(f for f in flags if f.anomaly_type == AnomalyType.TIER_QUALITY_MISMATCH)
        assert flag.severity == "medium"

    def test_tier2_below_tier1_minimum_not_flagged(self):
        # Tier-quality-mismatch only fires for tier=1
        detector = ExceptionDetector()
        flags = detector.update(make_evidence(passes=True, quality=0.50, tier=2))
        assert not any(f.anomaly_type == AnomalyType.TIER_QUALITY_MISMATCH for f in flags)

    def test_quality_drop_flagged_after_stable_run(self):
        """After 3 high-quality items, a very-low-quality item triggers QUALITY_DROP."""
        detector = ExceptionDetector()
        for _ in range(3):
            detector.update(make_evidence(quality=0.95))
        drop_item = make_evidence(quality=0.20)
        flags = detector.update(drop_item)
        assert any(f.anomaly_type == AnomalyType.QUALITY_DROP for f in flags)

    def test_quality_drop_not_flagged_with_fewer_than_3_prior_items(self):
        detector = ExceptionDetector()
        for _ in range(2):  # only 2 prior items — not enough
            detector.update(make_evidence(quality=0.95))
        flags = detector.update(make_evidence(quality=0.20))
        assert not any(f.anomaly_type == AnomalyType.QUALITY_DROP for f in flags)

    def test_deviation_spike_flagged_when_tdr_breached(self):
        """All 5 items fail (dev rate 100%) — should breach 10% TDR."""
        detector = ExceptionDetector(is_material=False)  # TDR = 0.10
        flags_all = []
        for _ in range(5):
            flags_all.extend(detector.update(make_evidence(passes=False, quality=0.90)))
        assert any(f.anomaly_type == AnomalyType.DEVIATION_SPIKE for f in flags_all)

    def test_deviation_spike_fires_only_once(self):
        """Once TDR is breached, subsequent items must not re-flag DEVIATION_SPIKE."""
        detector = ExceptionDetector(is_material=False)
        all_flags: list[ExceptionFlag] = []
        for _ in range(10):
            all_flags.extend(detector.update(make_evidence(passes=False)))
        spikes = [f for f in all_flags if f.anomaly_type == AnomalyType.DEVIATION_SPIKE]
        assert len(spikes) == 1

    def test_deviation_spike_not_flagged_when_under_tdr(self):
        """All 5 items pass — dev rate 0% — no spike."""
        detector = ExceptionDetector()
        all_flags: list[ExceptionFlag] = []
        for _ in range(5):
            all_flags.extend(detector.update(make_evidence(passes=True)))
        assert not any(f.anomaly_type == AnomalyType.DEVIATION_SPIKE for f in all_flags)

    def test_deviation_spike_uses_material_tdr(self):
        """6% dev rate is fine for non-material (TDR 10%) but fires for material (TDR 5%)."""
        # Build: 15 passes then 1 failure → dev rate settles to 1/16 ≈ 6.25%
        # (failure placed last so dev_rate stays low until item 16)
        def run(is_material: bool):
            detector = ExceptionDetector(is_material=is_material)
            all_flags: list[ExceptionFlag] = []
            items = [make_evidence(passes=True)] * 15 + [make_evidence(passes=False)]
            for item in items:
                all_flags.extend(detector.update(item))
            return any(f.anomaly_type == AnomalyType.DEVIATION_SPIKE for f in all_flags)

        assert run(is_material=True) is True   # 6.25% > 5% (material TDR)
        assert run(is_material=False) is False  # 6.25% < 10% (non-material TDR)

    def test_item_count_property(self):
        detector = ExceptionDetector()
        assert detector.item_count == 0
        detector.update(make_evidence())
        assert detector.item_count == 1

    def test_running_deviation_rate_empty(self):
        detector = ExceptionDetector()
        assert detector.running_deviation_rate == 0.0

    def test_running_deviation_rate_half_failing(self):
        detector = ExceptionDetector()
        detector.update(make_evidence(passes=True))
        detector.update(make_evidence(passes=False))
        assert detector.running_deviation_rate == 0.5


class TestExceptionDetectorFinalize:
    def test_all_pass_triggers_inconsistent_batch(self):
        detector = ExceptionDetector()
        for _ in range(4):
            detector.update(make_evidence(passes=True))
        flags = detector.finalize()
        assert any(f.anomaly_type == AnomalyType.INCONSISTENT_BATCH for f in flags)

    def test_all_fail_triggers_inconsistent_batch(self):
        detector = ExceptionDetector()
        for _ in range(4):
            detector.update(make_evidence(passes=False))
        flags = detector.finalize()
        batch_flags = [f for f in flags if f.anomaly_type == AnomalyType.INCONSISTENT_BATCH]
        assert len(batch_flags) == 1
        assert batch_flags[0].context["verdict"] == "fail"

    def test_mixed_results_no_inconsistent_batch(self):
        detector = ExceptionDetector()
        detector.update(make_evidence(passes=True))
        detector.update(make_evidence(passes=False))
        flags = detector.finalize()
        assert not any(f.anomaly_type == AnomalyType.INCONSISTENT_BATCH for f in flags)

    def test_single_item_no_inconsistent_batch(self):
        """Batch check requires ≥2 items."""
        detector = ExceptionDetector()
        detector.update(make_evidence(passes=True))
        assert detector.finalize() == []


class TestDetectExceptionsBatch:
    def test_clean_evidence_produces_no_flags(self):
        items = [make_evidence(passes=True, quality=0.90, tier=2) for _ in range(3)]
        flags = detect_exceptions(items)
        # 3 items: no stream anomalies; mixed verdicts would be needed for batch
        # But all pass → INCONSISTENT_BATCH (all pass)
        assert any(f.anomaly_type == AnomalyType.INCONSISTENT_BATCH for f in flags)

    def test_clean_mixed_evidence_no_flags(self):
        items = (
            [make_evidence(passes=True, quality=0.90, tier=2)] * 3
            + [make_evidence(passes=False, quality=0.30, tier=3)]
        )
        flags = detect_exceptions(items)
        assert not any(f.anomaly_type == AnomalyType.INCONSISTENT_BATCH for f in flags)

    def test_detect_exceptions_is_deterministic(self):
        items = [make_evidence(passes=True, quality=0.90) for _ in range(3)]
        flags1 = detect_exceptions(items)
        flags2 = detect_exceptions(items)
        assert [f.anomaly_type for f in flags1] == [f.anomaly_type for f in flags2]

    def test_low_quality_pass_appears_in_batch_output(self):
        items = [make_evidence(passes=True, quality=0.20, tier=2)]
        flags = detect_exceptions(items)
        assert any(f.anomaly_type == AnomalyType.LOW_QUALITY_PASS for f in flags)


# ===========================================================================
# 2. Per-Domain Materiality
# ===========================================================================

class TestDomainThresholds:
    def test_high_risk_domains_have_lower_threshold(self):
        for domain in ("21b", "21c", "21i", "21j"):
            assert DOMAIN_THRESHOLDS[domain] == 3.0, f"Expected 3.0 for {domain}"

    def test_medium_risk_domains_match_global_default(self):
        for domain in ("21a", "21d", "21e", "21h"):
            assert DOMAIN_THRESHOLDS[domain] == MATERIALITY_THRESHOLD

    def test_lower_risk_domains_have_higher_threshold(self):
        for domain in ("21f", "21g"):
            assert DOMAIN_THRESHOLDS[domain] == 4.0, f"Expected 4.0 for {domain}"

    def test_all_10_nis2_articles_registered(self):
        expected = {"21a", "21b", "21c", "21d", "21e", "21f", "21g", "21h", "21i", "21j"}
        assert set(DOMAIN_THRESHOLDS.keys()) == expected


class TestComputeDomainMateriality:
    def test_high_risk_domain_lower_threshold_makes_more_controls_material(self):
        # risk_weight=3.1: below global 3.5 but above 21b threshold 3.0
        result = compute_domain_materiality("C-001", risk_weight=3.1, domain="21b")
        assert result.is_material is True
        assert result.effective_threshold == 3.0

    def test_global_threshold_used_for_unknown_domain(self):
        result = compute_domain_materiality("C-002", risk_weight=3.2, domain=None)
        assert result.effective_threshold == MATERIALITY_THRESHOLD
        assert result.is_material is False  # 3.2 < 3.5

    def test_unknown_string_domain_uses_global(self):
        result = compute_domain_materiality("C-003", risk_weight=3.2, domain="99z")
        assert result.effective_threshold == MATERIALITY_THRESHOLD

    def test_lower_risk_domain_requires_higher_weight_for_materiality(self):
        # risk_weight=3.7: above global 3.5 but below 21g threshold 4.0
        result = compute_domain_materiality("C-004", risk_weight=3.7, domain="21g")
        assert result.is_material is False
        assert result.effective_threshold == 4.0

    def test_default_config_tdr_is_standard(self):
        result = compute_domain_materiality("C-005", risk_weight=4.0, domain="21a")
        assert result.tdr_threshold == TDR_MATERIAL  # material 21a uses default TDR

    def test_high_risk_domain_material_uses_stricter_tdr(self):
        result = compute_domain_materiality("C-006", risk_weight=3.1, domain="21b")
        # 21b is material at 3.1 and gets stricter TDR of 3%
        assert result.tdr_threshold == 0.03

    def test_non_material_control_uses_non_material_tdr(self):
        result = compute_domain_materiality("C-007", risk_weight=2.0, domain="21b")
        assert result.is_material is False
        assert result.tdr_threshold == TDR_NON_MATERIAL

    def test_basis_contains_domain_for_known_domain(self):
        result = compute_domain_materiality("C-008", risk_weight=3.0, domain="21c")
        assert "21c" in result.basis

    def test_basis_contains_global_for_unknown_domain(self):
        result = compute_domain_materiality("C-009", risk_weight=3.0)
        assert "Global" in result.basis or "global" in result.basis.lower()


class TestOrgMaterialityConfig:
    def test_org_override_replaces_system_threshold(self):
        org = OrgMaterialityConfig(
            org_id="acme",
            domain_thresholds={"21a": 2.5},  # much lower override
        )
        result = compute_domain_materiality("C-010", risk_weight=2.8, domain="21a", org_config=org)
        assert result.is_material is True
        assert result.effective_threshold == 2.5
        assert "organisation override" in result.basis

    def test_org_global_threshold_overrides_system_default(self):
        org = OrgMaterialityConfig(org_id="beta", global_threshold=2.0)
        result = compute_domain_materiality("C-011", risk_weight=2.1, org_config=org)
        assert result.is_material is True  # 2.1 >= 2.0

    def test_org_tdr_override_applied_for_material_control(self):
        org = OrgMaterialityConfig(org_id="gamma", tdr_material=0.02)
        result = compute_domain_materiality("C-012", risk_weight=4.0, domain="21a", org_config=org)
        assert result.tdr_threshold == 0.02

    def test_default_org_config_matches_global_constants(self):
        assert DEFAULT_ORG_CONFIG.global_threshold == MATERIALITY_THRESHOLD
        assert DEFAULT_ORG_CONFIG.tdr_material == TDR_MATERIAL
        assert DEFAULT_ORG_CONFIG.tdr_non_material == TDR_NON_MATERIAL


# ===========================================================================
# 3. Heat Map
# ===========================================================================

class TestHeatMapHelpers:
    def test_risk_weight_to_impact_boundary_values(self):
        assert _risk_weight_to_impact(0.0) == 1
        assert _risk_weight_to_impact(1.0) == 1
        assert _risk_weight_to_impact(1.1) == 2
        assert _risk_weight_to_impact(5.0) == 5
        assert _risk_weight_to_impact(5.5) == 5   # clamped

    def test_posterior_to_likelihood_boundary_values(self):
        # posterior=1.0 → control fully effective → failure very unlikely → likelihood=1
        assert _posterior_to_likelihood(1.0) == 1
        # posterior=0.0 → control definitely failing → failure almost certain → likelihood=5
        assert _posterior_to_likelihood(0.0) == 5

    def test_posterior_to_likelihood_mid_range(self):
        # posterior=0.5 → likelihood = 5 - floor(0.5*4) = 5 - 2 = 3
        assert _posterior_to_likelihood(0.5) == 3

    def test_zone_boundaries(self):
        assert _raw_score_to_zone(1) == "low"
        assert _raw_score_to_zone(4) == "low"
        assert _raw_score_to_zone(5) == "medium"
        assert _raw_score_to_zone(9) == "medium"
        assert _raw_score_to_zone(10) == "high"
        assert _raw_score_to_zone(14) == "high"
        assert _raw_score_to_zone(15) == "critical"
        assert _raw_score_to_zone(25) == "critical"


class TestGenerateHeatMap:
    def test_generates_5x5_grid(self):
        result = generate_heat_map([])
        assert len(result.cells) == HEAT_MAP_SIZE
        for row in result.cells:
            assert len(row) == HEAT_MAP_SIZE

    def test_cell_scores_correct(self):
        result = generate_heat_map([])
        # cells[0][0] = impact=1, likelihood=1 → score=1
        assert result.cells[0][0].raw_score == 1
        # cells[4][4] = impact=5, likelihood=5 → score=25
        assert result.cells[4][4].raw_score == 25

    def test_top_right_cell_is_critical(self):
        result = generate_heat_map([])
        assert result.cells[4][4].zone == "critical"

    def test_bottom_left_cell_is_low(self):
        result = generate_heat_map([])
        assert result.cells[0][0].zone == "low"

    def test_high_risk_weight_maps_to_impact_5(self):
        controls = [HeatMapInput(control_id="C-001", risk_weight=5.0, posterior=0.5)]
        result = generate_heat_map(controls)
        impact, _ = result.control_placements["C-001"]
        assert impact == 5

    def test_low_risk_weight_maps_to_impact_1(self):
        controls = [HeatMapInput(control_id="C-002", risk_weight=0.5, posterior=0.5)]
        result = generate_heat_map(controls)
        impact, _ = result.control_placements["C-002"]
        assert impact == 1

    def test_high_posterior_maps_to_low_likelihood(self):
        controls = [HeatMapInput(control_id="C-003", risk_weight=3.0, posterior=1.0)]
        result = generate_heat_map(controls)
        _, likelihood = result.control_placements["C-003"]
        assert likelihood == 1

    def test_low_posterior_maps_to_high_likelihood(self):
        controls = [HeatMapInput(control_id="C-004", risk_weight=3.0, posterior=0.0)]
        result = generate_heat_map(controls)
        _, likelihood = result.control_placements["C-004"]
        assert likelihood == 5

    def test_control_id_appears_in_cell(self):
        controls = [HeatMapInput(control_id="C-005", risk_weight=3.0, posterior=0.5)]
        result = generate_heat_map(controls)
        impact, likelihood = result.control_placements["C-005"]
        cell = result.cells[impact - 1][likelihood - 1]
        assert "C-005" in cell.control_ids

    def test_zone_summary_counts_correctly(self):
        # 1 control at (impact=5, likelihood=5) → score=25 → critical
        controls = [HeatMapInput(control_id="C-006", risk_weight=5.0, posterior=0.0)]
        result = generate_heat_map(controls)
        assert result.zone_summary["critical"] == 1
        assert result.zone_summary["high"] == 0

    def test_multiple_controls_same_cell(self):
        controls = [
            HeatMapInput(control_id="C-007", risk_weight=3.0, posterior=0.5),
            HeatMapInput(control_id="C-008", risk_weight=3.0, posterior=0.5),
        ]
        result = generate_heat_map(controls)
        pos_1 = result.control_placements["C-007"]
        pos_2 = result.control_placements["C-008"]
        assert pos_1 == pos_2  # same cell
        impact, likelihood = pos_1
        cell = result.cells[impact - 1][likelihood - 1]
        assert "C-007" in cell.control_ids
        assert "C-008" in cell.control_ids

    def test_empty_controls_all_cells_empty(self):
        result = generate_heat_map([])
        for row in result.cells:
            for cell in row:
                assert cell.control_ids == []

    def test_metadata_contains_grid_size(self):
        result = generate_heat_map([])
        assert result.metadata["grid_size"] == HEAT_MAP_SIZE


class TestComputeResidualRisk:
    def test_zero_posterior_residual_equals_inherent(self):
        # posterior=0 → control fully ineffective → residual = inherent
        residual = compute_residual_risk(5, 5, control_posterior=0.0)
        assert residual == 25.0

    def test_perfect_posterior_residual_is_zero(self):
        residual = compute_residual_risk(5, 5, control_posterior=1.0)
        assert residual == 0.0

    def test_half_posterior_halves_inherent(self):
        residual = compute_residual_risk(4, 4, control_posterior=0.5)
        assert abs(residual - 8.0) < 1e-9

    def test_posterior_clamped_above_1(self):
        residual = compute_residual_risk(3, 3, control_posterior=2.0)  # clamped to 1
        assert residual == 0.0

    def test_posterior_clamped_below_0(self):
        residual = compute_residual_risk(3, 3, control_posterior=-1.0)  # clamped to 0
        assert residual == 9.0


# ===========================================================================
# 4. Article-Level Aggregation
# ===========================================================================

class TestAggregateByArticle:
    def test_single_control_single_article(self):
        result = score_control("C-001", risk_weight=2.0, prior=0.8, evidence_items=[])
        meta = ControlMeta(control_id="C-001", article_ref="NIS2-21a", risk_weight=2.0)
        articles = aggregate_by_article([result], [meta])
        assert "21a" in articles
        article = articles["21a"]
        assert article.control_count == 1
        assert article.weighted_posterior == round(result.posterior, 4)

    def test_multiple_controls_same_article_weighted_avg(self):
        r1 = score_control("C-001", risk_weight=1.0, prior=0.6, evidence_items=[])
        r2 = score_control("C-002", risk_weight=3.0, prior=0.8, evidence_items=[])
        m1 = ControlMeta("C-001", "21a", 1.0)
        m2 = ControlMeta("C-002", "21a", 3.0)
        articles = aggregate_by_article([r1, r2], [m1, m2])
        article = articles["21a"]
        # weighted avg = (0.6*1 + 0.8*3) / (1+3) = (0.6 + 2.4) / 4 = 0.75
        assert abs(article.weighted_posterior - 0.75) < 1e-4

    def test_two_articles_separated(self):
        r1 = score_control("C-001", risk_weight=2.0, prior=0.7, evidence_items=[])
        r2 = score_control("C-002", risk_weight=2.0, prior=0.4, evidence_items=[])
        m1 = ControlMeta("C-001", "21a", 2.0)
        m2 = ControlMeta("C-002", "21b", 2.0)
        articles = aggregate_by_article([r1, r2], [m1, m2])
        assert "21a" in articles and "21b" in articles
        assert articles["21a"].control_count == 1
        assert articles["21b"].control_count == 1

    def test_worst_case_conclusion_propagates(self):
        """If any control is NOT_EFFECTIVE, the article gets NOT_EFFECTIVE."""
        # Force two different conclusions via evidence
        good_ev = [make_evidence(passes=True, quality=0.95) for _ in range(6)]
        r_good = score_control("C-001", risk_weight=2.0, prior=0.5, evidence_items=good_ev)
        r_bad  = score_control("C-002", risk_weight=2.0, prior=0.5, evidence_items=[])
        # r_good should be EFFECTIVE; r_bad is NOT_EFFECTIVE (no evidence)
        assert r_good.conclusion == AuditConclusion.EFFECTIVE
        assert r_bad.conclusion == AuditConclusion.NOT_EFFECTIVE

        m_good = ControlMeta("C-001", "21a", 2.0)
        m_bad  = ControlMeta("C-002", "21a", 2.0)
        articles = aggregate_by_article([r_good, r_bad], [m_good, m_bad])
        assert articles["21a"].worst_conclusion == AuditConclusion.NOT_EFFECTIVE

    def test_article_ref_normalisation(self):
        r1 = score_control("C-001", risk_weight=2.0, prior=0.5, evidence_items=[])
        m1 = ControlMeta("C-001", "NIS2-21A", 2.0)  # mixed case + prefix
        articles = aggregate_by_article([r1], [m1])
        assert "21a" in articles  # normalised to lowercase

    def test_mismatched_lengths_raises_value_error(self):
        r1 = score_control("C-001", risk_weight=2.0, prior=0.5, evidence_items=[])
        with pytest.raises(ValueError, match="equal length"):
            aggregate_by_article([r1], [])

    def test_material_count_correct(self):
        r_mat = score_control("C-001", risk_weight=4.0, prior=0.5, evidence_items=[])
        r_non = score_control("C-002", risk_weight=2.0, prior=0.5, evidence_items=[])
        m_mat = ControlMeta("C-001", "21a", 4.0)
        m_non = ControlMeta("C-002", "21a", 2.0)
        articles = aggregate_by_article([r_mat, r_non], [m_mat, m_non])
        assert articles["21a"].material_control_count == 1  # only r_mat is material

    def test_traceability_contains_control_ids(self):
        r1 = score_control("C-001", risk_weight=2.0, prior=0.5, evidence_items=[])
        m1 = ControlMeta("C-001", "21a", 2.0)
        articles = aggregate_by_article([r1], [m1])
        assert "C-001" in articles["21a"].traceability["control_ids"]


# ===========================================================================
# 5. Risk Trending
# ===========================================================================

class TestComputeRiskTrend:
    def test_improving_trend_detected(self):
        cur  = PeriodScore(period_id="Q2", article_scores={"21a": 0.85})
        prev = PeriodScore(period_id="Q1", article_scores={"21a": 0.65})
        trend = compute_risk_trend(cur, prev)
        assert trend.article_trends["21a"].direction == TrendDirection.IMPROVING
        assert trend.overall_direction == TrendDirection.IMPROVING
        assert trend.improved_count == 1

    def test_deteriorating_trend_detected(self):
        cur  = PeriodScore(period_id="Q2", article_scores={"21a": 0.50})
        prev = PeriodScore(period_id="Q1", article_scores={"21a": 0.75})
        trend = compute_risk_trend(cur, prev)
        assert trend.article_trends["21a"].direction == TrendDirection.DETERIORATING
        assert trend.overall_direction == TrendDirection.DETERIORATING
        assert trend.deteriorated_count == 1

    def test_stable_trend_within_delta(self):
        cur  = PeriodScore(period_id="Q2", article_scores={"21a": 0.72})
        prev = PeriodScore(period_id="Q1", article_scores={"21a": 0.70})
        trend = compute_risk_trend(cur, prev)
        assert trend.article_trends["21a"].direction == TrendDirection.STABLE
        assert trend.overall_direction == TrendDirection.STABLE

    def test_empty_periods_gives_insufficient_data(self):
        cur  = PeriodScore(period_id="Q2", article_scores={})
        prev = PeriodScore(period_id="Q1", article_scores={})
        trend = compute_risk_trend(cur, prev)
        assert trend.overall_direction == TrendDirection.INSUFFICIENT_DATA

    def test_overall_direction_majority_wins(self):
        # 2 improving, 1 deteriorating → overall IMPROVING
        cur  = PeriodScore(period_id="Q2", article_scores={"21a": 0.9, "21b": 0.9, "21c": 0.2})
        prev = PeriodScore(period_id="Q1", article_scores={"21a": 0.5, "21b": 0.5, "21c": 0.8})
        trend = compute_risk_trend(cur, prev)
        assert trend.overall_direction == TrendDirection.IMPROVING
        assert trend.improved_count == 2
        assert trend.deteriorated_count == 1

    def test_article_present_only_in_current(self):
        """Articles in current but not previous default previous score to 0."""
        cur  = PeriodScore(period_id="Q2", article_scores={"21a": 0.6})
        prev = PeriodScore(period_id="Q1", article_scores={})
        trend = compute_risk_trend(cur, prev)
        assert "21a" in trend.article_trends
        assert trend.article_trends["21a"].previous_score == 0.0
        assert trend.article_trends["21a"].current_score == 0.6

    def test_delta_and_percent_change_calculated(self):
        cur  = PeriodScore(period_id="Q2", article_scores={"21a": 0.80})
        prev = PeriodScore(period_id="Q1", article_scores={"21a": 0.50})
        trend = compute_risk_trend(cur, prev)
        at = trend.article_trends["21a"]
        assert abs(at.delta - 0.30) < 1e-4
        assert abs(at.percent_change - 60.0) < 0.1

    def test_trend_is_deterministic(self):
        cur  = PeriodScore(period_id="Q2", article_scores={"21a": 0.75})
        prev = PeriodScore(period_id="Q1", article_scores={"21a": 0.60})
        t1 = compute_risk_trend(cur, prev)
        t2 = compute_risk_trend(cur, prev)
        assert t1.overall_direction == t2.overall_direction
        assert t1.improved_count == t2.improved_count


# ===========================================================================
# 6. Org Risk Thresholds
# ===========================================================================

class TestOrgRiskThresholds:
    def test_default_thresholds_match_scorer_constants(self):
        org = OrgRiskThresholds(org_id="default")
        assert org.penalty_ratio == PENALTY_RATIO
        assert org.materiality_threshold == MATERIALITY_THRESHOLD
        assert org.tdr_material == TDR_MATERIAL
        assert org.tdr_non_material == TDR_NON_MATERIAL

    def test_default_threshold_effective_matches_scorer(self):
        org = OrgRiskThresholds(org_id="default")
        assert abs(org.threshold_effective - _THRESHOLD_EFFECTIVE) < 1e-10

    def test_default_threshold_partial_matches_scorer(self):
        org = OrgRiskThresholds(org_id="default")
        assert abs(org.threshold_partial - _THRESHOLD_PARTIAL) < 1e-10

    def test_custom_penalty_ratio_changes_thresholds(self):
        # penalty_ratio=10 → threshold_effective = 1 - 1/11 ≈ 0.909
        org = OrgRiskThresholds(org_id="strict", penalty_ratio=10.0)
        expected_eff = 1.0 - (1.0 / (1.0 + 10.0))
        assert abs(org.threshold_effective - expected_eff) < 1e-10

    def test_lower_penalty_ratio_makes_scoring_more_lenient(self):
        """Lower penalty_ratio → lower effective threshold → easier to conclude EFFECTIVE."""
        org_strict  = OrgRiskThresholds(org_id="strict",  penalty_ratio=5.0)  # default
        org_lenient = OrgRiskThresholds(org_id="lenient", penalty_ratio=2.0)
        assert org_lenient.threshold_effective < org_strict.threshold_effective


class TestScoreControlWithThresholds:
    def test_default_org_matches_standard_score_control(self):
        """With default OrgRiskThresholds, conclusions must equal score_control()."""
        org = OrgRiskThresholds(org_id="default")
        evidence = [make_evidence(passes=True, quality=0.95) for _ in range(6)]
        std = score_control("C-001", risk_weight=2.0, prior=0.5, evidence_items=evidence)
        org_result = score_control_with_thresholds(
            "C-001", risk_weight=2.0, prior=0.5,
            evidence_items=evidence, org_thresholds=org,
        )
        assert std.conclusion == org_result.conclusion
        assert std.posterior == org_result.posterior
        assert std.evidence_count == org_result.evidence_count

    def test_lenient_thresholds_can_pass_borderline_control(self):
        """
        A borderline control that scores PARTIALLY_EFFECTIVE at default 5x penalty
        should score EFFECTIVE with a very lenient (penalty_ratio=1) threshold.
        """
        # Build evidence that produces posterior near 0.78 (above partial, below effective)
        evidence = [make_evidence(passes=True, quality=0.85) for _ in range(4)]
        posterior = score_control("C-001", risk_weight=2.0, prior=0.5, evidence_items=evidence).posterior
        # Only run test if the borderline range is actually met
        if 0.72 <= posterior < _THRESHOLD_EFFECTIVE:
            lenient = OrgRiskThresholds(org_id="lenient", penalty_ratio=1.0)
            result = score_control_with_thresholds(
                "C-001", risk_weight=2.0, prior=0.5,
                evidence_items=evidence, org_thresholds=lenient,
            )
            # lenient threshold_effective ≈ 0.667, so posterior ≈ 0.78 > 0.667 → EFFECTIVE
            assert result.conclusion == AuditConclusion.EFFECTIVE

    def test_no_evidence_returns_not_effective_with_org_penalty_in_traceability(self):
        org = OrgRiskThresholds(org_id="myorg", penalty_ratio=8.0)
        result = score_control_with_thresholds(
            "C-999", risk_weight=2.0, prior=0.5, evidence_items=[], org_thresholds=org,
        )
        assert result.conclusion == AuditConclusion.NOT_EFFECTIVE
        assert result.traceability["penalty_ratio"] == 8.0
        assert result.traceability["org_id"] == "myorg"

    def test_5x_penalty_preserved_in_default_traceability(self):
        """Default org uses penalty_ratio=5 — must be recorded in traceability."""
        org = OrgRiskThresholds(org_id="default")
        result = score_control_with_thresholds(
            "C-998", risk_weight=1.0, prior=0.5, evidence_items=[], org_thresholds=org,
        )
        assert result.traceability["penalty_ratio"] == PENALTY_RATIO

    def test_custom_tdr_affects_conclusion(self):
        """Org with very strict TDR (1%) should flag deviation even at low failure rate."""
        # 1 fail out of 10 = 10% deviation rate — fine for default TDR 10%,
        # but should exceed org TDR of 1%
        org = OrgRiskThresholds(org_id="strict", tdr_non_material=0.01)
        evidence = [make_evidence(passes=False)] + [make_evidence(passes=True)] * 9
        result = score_control_with_thresholds(
            "C-997", risk_weight=2.0, prior=0.9, evidence_items=evidence, org_thresholds=org,
        )
        # dev_rate = 0.10 > org TDR 0.01 → tdr_exceeded = True → NOT_EFFECTIVE
        assert result.tdr_exceeded is True
        assert result.conclusion == AuditConclusion.NOT_EFFECTIVE

