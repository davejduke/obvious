"""Unit tests — Documentation Engine (SHA-256 hash chain, working paper)."""
import hashlib
import pytest
from uuid import uuid4
from engine.documentation.chain import (
    EvidenceHashChain,
    WorkingPaper,
    ChainLink,
    compute_content_hash,
    compute_link_hash,
    GENESIS_HASH,
)


class TestComputeContentHash:
    def test_deterministic(self):
        h1 = compute_content_hash("hello")
        h2 = compute_content_hash("hello")
        assert h1 == h2

    def test_bytes_input(self):
        h = compute_content_hash(b"hello")
        expected = hashlib.sha256(b"hello").hexdigest()
        assert h == expected

    def test_str_input(self):
        h = compute_content_hash("hello")
        expected = hashlib.sha256(b"hello").hexdigest()
        assert h == expected

    def test_different_content_different_hash(self):
        assert compute_content_hash("a") != compute_content_hash("b")


class TestEvidenceHashChain:
    def test_empty_chain_has_genesis_tail(self):
        chain = EvidenceHashChain()
        assert chain.tail_hash == GENESIS_HASH
        assert chain.length == 0
        assert chain.verify()

    def test_append_single_link(self):
        chain = EvidenceHashChain()
        link = chain.append(evidence_id=uuid4(), content="evidence text")
        assert chain.length == 1
        assert link.link_index == 0
        assert link.previous_hash == GENESIS_HASH

    def test_chain_grows_sequentially(self):
        chain = EvidenceHashChain()
        for i in range(5):
            chain.append(evidence_id=uuid4(), content=f"item {i}")
        assert chain.length == 5
        for idx, link in enumerate(chain.links()):
            assert link.link_index == idx

    def test_verify_intact_chain(self):
        chain = EvidenceHashChain()
        for _ in range(3):
            chain.append(uuid4(), "content")
        assert chain.verify() is True

    def test_tampered_chain_fails_verify(self):
        chain = EvidenceHashChain()
        chain.append(uuid4(), "original")
        chain.append(uuid4(), "second")
        # Directly tamper with a link hash
        chain._links[0].content_hash = "tampered" * 8  # wrong hash
        assert chain.verify() is False

    def test_previous_hash_linkage(self):
        chain = EvidenceHashChain()
        l0 = chain.append(uuid4(), "first")
        l1 = chain.append(uuid4(), "second")
        assert l1.previous_hash == l0.link_hash

    def test_tail_hash_is_last_link_hash(self):
        chain = EvidenceHashChain()
        for i in range(4):
            last = chain.append(uuid4(), f"item {i}")
        assert chain.tail_hash == last.link_hash

    def test_metadata_stored(self):
        chain = EvidenceHashChain()
        link = chain.append(uuid4(), "content", metadata={"source": "SIEM"})
        assert link.metadata["source"] == "SIEM"


class TestWorkingPaper:
    def test_finalize_produces_serialisable_dict(self):
        paper = WorkingPaper(
            control_id="NIS2-21a",
            conclusion="Effective",
            confidence_pct=87.5,
            posterior=0.92,
        )
        result = paper.finalize()
        assert result["control_id"] == "NIS2-21a"
        assert result["conclusion"] == "Effective"
        assert result["confidence_pct"] == 87.5

    def test_finalize_includes_chain_validity(self):
        paper = WorkingPaper(control_id="C-001")
        paper.chain.append(uuid4(), "evidence A")
        result = paper.finalize()
        assert result["evidence_chain"]["chain_valid"] is True
        assert result["evidence_chain"]["length"] == 1

    def test_narrative_placeholder_present(self):
        paper = WorkingPaper(control_id="C-001", narrative_generated_by="placeholder")
        result = paper.finalize()
        assert result["narrative_generated_by"] == "placeholder"


# ---------------------------------------------------------------------------
# Bedrock stub tests
# ---------------------------------------------------------------------------

from engine.documentation.bedrock import (
    BedrockConfig,
    BedrockResponse,
    MockBedrockClient,
    NarrativeSection,
    NarrativeTone,
    create_bedrock_client,
)


class TestBedrockConfig:
    def test_default_model_id(self):
        cfg = BedrockConfig()
        assert "claude" in cfg.model_id
        assert "sonnet" in cfg.model_id.lower()

    def test_to_invoke_body_structure(self):
        cfg = BedrockConfig()
        body = cfg.to_invoke_body("test prompt")
        assert body["anthropic_version"] == "bedrock-2023-05-31"
        assert body["messages"][0]["role"] == "user"
        assert body["messages"][0]["content"] == "test prompt"
        assert body["max_tokens"] == cfg.max_tokens

    def test_no_api_key_fields(self):
        cfg = BedrockConfig()
        assert not hasattr(cfg, "api_key")
        assert not hasattr(cfg, "secret_key")


class TestMockBedrockClient:
    def test_is_stub_no_network(self):
        client = MockBedrockClient()
        # Should return immediately without network I/O
        resp = client.generate_text("formal audit", NarrativeSection.EXECUTIVE_SUMMARY)
        assert resp.is_mock is True

    def test_returns_bedrock_response(self):
        client = MockBedrockClient()
        resp = client.generate_text("test", NarrativeSection.METHODOLOGY)
        assert isinstance(resp, BedrockResponse)
        assert resp.text != ""

    def test_call_count_increments(self):
        client = MockBedrockClient()
        assert client.call_count == 0
        client.generate_text("x", NarrativeSection.FINDINGS)
        client.generate_text("x", NarrativeSection.RECOMMENDATIONS)
        assert client.call_count == 2

    def test_tone_detection_executive(self):
        client = MockBedrockClient()
        resp = client.generate_text(
            "Write in executive tone for the board", NarrativeSection.EXECUTIVE_SUMMARY
        )
        assert resp.is_mock is True
        # Executive tone narrative is shorter / more concise
        assert len(resp.text) > 10

    def test_tone_detection_technical(self):
        client = MockBedrockClient()
        resp = client.generate_text(
            "Write in technical tone", NarrativeSection.METHODOLOGY
        )
        assert resp.is_mock is True

    def test_all_sections_return_text(self):
        client = MockBedrockClient()
        for section in NarrativeSection:
            resp = client.generate_text("formal", section)
            assert resp.text != "", f"Empty text for section {section}"

    def test_model_id_includes_mock_marker(self):
        client = MockBedrockClient()
        resp = client.generate_text("x", NarrativeSection.FINDINGS)
        assert "#mock" in resp.model_id

    def test_create_bedrock_client_factory_default(self):
        client = create_bedrock_client()
        assert isinstance(client, MockBedrockClient)

    def test_create_bedrock_client_factory_real_raises(self):
        import pytest
        with pytest.raises(NotImplementedError):
            create_bedrock_client(use_real=True)


# ---------------------------------------------------------------------------
# Narrative generator tests
# ---------------------------------------------------------------------------

from engine.documentation.narrative import (
    NarrativeContext,
    NarrativeGenerator,
    NarrativeResult,
    WorkingPaperNarratives,
)


def _sample_context() -> NarrativeContext:
    return NarrativeContext(
        conclusion="Effective",
        confidence_pct=87.5,
        posterior=0.912,
        control_count=10,
        sufficiency_verdict="sufficient",
        sufficiency_ratio=0.85,
        evidence_count=5,
        chain_length=5,
        evidence_types="manual upload and SIEM integration",
        framework="NIS 2 Article 21",
        period_start="2024-01-01",
        period_end="2024-06-30",
        finding_count=3,
        critical_count=1,
        high_count=1,
        medium_count=1,
        low_count=0,
    )


class TestNarrativeContext:
    def test_as_format_kwargs_returns_expected_keys(self):
        ctx = _sample_context()
        kwargs = ctx.as_format_kwargs()
        required_keys = [
            "conclusion", "confidence_pct", "posterior", "control_count",
            "sufficiency_verdict", "sufficiency_ratio", "evidence_count",
            "chain_length", "evidence_types", "framework",
            "period_start", "period_end", "finding_count",
            "critical_count", "high_count", "medium_count", "low_count",
        ]
        for key in required_keys:
            assert key in kwargs, f"Missing key: {key}"

    def test_confidence_pct_rounded(self):
        ctx = _sample_context()
        ctx.confidence_pct = 87.567
        assert ctx.as_format_kwargs()["confidence_pct"] == 87.6


class TestNarrativeGenerator:
    def test_generate_executive_summary(self):
        gen = NarrativeGenerator()
        result = gen.generate_executive_summary(_sample_context(), NarrativeTone.FORMAL)
        assert isinstance(result, NarrativeResult)
        assert result.text != ""
        assert result.section == NarrativeSection.EXECUTIVE_SUMMARY
        assert result.tone == NarrativeTone.FORMAL

    def test_generate_methodology(self):
        gen = NarrativeGenerator()
        result = gen.generate_methodology(_sample_context(), NarrativeTone.TECHNICAL)
        assert result.section == NarrativeSection.METHODOLOGY
        assert result.tone == NarrativeTone.TECHNICAL

    def test_generate_findings(self):
        gen = NarrativeGenerator()
        result = gen.generate_findings(_sample_context(), NarrativeTone.EXECUTIVE)
        assert result.section == NarrativeSection.FINDINGS
        assert result.tone == NarrativeTone.EXECUTIVE

    def test_generate_recommendations(self):
        gen = NarrativeGenerator()
        result = gen.generate_recommendations(_sample_context())
        assert result.section == NarrativeSection.RECOMMENDATIONS

    def test_generate_all_returns_all_sections(self):
        gen = NarrativeGenerator()
        narratives = gen.generate_all(_sample_context(), NarrativeTone.FORMAL)
        assert isinstance(narratives, WorkingPaperNarratives)
        assert narratives.executive_summary.text != ""
        assert narratives.methodology.text != ""
        assert narratives.findings.text != ""
        assert narratives.recommendations.text != ""
        assert narratives.tone == NarrativeTone.FORMAL

    def test_narrative_resolves_conclusion_placeholder(self):
        """Deterministic conclusion must appear in resolved narrative text."""
        gen = NarrativeGenerator()
        ctx = _sample_context()
        ctx.conclusion = "Effective"
        result = gen.generate_executive_summary(ctx, NarrativeTone.FORMAL)
        # The mock template for formal exec summary contains {conclusion}
        assert "Effective" in result.text

    def test_narrative_resolves_framework_placeholder(self):
        gen = NarrativeGenerator()
        ctx = _sample_context()
        ctx.framework = "ISO 27001"
        result = gen.generate_executive_summary(ctx, NarrativeTone.FORMAL)
        assert "ISO 27001" in result.text

    def test_llm_never_sets_conclusion_in_context(self):
        """The NarrativeGenerator must not modify NarrativeContext."""
        gen = NarrativeGenerator()
        ctx = _sample_context()
        original_conclusion = ctx.conclusion
        gen.generate_all(ctx)
        assert ctx.conclusion == original_conclusion  # unchanged

    def test_is_mock_flag_set(self):
        gen = NarrativeGenerator()
        result = gen.generate_findings(_sample_context())
        assert result.is_mock is True

    def test_to_dict_serialisable(self):
        gen = NarrativeGenerator()
        narratives = gen.generate_all(_sample_context())
        d = narratives.to_dict()
        assert "executive_summary" in d
        assert "methodology" in d
        assert "findings" in d
        assert "recommendations" in d
        assert d["tone"] == NarrativeTone.FORMAL.value

    def test_custom_mock_client_injected(self):
        mock_client = MockBedrockClient()
        gen = NarrativeGenerator(client=mock_client)
        gen.generate_all(_sample_context())
        assert mock_client.call_count == 4  # one per section


# ---------------------------------------------------------------------------
# Template engine tests
# ---------------------------------------------------------------------------

from engine.documentation.templates import (
    EngagementType,
    RenderedWorkingPaper,
    SectionType,
    TemplateEngine,
    WorkingPaperTemplate,
)


IIA_REQUIRED_SECTIONS = {
    SectionType.COVER_PAGE,
    SectionType.SCOPE,
    SectionType.METHODOLOGY,
    SectionType.FINDINGS_MATRIX,
    SectionType.EVIDENCE_REFERENCES,
    SectionType.CONCLUSIONS,
    SectionType.RECOMMENDATIONS,
}


class TestTemplateEngine:
    def test_get_compliance_template(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        assert isinstance(tmpl, WorkingPaperTemplate)
        assert tmpl.engagement_type == EngagementType.COMPLIANCE

    def test_get_it_template(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.IT)
        assert tmpl.engagement_type == EngagementType.IT

    def test_all_templates_available(self):
        engine = TemplateEngine()
        templates = engine.get_all_templates()
        assert set(templates.keys()) == {et.value for et in EngagementType}

    def test_compliance_template_has_all_iia_sections(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        section_types = {s.section_type for s in tmpl.sections}
        assert IIA_REQUIRED_SECTIONS == section_types

    def test_iia_standard_reference_present(self):
        """Every section must have an IIA standard reference."""
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        for sec in tmpl.sections:
            if sec.include_iia_reference:
                assert sec.iia_standard_ref.startswith("IIA Standard 4.1")

    def test_findings_section_has_management_response(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        findings_sec = tmpl.get_section(SectionType.FINDINGS_MATRIX)
        assert findings_sec is not None
        assert findings_sec.include_management_response is True

    def test_methodology_section_has_narrative(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        meth = tmpl.get_section(SectionType.METHODOLOGY)
        assert meth is not None
        assert meth.include_narrative is True
        assert meth.narrative_section_key == "methodology"

    def test_sections_ordered_correctly(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        orders = [s.order for s in tmpl.ordered_sections]
        assert orders == sorted(orders)

    def test_to_dict_serialisable(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.IT)
        d = tmpl.to_dict()
        assert d["iia_standard"] == "IIA Standard 4.1"
        assert len(d["sections"]) == 7

    def test_customize_removes_section(self):
        engine = TemplateEngine()
        base = engine.get_template(EngagementType.COMPLIANCE)
        custom = engine.customize_template(
            base, remove_section_types=[SectionType.SCOPE]
        )
        section_types = {s.section_type for s in custom.sections}
        assert SectionType.SCOPE not in section_types
        # Base not mutated
        assert any(s.section_type == SectionType.SCOPE for s in base.sections)

    def test_customize_adds_section(self):
        from engine.documentation.templates import SectionConfig
        engine = TemplateEngine()
        base = engine.get_template(EngagementType.COMPLIANCE)
        extra = SectionConfig(
            section_type=SectionType.SCOPE, title="Extra", order=99
        )
        custom = engine.customize_template(base, add_sections=[extra])
        assert len(custom.sections) == len(base.sections) + 1

    def test_customize_returns_new_template_id(self):
        engine = TemplateEngine()
        base = engine.get_template(EngagementType.COMPLIANCE)
        custom = engine.customize_template(base, name="Custom")
        assert custom.template_id != base.template_id

    def test_render_without_narratives(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        data = {
            "conclusion": "Effective",
            "confidence_pct": 87.5,
            "posterior": 0.91,
            "sufficiency_verdict": "sufficient",
            "chain_valid": True,
        }
        rendered = engine.render(tmpl, data)
        assert isinstance(rendered, RenderedWorkingPaper)
        assert len(rendered.sections) == 7

    def test_render_with_narratives_embeds_text(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        narratives = {"methodology": "This is the methodology narrative."}
        rendered = engine.render(tmpl, {}, narratives=narratives)
        meth_section = next(
            s for s in rendered.sections
            if s.config.section_type == SectionType.METHODOLOGY
        )
        assert meth_section.narrative_text == "This is the methodology narrative."

    def test_rendered_working_paper_to_dict(self):
        engine = TemplateEngine()
        tmpl = engine.get_template(EngagementType.COMPLIANCE)
        rendered = engine.render(tmpl, {})
        d = rendered.to_dict()
        assert "sections" in d
        assert len(d["sections"]) == 7


# ---------------------------------------------------------------------------
# Enhanced working paper + version history tests
# ---------------------------------------------------------------------------

from engine.documentation.working_paper import (
    EnhancedWorkingPaper,
    PaperVersion,
    VersionHistory,
    build_working_paper,
)


class TestVersionHistory:
    def test_empty_history(self):
        h = VersionHistory()
        assert h.count == 0
        assert h.latest is None

    def test_append_version(self):
        h = VersionHistory()
        v = PaperVersion(version_number=1, conclusion="Effective")
        h.append(v)
        assert h.count == 1
        assert h.latest is v

    def test_sequential_version_enforcement(self):
        import pytest
        h = VersionHistory()
        h.append(PaperVersion(version_number=1))
        with pytest.raises(ValueError, match="Expected version 2"):
            h.append(PaperVersion(version_number=5))  # wrong sequence

    def test_multiple_versions(self):
        h = VersionHistory()
        for i in range(1, 4):
            h.append(PaperVersion(version_number=i))
        assert h.count == 3
        assert h.latest.version_number == 3

    def test_to_dict(self):
        h = VersionHistory()
        h.append(PaperVersion(version_number=1, conclusion="Effective"))
        d = h.to_dict()
        assert d["count"] == 1
        assert d["versions"][0]["conclusion"] == "Effective"


class TestEnhancedWorkingPaper:
    def test_create_paper(self):
        paper = build_working_paper("NIS2-21b")
        assert paper.control_id == "NIS2-21b"
        assert paper.history.count == 0

    def test_chain_accessible(self):
        paper = build_working_paper("C-001")
        paper.chain.append(uuid4(), "evidence content")
        assert paper.chain.length == 1

    def test_set_deterministic_outputs(self):
        paper = build_working_paper("C-001")
        paper.set_deterministic_outputs(
            conclusion="Not Effective",
            confidence_pct=23.0,
            posterior=0.21,
            sufficiency_verdict="insufficient",
            sufficiency_ratio=0.45,
        )
        snapshot = paper.finalize()
        # LLM does NOT produce these values — deterministic engine does
        assert snapshot["conclusion"] == "Not Effective"
        assert snapshot["confidence_pct"] == 23.0

    def test_llm_does_not_override_conclusion(self):
        """Critical constraint: LLM text must never change the stored conclusion."""
        paper = build_working_paper("C-001")
        paper.set_deterministic_outputs(
            conclusion="Effective",
            confidence_pct=90.0,
            posterior=0.92,
            sufficiency_verdict="sufficient",
            sufficiency_ratio=0.88,
        )
        ctx = _sample_context()
        ctx.conclusion = "Effective"  # Must match deterministic value
        paper.generate_narratives(ctx, NarrativeTone.FORMAL)
        snapshot = paper.finalize()
        # Conclusion in snapshot comes from set_deterministic_outputs, not LLM
        assert snapshot["conclusion"] == "Effective"

    def test_generate_narratives_all_sections(self):
        paper = build_working_paper("C-001")
        narratives = paper.generate_narratives(_sample_context(), NarrativeTone.FORMAL)
        assert narratives.executive_summary.text != ""
        assert narratives.methodology.text != ""
        assert narratives.findings.text != ""
        assert narratives.recommendations.text != ""

    def test_save_version_creates_snapshot(self):
        paper = build_working_paper("C-001")
        paper.set_deterministic_outputs(
            conclusion="Effective", confidence_pct=88.0, posterior=0.91,
            sufficiency_verdict="sufficient", sufficiency_ratio=0.87,
        )
        v = paper.save_version(created_by="alice", change_note="Initial version")
        assert v.version_number == 1
        assert v.created_by == "alice"
        assert v.change_note == "Initial version"
        assert v.conclusion == "Effective"
        assert paper.history.count == 1

    def test_multiple_versions_track_changes(self):
        paper = build_working_paper("C-001")
        paper.set_deterministic_outputs(
            conclusion="Not Effective", confidence_pct=35.0, posterior=0.32,
            sufficiency_verdict="insufficient", sufficiency_ratio=0.5,
        )
        paper.save_version(change_note="Draft")
        paper.set_deterministic_outputs(
            conclusion="Effective", confidence_pct=89.0, posterior=0.91,
            sufficiency_verdict="sufficient", sufficiency_ratio=0.87,
        )
        paper.save_version(change_note="After remediation")
        assert paper.history.count == 2
        assert paper.history.versions[0].conclusion == "Not Effective"
        assert paper.history.versions[1].conclusion == "Effective"

    def test_version_includes_chain_state(self):
        paper = build_working_paper("C-001")
        paper.chain.append(uuid4(), "ev1")
        paper.chain.append(uuid4(), "ev2")
        paper.set_deterministic_outputs(
            conclusion="Effective", confidence_pct=88.0, posterior=0.9,
            sufficiency_verdict="sufficient", sufficiency_ratio=0.8,
        )
        v = paper.save_version()
        assert v.chain_length == 2
        assert v.chain_valid is True
        assert v.chain_tail_hash == paper.chain.tail_hash

    def test_version_with_narrative(self):
        paper = build_working_paper("C-001")
        paper.set_deterministic_outputs(
            conclusion="Effective", confidence_pct=88.0, posterior=0.9,
            sufficiency_verdict="sufficient", sufficiency_ratio=0.8,
        )
        narratives = paper.generate_narratives(_sample_context())
        v = paper.save_version(narratives=narratives)
        assert v.narrative_executive_summary != ""
        assert v.narrative_is_mock is True

    def test_render_produces_rendered_paper(self):
        paper = build_working_paper("C-001")
        paper.set_deterministic_outputs(
            conclusion="Effective", confidence_pct=88.0, posterior=0.9,
            sufficiency_verdict="sufficient", sufficiency_ratio=0.8,
        )
        narratives = paper.generate_narratives(_sample_context())
        rendered = paper.render(
            deterministic_data={
                "conclusion": "Effective",
                "confidence_pct": 88.0,
                "posterior": 0.9,
                "sufficiency_verdict": "sufficient",
                "chain_valid": True,
                "finding_count": 3,
                "findings": [],
                "evidence_chain": [],
                "evidence_count": 5,
            },
            narratives=narratives,
        )
        assert len(rendered.sections) == 7
        section_types = {s.config.section_type for s in rendered.sections}
        assert IIA_REQUIRED_SECTIONS == section_types

    def test_finalize_includes_version_history(self):
        paper = build_working_paper("C-001")
        paper.set_deterministic_outputs(
            conclusion="Effective", confidence_pct=88.0, posterior=0.9,
            sufficiency_verdict="sufficient", sufficiency_ratio=0.8,
        )
        paper.save_version(change_note="v1")
        snapshot = paper.finalize()
        assert snapshot["version_history"]["count"] == 1
        assert snapshot["version_history"]["versions"][0]["change_note"] == "v1"

    def test_finalize_includes_evidence_chain_status(self):
        paper = build_working_paper("C-001")
        paper.chain.append(uuid4(), "evidence")
        paper.set_deterministic_outputs(
            conclusion="Effective", confidence_pct=80.0, posterior=0.85,
            sufficiency_verdict="sufficient", sufficiency_ratio=0.75,
        )
        snapshot = paper.finalize()
        assert snapshot["evidence_chain"]["length"] == 1
        assert snapshot["evidence_chain"]["chain_valid"] is True
