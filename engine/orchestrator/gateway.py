"""FastAPI Gateway — HTTP interface for the reasoning engine.

Exposes:
  POST /reason          — Run the full 5-component pipeline
  POST /reason/batch    — Run multiple controls in sequence
  GET  /health          — Liveness probe
  GET  /info            — Engine metadata

ALL conclusions are deterministic. LLM is not invoked by this gateway.
"""
from __future__ import annotations

import uuid
from typing import Any, Optional

from fastapi import FastAPI, HTTPException, status
from pydantic import BaseModel, Field, field_validator

from engine.orchestrator.pipeline import (
    ControlInput,
    EvidenceInput,
    ReasoningPipeline,
    ReasoningRequest,
    ReasoningResult,
)

app = FastAPI(
    title="AIAUDITOR Reasoning Engine",
    description=(
        "Deterministic NIS 2 audit reasoning: Bayesian scoring, "
        "Cochran sampling, DAG scope resolution, 5x asymmetric penalty."
    ),
    version="0.1.0",
)

_pipeline = ReasoningPipeline()


# ---------------------------------------------------------------------------
# Request / response Pydantic models (HTTP layer)
# ---------------------------------------------------------------------------

class EvidenceInputDTO(BaseModel):
    evidence_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    content: str
    tier: int = Field(ge=1, le=6)
    quality_score: float = Field(ge=0.0, le=1.0)
    passes: bool = True
    metadata: dict[str, Any] = Field(default_factory=dict)

    @field_validator("evidence_id")
    @classmethod
    def parse_uuid(cls, v: str) -> str:
        uuid.UUID(v)  # validate it is a valid UUID
        return v


class ControlInputDTO(BaseModel):
    control_id: str
    title: str
    article_ref: str
    risk_weight: float = Field(ge=0.0, le=5.0)
    prior: float = Field(default=0.5, ge=0.0, le=1.0)
    description: str = ""
    population_size: Optional[int] = Field(default=None, ge=1)


class ReasoningRequestDTO(BaseModel):
    engagement_id: Optional[str] = None
    control: ControlInputDTO
    evidence_items: list[EvidenceInputDTO] = Field(default_factory=list)
    confidence_level: float = Field(default=0.95)
    expected_deviation_rate: float = Field(default=0.05, gt=0.0, lt=1.0)
    budget_hours: float = Field(default=40.0, gt=0.0)

    @field_validator("confidence_level")
    @classmethod
    def validate_confidence(cls, v: float) -> float:
        if v not in (0.90, 0.95, 0.99):
            raise ValueError("confidence_level must be 0.90, 0.95, or 0.99")
        return v


class ReasoningResponseDTO(BaseModel):
    request_id: str
    engagement_id: Optional[str]
    control_id: str
    article_ref: str
    in_scope: bool
    conclusion: str
    confidence_pct: float
    sufficiency_verdict: str
    sufficiency_ratio: float
    required_sample_size: int
    evidence_count: int
    posterior: float
    tdr_exceeded: bool
    working_paper_id: str
    chain_length: int
    chain_valid: bool
    chain_tail_hash: str
    pipeline_trace: dict[str, Any]


def _to_domain_request(dto: ReasoningRequestDTO) -> ReasoningRequest:
    return ReasoningRequest(
        engagement_id=uuid.UUID(dto.engagement_id) if dto.engagement_id else None,
        control=ControlInput(
            control_id=dto.control.control_id,
            title=dto.control.title,
            article_ref=dto.control.article_ref,
            risk_weight=dto.control.risk_weight,
            prior=dto.control.prior,
            description=dto.control.description,
            population_size=dto.control.population_size,
        ),
        evidence_items=[
            EvidenceInput(
                evidence_id=uuid.UUID(e.evidence_id),
                content=e.content,
                tier=e.tier,
                quality_score=e.quality_score,
                passes=e.passes,
                metadata=e.metadata,
            )
            for e in dto.evidence_items
        ],
        confidence_level=dto.confidence_level,
        expected_deviation_rate=dto.expected_deviation_rate,
        budget_hours=dto.budget_hours,
    )


def _to_response_dto(result: ReasoningResult) -> ReasoningResponseDTO:
    wp = result.working_paper
    chain_info = wp.get("evidence_chain", {})
    return ReasoningResponseDTO(
        request_id=str(result.request_id),
        engagement_id=str(result.engagement_id) if result.engagement_id else None,
        control_id=result.control_id,
        article_ref=result.article_ref,
        in_scope=result.in_scope,
        conclusion=result.conclusion.value,
        confidence_pct=result.confidence_pct,
        sufficiency_verdict=result.sufficiency_verdict,
        sufficiency_ratio=result.sufficiency_ratio,
        required_sample_size=result.required_sample_size,
        evidence_count=result.evidence_count,
        posterior=result.risk_score.posterior,
        tdr_exceeded=result.risk_score.tdr_exceeded,
        working_paper_id=wp.get("paper_id", ""),
        chain_length=chain_info.get("length", 0),
        chain_valid=chain_info.get("chain_valid", False),
        chain_tail_hash=chain_info.get("tail_hash", ""),
        pipeline_trace=result.pipeline_trace,
    )


# ---------------------------------------------------------------------------
# Routes
# ---------------------------------------------------------------------------

@app.get("/health", status_code=status.HTTP_200_OK)
def health() -> dict[str, str]:
    """Liveness probe."""
    return {"status": "ok", "engine": "aiauditor-reasoning"}


@app.get("/info")
def info() -> dict[str, Any]:
    """Engine metadata."""
    return {
        "name": "AIAUDITOR Reasoning Engine",
        "version": "0.1.0",
        "components": [
            "scope",
            "economy",
            "risk",
            "quality",
            "documentation",
            "orchestrator",
        ],
        "deterministic": True,
        "llm_role": "narrative_text_generation_only",
        "scoring": {
            "method": "bayesian",
            "sampling": "cochran",
            "penalty": "5x_asymmetric_false_negative",
            "scope_resolution": "dag_topological",
        },
    }


@app.post("/reason", response_model=ReasoningResponseDTO, status_code=status.HTTP_200_OK)
def reason(request_dto: ReasoningRequestDTO) -> ReasoningResponseDTO:
    """Run the full deterministic reasoning pipeline for one control."""
    try:
        domain_request = _to_domain_request(request_dto)
        result = _pipeline.run(domain_request)
        return _to_response_dto(result)
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_422_UNPROCESSABLE_ENTITY, detail=str(exc))
    except Exception as exc:
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(exc))


@app.post("/reason/batch", status_code=status.HTTP_200_OK)
def reason_batch(requests: list[ReasoningRequestDTO]) -> list[ReasoningResponseDTO]:
    """Run reasoning pipeline for multiple controls. Each is independent."""
    if not requests:
        return []
    if len(requests) > 50:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="Batch size must not exceed 50 items",
        )
    results = []
    for req_dto in requests:
        try:
            domain_request = _to_domain_request(req_dto)
            result = _pipeline.run(domain_request)
            results.append(_to_response_dto(result))
        except ValueError as exc:
            raise HTTPException(
                status_code=status.HTTP_422_UNPROCESSABLE_ENTITY, detail=str(exc)
            )
    return results
