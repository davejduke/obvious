"""Working Paper Template Engine — IIA Standard 4.1 compliant templates.

Provides customisable per-engagement-type templates with the required sections:
  1. Cover Page
  2. Scope
  3. Methodology
  4. Findings Matrix
  5. Evidence References
  6. Conclusions
  7. Recommendations

Each engagement type (compliance, operational, IT, financial) has a default
template configuration that can be further customised at runtime.

IIA Standard 4.1 compliance
----------------------------
All working papers include:
- Auditor identification
- Engagement objective
- Audit period
- Evidence basis for conclusions
- Risk-based conclusion wording
- Management response field
- Cross-reference to supporting work papers
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Optional
from uuid import UUID, uuid4
from datetime import datetime, timezone


# ---------------------------------------------------------------------------
# Enumerations
# ---------------------------------------------------------------------------

class EngagementType(str, Enum):
    """Audit engagement type — determines default template configuration."""
    COMPLIANCE = "compliance"      # Regulatory compliance (NIS 2, ISO 27001)
    OPERATIONAL = "operational"    # Process and control effectiveness
    IT = "it"                      # IT general controls, cyber security
    FINANCIAL = "financial"        # Financial controls and assertions


class SectionType(str, Enum):
    """Standard working paper section identifiers."""
    COVER_PAGE = "cover_page"
    SCOPE = "scope"
    METHODOLOGY = "methodology"
    FINDINGS_MATRIX = "findings_matrix"
    EVIDENCE_REFERENCES = "evidence_references"
    CONCLUSIONS = "conclusions"
    RECOMMENDATIONS = "recommendations"


# ---------------------------------------------------------------------------
# Section configuration
# ---------------------------------------------------------------------------

@dataclass
class SectionConfig:
    """Configuration for a single template section."""
    section_type: SectionType
    title: str
    required: bool = True
    include_iia_reference: bool = False   # Print IIA standard reference in header
    iia_standard_ref: str = ""            # e.g. "IIA Standard 4.1"
    include_management_response: bool = False
    include_narrative: bool = False        # If True, LLM narrative is embedded
    narrative_section_key: str = ""        # Maps to NarrativeSection enum value
    order: int = 0                         # Rendering order (ascending)
    custom_fields: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "section_type": self.section_type.value,
            "title": self.title,
            "required": self.required,
            "iia_standard_ref": self.iia_standard_ref,
            "include_management_response": self.include_management_response,
            "include_narrative": self.include_narrative,
            "narrative_section_key": self.narrative_section_key,
            "order": self.order,
        }


# ---------------------------------------------------------------------------
# Working paper template
# ---------------------------------------------------------------------------

@dataclass
class WorkingPaperTemplate:
    """A complete template definition for one engagement type.

    Contains ordered section configurations and metadata for IIA 4.1 compliance.
    """
    template_id: UUID = field(default_factory=uuid4)
    engagement_type: EngagementType = EngagementType.COMPLIANCE
    name: str = ""
    description: str = ""
    iia_standard: str = "IIA Standard 4.1"
    version: str = "1.0"
    sections: list[SectionConfig] = field(default_factory=list)
    created_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )

    @property
    def ordered_sections(self) -> list[SectionConfig]:
        """Return sections sorted by their rendering order."""
        return sorted(self.sections, key=lambda s: s.order)

    def get_section(self, section_type: SectionType) -> Optional[SectionConfig]:
        """Look up a section by type."""
        for sec in self.sections:
            if sec.section_type == section_type:
                return sec
        return None

    def to_dict(self) -> dict[str, Any]:
        return {
            "template_id": str(self.template_id),
            "engagement_type": self.engagement_type.value,
            "name": self.name,
            "description": self.description,
            "iia_standard": self.iia_standard,
            "version": self.version,
            "sections": [s.to_dict() for s in self.ordered_sections],
            "created_at": self.created_at,
        }


# ---------------------------------------------------------------------------
# Rendered section — populated at working paper generation time
# ---------------------------------------------------------------------------

@dataclass
class RenderedSection:
    """A section fully rendered with deterministic data and LLM narrative."""
    config: SectionConfig
    content: dict[str, Any] = field(default_factory=dict)  # Deterministic data
    narrative_text: str = ""                                # LLM-generated text
    narrative_is_mock: bool = True
    rendered_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )

    def to_dict(self) -> dict[str, Any]:
        return {
            "section_type": self.config.section_type.value,
            "title": self.config.title,
            "iia_standard_ref": self.config.iia_standard_ref,
            "content": self.content,
            "narrative_text": self.narrative_text,
            "narrative_is_mock": self.narrative_is_mock,
            "rendered_at": self.rendered_at,
        }


@dataclass
class RenderedWorkingPaper:
    """The fully rendered working paper — ready for PDF generation."""
    paper_id: UUID = field(default_factory=uuid4)
    engagement_id: Optional[UUID] = None
    template: Optional[WorkingPaperTemplate] = None
    sections: list[RenderedSection] = field(default_factory=list)
    rendered_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )

    def to_dict(self) -> dict[str, Any]:
        return {
            "paper_id": str(self.paper_id),
            "engagement_id": str(self.engagement_id) if self.engagement_id else None,
            "template": self.template.to_dict() if self.template else None,
            "sections": [s.to_dict() for s in self.sections],
            "rendered_at": self.rendered_at,
        }


# ---------------------------------------------------------------------------
# Default templates per engagement type
# ---------------------------------------------------------------------------

def _iia_cover_page(order: int = 0) -> SectionConfig:
    return SectionConfig(
        section_type=SectionType.COVER_PAGE,
        title="Cover Page",
        required=True,
        include_iia_reference=True,
        iia_standard_ref="IIA Standard 4.1",
        order=order,
        custom_fields={
            "fields": [
                "engagement_title", "org_name", "auditor_name", "auditor_email",
                "period_start", "period_end", "classification", "version",
                "report_date",
            ]
        },
    )


def _iia_scope(order: int = 1) -> SectionConfig:
    return SectionConfig(
        section_type=SectionType.SCOPE,
        title="Scope & Objectives",
        required=True,
        include_iia_reference=True,
        iia_standard_ref="IIA Standard 4.1 §6",
        order=order,
        custom_fields={"fields": ["objective", "in_scope_controls", "out_of_scope", "period"]},
    )


def _iia_methodology(order: int = 2, include_narrative: bool = True) -> SectionConfig:
    return SectionConfig(
        section_type=SectionType.METHODOLOGY,
        title="Audit Methodology",
        required=True,
        include_iia_reference=True,
        iia_standard_ref="IIA Standard 4.1 §7",
        include_narrative=include_narrative,
        narrative_section_key="methodology",
        order=order,
        custom_fields={
            "fields": [
                "sampling_method", "evidence_types", "risk_model",
                "quality_model", "sufficiency_formula",
            ]
        },
    )


def _iia_findings_matrix(order: int = 3) -> SectionConfig:
    return SectionConfig(
        section_type=SectionType.FINDINGS_MATRIX,
        title="Findings Matrix",
        required=True,
        include_iia_reference=True,
        iia_standard_ref="IIA Standard 4.1 §8",
        include_narrative=True,
        narrative_section_key="findings",
        include_management_response=True,
        order=order,
        custom_fields={
            "columns": [
                "ref", "title", "severity", "control_ref",
                "description", "evidence_count", "recommendation",
            ]
        },
    )


def _iia_evidence_references(order: int = 4) -> SectionConfig:
    return SectionConfig(
        section_type=SectionType.EVIDENCE_REFERENCES,
        title="Evidence References",
        required=True,
        include_iia_reference=True,
        iia_standard_ref="IIA Standard 4.1 §9",
        order=order,
        custom_fields={
            "columns": [
                "evidence_id", "title", "source_type",
                "collected_at", "hash", "chain_index",
            ]
        },
    )


def _iia_conclusions(order: int = 5, include_narrative: bool = True) -> SectionConfig:
    return SectionConfig(
        section_type=SectionType.CONCLUSIONS,
        title="Conclusions",
        required=True,
        include_iia_reference=True,
        iia_standard_ref="IIA Standard 4.1 §10",
        include_narrative=include_narrative,
        narrative_section_key="executive_summary",
        order=order,
        custom_fields={
            "fields": [
                "conclusion", "confidence_pct", "posterior",
                "sufficiency_verdict", "chain_valid",
            ]
        },
    )


def _iia_recommendations(order: int = 6) -> SectionConfig:
    return SectionConfig(
        section_type=SectionType.RECOMMENDATIONS,
        title="Recommendations",
        required=True,
        include_iia_reference=True,
        iia_standard_ref="IIA Standard 4.1 §11",
        include_narrative=True,
        narrative_section_key="recommendations",
        include_management_response=True,
        order=order,
        custom_fields={
            "columns": ["ref", "priority", "action", "owner", "due_date", "status"]
        },
    )


# Pre-built default templates

_DEFAULT_TEMPLATES: dict[EngagementType, WorkingPaperTemplate] = {
    EngagementType.COMPLIANCE: WorkingPaperTemplate(
        engagement_type=EngagementType.COMPLIANCE,
        name="NIS 2 / Regulatory Compliance Working Paper",
        description=(
            "Standard working paper for regulatory compliance engagements "
            "(NIS 2 Article 21, ISO 27001, DORA). IIA Standard 4.1 compliant."
        ),
        sections=[
            _iia_cover_page(0),
            _iia_scope(1),
            _iia_methodology(2),
            _iia_findings_matrix(3),
            _iia_evidence_references(4),
            _iia_conclusions(5),
            _iia_recommendations(6),
        ],
    ),
    EngagementType.OPERATIONAL: WorkingPaperTemplate(
        engagement_type=EngagementType.OPERATIONAL,
        name="Operational Controls Working Paper",
        description=(
            "Working paper for operational process and control effectiveness "
            "engagements. IIA Standard 4.1 compliant."
        ),
        sections=[
            _iia_cover_page(0),
            _iia_scope(1),
            _iia_methodology(2),
            _iia_findings_matrix(3),
            _iia_evidence_references(4),
            _iia_conclusions(5),
            _iia_recommendations(6),
        ],
    ),
    EngagementType.IT: WorkingPaperTemplate(
        engagement_type=EngagementType.IT,
        name="IT General Controls & Cybersecurity Working Paper",
        description=(
            "Working paper for IT general controls, cybersecurity, and "
            "technical risk engagements. IIA Standard 4.1 compliant."
        ),
        sections=[
            _iia_cover_page(0),
            _iia_scope(1),
            _iia_methodology(2, include_narrative=True),
            _iia_findings_matrix(3),
            _iia_evidence_references(4),
            _iia_conclusions(5),
            _iia_recommendations(6),
        ],
    ),
    EngagementType.FINANCIAL: WorkingPaperTemplate(
        engagement_type=EngagementType.FINANCIAL,
        name="Financial Controls Working Paper",
        description=(
            "Working paper for financial controls and assertion testing. "
            "IIA Standard 4.1 compliant."
        ),
        sections=[
            _iia_cover_page(0),
            _iia_scope(1),
            _iia_methodology(2),
            _iia_findings_matrix(3),
            _iia_evidence_references(4),
            _iia_conclusions(5, include_narrative=False),  # formal language only
            _iia_recommendations(6),
        ],
    ),
}


# ---------------------------------------------------------------------------
# Template Engine
# ---------------------------------------------------------------------------

class TemplateEngine:
    """Builds and renders IIA Standard 4.1 compliant working paper templates.

    Usage
    -----
    engine = TemplateEngine()
    template = engine.get_template(EngagementType.COMPLIANCE)
    rendered = engine.render(template, deterministic_data, narratives)
    """

    def get_template(
        self,
        engagement_type: EngagementType = EngagementType.COMPLIANCE,
    ) -> WorkingPaperTemplate:
        """Return the default template for an engagement type."""
        return _DEFAULT_TEMPLATES[engagement_type]

    def get_all_templates(self) -> dict[str, WorkingPaperTemplate]:
        """Return all built-in templates keyed by engagement type value."""
        return {et.value: tmpl for et, tmpl in _DEFAULT_TEMPLATES.items()}

    def customize_template(
        self,
        base: WorkingPaperTemplate,
        *,
        name: Optional[str] = None,
        add_sections: Optional[list[SectionConfig]] = None,
        remove_section_types: Optional[list[SectionType]] = None,
        update_fields: Optional[dict[str, Any]] = None,
    ) -> WorkingPaperTemplate:
        """Return a new template derived from ``base`` with modifications.

        The base template is NOT mutated — a fresh copy is returned.
        """
        import copy
        customised = copy.deepcopy(base)
        customised.template_id = uuid4()  # New identity
        if name:
            customised.name = name
        if remove_section_types:
            customised.sections = [
                s for s in customised.sections
                if s.section_type not in remove_section_types
            ]
        if add_sections:
            customised.sections.extend(add_sections)
        if update_fields:
            for key, val in update_fields.items():
                if hasattr(customised, key):
                    setattr(customised, key, val)
        return customised

    def render(
        self,
        template: WorkingPaperTemplate,
        deterministic_data: dict[str, Any],
        narratives: Optional[dict[str, str]] = None,
        engagement_id: Optional[UUID] = None,
    ) -> RenderedWorkingPaper:
        """Render a template with deterministic data and optional LLM narratives.

        Parameters
        ----------
        template:
            The WorkingPaperTemplate to render.
        deterministic_data:
            Dict with keys matching section content requirements.
            ALL audit conclusions and scores MUST come from this dict —
            never from the narratives dict.
        narratives:
            Optional dict keyed by NarrativeSection.value with LLM-generated
            text strings.  If a section has include_narrative=True but no
            matching narrative is provided, the field is left empty.
        engagement_id:
            UUID of the engagement for the rendered paper.
        """
        narratives = narratives or {}
        rendered_sections: list[RenderedSection] = []

        for sec_config in template.ordered_sections:
            narrative_text = ""
            narrative_is_mock = False
            if sec_config.include_narrative and sec_config.narrative_section_key:
                narrative_text = narratives.get(sec_config.narrative_section_key, "")
                narrative_is_mock = bool(narrative_text)  # mock until real LLM

            section_content = self._extract_section_content(
                sec_config.section_type, deterministic_data
            )
            rendered_sections.append(
                RenderedSection(
                    config=sec_config,
                    content=section_content,
                    narrative_text=narrative_text,
                    narrative_is_mock=narrative_is_mock,
                )
            )

        return RenderedWorkingPaper(
            engagement_id=engagement_id,
            template=template,
            sections=rendered_sections,
        )

    def _extract_section_content(
        self, section_type: SectionType, data: dict[str, Any]
    ) -> dict[str, Any]:
        """Extract the relevant subset of deterministic_data for a section."""
        section_keys: dict[SectionType, list[str]] = {
            SectionType.COVER_PAGE: [
                "engagement_title", "org_name", "auditor_name", "auditor_email",
                "period_start", "period_end", "classification", "version", "report_date",
            ],
            SectionType.SCOPE: ["objective", "in_scope_controls", "out_of_scope", "period"],
            SectionType.METHODOLOGY: [
                "sampling_method", "evidence_types", "risk_model",
                "quality_model", "sufficiency_formula",
            ],
            SectionType.FINDINGS_MATRIX: ["findings", "finding_count"],
            SectionType.EVIDENCE_REFERENCES: ["evidence_chain", "evidence_count"],
            SectionType.CONCLUSIONS: [
                "conclusion", "confidence_pct", "posterior",
                "sufficiency_verdict", "chain_valid",
            ],
            SectionType.RECOMMENDATIONS: ["recommendations", "finding_count"],
        }
        keys = section_keys.get(section_type, [])
        return {k: data[k] for k in keys if k in data}

