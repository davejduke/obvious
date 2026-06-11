"""Engine Orchestrator — Pipeline coordinator and conclusion assembly.

Coordinates the 5 deterministic reasoning components:
  Scope Engine → Economy Engine → Risk Engine → Quality Engine → Documentation Engine

All audit conclusions are computed deterministically. No LLM is invoked.
LLM narrative generation is a placeholder stub here; populated in production
by Documentation Engine via AWS Bedrock.

Conclusion Assembly
-------------------
Final conclusion = Risk Engine conclusion, with Quality Engine sufficiency
as a confidence modifier:
  - If sufficiency = INSUFFICIENT → downgrade conclusion by one tier
    (Effective → Partially Effective, Partially Effective → Not Effective)
  - If sufficiency = MARGINAL → confidence reduced by 20%
  - If sufficiency = SUFFICIENT → no adjustment
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional
from uuid import UUID, uuid4

from engine.documentation.chain import EvidenceHashChain, WorkingPaper, compute_content_hash
from engine.quality.sampler import (
    EvidenceQualityInput,
    SamplingParameters,
    SufficiencyVerdict,
    assess_sufficiency,
    cochran_sample_size,
)
from engine.risk.scorer import AuditConclusion, EvidenceItem, RiskScoreResult, score_control
from engine.scope.dag import AuditScopeDAG, BoundaryStatus, ScopeNode


# ---------------------------------------------------------------------------
# Request / response schemas
# ---------------------------------------------------------------------------

@dataclass
class EvidenceInput:
    """Evidence item submitted to the orchestrator."""
    evidence_id: UUID
    content: str            # Raw text / serialised content for hashing
    tier: int               # 1–6 evidence tier
    quality_score: float    # 0.0–1.0
    passes: bool = True     # Whether evidence supports control effectiveness
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class ControlInput:
    """A NIS 2 control to be audited."""
    control_id: str
    title: str
    article_ref: str         # e.g. "NIS2-21a"
    risk_weight: float       # 0.0–5.0
    prior: float = 0.5       # Prior probability of effectiveness (default: neutral)
    description: str = ""
    population_size: Optional[int] = None  # For Cochran finite correction


@dataclass
class ReasoningRequest:
    """Full input to the reasoning pipeline."""
    request_id: UUID = field(default_factory=uuid4)
    engagement_id: Optional[UUID] = None
    control: ControlInput = field(default_factory=lambda: ControlInput("", "", "", 1.0))
    evidence_items: list[EvidenceInput] = field(default_factory=list)
    confidence_level: float = 0.95    # For Cochran sampling
    expected_deviation_rate: float = 0.05
    budget_hours: float = 40.0


@dataclass
class ReasoningResult:
    """Full output from the reasoning pipeline — deterministic."""
    request_id: UUID
    engagement_id: Optional[UUID]
    control_id: str
    article_ref: str

    # Scope
    in_scope: bool

    # Risk scoring
    risk_score: RiskScoreResult

    # Quality / sampling
    required_sample_size: int
    evidence_count: int
    sufficiency_verdict: str
    sufficiency_ratio: float

    # Final conclusion (after sufficiency adjustment)
    conclusion: AuditConclusion
    confidence_pct: float          # 0–100, rounded to 1 decimal

    # Documentation
    working_paper: dict[str, Any]  # Serialised WorkingPaper.finalize()

    # Traceability
    pipeline_trace: dict[str, Any] = field(default_factory=dict)


# ---------------------------------------------------------------------------
# Conclusion downgrade logic
# ---------------------------------------------------------------------------

_CONCLUSION_ORDER: list[AuditConclusion] = [
    AuditConclusion.EFFECTIVE,
    AuditConclusion.PARTIALLY_EFFECTIVE,
    AuditConclusion.NOT_EFFECTIVE,
]


def _downgrade_conclusion(conclusion: AuditConclusion, steps: int = 1) -> AuditConclusion:
    """Downgrade conclusion by `steps` tiers (maximum = NOT_EFFECTIVE)."""
    idx = _CONCLUSION_ORDER.index(conclusion)
    return _CONCLUSION_ORDER[min(idx + steps, len(_CONCLUSION_ORDER) - 1)]


def _adjust_for_sufficiency(
    conclusion: AuditConclusion,
    confidence: float,
    verdict: SufficiencyVerdict,
) -> tuple[AuditConclusion, float]:
    """Apply sufficiency modifier to the conclusion and confidence."""
    if verdict == SufficiencyVerdict.INSUFFICIENT:
        conclusion = _downgrade_conclusion(conclusion, steps=1)
        confidence = confidence * 0.60
    elif verdict == SufficiencyVerdict.MARGINAL:
        confidence = confidence * 0.80
    return conclusion, confidence


# ---------------------------------------------------------------------------
# Pipeline
# ---------------------------------------------------------------------------

class ReasoningPipeline:
    """Coordinates all 5 deterministic reasoning components.

    Usage
    -----
    pipeline = ReasoningPipeline()
    result = pipeline.run(request)
    """

    def __init__(self) -> None:
        self._dag = AuditScopeDAG()

    def run(self, request: ReasoningRequest) -> ReasoningResult:
        """Execute the full reasoning pipeline for one control + evidence set."""
        ctrl = request.control

        # ------------------------------------------------------------------
        # 1. Scope Engine — register control in DAG, mark in-scope
        # ------------------------------------------------------------------
        scope_node = ScopeNode(
            node_id=uuid4(),
            control_id=ctrl.control_id,
            title=ctrl.title,
            article_ref=ctrl.article_ref,
            risk_weight=ctrl.risk_weight,
            boundary_status=BoundaryStatus.IN_SCOPE,
        )
        # Use a fresh DAG per request (pipeline is stateless per call)
        dag = AuditScopeDAG()
        dag.add_node(scope_node)
        dag.set_boundary(scope_node.node_id, BoundaryStatus.IN_SCOPE)
        in_scope = scope_node.is_in_scope()

        # ------------------------------------------------------------------
        # 2. Quality Engine — Cochran sample size + sufficiency
        # ------------------------------------------------------------------
        sampling_params = SamplingParameters(
            confidence_level=request.confidence_level,
            expected_deviation_rate=request.expected_deviation_rate,
            tolerable_deviation_rate=min(
                0.10 if ctrl.risk_weight < 3.5 else 0.05,
                0.99,
            ),
            population_size=ctrl.population_size,
        )
        sample_result = cochran_sample_size(sampling_params)
        required_n = sample_result.n_final

        quality_inputs = [
            EvidenceQualityInput(
                evidence_id=e.evidence_id,
                tier=e.tier,
                quality_score=e.quality_score,
                passes=e.passes,
            )
            for e in request.evidence_items
        ]
        sufficiency = assess_sufficiency(ctrl.control_id, quality_inputs, required_n)

        # ------------------------------------------------------------------
        # 3. Risk Engine — Bayesian scoring with 5x asymmetric penalty
        # ------------------------------------------------------------------
        risk_inputs = [
            EvidenceItem(
                evidence_id=e.evidence_id,
                passes=e.passes,
                quality_score=e.quality_score,
                tier=e.tier,
            )
            for e in request.evidence_items
        ]
        risk_score = score_control(
            control_id=ctrl.control_id,
            risk_weight=ctrl.risk_weight,
            prior=ctrl.prior,
            evidence_items=risk_inputs,
        )

        # ------------------------------------------------------------------
        # 4. Conclusion assembly — adjust for sufficiency
        # ------------------------------------------------------------------
        raw_conclusion = risk_score.conclusion
        raw_confidence = risk_score.confidence
        final_conclusion, adjusted_confidence = _adjust_for_sufficiency(
            raw_conclusion, raw_confidence, sufficiency.verdict
        )
        confidence_pct = round(adjusted_confidence * 100, 1)

        # ------------------------------------------------------------------
        # 5. Documentation Engine — SHA-256 hash chain + working paper
        # ------------------------------------------------------------------
        chain = EvidenceHashChain()
        for e in request.evidence_items:
            chain.append(
                evidence_id=e.evidence_id,
                content=e.content,
                metadata=e.metadata,
            )

        paper = WorkingPaper(
            paper_id=uuid4(),
            control_id=ctrl.control_id,
            engagement_id=request.engagement_id,
            chain=chain,
            conclusion=final_conclusion.value,
            confidence_pct=confidence_pct,
            posterior=risk_score.posterior,
            sufficiency_verdict=sufficiency.verdict.value,
            sufficiency_ratio=sufficiency.sufficiency_ratio,
            narrative_text=(
                f"[LLM narrative placeholder] Control {ctrl.control_id} "
                f"({ctrl.article_ref}): {final_conclusion.value} "
                f"({confidence_pct}% confidence). "
                f"Evidence chain contains {chain.length} item(s)."
            ),
            narrative_generated_by="placeholder",
        )

        return ReasoningResult(
            request_id=request.request_id,
            engagement_id=request.engagement_id,
            control_id=ctrl.control_id,
            article_ref=ctrl.article_ref,
            in_scope=in_scope,
            risk_score=risk_score,
            required_sample_size=required_n,
            evidence_count=len(request.evidence_items),
            sufficiency_verdict=sufficiency.verdict.value,
            sufficiency_ratio=sufficiency.sufficiency_ratio,
            conclusion=final_conclusion,
            confidence_pct=confidence_pct,
            working_paper=paper.finalize(),
            pipeline_trace={
                "scope": {
                    "dag_node_id": str(scope_node.node_id),
                    "in_scope": in_scope,
                },
                "quality": {
                    "required_n": required_n,
                    "n_infinite": sample_result.n_infinite,
                    "finite_correction": sample_result.finite_correction_applied,
                    "sufficiency_ratio": sufficiency.sufficiency_ratio,
                    "tier_distribution": sufficiency.tier_distribution,
                },
                "risk": {
                    "prior": ctrl.prior,
                    "posterior": risk_score.posterior,
                    "raw_conclusion": raw_conclusion.value,
                    "tdr_exceeded": risk_score.tdr_exceeded,
                    "penalty_ratio": 5.0,
                },
                "conclusion_adjustment": {
                    "before": raw_conclusion.value,
                    "after": final_conclusion.value,
                    "sufficiency_verdict": sufficiency.verdict.value,
                    "confidence_before": round(raw_confidence * 100, 1),
                    "confidence_after": confidence_pct,
                },
                "chain": {
                    "length": chain.length,
                    "tail_hash": chain.tail_hash,
                    "valid": chain.verify(),
                },
            },
        )
