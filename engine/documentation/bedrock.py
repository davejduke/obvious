"""AWS Bedrock Claude Sonnet 3.7 STUB — mock client for narrative text generation.

IMPORTANT: This is a stub with NO real API credentials. All responses are
generated locally using deterministic templates. When real AWS credentials
are provisioned, replace MockBedrockClient with RealBedrockClient using
the same interface — the rest of the documentation engine is unaffected.

Constraint: LLM is text-generation ONLY. It NEVER produces audit conclusions,
risk scores, quality assessments, or evidence evaluations. Those always
come from the deterministic engines (Risk, Quality, Documentation).
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Optional, Protocol, runtime_checkable
from uuid import UUID, uuid4
from datetime import datetime, timezone


# ---------------------------------------------------------------------------
# Enumerations
# ---------------------------------------------------------------------------

class NarrativeTone(str, Enum):
    """Tone configuration for LLM-generated narrative text."""
    FORMAL = "formal"          # Board/committee language, third-person
    TECHNICAL = "technical"    # Detailed, precise, references specific controls
    EXECUTIVE = "executive"    # Concise, strategic impact focus, action-oriented


class NarrativeSection(str, Enum):
    """Sections of a working paper that receive LLM-generated narrative."""
    EXECUTIVE_SUMMARY = "executive_summary"
    METHODOLOGY = "methodology"
    FINDINGS = "findings"
    RECOMMENDATIONS = "recommendations"


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

@dataclass
class BedrockConfig:
    """Configuration for the AWS Bedrock Claude integration."""
    model_id: str = "anthropic.claude-sonnet-3-7-20250219-v1:0"  # Claude Sonnet 3.7
    region: str = "us-east-1"
    max_tokens: int = 4096
    temperature: float = 0.3     # Low temperature for consistent audit narrative
    top_p: float = 0.95
    # No API key fields — credentials injected via AWS role/env at deploy time

    def to_invoke_body(self, prompt: str) -> dict[str, Any]:
        """Build the request body for bedrock:invoke_model (Anthropic messages API)."""
        return {
            "anthropic_version": "bedrock-2023-05-31",
            "max_tokens": self.max_tokens,
            "temperature": self.temperature,
            "top_p": self.top_p,
            "messages": [{"role": "user", "content": prompt}],
        }


# ---------------------------------------------------------------------------
# Client protocol (interface)
# ---------------------------------------------------------------------------

@runtime_checkable
class BedrockClientProtocol(Protocol):
    """Interface that real and mock Bedrock clients must satisfy."""

    def generate_text(self, prompt: str, section: NarrativeSection) -> "BedrockResponse":
        """Invoke the model and return the narrative text response."""
        ...


# ---------------------------------------------------------------------------
# Response model
# ---------------------------------------------------------------------------

@dataclass
class BedrockResponse:
    """Parsed response from a Bedrock model invocation."""
    response_id: UUID = field(default_factory=uuid4)
    text: str = ""
    model_id: str = ""
    input_tokens: int = 0
    output_tokens: int = 0
    is_mock: bool = False
    generated_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )

    def to_dict(self) -> dict[str, Any]:
        return {
            "response_id": str(self.response_id),
            "model_id": self.model_id,
            "text": self.text,
            "input_tokens": self.input_tokens,
            "output_tokens": self.output_tokens,
            "is_mock": self.is_mock,
            "generated_at": self.generated_at,
        }


# ---------------------------------------------------------------------------
# Mock narratives keyed by (section, tone)
# ---------------------------------------------------------------------------

_MOCK_NARRATIVES: dict[tuple[NarrativeSection, NarrativeTone], str] = {
    (NarrativeSection.EXECUTIVE_SUMMARY, NarrativeTone.FORMAL): (
        "This report presents the findings of an independent audit conducted in accordance "
        "with the Institute of Internal Auditors (IIA) International Standards for the "
        "Professional Practice of Internal Auditing. The engagement assessed {control_count} "
        "controls across the {framework} framework during the period {period_start} to "
        "{period_end}. Based on the evidence examined — comprising {evidence_count} items — "
        "the audit team concludes that {conclusion}. Confidence in this determination stands "
        "at {confidence_pct}%. Full findings are detailed in subsequent sections."
    ),
    (NarrativeSection.EXECUTIVE_SUMMARY, NarrativeTone.TECHNICAL): (
        "Audit scope: {framework} controls ({control_count} controls under test). Evidence "
        "corpus: {evidence_count} items, hash-chain verified (SHA-256, chain length "
        "{chain_length}). Bayesian posterior probability: {posterior:.4f} (asymmetric "
        "5x false-negative penalty applied). Sufficiency verdict: {sufficiency_verdict} "
        "({sufficiency_ratio:.1%} of required evidence collected). "
        "Conclusion: {conclusion} at {confidence_pct}% confidence."
    ),
    (NarrativeSection.EXECUTIVE_SUMMARY, NarrativeTone.EXECUTIVE): (
        "We audited {framework} compliance for the period {period_start}–{period_end}. "
        "After reviewing {evidence_count} evidence items across {control_count} controls, "
        "our determination is: {conclusion} ({confidence_pct}% confidence). "
        "{finding_count} findings were identified — please see the recommendations section "
        "for prioritised remediation actions."
    ),
    (NarrativeSection.METHODOLOGY, NarrativeTone.FORMAL): (
        "The audit was conducted in accordance with IIA Standard 4.1 and applicable "
        "professional standards. Evidence was collected through {evidence_types} and "
        "subject to quality assessment using the Cochran 1977 statistical sampling formula. "
        "Risk scoring employed a Bayesian inference model with a 5x asymmetric "
        "false-negative penalty to account for the elevated materiality risk of cybersecurity "
        "control failures. All conclusions are the product of deterministic computation; "
        "this narrative provides human-readable context only."
    ),
    (NarrativeSection.METHODOLOGY, NarrativeTone.TECHNICAL): (
        "Evidence corpus assembled via {evidence_types}. Quality scoring: Cochran 1977 "
        "formula, sufficiency threshold {sufficiency_ratio:.2f}. Risk scoring: Bayesian "
        "update over {control_count} controls, likelihood_pass=0.92, likelihood_fail=0.15, "
        "PENALTY_RATIO=5.0. Hash chain: SHA-256, chain length {chain_length}. "
        "TDR: material controls <=5%, non-material <=10%. All conclusions deterministic."
    ),
    (NarrativeSection.METHODOLOGY, NarrativeTone.EXECUTIVE): (
        "We used a rigorous, standards-based methodology. Evidence from {evidence_types} "
        "was statistically sampled and independently verified. Risk was scored using "
        "industry-standard Bayesian inference with additional weighting for high-impact "
        "failures. All findings are fact-based and reproducible."
    ),
    (NarrativeSection.FINDINGS, NarrativeTone.FORMAL): (
        "The audit identified {finding_count} findings across the {framework} control "
        "framework. Findings are presented in descending order of severity. Each finding "
        "is supported by documented evidence forming an unbroken SHA-256 hash chain, "
        "ensuring tamper-evident integrity. Control conclusions — Effective, Partially "
        "Effective, or Not Effective — are derived exclusively from deterministic risk "
        "scoring and are not subject to LLM inference."
    ),
    (NarrativeSection.FINDINGS, NarrativeTone.TECHNICAL): (
        "{finding_count} findings detected. Critical: {critical_count}, High: {high_count}, "
        "Medium: {medium_count}, Low: {low_count}. Evidence hash-chain length: {chain_length}. "
        "Posterior probability confirms deterministic conclusion: {conclusion}. "
        "No LLM inference applied to findings or evidence evaluation."
    ),
    (NarrativeSection.FINDINGS, NarrativeTone.EXECUTIVE): (
        "{finding_count} issues found. The most significant require immediate attention: "
        "see critical and high-severity items below. Management should prioritise addressing "
        "the {critical_count} critical finding(s) within 30 days."
    ),
    (NarrativeSection.RECOMMENDATIONS, NarrativeTone.FORMAL): (
        "The audit team respectfully submits the following recommendations for management's "
        "consideration. Each recommendation addresses a specific control deficiency identified "
        "during fieldwork. Management is requested to provide a formal response confirming "
        "acceptance, partial acceptance, or rejection of each recommendation, together with "
        "a proposed remediation timeline. All remediation actions should be verified in a "
        "subsequent follow-up engagement."
    ),
    (NarrativeSection.RECOMMENDATIONS, NarrativeTone.TECHNICAL): (
        "Remediation actions mapped to {finding_count} findings. Priority order: Critical "
        "({critical_count}) -> High ({high_count}) -> Medium ({medium_count}) -> "
        "Low ({low_count}). Implementation verification should include re-execution of the "
        "evidence hash-chain and Bayesian re-scoring to confirm posterior exceeds the "
        "0.833 effectiveness threshold. TDR re-test required for all material controls."
    ),
    (NarrativeSection.RECOMMENDATIONS, NarrativeTone.EXECUTIVE): (
        "We recommend {finding_count} actions to remediate the identified gaps. "
        "Address the {critical_count} critical item(s) immediately — these represent the "
        "highest risk to {framework} compliance. A follow-up review should be scheduled "
        "within 90 days of remediation completion to confirm effectiveness."
    ),
}


# ---------------------------------------------------------------------------
# Mock client — no network I/O, no credentials required
# ---------------------------------------------------------------------------

class MockBedrockClient:
    """Stub Bedrock client — returns templated mock narratives without network calls.

    This satisfies BedrockClientProtocol and is the default implementation
    until real AWS credentials are provisioned.  The stub is intentionally
    realistic so downstream template and PDF tests exercise the full pipeline.

    Swap for a real client via dependency injection:
        generator = NarrativeGenerator(client=RealBedrockClient(config))
    """

    def __init__(self, config: Optional[BedrockConfig] = None) -> None:
        self._config = config or BedrockConfig()
        self._call_count: int = 0

    def generate_text(self, prompt: str, section: NarrativeSection) -> BedrockResponse:
        """Return a mock narrative for the requested section.

        The prompt is accepted but not forwarded — this is a stub.
        Template placeholders in the mock text are resolved by NarrativeGenerator.
        """
        self._call_count += 1
        tone = self._extract_tone_from_prompt(prompt)
        key = (section, tone)
        text = _MOCK_NARRATIVES.get(
            key,
            _MOCK_NARRATIVES.get((section, NarrativeTone.FORMAL), "[narrative placeholder]"),
        )
        return BedrockResponse(
            text=text,
            model_id=self._config.model_id + "#mock",
            input_tokens=len(prompt.split()),
            output_tokens=len(text.split()),
            is_mock=True,
        )

    @staticmethod
    def _extract_tone_from_prompt(prompt: str) -> NarrativeTone:
        """Detect the requested tone from the prompt string."""
        lower = prompt.lower()
        if "executive" in lower:
            return NarrativeTone.EXECUTIVE
        if "technical" in lower:
            return NarrativeTone.TECHNICAL
        return NarrativeTone.FORMAL

    @property
    def call_count(self) -> int:
        """Number of generate_text calls — useful for test assertions."""
        return self._call_count


# ---------------------------------------------------------------------------
# Factory
# ---------------------------------------------------------------------------

def create_bedrock_client(
    config: Optional[BedrockConfig] = None,
    *,
    use_real: bool = False,
) -> MockBedrockClient:
    """Factory for creating a Bedrock client.

    Parameters
    ----------
    config:
        BedrockConfig instance.  Defaults to BedrockConfig().
    use_real:
        When False (default), returns MockBedrockClient — safe for
        development and testing.  When True, would return a RealBedrockClient
        (not yet implemented — requires AWS SDK and credentials).
    """
    if use_real:
        raise NotImplementedError(
            "RealBedrockClient not yet implemented. "
            "Configure AWS credentials and implement when ready to connect."
        )
    return MockBedrockClient(config)

