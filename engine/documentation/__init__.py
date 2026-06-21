"""Documentation Engine — SHA-256 hash chain, evidence assembly, working papers.

Phase 3 additions:
- AWS Bedrock Claude Sonnet 3.7 stub (MockBedrockClient)
- LLM narrative generation (NarrativeGenerator)
- Working paper template engine (TemplateEngine, IIA Standard 4.1)
- Enhanced working paper with version history (EnhancedWorkingPaper)
"""
from engine.documentation.chain import (
    EvidenceHashChain,
    WorkingPaper,
    ChainLink,
    compute_content_hash,
    compute_link_hash,
    GENESIS_HASH,
)
from engine.documentation.bedrock import (
    BedrockConfig,
    BedrockResponse,
    MockBedrockClient,
    NarrativeSection,
    NarrativeTone,
    create_bedrock_client,
)
from engine.documentation.narrative import (
    NarrativeContext,
    NarrativeGenerator,
    NarrativeResult,
    WorkingPaperNarratives,
)
from engine.documentation.templates import (
    EngagementType,
    RenderedSection,
    RenderedWorkingPaper,
    SectionConfig,
    SectionType,
    TemplateEngine,
    WorkingPaperTemplate,
)
from engine.documentation.working_paper import (
    EnhancedWorkingPaper,
    PaperVersion,
    VersionHistory,
    build_working_paper,
)

__all__ = [
    # Phase 1 — chain
    "EvidenceHashChain",
    "WorkingPaper",
    "ChainLink",
    "compute_content_hash",
    "compute_link_hash",
    "GENESIS_HASH",
    # Phase 3 — Bedrock stub
    "BedrockConfig",
    "BedrockResponse",
    "MockBedrockClient",
    "NarrativeSection",
    "NarrativeTone",
    "create_bedrock_client",
    # Phase 3 — narrative generator
    "NarrativeContext",
    "NarrativeGenerator",
    "NarrativeResult",
    "WorkingPaperNarratives",
    # Phase 3 — template engine
    "EngagementType",
    "RenderedSection",
    "RenderedWorkingPaper",
    "SectionConfig",
    "SectionType",
    "TemplateEngine",
    "WorkingPaperTemplate",
    # Phase 3 — enhanced working paper
    "EnhancedWorkingPaper",
    "PaperVersion",
    "VersionHistory",
    "build_working_paper",
]
