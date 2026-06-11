"""Unit tests — Risk Engine (Bayesian scoring, materiality, TDR, 5x penalty)."""
import pytest
from uuid import uuid4
from engine.risk.scorer import (
    AuditConclusion,
    EvidenceItem,
    RiskScoreResult,
    PENALTY_RATIO,
    MATERIALITY_THRESHOLD,
    TDR_MATERIAL,
    TDR_NON_MATERIAL,
    bayesian_update,
    compute_deviation_rate,
    score_control,
    _THRESHOLD_EFFECTIVE,
    _THRESHOLD_PARTIAL,
)


def make_evidence(passes: bool, quality: float = 0.9, tier: int = 1) -> EvidenceItem:
    return EvidenceItem(evidence_id=uuid4(), passes=passes, quality_score=quality, tier=tier)


class TestConstants:
    def test_penalty_ratio(self):
        assert PENALTY_RATIO == 5.0

    def test_effective_threshold_with_5x_penalty(self):
        # threshold_effective = 1 - 1/(1+5) = 1 - 1/6 ≈ 0.8333
        expected = 1.0 - (1.0 / (1.0 + 5.0))
        assert abs(_THRESHOLD_EFFECTIVE - expected) < 1e-10

    def test_partial_threshold_with_5x_penalty(self):
        # threshold_partial = 1 - 1/(1+2.5) ≈ 0.7143
        expected = 1.0 - (1.0 / (1.0 + 2.5))
        assert abs(_THRESHOLD_PARTIAL - expected) < 1e-10

    def test_materiality_threshold(self):
        assert MATERIALITY_THRESHOLD == 3.5

    def test_tdr_values(self):
        assert TDR_MATERIAL == 0.05
        assert TDR_NON_MATERIAL == 0.10


class TestBayesianUpdate:
    def test_neutral_prior_no_evidence(self):
        posterior = bayesian_update(0.5, [])
        assert posterior == 0.5

    def test_strong_positive_evidence_increases_posterior(self):
        evidence = [make_evidence(passes=True, quality=1.0, tier=1) for _ in range(5)]
        posterior = bayesian_update(0.5, evidence)
        assert posterior > 0.9

    def test_strong_negative_evidence_decreases_posterior(self):
        evidence = [make_evidence(passes=False, quality=1.0, tier=1) for _ in range(5)]
        posterior = bayesian_update(0.5, evidence)
        assert posterior < 0.1

    def test_posterior_stays_in_bounds(self):
        evidence = [make_evidence(passes=True, quality=1.0) for _ in range(20)]
        posterior = bayesian_update(0.5, evidence)
        assert 0.0 <= posterior <= 1.0

    def test_invalid_prior_raises(self):
        with pytest.raises(ValueError, match="Prior must be in"):
            bayesian_update(1.5, [])

    def test_low_quality_evidence_has_less_impact(self):
        high_quality = [make_evidence(True, quality=0.95) for _ in range(3)]
        low_quality = [make_evidence(True, quality=0.1) for _ in range(3)]
        post_high = bayesian_update(0.5, high_quality)
        post_low = bayesian_update(0.5, low_quality)
        assert post_high > post_low

    def test_update_is_deterministic(self):
        evidence = [make_evidence(True), make_evidence(False), make_evidence(True)]
        p1 = bayesian_update(0.5, evidence)
        p2 = bayesian_update(0.5, evidence)
        assert p1 == p2


class TestDeviationRate:
    def test_all_passing(self):
        evidence = [make_evidence(True) for _ in range(10)]
        assert compute_deviation_rate(evidence) == 0.0

    def test_all_failing(self):
        evidence = [make_evidence(False) for _ in range(10)]
        assert compute_deviation_rate(evidence) == 1.0

    def test_partial_failure(self):
        evidence = [make_evidence(True)] * 8 + [make_evidence(False)] * 2
        rate = compute_deviation_rate(evidence)
        assert abs(rate - 0.2) < 1e-10

    def test_empty_evidence(self):
        assert compute_deviation_rate([]) == 0.0


class TestScoreControl:
    def test_effective_conclusion_with_strong_evidence(self):
        evidence = [make_evidence(passes=True, quality=0.95, tier=1) for _ in range(6)]
        result = score_control("C-001", risk_weight=2.0, prior=0.5, evidence_items=evidence)
        assert result.conclusion == AuditConclusion.EFFECTIVE
        assert result.confidence > 0.0
        assert result.penalty_applied is True

    def test_not_effective_conclusion_with_poor_evidence(self):
        evidence = [make_evidence(passes=False, quality=0.9, tier=1) for _ in range(6)]
        result = score_control("C-002", risk_weight=2.0, prior=0.5, evidence_items=evidence)
        assert result.conclusion == AuditConclusion.NOT_EFFECTIVE

    def test_no_evidence_returns_not_effective(self):
        result = score_control("C-003", risk_weight=2.0, prior=0.5, evidence_items=[])
        assert result.conclusion == AuditConclusion.NOT_EFFECTIVE
        assert result.evidence_count == 0

    def test_material_control_flagged(self):
        result = score_control("C-004", risk_weight=4.0, prior=0.5, evidence_items=[])
        assert result.is_material is True
        assert result.tdr_threshold == TDR_MATERIAL

    def test_non_material_control_flagged(self):
        result = score_control("C-005", risk_weight=2.0, prior=0.5, evidence_items=[])
        assert result.is_material is False
        assert result.tdr_threshold == TDR_NON_MATERIAL

    def test_tdr_exceeded_triggers_not_effective(self):
        # 4 out of 5 fail → dev_rate 0.80 >> TDR 0.10
        evidence = [make_evidence(False)] * 4 + [make_evidence(True)]
        result = score_control("C-006", risk_weight=2.0, prior=0.9, evidence_items=evidence)
        assert result.tdr_exceeded is True
        assert result.conclusion == AuditConclusion.NOT_EFFECTIVE

    def test_traceability_contains_evidence_ids(self):
        ev = [make_evidence(True) for _ in range(3)]
        result = score_control("C-007", risk_weight=1.0, prior=0.5, evidence_items=ev)
        assert "evidence_ids" in result.traceability
        assert len(result.traceability["evidence_ids"]) == 3

    def test_5x_penalty_threshold_encoded_in_traceability(self):
        result = score_control("C-008", risk_weight=1.0, prior=0.5, evidence_items=[])
        assert result.traceability["penalty_ratio"] == 5.0
