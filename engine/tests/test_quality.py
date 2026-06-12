"""Unit tests — Quality Engine (Cochran sampling, tier scoring, sufficiency)."""
import math
import pytest
from uuid import uuid4
from engine.quality.sampler import (
    cochran_sample_size,
    assess_sufficiency,
    score_evidence_tier,
    SamplingParameters,
    EvidenceQualityInput,
    SufficiencyVerdict,
    TIER_WEIGHTS,
    Z_SCORES,
)


class TestCochranSampleSize:
    def test_standard_calculation(self):
        # n₀ = (1.96² * 0.05 * 0.95) / 0.10² = 3.8416 * 0.0475 / 0.01 = 18.2476 → ceil = 19
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
        # Use params that produce a larger n0 so finite correction has noticeable effect
        # n₀ = 2.5758² * 0.02 * 0.98 / 0.03² = 144.49 → 145, with N=200: n=85
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
        base = SamplingParameters(
            confidence_level=0.95,
            expected_deviation_rate=0.05,
            tolerable_deviation_rate=0.10,
        )
        higher = SamplingParameters(
            confidence_level=0.99,
            expected_deviation_rate=0.05,
            tolerable_deviation_rate=0.10,
        )
        n_95 = cochran_sample_size(base).n_infinite
        n_99 = cochran_sample_size(higher).n_infinite
        assert n_99 > n_95

    def test_unsupported_confidence_raises(self):
        with pytest.raises(ValueError, match="not supported"):
            cochran_sample_size(SamplingParameters(
                confidence_level=0.80,
                expected_deviation_rate=0.05,
                tolerable_deviation_rate=0.10,
            ))

    def test_tdr_not_greater_than_expected_raises(self):
        with pytest.raises(ValueError, match="must be >"):
            cochran_sample_size(SamplingParameters(
                confidence_level=0.95,
                expected_deviation_rate=0.10,
                tolerable_deviation_rate=0.05,
            ))

    def test_invalid_expected_rate_raises(self):
        with pytest.raises(ValueError):
            cochran_sample_size(SamplingParameters(
                confidence_level=0.95,
                expected_deviation_rate=0.0,
                tolerable_deviation_rate=0.10,
            ))

    def test_z_scores_correct(self):
        assert Z_SCORES[0.90] == pytest.approx(1.6449, abs=1e-3)
        assert Z_SCORES[0.95] == pytest.approx(1.9600, abs=1e-3)
        assert Z_SCORES[0.99] == pytest.approx(2.5758, abs=1e-3)

    def test_result_is_deterministic(self):
        params = SamplingParameters(0.95, 0.05, 0.10)
        r1 = cochran_sample_size(params)
        r2 = cochran_sample_size(params)
        assert r1.n_final == r2.n_final

    def test_small_population_finite_correction(self):
        # n0=19 with N=25: n = 19 / (1 + 18/25) = 19/1.72 ≈ 11.05 → 12
        params = SamplingParameters(
            confidence_level=0.95,
            expected_deviation_rate=0.05,
            tolerable_deviation_rate=0.10,
            population_size=25,
        )
        result = cochran_sample_size(params)
        assert result.finite_correction_applied
        assert result.n_final == 12


class TestEvidenceTierScoring:
    def test_tier_1_highest_weight(self):
        assert TIER_WEIGHTS[1] == 1.00

    def test_tier_6_lowest_weight(self):
        assert TIER_WEIGHTS[6] == 0.25

    def test_decreasing_weights(self):
        weights = [TIER_WEIGHTS[t] for t in range(1, 7)]
        assert weights == sorted(weights, reverse=True)

    def test_score_evidence_tier(self):
        score = score_evidence_tier(tier=1, quality_score=0.9)
        assert score == pytest.approx(0.9 * 1.00)

    def test_score_evidence_tier_6(self):
        score = score_evidence_tier(tier=6, quality_score=1.0)
        assert score == pytest.approx(0.25)

    def test_invalid_tier_raises(self):
        with pytest.raises(ValueError, match="1\u20136"):
            score_evidence_tier(tier=7, quality_score=0.5)

    def test_invalid_quality_raises(self):
        with pytest.raises(ValueError, match="quality_score"):
            score_evidence_tier(tier=1, quality_score=1.5)


class TestSufficiency:
    def _make_items(self, count: int, tier: int = 1, quality: float = 0.9):
        return [EvidenceQualityInput(uuid4(), tier=tier, quality_score=quality) for _ in range(count)]

    def test_sufficient_verdict(self):
        # required_n=5, supply 8 tier-1 items at quality 0.9 → weighted_sum = 8*0.9 = 7.2 → ratio=1.44 → cap to 1.0
        items = self._make_items(8, tier=1, quality=0.9)
        result = assess_sufficiency("C-001", items, required_sample_size=5)
        assert result.verdict == SufficiencyVerdict.SUFFICIENT

    def test_insufficient_verdict_no_evidence(self):
        result = assess_sufficiency("C-001", [], required_sample_size=10)
        assert result.verdict == SufficiencyVerdict.INSUFFICIENT
        assert result.sufficiency_ratio == 0.0

    def test_marginal_verdict(self):
        # required_n=10, supply 8 tier-1 at 0.9 → 7.2 / 10 = 0.72 → marginal (>= 0.60, < 0.80)
        items = self._make_items(8, tier=1, quality=0.9)
        result = assess_sufficiency("C-001", items, required_sample_size=10)
        assert result.verdict == SufficiencyVerdict.MARGINAL

    def test_tier_distribution_counted(self):
        items = (
            self._make_items(3, tier=1)
            + self._make_items(2, tier=3)
        )
        result = assess_sufficiency("C-001", items, required_sample_size=5)
        assert result.tier_distribution[1] == 3
        assert result.tier_distribution[3] == 2

    def test_invalid_required_sample_size_raises(self):
        with pytest.raises(ValueError, match="positive"):
            assess_sufficiency("C-001", [], required_sample_size=0)
