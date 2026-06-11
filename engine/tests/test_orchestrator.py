"""Unit tests — Engine Orchestrator (pipeline, conclusion assembly, FastAPI gateway)."""
import pytest
from uuid import uuid4
from fastapi.testclient import TestClient

from engine.orchestrator.pipeline import (
    ControlInput,
    EvidenceInput,
    ReasoningPipeline,
    ReasoningRequest,
    _downgrade_conclusion,
    _adjust_for_sufficiency,
)
from engine.orchestrator.gateway import app
from engine.risk.scorer import AuditConclusion
from engine.quality.sampler import SufficiencyVerdict


client = TestClient(app)


def make_evidence(passes: bool = True, tier: int = 1, quality: float = 0.9) -> EvidenceInput:
    return EvidenceInput(
        evidence_id=uuid4(),
        content=f"evidence content passes={passes}",
        tier=tier,
        quality_score=quality,
        passes=passes,
    )


def make_control(risk_weight: float = 2.0, prior: float = 0.5) -> ControlInput:
    return ControlInput(
        control_id="NIS2-21a-001",
        title="Risk analysis and information security policies",
        article_ref="NIS2-21a",
        risk_weight=risk_weight,
        prior=prior,
    )


class TestConclusionAdjustment:
    def test_downgrade_effective_to_partial(self):
        result = _downgrade_conclusion(AuditConclusion.EFFECTIVE, steps=1)
        assert result == AuditConclusion.PARTIALLY_EFFECTIVE

    def test_downgrade_partial_to_not_effective(self):
        result = _downgrade_conclusion(AuditConclusion.PARTIALLY_EFFECTIVE, steps=1)
        assert result == AuditConclusion.NOT_EFFECTIVE

    def test_downgrade_not_effective_stays(self):
        result = _downgrade_conclusion(AuditConclusion.NOT_EFFECTIVE, steps=1)
        assert result == AuditConclusion.NOT_EFFECTIVE

    def test_insufficient_sufficiency_downgrades(self):
        conclusion, confidence = _adjust_for_sufficiency(
            AuditConclusion.EFFECTIVE, 0.9, SufficiencyVerdict.INSUFFICIENT
        )
        assert conclusion == AuditConclusion.PARTIALLY_EFFECTIVE
        assert confidence < 0.9

    def test_marginal_sufficiency_reduces_confidence(self):
        conclusion, confidence = _adjust_for_sufficiency(
            AuditConclusion.EFFECTIVE, 0.9, SufficiencyVerdict.MARGINAL
        )
        assert conclusion == AuditConclusion.EFFECTIVE
        assert abs(confidence - 0.72) < 1e-10

    def test_sufficient_no_change(self):
        conclusion, confidence = _adjust_for_sufficiency(
            AuditConclusion.EFFECTIVE, 0.9, SufficiencyVerdict.SUFFICIENT
        )
        assert conclusion == AuditConclusion.EFFECTIVE
        assert confidence == 0.9


class TestReasoningPipeline:
    def test_full_pipeline_run(self):
        pipeline = ReasoningPipeline()
        evidence = [make_evidence(True) for _ in range(8)]
        request = ReasoningRequest(
            control=make_control(risk_weight=2.0, prior=0.5),
            evidence_items=evidence,
            confidence_level=0.95,
            expected_deviation_rate=0.05,
        )
        result = pipeline.run(request)
        assert result.control_id == "NIS2-21a-001"
        assert result.in_scope is True
        assert result.conclusion in list(AuditConclusion)
        assert 0.0 <= result.confidence_pct <= 100.0
        assert result.evidence_count == 8

    def test_pipeline_with_no_evidence_returns_not_effective(self):
        pipeline = ReasoningPipeline()
        request = ReasoningRequest(
            control=make_control(),
            evidence_items=[],
        )
        result = pipeline.run(request)
        assert result.conclusion == AuditConclusion.NOT_EFFECTIVE

    def test_pipeline_deterministic(self):
        pipeline = ReasoningPipeline()
        evidence = [make_evidence(True, quality=0.85) for _ in range(5)]
        ctrl = ControlInput("C-001", "Control", "NIS2-21b", 2.0, prior=0.5)
        req = ReasoningRequest(control=ctrl, evidence_items=evidence)
        r1 = pipeline.run(req)
        r2 = pipeline.run(req)
        assert r1.conclusion == r2.conclusion
        assert r1.confidence_pct == r2.confidence_pct

    def test_working_paper_chain_valid(self):
        pipeline = ReasoningPipeline()
        evidence = [make_evidence(True) for _ in range(3)]
        result = pipeline.run(ReasoningRequest(
            control=make_control(),
            evidence_items=evidence,
        ))
        chain_info = result.working_paper["evidence_chain"]
        assert chain_info["chain_valid"] is True
        assert chain_info["length"] == 3

    def test_pipeline_trace_present(self):
        pipeline = ReasoningPipeline()
        result = pipeline.run(ReasoningRequest(
            control=make_control(),
            evidence_items=[make_evidence()],
        ))
        trace = result.pipeline_trace
        assert "scope" in trace
        assert "quality" in trace
        assert "risk" in trace
        assert "conclusion_adjustment" in trace
        assert "chain" in trace

    def test_penalty_ratio_in_trace(self):
        pipeline = ReasoningPipeline()
        result = pipeline.run(ReasoningRequest(
            control=make_control(),
            evidence_items=[make_evidence()],
        ))
        assert result.pipeline_trace["risk"]["penalty_ratio"] == 5.0


class TestFastAPIGateway:
    def test_health_endpoint(self):
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"

    def test_info_endpoint(self):
        resp = client.get("/info")
        data = resp.json()
        assert data["deterministic"] is True
        assert data["scoring"]["penalty"] == "5x_asymmetric_false_negative"

    def test_reason_endpoint_success(self):
        payload = {
            "control": {
                "control_id": "NIS2-21a-001",
                "title": "Risk analysis",
                "article_ref": "NIS2-21a",
                "risk_weight": 2.0,
                "prior": 0.5,
            },
            "evidence_items": [
                {
                    "evidence_id": str(uuid4()),
                    "content": "SIEM log export shows access controls",
                    "tier": 1,
                    "quality_score": 0.9,
                    "passes": True,
                }
            ],
            "confidence_level": 0.95,
            "expected_deviation_rate": 0.05,
        }
        resp = client.post("/reason", json=payload)
        assert resp.status_code == 200
        data = resp.json()
        assert data["control_id"] == "NIS2-21a-001"
        assert data["conclusion"] in ["Effective", "Partially Effective", "Not Effective"]
        assert "chain_valid" in data

    def test_reason_endpoint_no_evidence(self):
        payload = {
            "control": {
                "control_id": "C-001",
                "title": "Test control",
                "article_ref": "NIS2-21b",
                "risk_weight": 1.0,
            },
            "evidence_items": [],
            "confidence_level": 0.95,
            "expected_deviation_rate": 0.05,
        }
        resp = client.post("/reason", json=payload)
        assert resp.status_code == 200
        assert resp.json()["conclusion"] == "Not Effective"

    def test_reason_invalid_confidence_level(self):
        payload = {
            "control": {
                "control_id": "C-001",
                "title": "Test",
                "article_ref": "NIS2-21a",
                "risk_weight": 1.0,
            },
            "evidence_items": [],
            "confidence_level": 0.80,  # Not in {0.90, 0.95, 0.99}
            "expected_deviation_rate": 0.05,
        }
        resp = client.post("/reason", json=payload)
        assert resp.status_code == 422

    def test_batch_endpoint(self):
        single = {
            "control": {
                "control_id": "B-001",
                "title": "Batch control",
                "article_ref": "NIS2-21c",
                "risk_weight": 1.5,
            },
            "evidence_items": [],
            "confidence_level": 0.95,
            "expected_deviation_rate": 0.05,
        }
        resp = client.post("/reason/batch", json=[single, single])
        assert resp.status_code == 200
        assert len(resp.json()) == 2

    def test_batch_empty_returns_empty(self):
        resp = client.post("/reason/batch", json=[])
        assert resp.status_code == 200
        assert resp.json() == []

    def test_batch_exceeds_limit(self):
        single = {
            "control": {
                "control_id": "B-001",
                "title": "x",
                "article_ref": "NIS2-21a",
                "risk_weight": 1.0,
            },
            "evidence_items": [],
            "confidence_level": 0.95,
            "expected_deviation_rate": 0.05,
        }
        resp = client.post("/reason/batch", json=[single] * 51)
        assert resp.status_code == 422
