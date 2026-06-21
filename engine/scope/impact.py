"""Scope Engine — scope change impact analysis.

Given a ``ScopeDiff`` (before → after), computes which findings and evidence
items are affected when controls move in or out of scope.

Usage::

    analyzer = ScopeImpactAnalyzer()
    impact = analyzer.analyze(
        diff=diff,
        findings=[FindingRef("f-001", "NIS2-21a")],
        evidence=[EvidenceRef("ev-001", "NIS2-21a")],
    )
    if impact.has_impact:
        print(impact.to_dict())
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from engine.scope.dag import BoundaryStatus
from engine.scope.history import ScopeDiff


@dataclass
class FindingRef:
    """Lightweight reference to a finding, keyed by the control it belongs to."""

    finding_id: str
    control_id: str


@dataclass
class EvidenceRef:
    """Lightweight reference to an evidence item keyed by control ID."""

    evidence_id: str
    control_id: str


@dataclass
class ImpactedFinding:
    finding_id: str
    control_id: str
    impact_reason: str  # "control_removed_from_scope" | "control_added_to_scope"


@dataclass
class ImpactedEvidence:
    evidence_id: str
    control_id: str
    impact_reason: str


@dataclass
class ScopeChangeImpact:
    """Result of a scope change impact analysis."""

    diff: ScopeDiff
    affected_findings: list[ImpactedFinding] = field(default_factory=list)
    affected_evidence: list[ImpactedEvidence] = field(default_factory=list)

    @property
    def has_impact(self) -> bool:
        """True if any findings or evidence are affected."""
        return bool(self.affected_findings or self.affected_evidence)

    def to_dict(self) -> dict[str, Any]:
        return {
            "diff": self.diff.to_dict(),
            "affected_findings_count": len(self.affected_findings),
            "affected_evidence_count": len(self.affected_evidence),
            "affected_findings": [
                {
                    "finding_id": f.finding_id,
                    "control_id": f.control_id,
                    "impact_reason": f.impact_reason,
                }
                for f in self.affected_findings
            ],
            "affected_evidence": [
                {
                    "evidence_id": e.evidence_id,
                    "control_id": e.control_id,
                    "impact_reason": e.impact_reason,
                }
                for e in self.affected_evidence
            ],
        }


class ScopeImpactAnalyzer:
    """Analyses the business impact of scope changes on findings and evidence.

    The analyzer is stateless: each ``analyze`` call is independent.
    """

    def analyze(
        self,
        diff: ScopeDiff,
        findings: list[FindingRef],
        evidence: list[EvidenceRef],
    ) -> ScopeChangeImpact:
        """Compute impact on findings and evidence given a scope diff.

        Args:
            diff: The ``ScopeDiff`` produced by ``ScopeVersionHistory.diff``.
            findings: All findings currently linked to any control.
            evidence: All evidence items currently linked to any control.

        Returns:
            ``ScopeChangeImpact`` with affected findings and evidence annotated
            with a human-readable ``impact_reason``.
        """
        impact = ScopeChangeImpact(diff=diff)

        # Build lookup sets for each change category
        removed_controls = {c.control_id for c in diff.removed}
        added_controls = {c.control_id for c in diff.added}
        newly_out_of_scope = {
            c.control_id
            for c in diff.modified
            if c.new_boundary == BoundaryStatus.OUT_OF_SCOPE
            and c.old_boundary == BoundaryStatus.IN_SCOPE
        }
        newly_in_scope = {
            c.control_id
            for c in diff.modified
            if c.new_boundary == BoundaryStatus.IN_SCOPE
            and c.old_boundary != BoundaryStatus.IN_SCOPE
        }

        scope_reduced = removed_controls | newly_out_of_scope
        scope_expanded = added_controls | newly_in_scope

        for f in findings:
            if f.control_id in scope_reduced:
                impact.affected_findings.append(
                    ImpactedFinding(
                        finding_id=f.finding_id,
                        control_id=f.control_id,
                        impact_reason="control_removed_from_scope",
                    )
                )
            elif f.control_id in scope_expanded:
                impact.affected_findings.append(
                    ImpactedFinding(
                        finding_id=f.finding_id,
                        control_id=f.control_id,
                        impact_reason="control_added_to_scope",
                    )
                )

        for e in evidence:
            if e.control_id in scope_reduced:
                impact.affected_evidence.append(
                    ImpactedEvidence(
                        evidence_id=e.evidence_id,
                        control_id=e.control_id,
                        impact_reason="control_removed_from_scope",
                    )
                )
            elif e.control_id in scope_expanded:
                impact.affected_evidence.append(
                    ImpactedEvidence(
                        evidence_id=e.evidence_id,
                        control_id=e.control_id,
                        impact_reason="control_added_to_scope",
                    )
                )

        return impact

