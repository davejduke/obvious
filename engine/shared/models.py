"""AIAUDITOR Engine — Shared data models and enumerations."""
from __future__ import annotations

from enum import Enum
from typing import Any, Optional
from uuid import UUID
from datetime import date, datetime

from pydantic import BaseModel, Field


class Persona(str, Enum):
    """7-persona RBAC model."""
    INTERNAL_AUDITOR = "internal_auditor"
    CAE = "cae"
    AUDIT_COMMITTEE = "audit_committee"
    AUDITEE_CISO = "auditee_ciso"
    COSOURCED_PROVIDER = "cosourced_provider"
    PTWG_MEMBER = "ptwg_member"
    BETA_TESTER = "beta_tester"


class EngagementStatus(str, Enum):
    PLANNING = "planning"
    FIELDWORK = "fieldwork"
    REVIEW = "review"
    REPORTING = "reporting"
    COMPLETED = "completed"
    CANCELLED = "cancelled"


class FindingSeverity(str, Enum):
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFORMATIONAL = "informational"


class RiskRating(str, Enum):
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFORMATIONAL = "informational"


class EvidenceSourceType(str, Enum):
    MANUAL_UPLOAD = "manual_upload"
    API_INTEGRATION = "api_integration"
    AUTOMATED_SCAN = "automated_scan"
    SCREENSHOT = "screenshot"
    LOG_EXPORT = "log_export"
    CONFIGURATION_EXPORT = "configuration_export"


class NIS2Article(str, Enum):
    """NIS 2 Article 21 sub-clauses."""
    A = "21a"  # Risk analysis
    B = "21b"  # Incident handling
    C = "21c"  # Business continuity
    D = "21d"  # Supply chain
    E = "21e"  # Development security
    F = "21f"  # Effectiveness assessment
    G = "21g"  # Cyber hygiene
    H = "21h"  # Cryptography
    I = "21i"  # HR security & access control
    J = "21j"  # MFA & secure communications


class ControlModel(BaseModel):
    id: UUID
    framework_id: UUID
    org_id: UUID
    parent_id: Optional[UUID] = None
    control_id: str
    title: str
    description: Optional[str] = None
    domain: Optional[str] = None
    article_ref: Optional[str] = None
    risk_weight: float = Field(ge=0.0, le=5.0)
    tags: list[str] = []


class EvidenceModel(BaseModel):
    id: UUID
    org_id: UUID
    engagement_id: UUID
    control_id: UUID
    title: str
    source_type: EvidenceSourceType
    content_text: Optional[str] = None
    collection_date: datetime
    period_start: Optional[date] = None
    period_end: Optional[date] = None
    metadata: dict[str, Any] = {}


class QualityScore(BaseModel):
    evidence_id: UUID
    completeness_score: float = Field(ge=0.0, le=1.0)
    accuracy_score: float = Field(ge=0.0, le=1.0)
    timeliness_score: float = Field(ge=0.0, le=1.0)
    relevance_score: float = Field(ge=0.0, le=1.0)
    aggregate_score: float = Field(ge=0.0, le=1.0)
    deficiencies: list[str] = []


class FindingModel(BaseModel):
    id: UUID
    org_id: UUID
    engagement_id: UUID
    control_id: UUID
    finding_ref: str
    title: str
    description: str
    root_cause: Optional[str] = None
    impact: Optional[str] = None
    severity: FindingSeverity
    status: str = "open"
    evidence_ids: list[UUID] = []
    tags: list[str] = []

