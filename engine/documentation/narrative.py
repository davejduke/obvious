"""Narrative Generator — LLM-powered human-readable text for working paper sections.

This module orchestrates calls to the Bedrock stub and resolves template
variables from deterministic engine outputs before storing the final text.

Critical constraint
-------------------
The LLM generates text ONLY.  All values passed to it (conclusion, scores,
counts) come exclusively from the deterministic Risk/Quality/Documentation
engines.  The LLM never computes, infers, or modifies those values.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional
from uuid import UUID, uuid4
from datetime import datetime, timezone

from engine.documentation.bedrock import (
    BedrockConfig,
    BedrockResponse,
    MockBedrockClient,
    NarrativeSection,
    NarrativeTone,
    create_bedrock_client,
)


# ---------------------------------------------------------------------------
# Request / Response models
# ---------------------------------------------------------------------------

@dataclass
class NarrativeContext:
    """All deterministic inputs used to fill narrative template variables.

    These values MUST come from deterministic engines. LLM never derives them.
    """
    # From Risk Engine output
    conclusion: str = ""              # e.g. "Effective"
    confidence_pct: float = 0.0       # 0–100
    posterior: float = 0.0            # Bayesian posterior 0.0–1.0
    control_count: int = 1

    # From Quality Engine output
    sufficiency_verdict: str = ""     # "sufficient"|"marginal"|"insufficient"
    sufficiency_ratio: float = 0.0    # 0.0–1.0

    # From Documentation Engine
    evidence_count: int = 0
    chain_length: int = 0
    evidence_types: str = "manual upload and API integration"  # human-readable

    # Engagement metadata
    framework: str = "NIS 2 Article 21"
    period_start: str = ""
    period_end: str = ""

    # Finding counts (from deterministic finding list)
    finding_count: int = 0
    critical_count: int = 0
    high_count: int = 0
    medium_count: int = 0
    low_count: int = 0

    def as_format_kwargs(self) -> dict[str, Any]:
        """Return a dict suitable for str.format_map() on narrative templates."""
        return {
            "conclusion": self.conclusion,
            "confidence_pct": round(self.confidence_pct, 1),
            "posterior": self.posterior,
            "control_count": self.control_count,
            "sufficiency_verdict": self.sufficiency_verdict,
            "sufficiency_ratio": self.sufficiency_ratio,
            "evidence_count": self.evidence_count,
            "chain_length": self.chain_length,
            "evidence_types": self.evidence_types,
            "framework": self.framework,
            "period_start": self.period_start,
            "period_end": self.period_end,
            "finding_count": self.finding_count,
            "critical_count": self.critical_count,
            "high_count": self.high_count,
            "medium_count": self.medium_count,
            "low_count": self.low_count,
        }


@dataclass
class NarrativeResult:
    """Resolved narrative text for one working paper section."""
    request_id: UUID = field(default_factory=uuid4)
    section: NarrativeSection = NarrativeSection.EXECUTIVE_SUMMARY
    tone: NarrativeTone = NarrativeTone.FORMAL
    text: str = ""                  # Final resolved text (placeholders filled)
    raw_template: str = ""          # Raw template from LLM before variable substitution
    model_id: str = ""
    is_mock: bool = True
    generated_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )

    def to_dict(self) -> dict[str, Any]:
        return {
            "request_id": str(self.request_id),
            "section": self.section.value,
            "tone": self.tone.value,
            "text": self.text,
            "model_id": self.model_id,
            "is_mock": self.is_mock,
            "generated_at": self.generated_at,
        }


@dataclass
class WorkingPaperNarratives:
    """Complete set of LLM narratives for all four working paper sections."""
    executive_summary: NarrativeResult = field(
        default_factory=lambda: NarrativeResult(section=NarrativeSection.EXECUTIVE_SUMMARY)
    )
    methodology: NarrativeResult = field(
        default_factory=lambda: NarrativeResult(section=NarrativeSection.METHODOLOGY)
    )
    findings: NarrativeResult = field(
        default_factory=lambda: NarrativeResult(section=NarrativeSection.FINDINGS)
    )
    recommendations: NarrativeResult = field(
        default_factory=lambda: NarrativeResult(section=NarrativeSection.RECOMMENDATIONS)
    )
    tone: NarrativeTone = NarrativeTone.FORMAL
    generated_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )

    def to_dict(self) -> dict[str, Any]:
        return {
            "tone": self.tone.value,
            "generated_at": self.generated_at,
            "executive_summary": self.executive_summary.to_dict(),
            "methodology": self.methodology.to_dict(),
            "findings": self.findings.to_dict(),
            "recommendations": self.recommendations.to_dict(),
        }


# ---------------------------------------------------------------------------
# Narrative Generator
# ---------------------------------------------------------------------------

class NarrativeGenerator:
    """Generates human-readable narrative text for working paper sections.

    Uses a BedrockClient (mock by default) to obtain narrative templates and
    resolves deterministic values from NarrativeContext into the final text.

    All conclusions, scores, and counts MUST be pre-computed by deterministic
    engines and passed in via NarrativeContext — this class never derives them.
    """

    def __init__(
        self,
        client: Optional[MockBedrockClient] = None,
        config: Optional[BedrockConfig] = None,
    ) -> None:
        self._client = client or create_bedrock_client(config)

    # ------------------------------------------------------------------
    # Individual section generators
    # ------------------------------------------------------------------

    def generate_executive_summary(
        self, context: NarrativeContext, tone: NarrativeTone = NarrativeTone.FORMAL
    ) -> NarrativeResult:
        """Generate the executive summary narrative."""
        return self._generate(NarrativeSection.EXECUTIVE_SUMMARY, context, tone)

    def generate_methodology(
        self, context: NarrativeContext, tone: NarrativeTone = NarrativeTone.FORMAL
    ) -> NarrativeResult:
        """Generate the methodology narrative."""
        return self._generate(NarrativeSection.METHODOLOGY, context, tone)

    def generate_findings(
        self, context: NarrativeContext, tone: NarrativeTone = NarrativeTone.FORMAL
    ) -> NarrativeResult:
        """Generate the findings narrative."""
        return self._generate(NarrativeSection.FINDINGS, context, tone)

    def generate_recommendations(
        self, context: NarrativeContext, tone: NarrativeTone = NarrativeTone.FORMAL
    ) -> NarrativeResult:
        """Generate the recommendations narrative."""
        return self._generate(NarrativeSection.RECOMMENDATIONS, context, tone)

    # ------------------------------------------------------------------
    # Convenience: generate all four sections at once
    # ------------------------------------------------------------------

    def generate_all(
        self, context: NarrativeContext, tone: NarrativeTone = NarrativeTone.FORMAL
    ) -> WorkingPaperNarratives:
        """Generate all four working paper narratives in a single call."""
        return WorkingPaperNarratives(
            executive_summary=self.generate_executive_summary(context, tone),
            methodology=self.generate_methodology(context, tone),
            findings=self.generate_findings(context, tone),
            recommendations=self.generate_recommendations(context, tone),
            tone=tone,
        )

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------

    def _build_prompt(self, section: NarrativeSection, tone: NarrativeTone) -> str:
        """Build the prompt sent to the Bedrock model."""
        return (
            f"Generate a {tone.value} tone {section.value.replace('_', ' ')} section "
            f"for an IIA Standard 4.1 compliant audit working paper. "
            f"Write in {tone.value} style. Use placeholders for variable data. "
            f"Output narrative text only, no headings."
        )

    def _generate(
        self,
        section: NarrativeSection,
        context: NarrativeContext,
        tone: NarrativeTone,
    ) -> NarrativeResult:
        """Core generation: call Bedrock, resolve template variables, return result."""
        prompt = self._build_prompt(section, tone)
        response: BedrockResponse = self._client.generate_text(prompt, section)

        # Resolve placeholder variables with deterministic values from context
        try:
            resolved = response.text.format_map(context.as_format_kwargs())
        except (KeyError, ValueError):
            # Safety net: if template resolution fails, keep raw text
            resolved = response.text

        return NarrativeResult(
            section=section,
            tone=tone,
            text=resolved,
            raw_template=response.text,
            model_id=response.model_id,
            is_mock=response.is_mock,
        )

