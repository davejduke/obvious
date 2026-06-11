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
