"""Documentation Engine — SHA-256 hash chain, evidence assembly, working papers.

Deterministic component — builds tamper-evident evidence chains.
LLM (AWS Bedrock Claude Sonnet 3.7) is used ONLY for narrative text generation
in the working paper; the chain itself, hashes, and conclusions are computed
algorithmically and never by LLM.

Hash Chain
----------
Each link in the chain is:
    hash_i = SHA-256( hash_{i-1} || evidence_id_i || content_hash_i || timestamp_i )

The chain starts with a genesis hash (all-zero 32 bytes).
Any mutation to any link invalidates all subsequent links — providing integrity proof.

Working Paper Schema
--------------------
A working paper records:
- Control under audit
- Evidence chain (ordered, hash-linked)
- Risk score result (from Risk Engine)
- Sufficiency result (from Quality Engine)
- Conclusion (deterministic)
- Narrative text placeholder (populated by LLM in production)
"""
from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Optional
from uuid import UUID, uuid4


# Genesis hash: all zeros (32 bytes = 64 hex chars)
GENESIS_HASH = "0" * 64


# ---------------------------------------------------------------------------
# Evidence chain link
# ---------------------------------------------------------------------------

@dataclass
class ChainLink:
    """A single link in the evidence hash chain."""
    link_index: int
    evidence_id: UUID
    content_hash: str         # SHA-256 of the evidence content bytes
    timestamp: str            # ISO-8601 UTC
    previous_hash: str        # Hash of the previous link (or GENESIS_HASH)
    link_hash: str            # SHA-256 of this link's canonical representation
    metadata: dict[str, Any] = field(default_factory=dict)

    def to_canonical(self) -> str:
        """Produce a deterministic canonical representation for hashing."""
        return json.dumps(
            {
                "link_index": self.link_index,
                "evidence_id": str(self.evidence_id),
                "content_hash": self.content_hash,
                "timestamp": self.timestamp,
                "previous_hash": self.previous_hash,
            },
            sort_keys=True,
            separators=(",", ":"),
        )


def compute_content_hash(content: str | bytes) -> str:
    """Return the SHA-256 hex digest of content."""
    if isinstance(content, str):
        content = content.encode("utf-8")
    return hashlib.sha256(content).hexdigest()


def compute_link_hash(
    previous_hash: str,
    evidence_id: UUID,
    content_hash: str,
    timestamp: str,
    link_index: int,
) -> str:
    """Compute the SHA-256 hash for a chain link."""
    canonical = json.dumps(
        {
            "link_index": link_index,
            "evidence_id": str(evidence_id),
            "content_hash": content_hash,
            "timestamp": timestamp,
            "previous_hash": previous_hash,
        },
        sort_keys=True,
        separators=(",", ":"),
    )
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()


# ---------------------------------------------------------------------------
# Evidence hash chain
# ---------------------------------------------------------------------------

class EvidenceHashChain:
    """Ordered, tamper-evident SHA-256 hash chain over evidence items.

    Usage
    -----
    chain = EvidenceHashChain()
    chain.append(evidence_id=..., content=b"...", metadata={...})
    valid = chain.verify()
    """

    def __init__(self) -> None:
        self._links: list[ChainLink] = []
        self._tail_hash: str = GENESIS_HASH

    def append(
        self,
        evidence_id: UUID,
        content: str | bytes,
        metadata: Optional[dict[str, Any]] = None,
        timestamp: Optional[str] = None,
    ) -> ChainLink:
        """Append a new evidence item to the chain."""
        ts = timestamp or datetime.now(timezone.utc).isoformat()
        content_hash = compute_content_hash(content)
        idx = len(self._links)
        link_hash = compute_link_hash(
            previous_hash=self._tail_hash,
            evidence_id=evidence_id,
            content_hash=content_hash,
            timestamp=ts,
            link_index=idx,
        )
        link = ChainLink(
            link_index=idx,
            evidence_id=evidence_id,
            content_hash=content_hash,
            timestamp=ts,
            previous_hash=self._tail_hash,
            link_hash=link_hash,
            metadata=metadata or {},
        )
        self._links.append(link)
        self._tail_hash = link_hash
        return link

    def verify(self) -> bool:
        """Recompute all link hashes and verify the chain is unbroken."""
        prev_hash = GENESIS_HASH
        for link in self._links:
            expected = compute_link_hash(
                previous_hash=prev_hash,
                evidence_id=link.evidence_id,
                content_hash=link.content_hash,
                timestamp=link.timestamp,
                link_index=link.link_index,
            )
            if expected != link.link_hash:
                return False
            prev_hash = link.link_hash
        return True

    @property
    def tail_hash(self) -> str:
        return self._tail_hash

    @property
    def length(self) -> int:
        return len(self._links)

    def links(self) -> list[ChainLink]:
        return list(self._links)


# ---------------------------------------------------------------------------
# Working paper
# ---------------------------------------------------------------------------

@dataclass
class WorkingPaper:
    """A complete audit working paper for one control."""
    paper_id: UUID = field(default_factory=uuid4)
    control_id: str = ""
    engagement_id: Optional[UUID] = None
    chain: EvidenceHashChain = field(default_factory=EvidenceHashChain)

    # Risk Engine output
    conclusion: str = ""           # e.g. "Effective"
    confidence_pct: float = 0.0    # 0–100
    posterior: float = 0.0

    # Quality Engine output
    sufficiency_verdict: str = ""  # "sufficient" | "marginal" | "insufficient"
    sufficiency_ratio: float = 0.0

    # Narrative (LLM-generated placeholder — AWS Bedrock in production)
    narrative_text: str = ""
    narrative_generated_by: str = "placeholder"  # "llm" when populated

    created_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )
    metadata: dict[str, Any] = field(default_factory=dict)

    def finalize(self) -> dict[str, Any]:
        """Produce a serialisable, immutable snapshot of the working paper."""
        return {
            "paper_id": str(self.paper_id),
            "control_id": self.control_id,
            "engagement_id": str(self.engagement_id) if self.engagement_id else None,
            "conclusion": self.conclusion,
            "confidence_pct": round(self.confidence_pct, 2),
            "posterior": round(self.posterior, 4),
            "sufficiency_verdict": self.sufficiency_verdict,
            "sufficiency_ratio": round(self.sufficiency_ratio, 4),
            "evidence_chain": {
                "length": self.chain.length,
                "tail_hash": self.chain.tail_hash,
                "links": [
                    {
                        "index": lnk.link_index,
                        "evidence_id": str(lnk.evidence_id),
                        "content_hash": lnk.content_hash,
                        "timestamp": lnk.timestamp,
                        "link_hash": lnk.link_hash,
                    }
                    for lnk in self.chain.links()
                ],
                "chain_valid": self.chain.verify(),
            },
            "narrative_text": self.narrative_text,
            "narrative_generated_by": self.narrative_generated_by,
            "created_at": self.created_at,
            "metadata": self.metadata,
        }
