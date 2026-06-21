"""Economy Engine — Work program templates (reusable across engagements).

Templates define the canonical set of work steps for a given control objective
type.  When a new engagement is created, templates are instantiated into
concrete WorkItem objects with engagement-specific UUIDs and scaling factors.

All template logic is deterministic: no LLM is consulted.

Built-in templates
------------------
Five NIS 2 Article 21 templates ship with the engine:
  - access_control          (NIS2-21b)
  - incident_response       (NIS2-21a)
  - vulnerability_management (NIS2-21e)
  - supply_chain            (NIS2-21d)
  - cryptography            (NIS2-21c)

Custom templates can be registered via TEMPLATE_REGISTRY.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional
from uuid import UUID, uuid4

from engine.economy.work_program import WorkItem


@dataclass(frozen=True)
class TemplateStep:
    """A single reusable step definition within a work program template."""
    step_key: str          # stable identifier within the template (e.g. "policy_review")
    title: str
    description: str
    base_hours: float      # default estimated hours; scaled by hours_multiplier
    evidence_types: list[str] = field(default_factory=list)
    sequence: int = 0      # ordering within the template (lower = earlier)


@dataclass
class WorkProgramTemplate:
    """A reusable, versionable template for a class of control objectives.

    Multiple engagements share the same template definition while maintaining
    independent WorkProgram instances (instantiated via ``instantiate``).
    """
    template_id: UUID = field(default_factory=uuid4)
    name: str = ""
    control_objective_type: str = ""   # e.g. "access_control"
    version: str = "1.0"
    framework_refs: list[str] = field(default_factory=list)  # e.g. ["NIS2-21a"]
    steps: list[TemplateStep] = field(default_factory=list)
    metadata: dict[str, Any] = field(default_factory=dict)

    def instantiate(
        self,
        control_id: str,
        risk_weight: float = 1.0,
        article_ref: Optional[str] = None,
        hours_multiplier: float = 1.0,
    ) -> list[WorkItem]:
        """Instantiate template steps into concrete WorkItems.

        Parameters
        ----------
        control_id:
            The specific control being audited (e.g. "NIS2-21a").
        risk_weight:
            Risk weight for the control (0-5); copied to each WorkItem.
        article_ref:
            Optional framework article reference; falls back to
            self.framework_refs[0] if omitted.
        hours_multiplier:
            Scale factor applied to each step's base_hours.  Use values > 1
            for complex or high-risk engagements.  Minimum result is 1.0 h.

        Returns
        -------
        List of WorkItems in template step sequence order.
        """
        effective_ref = article_ref or (
            self.framework_refs[0] if self.framework_refs else None
        )
        sorted_steps = sorted(self.steps, key=lambda s: (s.sequence, s.step_key))
        items: list[WorkItem] = []
        for step in sorted_steps:
            estimated_hours = max(1.0, step.base_hours * hours_multiplier)
            items.append(
                WorkItem(
                    item_id=uuid4(),
                    control_id=control_id,
                    title=f"{step.title} — {control_id}",
                    description=step.description,
                    estimated_hours=estimated_hours,
                    risk_weight=risk_weight,
                    evidence_required=list(step.evidence_types),
                    article_ref=effective_ref,
                )
            )
        return items


# ---------------------------------------------------------------------------
# Built-in NIS 2 template library
# ---------------------------------------------------------------------------

def _make_template(
    name: str,
    obj_type: str,
    framework_refs: list[str],
    steps: list[tuple[str, str, str, float, list[str]]],
) -> WorkProgramTemplate:
    """Helper: construct a WorkProgramTemplate from a compact step list.

    Each step tuple is (step_key, title, description, base_hours, evidence_types).
    """
    template_steps = [
        TemplateStep(
            step_key=key,
            title=title,
            description=desc,
            base_hours=hrs,
            evidence_types=evidence,
            sequence=i,
        )
        for i, (key, title, desc, hrs, evidence) in enumerate(steps)
    ]
    return WorkProgramTemplate(
        name=name,
        control_objective_type=obj_type,
        framework_refs=framework_refs,
        steps=template_steps,
    )


TEMPLATE_ACCESS_CONTROL = _make_template(
    name="Access Control Audit",
    obj_type="access_control",
    framework_refs=["NIS2-21b"],
    steps=[
        (
            "policy_review",
            "Policy and Procedure Review",
            "Review access control policies, procedures, and standards for completeness.",
            2.0,
            ["policy", "documentation"],
        ),
        (
            "privilege_inventory",
            "Privileged Account Inventory",
            "Enumerate all privileged accounts and verify least-privilege assignments.",
            4.0,
            ["system_report", "hr_data"],
        ),
        (
            "access_log_analysis",
            "Access Log Analysis",
            "Analyse authentication and authorisation logs for anomalies.",
            6.0,
            ["log_data", "siem_export"],
        ),
        (
            "recertification_check",
            "Access Recertification Verification",
            "Verify periodic access reviews are performed and documented.",
            2.0,
            ["documentation", "approval_record"],
        ),
    ],
)

TEMPLATE_INCIDENT_RESPONSE = _make_template(
    name="Incident Response Audit",
    obj_type="incident_response",
    framework_refs=["NIS2-21a"],
    steps=[
        (
            "ir_plan_review",
            "IR Plan Documentation Review",
            "Review incident response plan for completeness and currency.",
            2.0,
            ["documentation"],
        ),
        (
            "tabletop_evidence",
            "Tabletop Exercise Evidence Review",
            "Review evidence of recent incident response tabletop exercises.",
            3.0,
            ["documentation", "meeting_minutes"],
        ),
        (
            "incident_log_review",
            "Incident Log and Metrics Review",
            "Analyse historical incident records for detection and response times.",
            4.0,
            ["log_data", "metrics_report"],
        ),
        (
            "notification_compliance",
            "Regulatory Notification Compliance",
            "Verify 72-hour notification obligations are met for significant incidents.",
            2.0,
            ["documentation", "correspondence"],
        ),
    ],
)

TEMPLATE_VULNERABILITY_MANAGEMENT = _make_template(
    name="Vulnerability Management Audit",
    obj_type="vulnerability_management",
    framework_refs=["NIS2-21e"],
    steps=[
        (
            "scan_frequency_check",
            "Scan Schedule Verification",
            "Confirm vulnerability scans run at required frequencies.",
            2.0,
            ["system_report", "schedule"],
        ),
        (
            "remediation_sla_review",
            "Remediation SLA Review",
            "Verify critical/high vulnerabilities are remediated within SLA.",
            4.0,
            ["ticket_data", "metrics_report"],
        ),
        (
            "patch_process_review",
            "Patch Management Process Review",
            "Review patch management procedures and exception handling.",
            2.0,
            ["documentation", "policy"],
        ),
        (
            "pentest_evidence",
            "Penetration Test Evidence Review",
            "Review most recent penetration test results and remediation tracking.",
            3.0,
            ["pentest_report", "documentation"],
        ),
    ],
)

TEMPLATE_SUPPLY_CHAIN = _make_template(
    name="Supply Chain Security Audit",
    obj_type="supply_chain",
    framework_refs=["NIS2-21d"],
    steps=[
        (
            "vendor_inventory",
            "Vendor Inventory and Criticality Assessment",
            "Enumerate critical third-party suppliers and their security classifications.",
            3.0,
            ["documentation", "contract"],
        ),
        (
            "due_diligence_review",
            "Supplier Due Diligence Review",
            "Review security assessments, questionnaires, or certifications for key vendors.",
            5.0,
            ["assessment_report", "documentation"],
        ),
        (
            "contract_clause_check",
            "Security Contract Clause Verification",
            "Confirm supply-chain security obligations are embedded in contracts.",
            2.0,
            ["contract", "documentation"],
        ),
    ],
)

TEMPLATE_CRYPTOGRAPHY = _make_template(
    name="Cryptography and Key Management Audit",
    obj_type="cryptography",
    framework_refs=["NIS2-21c"],
    steps=[
        (
            "crypto_policy_review",
            "Cryptography Policy Review",
            "Review cryptography standards, approved algorithms, and key management policy.",
            2.0,
            ["policy", "documentation"],
        ),
        (
            "key_inventory",
            "Key Inventory and Lifecycle Check",
            "Audit cryptographic key inventory, rotation schedules, and expiry tracking.",
            4.0,
            ["system_report", "documentation"],
        ),
        (
            "tls_config_review",
            "TLS Configuration Assessment",
            "Verify TLS versions, cipher suites, and certificate validity.",
            3.0,
            ["technical_scan", "system_report"],
        ),
    ],
)


# ---------------------------------------------------------------------------
# Template registry
# ---------------------------------------------------------------------------

# Maps control_objective_type -> WorkProgramTemplate
TEMPLATE_REGISTRY: dict[str, WorkProgramTemplate] = {
    t.control_objective_type: t
    for t in [
        TEMPLATE_ACCESS_CONTROL,
        TEMPLATE_INCIDENT_RESPONSE,
        TEMPLATE_VULNERABILITY_MANAGEMENT,
        TEMPLATE_SUPPLY_CHAIN,
        TEMPLATE_CRYPTOGRAPHY,
    ]
}


def get_template(control_objective_type: str) -> Optional[WorkProgramTemplate]:
    """Look up a built-in template by control objective type.

    Returns None if no matching template is registered.
    """
    return TEMPLATE_REGISTRY.get(control_objective_type)


def list_templates() -> list[WorkProgramTemplate]:
    """Return all registered templates in deterministic (alphabetical) order."""
    return sorted(TEMPLATE_REGISTRY.values(), key=lambda t: t.control_objective_type)
