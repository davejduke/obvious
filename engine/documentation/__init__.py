"""Documentation Engine — SHA-256 hash chain, evidence assembly, working papers."""
from engine.documentation.chain import (
    EvidenceHashChain,
    WorkingPaper,
    ChainLink,
    compute_content_hash,
    compute_link_hash,
    GENESIS_HASH,
)

__all__ = [
    "EvidenceHashChain",
    "WorkingPaper",
    "ChainLink",
    "compute_content_hash",
    "compute_link_hash",
    "GENESIS_HASH",
]
