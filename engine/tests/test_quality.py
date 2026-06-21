"""Unit tests — Quality Engine (Cochran sampling, tier scoring, sufficiency,
hard floors, cross-validation, conflict resolution, gate enforcement, trending).
"""
import math
from uuid import uuid4

import pytest
from engine.quality.conflict_resolver import (
    ConflictType,
    detect_conflicts,
    resolve_conflicts,
)
from engine.quality.cross_validator import (
    AnnotatedEvidenceItem,
    cross_validate_evidence,
)
from engine.quality.floors import (
    DEFAULT_MINIMUM_EVIDENCE,
    QualityFloorConfig,
    check_quality_floor,
    get_floor_for_control,
    recalculate_floor,
)
from engine.quality.gate import (
    GateBlockReason,
    QualityGateConfig,
    enforce_quality_gate,
)
from engine.quality.sampler import (
    TIER_WEIGHTS,
    Z_SCORES,
    EvidenceQualityInput,
    SamplingParameters,
    SufficiencyVerdict,
    assess_sufficiency,
    cochran_sample_size,
    score_evidence_tier,
)
from engine.quality.trending import (
    TREND_STABLE_THRESHOLD,
    PeriodScore,
    TrendDirection,
    add_period_score,
    compute_trend,
)

# ===========================================================================
# Helpers
# ===========================================================================


def _ev(tier: int = 1, quality: float = 0.9, passes: bool = True) -> EvidenceQualityInput:
    return EvidenceQualityInput(uuid4(), tier=tier, quality_score=quality, passes=passes)


def _aev(
    tier: int = 1,
    quality: float = 0.9,
    passes: bool = True,
    source_id: str = "SRC-A",
    assertion_key: str = "MFA_enabled",
) -> AnnotatedEvidenceItem:
    return AnnotatedEvidenceItem(
        evidence_id=uuid4(),
        tier=tier,
        quality_score=quality,
        passes=passes,
        source_id=source_id,
        assertion_key=assertion_key,
    )


def _period(
    period_id: str,
    ratio: float,
    count: int = 5,
    verdict: SufficiencyVerdict = SufficiencyVerdict.SUFFICIENT,
) -> PeriodScore:
    return PeriodScore(
        period_id=period_id,
        sufficiency_ratio=ratio,
        evidence_count=count,
        verdict=verdict,
    )


# ===========================================================================
# Cochran sampling (preserved)
# ===========================================================================


class TestCochranSampleSize:
    def test_standard_calculation(self):
        # n0 = (1.96^2 * 0.05 * 0.95) / 0.10^2 = 3.8416 * 0.0475 / 0.01 = 18.2476 -> ceil = 19
        params = SamplingParameters(
            confidence_level=0.95,
            expected_deviation_rate=0.05,
            tolerable_deviation_rate=0.10,
        )
        result = cochran_sample_size(params)
        assert result.n_infinite == 19
        assert result.n_final == 19  # no finite correction
        assert not result.finite_correction_applied

    def test_finite_population_correction_reduces_sample(self):
        # n0 = 2.5758^2 * 0.02 * 0.98 / 0.03^2 = 144.49 -> 145, with N=200: n < 145
        params = SamplingParameters(
            confidence_level=0.99,
            expected_deviation_rate=0.02,
            tolerable_deviation_rate=0.03,
            population_size=200,
        )
        result = cochran_sample_size(params)
        assert result.finite_correction_applied
        assert result.n_final < result.n_infinite

    def test_99_confidence_larger_than_95(self):
        base = SamplingParameters(0.95, 0.05, 0.10)
        higher = SamplingParameters(0.99, 0.05, 0.10)
        assert cochran_sample_size(higher).n_infinite > cochran_sample_size(base).n_infinite

    def test_unsupported_confidence_raises(self):
        with pytest.raises(ValueError, match="not supported"):
            cochran_sample_size(SamplingParameters(0.80, 0.05, 0.10))

    def test_tdr_not_greater_than_expected_raises(self):
        with pytest.raises(ValueError, match="must be >"):
            cochran_sample_size(SamplingParameters(0.95, 0.10, 0.05))

    def test_invalid_expected_rate_raises(self):
        with pytest.raises(ValueError):
            cochran_sample_size(SamplingParameters(0.95, 0.0, 0.10))

    def test_z_scores_correct(self):
        assert Z_SCORES[0.90] == pytest.approx(1.6449, abs=1e-3)
        assert Z_SCORES[0.95] == pytest.approx(1.9600, abs=1e-3)
        assert Z_SCORES[0.99] == pytest.approx(2.5758, abs=1e-3)

    def test_result_is_deterministic(self):
        params = SamplingParameters(0.95, 0.05, 0.10)
        assert cochran_sample_size(params).n_final == cochran_sample_size(params).n_final

    def test_small_population_finite_correction(self):
        # n0=19 with N=25: n = 19 / (1 + 18/25) = 19/1.72 ~ 11.05 -> 12
        params = SamplingParameters(0.95, 0.05, 0.10, population_size=25)
        result = cochran_sample_size(params)
        assert result.finite_correction_applied
        assert result.n_final == 12

    def test_cochran_formula_preserved_exact(self):
        """Verify the exact Cochran (1977) formula: n0 = (Z^2 * p * q) / e^2."""
        z = Z_SCORES[0.95]
        p, e = 0.05, 0.10
        q = 1.0 - p
        n0_expected = math.ceil((z**2 * p * q) / e**2)
        params = SamplingParameters(0.95, p, e)
        assert cochran_sample_size(params).n_infinite == n0_expected


# ===========================================================================
# Evidence tier scoring (preserved)
# ===========================================================================


class TestEvidenceTierScoring:
    def test_tier_1_highest_weight(self):
        assert TIER_WEIGHTS[1] == 1.00

    def test_tier_6_lowest_weight(self):
        assert TIER_WEIGHTS[6] == 0.25

    def test_decreasing_weights(self):
        weights = [TIER_WEIGHTS[t] for t in range(1, 7)]
        assert weights == sorted(weights, reverse=True)

    def test_score_evidence_tier(self):
        assert score_evidence_tier(tier=1, quality_score=0.9) == pytest.approx(0.9)

    def test_score_evidence_tier_6(self):
        assert score_evidence_tier(tier=6, quality_score=1.0) == pytest.approx(0.25)

    def test_invalid_tier_raises(self):
        with pytest.raises(ValueError, match="1\u20136"):
            score_evidence_tier(tier=7, quality_score=0.5)

    def test_invalid_quality_raises(self):
        with pytest.raises(ValueError, match="quality_score"):
            score_evidence_tier(tier=1, quality_score=1.5)


# ===========================================================================
# Evidence sufficiency (preserved)
# ===========================================================================


class TestSufficiency:
    def _make_items(self, count: int, tier: int = 1, quality: float = 0.9):
        return [_ev(tier, quality) for _ in range(count)]

    def test_sufficient_verdict(self):
        items = self._make_items(8, tier=1, quality=0.9)
        result = assess_sufficiency("C-001", items, required_sample_size=5)
        assert result.verdict == SufficiencyVerdict.SUFFICIENT

    def test_insufficient_verdict_no_evidence(self):
        result = assess_sufficiency("C-001", [], required_sample_size=10)
        assert result.verdict == SufficiencyVerdict.INSUFFICIENT
        assert result.sufficiency_ratio == 0.0

    def test_marginal_verdict(self):
        items = self._make_items(8, tier=1, quality=0.9)
        result = assess_sufficiency("C-001", items, required_sample_size=10)
        assert result.verdict == SufficiencyVerdict.MARGINAL

    def test_tier_distribution_counted(self):
        items = self._make_items(3, tier=1) + self._make_items(2, tier=3)
        result = assess_sufficiency("C-001", items, required_sample_size=5)
        assert result.tier_distribution[1] == 3
        assert result.tier_distribution[3] == 2

    def test_invalid_required_sample_size_raises(self):
        with pytest.raises(ValueError, match="positive"):
            assess_sufficiency("C-001", [], required_sample_size=0)


# ===========================================================================
# Hard quality floors
# ===========================================================================


class TestQualityFloorConfig:
    def test_default_minimum_is_3(self):
        config = QualityFloorConfig()
        assert config.default_minimum == DEFAULT_MINIMUM_EVIDENCE
        assert DEFAULT_MINIMUM_EVIDENCE == 3

    def test_invalid_default_raises(self):
        with pytest.raises(ValueError, match="default_minimum"):
            QualityFloorConfig(default_minimum=0)

    def test_invalid_override_raises(self):
        with pytest.raises(ValueError, match="per_control_override"):
            QualityFloorConfig(per_control_overrides={"C-001": 0})

    def test_per_control_override_applied(self):
        config = QualityFloorConfig(per_control_overrides={"C-001": 5})
        assert get_floor_for_control(config, "C-001") == 5
        assert get_floor_for_control(config, "C-002") == 3

    def test_override_not_required(self):
        config = QualityFloorConfig()
        assert get_floor_for_control(config, "any-control") == 3


class TestCheckQualityFloor:
    def test_floor_passed_exact_minimum(self):
        items = [_ev() for _ in range(3)]
        result = check_quality_floor("C-001", items)
        assert result.passed is True
        assert result.deficit == 0

    def test_floor_passed_above_minimum(self):
        items = [_ev() for _ in range(5)]
        result = check_quality_floor("C-001", items)
        assert result.passed is True
        assert result.deficit == 0

    def test_floor_failed_below_minimum(self):
        items = [_ev(), _ev()]
        result = check_quality_floor("C-001", items)
        assert result.passed is False
        assert result.deficit == 1

    def test_floor_failed_no_items(self):
        result = check_quality_floor("C-001", [])
        assert result.passed is False
        assert result.deficit == 3

    def test_floor_uses_default_config_when_none(self):
        items = [_ev() for _ in range(3)]
        result = check_quality_floor("C-001", items, config=None)
        assert result.required_minimum == 3

    def test_floor_uses_per_control_override(self):
        config = QualityFloorConfig(per_control_overrides={"C-001": 7})
        items = [_ev() for _ in range(5)]
        result = check_quality_floor("C-001", items, config)
        assert result.passed is False
        assert result.required_minimum == 7
        assert result.deficit == 2


class TestRecalculateFloor:
    def test_recalculate_after_add(self):
        items = [_ev(), _ev()]  # 2 items, floor 3 -> FAIL
        new_item = _ev()
        result = recalculate_floor(items, "C-001", added_items=[new_item])
        assert result.passed is True
        assert result.evidence_count == 3

    def test_recalculate_after_remove(self):
        items = [_ev() for _ in range(4)]  # 4 items -> PASS
        remove_id = items[0].evidence_id
        result = recalculate_floor(items, "C-001", removed_ids={remove_id})
        assert result.evidence_count == 3
        assert result.passed is True  # 3 == floor 3

    def test_recalculate_remove_below_floor(self):
        items = [_ev() for _ in range(3)]
        remove_id = items[0].evidence_id
        result = recalculate_floor(items, "C-001", removed_ids={remove_id})
        assert result.passed is False
        assert result.evidence_count == 2

    def test_recalculate_remove_then_add_net_zero(self):
        items = [_ev() for _ in range(3)]
        removed_id = items[0].evidence_id
        new_item = _ev()
        result = recalculate_floor(
            items, "C-001", added_items=[new_item], removed_ids={removed_id}
        )
        assert result.evidence_count == 3
        assert result.passed is True


# ===========================================================================
# Cross-validation
# ===========================================================================


class TestCrossValidation:
    def test_empty_items_returns_empty_list(self):
        assert cross_validate_evidence([]) == []

    def test_single_source_always_consistent(self):
        items = [_aev(source_id="SRC-A"), _aev(source_id="SRC-A")]
        results = cross_validate_evidence(items)
        assert len(results) == 1
        assert results[0].consistent is True
        assert results[0].consistency_score == 1.0

    def test_two_consistent_sources_both_pass(self):
        items = [
            _aev(passes=True, source_id="SRC-A"),
            _aev(passes=True, source_id="SRC-B"),
        ]
        results = cross_validate_evidence(items)
        assert results[0].consistent is True
        assert results[0].majority_passes is True
        assert results[0].inconsistent_source_ids == []

    def test_two_inconsistent_sources(self):
        items = [
            _aev(passes=True, source_id="SRC-A"),
            _aev(passes=False, source_id="SRC-B"),
        ]
        results = cross_validate_evidence(items)
        assert results[0].consistent is False
        assert results[0].consistency_score == pytest.approx(0.5)
        assert len(results[0].inconsistent_source_ids) == 1

    def test_three_sources_two_agree(self):
        items = [
            _aev(passes=True, source_id="SRC-A"),
            _aev(passes=True, source_id="SRC-B"),
            _aev(passes=False, source_id="SRC-C"),
        ]
        results = cross_validate_evidence(items)
        assert results[0].consistent is False
        assert results[0].consistency_score == pytest.approx(2 / 3, abs=1e-4)
        assert "SRC-C" in results[0].inconsistent_source_ids

    def test_multiple_assertion_keys_produce_separate_results(self):
        items = [
            _aev(assertion_key="MFA_enabled", source_id="SRC-A"),
            _aev(assertion_key="logging_on", source_id="SRC-A"),
        ]
        results = cross_validate_evidence(items)
        keys = {r.assertion_key for r in results}
        assert keys == {"MFA_enabled", "logging_on"}

    def test_source_count_correct(self):
        items = [
            _aev(source_id="SRC-A"),
            _aev(source_id="SRC-B"),
            _aev(source_id="SRC-C"),
        ]
        results = cross_validate_evidence(items)
        assert results[0].source_count == 3

    def test_avg_quality_calculated(self):
        items = [
            _aev(quality=0.8, source_id="SRC-A"),
            _aev(quality=0.8, source_id="SRC-A"),
        ]
        results = cross_validate_evidence(items)
        assert results[0].sources[0].avg_quality == pytest.approx(0.8)

    def test_result_is_deterministic(self):
        items = [_aev(source_id="SRC-A"), _aev(source_id="SRC-B")]
        r1 = cross_validate_evidence(items)
        r2 = cross_validate_evidence(items)
        assert r1[0].consistent == r2[0].consistent
        assert r1[0].consistency_score == r2[0].consistency_score


# ===========================================================================
# Conflict resolution
# ===========================================================================


class TestDetectConflicts:
    def test_no_conflicts_all_identical(self):
        items = [_ev(tier=1, quality=0.9), _ev(tier=1, quality=0.9)]
        assert detect_conflicts(items) == []

    def test_pass_fail_disagreement_detected(self):
        a = _ev(tier=1, quality=0.9, passes=True)
        b = _ev(tier=2, quality=0.9, passes=False)
        conflicts = detect_conflicts([a, b])
        assert len(conflicts) == 1
        assert conflicts[0].conflict_type == ConflictType.PASS_FAIL_DISAGREEMENT

    def test_quality_variance_conflict(self):
        a = _ev(tier=1, quality=0.9)
        b = _ev(tier=1, quality=0.5)
        conflicts = detect_conflicts([a, b])
        assert len(conflicts) == 1
        assert conflicts[0].conflict_type == ConflictType.QUALITY_VARIANCE

    def test_no_quality_variance_below_threshold(self):
        a = _ev(tier=1, quality=0.9)
        b = _ev(tier=1, quality=0.65)  # diff = 0.25, < 0.30 threshold
        assert detect_conflicts([a, b]) == []

    def test_quality_variance_at_threshold(self):
        a = _ev(tier=1, quality=0.9)
        b = _ev(tier=1, quality=0.6)  # diff exactly 0.30
        conflicts = detect_conflicts([a, b])
        assert len(conflicts) == 1
        assert conflicts[0].conflict_type == ConflictType.QUALITY_VARIANCE

    def test_prefer_lower_tier_number(self):
        # Tier 1 beats tier 3 (weight 1.00 > 0.70)
        a = _ev(tier=1, quality=0.5, passes=True)
        b = _ev(tier=3, quality=0.9, passes=False)
        conflicts = detect_conflicts([a, b])
        assert conflicts[0].preferred_id == a.evidence_id
        assert conflicts[0].flagged_id == b.evidence_id

    def test_prefer_higher_quality_on_same_tier(self):
        a = _ev(tier=2, quality=0.4, passes=True)
        b = _ev(tier=2, quality=0.9, passes=False)
        conflicts = detect_conflicts([a, b])
        assert conflicts[0].preferred_id == b.evidence_id
        assert conflicts[0].flagged_id == a.evidence_id

    def test_empty_items_no_conflicts(self):
        assert detect_conflicts([]) == []

    def test_single_item_no_conflicts(self):
        assert detect_conflicts([_ev()]) == []

    def test_custom_threshold(self):
        a = _ev(tier=1, quality=0.9)
        b = _ev(tier=1, quality=0.75)  # diff = 0.15, below default 0.30 but above 0.10
        # Default threshold: no conflict
        assert detect_conflicts([a, b]) == []
        # Custom threshold of 0.10: conflict
        conflicts = detect_conflicts([a, b], quality_variance_threshold=0.10)
        assert len(conflicts) == 1


class TestResolveConflicts:
    def test_flagged_items_excluded_from_accepted(self):
        a = _ev(tier=1, quality=0.9, passes=True)
        b = _ev(tier=3, quality=0.5, passes=False)
        result = resolve_conflicts("C-001", [a, b])
        assert b.evidence_id in result.flagged_item_ids
        assert all(it.evidence_id != b.evidence_id for it in result.accepted_items)

    def test_winner_in_accepted_set(self):
        a = _ev(tier=1, quality=0.9, passes=True)
        b = _ev(tier=3, quality=0.5, passes=False)
        result = resolve_conflicts("C-001", [a, b])
        assert any(it.evidence_id == a.evidence_id for it in result.accepted_items)

    def test_no_conflicts_all_accepted(self):
        items = [_ev() for _ in range(4)]
        result = resolve_conflicts("C-001", items)
        assert result.conflict_count == 0
        assert len(result.accepted_items) == 4
        assert not result.has_conflicts

    def test_has_conflicts_flag(self):
        a = _ev(passes=True)
        b = _ev(passes=False)
        result = resolve_conflicts("C-001", [a, b])
        assert result.has_conflicts is True

    def test_total_items_count(self):
        items = [_ev() for _ in range(5)]
        result = resolve_conflicts("C-001", items)
        assert result.total_items == 5

    def test_result_is_deterministic(self):
        a = _ev(tier=1, quality=0.9, passes=True)
        b = _ev(tier=3, quality=0.5, passes=False)
        r1 = resolve_conflicts("C-001", [a, b])
        r2 = resolve_conflicts("C-001", [a, b])
        assert r1.flagged_item_ids == r2.flagged_item_ids


# ===========================================================================
# Quality gate enforcement
# ===========================================================================


class TestQualityGate:
    def test_gate_passes_floor_met_and_sufficient(self):
        # 8 tier-1 items at quality 0.9, required_n=5 -> ratio 1.44 -> capped sufficient
        items = [_ev(tier=1, quality=0.9) for _ in range(8)]
        result = enforce_quality_gate("C-001", items, required_sample_size=5)
        assert result.passed is True
        assert result.block_reasons == []

    def test_gate_blocks_when_floor_not_met(self):
        items = [_ev()]  # only 1 item, floor=3
        result = enforce_quality_gate("C-001", items, required_sample_size=5)
        assert result.passed is False
        assert GateBlockReason.FLOOR_NOT_MET in result.block_reasons

    def test_gate_blocks_when_insufficient(self):
        # 1 tier-6 item at 0.5 quality, required_n=20 -> tiny ratio
        items = [_ev(tier=6, quality=0.5) for _ in range(3)]
        result = enforce_quality_gate("C-001", items, required_sample_size=20)
        assert GateBlockReason.INSUFFICIENT_EVIDENCE in result.block_reasons

    def test_gate_passes_marginal_by_default(self):
        # Marginal verdict should NOT block (only INSUFFICIENT blocks)
        # 8 tier-1 at 0.9, required_n=10 -> ratio=0.72 -> marginal
        items = [_ev(tier=1, quality=0.9) for _ in range(8)]
        result = enforce_quality_gate("C-001", items, required_sample_size=10)
        # May block on floor (8 >= 3) but not on insufficiency (marginal, not insufficient)
        assert GateBlockReason.INSUFFICIENT_EVIDENCE not in result.block_reasons

    def test_gate_does_not_block_on_conflicts_by_default(self):
        items = [_ev() for _ in range(5)]
        result = enforce_quality_gate(
            "C-001", items, required_sample_size=5, unresolved_conflict_count=3
        )
        assert GateBlockReason.UNRESOLVED_CONFLICTS not in result.block_reasons

    def test_gate_blocks_on_conflicts_when_configured(self):
        gate_cfg = QualityGateConfig(
            block_on_floor_failure=False,
            block_on_insufficient=False,
            block_on_conflicts=True,
        )
        items = [_ev() for _ in range(5)]
        result = enforce_quality_gate(
            "C-001", items, required_sample_size=5,
            gate_config=gate_cfg, unresolved_conflict_count=2,
        )
        assert result.passed is False
        assert GateBlockReason.UNRESOLVED_CONFLICTS in result.block_reasons

    def test_gate_multiple_block_reasons(self):
        # 0 items -> floor failure + insufficient
        result = enforce_quality_gate("C-001", [], required_sample_size=5)
        assert GateBlockReason.FLOOR_NOT_MET in result.block_reasons
        assert GateBlockReason.INSUFFICIENT_EVIDENCE in result.block_reasons
        assert result.passed is False

    def test_gate_carries_floor_and_sufficiency_results(self):
        items = [_ev(tier=1, quality=0.9) for _ in range(8)]
        result = enforce_quality_gate("C-001", items, required_sample_size=5)
        assert result.floor_result is not None
        assert result.sufficiency_result is not None

    def test_gate_custom_floor_config(self):
        # Custom floor of 10 with only 5 items
        floor_cfg = QualityFloorConfig(per_control_overrides={"C-001": 10})
        items = [_ev() for _ in range(5)]
        result = enforce_quality_gate(
            "C-001", items, required_sample_size=5, floor_config=floor_cfg
        )
        assert GateBlockReason.FLOOR_NOT_MET in result.block_reasons

    def test_gate_result_is_deterministic(self):
        items = [_ev(tier=1, quality=0.9) for _ in range(6)]
        r1 = enforce_quality_gate("C-001", items, required_sample_size=5)
        r2 = enforce_quality_gate("C-001", items, required_sample_size=5)
        assert r1.passed == r2.passed
        assert r1.block_reasons == r2.block_reasons

    def test_gate_preserves_cochran_in_context(self):
        """Gate integration with Cochran-derived required_sample_size."""
        from engine.quality.sampler import SamplingParameters, cochran_sample_size
        params = SamplingParameters(0.95, 0.05, 0.10)
        n = cochran_sample_size(params).n_final  # = 19
        # Provide 20 high-quality tier-1 items -> should pass gate
        items = [_ev(tier=1, quality=0.95) for _ in range(20)]
        result = enforce_quality_gate("C-001", items, required_sample_size=n)
        assert result.passed is True


# ===========================================================================
# Quality score trending
# ===========================================================================


class TestTrending:
    def test_empty_periods_raises(self):
        with pytest.raises(ValueError, match="must not be empty"):
            compute_trend("C-001", [])

    def test_single_period_insufficient_data(self):
        trend = compute_trend("C-001", [_period("Q1", 0.8)])
        assert trend.direction == TrendDirection.INSUFFICIENT_DATA
        assert trend.delta == 0.0

    def test_trend_improving(self):
        periods = [_period("Q1", 0.5), _period("Q2", 0.75), _period("Q3", 0.9)]
        trend = compute_trend("C-001", periods)
        assert trend.direction == TrendDirection.IMPROVING
        assert trend.delta == pytest.approx(0.4)

    def test_trend_declining(self):
        periods = [_period("Q1", 0.9), _period("Q2", 0.75), _period("Q3", 0.5)]
        trend = compute_trend("C-001", periods)
        assert trend.direction == TrendDirection.DECLINING
        assert trend.delta == pytest.approx(-0.4)

    def test_trend_stable_within_threshold(self):
        periods = [_period("Q1", 0.8), _period("Q2", 0.82)]
        trend = compute_trend("C-001", periods)
        assert trend.direction == TrendDirection.STABLE

    def test_trend_stable_at_threshold(self):
        # delta = TREND_STABLE_THRESHOLD exactly -> STABLE
        periods = [_period("Q1", 0.80), _period("Q2", round(0.80 + TREND_STABLE_THRESHOLD, 4))]
        trend = compute_trend("C-001", periods)
        assert trend.direction == TrendDirection.STABLE

    def test_trend_improving_just_above_threshold(self):
        delta = TREND_STABLE_THRESHOLD + 0.001
        periods = [_period("Q1", 0.50), _period("Q2", round(0.50 + delta, 4))]
        trend = compute_trend("C-001", periods)
        assert trend.direction == TrendDirection.IMPROVING

    def test_periods_preserved_in_order(self):
        p1 = _period("Q1", 0.5)
        p2 = _period("Q2", 0.9)
        trend = compute_trend("C-001", [p1, p2])
        assert trend.periods[0].period_id == "Q1"
        assert trend.periods[1].period_id == "Q2"

    def test_add_period_score_appends_and_recomputes(self):
        initial = compute_trend("C-001", [_period("Q1", 0.5), _period("Q2", 0.6)])
        assert initial.direction == TrendDirection.IMPROVING
        # Add a declining period
        updated = add_period_score(initial, _period("Q3", 0.3))
        assert updated.periods[-1].period_id == "Q3"
        # delta = 0.3 - 0.5 = -0.2 -> DECLINING
        assert updated.direction == TrendDirection.DECLINING

    def test_add_period_does_not_mutate_original(self):
        initial = compute_trend("C-001", [_period("Q1", 0.5), _period("Q2", 0.6)])
        add_period_score(initial, _period("Q3", 0.9))
        assert len(initial.periods) == 2

    def test_trend_result_is_deterministic(self):
        periods = [_period("Q1", 0.5), _period("Q2", 0.9)]
        t1 = compute_trend("C-001", periods)
        t2 = compute_trend("C-001", periods)
        assert t1.direction == t2.direction
        assert t1.delta == t2.delta

    def test_trend_delta_uses_earliest_and_latest(self):
        periods = [
            _period("Q1", 0.50),
            _period("Q2", 0.90),  # spike in the middle
            _period("Q3", 0.60),
        ]
        trend = compute_trend("C-001", periods)
        # delta = latest(0.60) - earliest(0.50) = 0.10 > 0.05 -> IMPROVING
        assert trend.delta == pytest.approx(0.10)
        assert trend.direction == TrendDirection.IMPROVING
