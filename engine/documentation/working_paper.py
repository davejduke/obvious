"""Enhanced Working Paper — version history, LLM narratives, template integration.

Builds on engine.documentation.chain.WorkingPaper to add:
- Version history (each save creates an immutable snapshot)
- LLM-generated narrative sections via NarrativeGenerator
- Template-aware rendering via TemplateEngine
- IIA Standard 4.1 compliance metadata

Constraint: All audit conclusions and scores are deterministic — the version
history records them alongside the LLM narrative text, but never replaces them.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional
from uuid import UUID, uuid4
from datetime import datetime, timezone

from engine.documentation.chain import (
    EvidenceHashChain,
    WorkingPaper,
)
from engine.documentation.bedrock import NarrativeTone
from engine.documentation.narrative import (
    NarrativeContext,
    NarrativeGenerator,
    WorkingPaperNarratives,
)
from engine.documentation.templates import (
    EngagementType,
    RenderedWorkingPaper,
    TemplateEngine,
    WorkingPaperTemplate,
)


# ---------------------------------------------------------------------------
# Version snapshot
# ---------------------------------------------------------------------------

@dataclass
class PaperVersion:
    """Immutable snapshot of a working paper at a point in time.

    Once created a PaperVersion is never modified — it represents the full
    state of the paper including deterministic outputs and any LLM narrative.
    """
    version_id: UUID = field(default_factory=uuid4)
    version_number: int = 1
    created_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )
    created_by: str = "system"
    change_note: str = ""

    # Deterministic outputs (from Risk/Quality/Documentation engines)
    conclusion: str = ""
    confidence_pct: float = 0.0
    posterior: float = 0.0
    sufficiency_verdict: str = ""
    sufficiency_ratio: float = 0.0
    chain_length: int = 0
    chain_tail_hash: str = ""
    chain_valid: bool = True

    # LLM narrative (text-generation only — never used for conclusions)
    narrative_tone: str = NarrativeTone.FORMAL.value
    narrative_executive_summary: str = ""
    narrative_methodology: str = ""
    narrative_findings: str = ""
    narrative_recommendations: str = ""
    narrative_is_mock: bool = True

    def to_dict(self) -> dict[str, Any]:
        return {
            "version_id": str(self.version_id),
            "version_number": self.version_number,
            "created_at": self.created_at,
            "created_by": self.created_by,
            "change_note": self.change_note,
            # Deterministic
            "conclusion": self.conclusion,
            "confidence_pct": round(self.confidence_pct, 2),
            "posterior": round(self.posterior, 4),
            "sufficiency_verdict": self.sufficiency_verdict,
            "sufficiency_ratio": round(self.sufficiency_ratio, 4),
            "chain_length": self.chain_length,
            "chain_tail_hash": self.chain_tail_hash,
            "chain_valid": self.chain_valid,
            # LLM narrative
            "narrative_tone": self.narrative_tone,
            "narrative_executive_summary": self.narrative_executive_summary,
            "narrative_methodology": self.narrative_methodology,
            "narrative_findings": self.narrative_findings,
            "narrative_recommendations": self.narrative_recommendations,
            "narrative_is_mock": self.narrative_is_mock,
        }


@dataclass
class VersionHistory:
    """Ordered, append-only log of working paper versions."""
    versions: list[PaperVersion] = field(default_factory=list)

    def append(self, version: PaperVersion) -> None:
        """Add a new version.  Raises if version_number is not sequential."""
        expected = len(self.versions) + 1
        if version.version_number != expected:
            raise ValueError(
                f"Expected version {expected}, got {version.version_number}"
            )
        self.versions.append(version)

    @property
    def latest(self) -> Optional[PaperVersion]:
        """Return the most recent version, or None if empty."""
        return self.versions[-1] if self.versions else None

    @property
    def count(self) -> int:
        return len(self.versions)

    def to_dict(self) -> dict[str, Any]:
        return {
            "count": self.count,
            "versions": [v.to_dict() for v in self.versions],
        }


# ---------------------------------------------------------------------------
# Enhanced working paper
# ---------------------------------------------------------------------------

class EnhancedWorkingPaper:
    """Working paper with version history, LLM narratives, and template rendering.

    This is the primary entry point for the Documentation Engine in Phase 3.
    It composes:
    - EvidenceHashChain (tamper-evident evidence)
    - NarrativeGenerator (LLM text — mock stub)
    - TemplateEngine (IIA 4.1 section layout)
    - VersionHistory (immutable paper snapshots)

    Constraint
    ----------
    The LLM generates ONLY narrative text.  Conclusions, scores, and verdicts
    are set via ``set_deterministic_outputs()`` from external engine results.
    This class never derives audit conclusions internally.
    """

    def __init__(
        self,
        control_id: str,
        engagement_id: Optional[UUID] = None,
        engagement_type: EngagementType = EngagementType.COMPLIANCE,
        narrative_generator: Optional[NarrativeGenerator] = None,
        template_engine: Optional[TemplateEngine] = None,
    ) -> None:
        self.paper_id: UUID = uuid4()
        self.control_id = control_id
        self.engagement_id = engagement_id
        self.engagement_type = engagement_type
        self.created_at = datetime.now(timezone.utc).isoformat()

        # Sub-components
        self._chain: EvidenceHashChain = EvidenceHashChain()
        self._narrative_generator = narrative_generator or NarrativeGenerator()
        self._template_engine = template_engine or TemplateEngine()
        self._history: VersionHistory = VersionHistory()

        # Deterministic outputs — set externally from engine results
        self._conclusion: str = ""
        self._confidence_pct: float = 0.0
        self._posterior: float = 0.0
        self._sufficiency_verdict: str = ""
        self._sufficiency_ratio: float = 0.0

        # Narrative cache
        self._last_narratives: Optional[WorkingPaperNarratives] = None

    # ------------------------------------------------------------------
    # Evidence chain delegation
    # ------------------------------------------------------------------

    @property
    def chain(self) -> EvidenceHashChain:
        """Access the underlying evidence hash chain."""
        return self._chain

    # ------------------------------------------------------------------
    # Deterministic outputs (must be set from engine results)
    # ------------------------------------------------------------------

    def set_deterministic_outputs(
        self,
        *,
        conclusion: str,
        confidence_pct: float,
        posterior: float,
        sufficiency_verdict: str,
        sufficiency_ratio: float,
    ) -> None:
        """Record the deterministic engine results for this working paper.

        MUST be called before generate_narratives() or save_version().
        These values come from the Risk Engine and Quality Engine — not LLM.
        """
        self._conclusion = conclusion
        self._confidence_pct = confidence_pct
        self._posterior = posterior
        self._sufficiency_verdict = sufficiency_verdict
        self._sufficiency_ratio = sufficiency_ratio

    # ------------------------------------------------------------------
    # Narrative generation
    # ------------------------------------------------------------------

    def generate_narratives(
        self,
        context: NarrativeContext,
        tone: NarrativeTone = NarrativeTone.FORMAL,
    ) -> WorkingPaperNarratives:
        """Generate LLM narrative text for all four sections.

        The NarrativeContext must be populated with deterministic values from
        the Risk, Quality, and Documentation engines before this call.
        """
        narratives = self._narrative_generator.generate_all(context, tone)
        self._last_narratives = narratives
        return narratives

    # ------------------------------------------------------------------
    # Version history
    # ------------------------------------------------------------------

    def save_version(
        self,
        *,
        created_by: str = "system",
        change_note: str = "",
        narratives: Optional[WorkingPaperNarratives] = None,
    ) -> PaperVersion:
        """Take an immutable snapshot of the current paper state.

        Call after updating deterministic outputs or regenerating narratives.
        Returns the new PaperVersion (also appended to self.history).
        """
        narratives = narratives or self._last_narratives
        version_number = self._history.count + 1

        version = PaperVersion(
            version_number=version_number,
            created_by=created_by,
            change_note=change_note,
            # Deterministic
            conclusion=self._conclusion,
            confidence_pct=self._confidence_pct,
            posterior=self._posterior,
            sufficiency_verdict=self._sufficiency_verdict,
            sufficiency_ratio=self._sufficiency_ratio,
            chain_length=self._chain.length,
            chain_tail_hash=self._chain.tail_hash,
            chain_valid=self._chain.verify(),
            # LLM narrative
            narrative_tone=narratives.tone.value if narratives else NarrativeTone.FORMAL.value,
            narrative_executive_summary=(
                narratives.executive_summary.text if narratives else ""
            ),
            narrative_methodology=narratives.methodology.text if narratives else "",
            narrative_findings=narratives.findings.text if narratives else "",
            narrative_recommendations=narratives.recommendations.text if narratives else "",
            narrative_is_mock=(
                narratives.executive_summary.is_mock if narratives else True
            ),
        )
        self._history.append(version)
        return version

    @property
    def history(self) -> VersionHistory:
        return self._history

    # ------------------------------------------------------------------
    # Template rendering
    # ------------------------------------------------------------------

    def render(
        self,
        deterministic_data: dict[str, Any],
        narratives: Optional[WorkingPaperNarratives] = None,
        template: Optional[WorkingPaperTemplate] = None,
    ) -> RenderedWorkingPaper:
        """Render this working paper using the template engine.

        Parameters
        ----------
        deterministic_data:
            Engagement and finding data from deterministic engines.
        narratives:
            LLM narratives.  Falls back to self._last_narratives if None.
        template:
            Template override.  Defaults to the configured engagement type.
        """
        narratives = narratives or self._last_narratives
        template = template or self._template_engine.get_template(self.engagement_type)

        # Build narratives dict for the template engine
        narrative_texts: dict[str, str] = {}
        if narratives:
            narrative_texts = {
                "executive_summary": narratives.executive_summary.text,
                "methodology": narratives.methodology.text,
                "findings": narratives.findings.text,
                "recommendations": narratives.recommendations.text,
            }

        return self._template_engine.render(
            template=template,
            deterministic_data=deterministic_data,
            narratives=narrative_texts,
            engagement_id=self.engagement_id,
        )

    # ------------------------------------------------------------------
    # Finalise (backwards-compatible with WorkingPaper.finalize())
    # ------------------------------------------------------------------

    def finalize(self, narratives: Optional[WorkingPaperNarratives] = None) -> dict[str, Any]:
        """Produce a serialisable snapshot including all engine outputs and narratives."""
        narratives = narratives or self._last_narratives
        return {
            "paper_id": str(self.paper_id),
            "control_id": self.control_id,
            "engagement_id": str(self.engagement_id) if self.engagement_id else None,
            "engagement_type": self.engagement_type.value,
            # Deterministic
            "conclusion": self._conclusion,
            "confidence_pct": round(self._confidence_pct, 2),
            "posterior": round(self._posterior, 4),
            "sufficiency_verdict": self._sufficiency_verdict,
            "sufficiency_ratio": round(self._sufficiency_ratio, 4),
            # Evidence chain
            "evidence_chain": {
                "length": self._chain.length,
                "tail_hash": self._chain.tail_hash,
                "chain_valid": self._chain.verify(),
            },
            # LLM narrative (text-generation only)
            "narratives": narratives.to_dict() if narratives else None,
            # Version history
            "version_history": self._history.to_dict(),
            "created_at": self.created_at,
        }


# ---------------------------------------------------------------------------
# Convenience factory
# ---------------------------------------------------------------------------

def build_working_paper(
    control_id: str,
    engagement_id: Optional[UUID] = None,
    engagement_type: EngagementType = EngagementType.COMPLIANCE,
) -> EnhancedWorkingPaper:
    """Create a new EnhancedWorkingPaper with default generators."""
    return EnhancedWorkingPaper(
        control_id=control_id,
        engagement_id=engagement_id,
        engagement_type=engagement_type,
    )

