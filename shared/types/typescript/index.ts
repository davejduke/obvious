// AIAUDITOR TypeScript shared types
// API contract types for the frontend

// ============================================================
// RBAC / Personas
// ============================================================
export type Persona =
  | "internal_auditor"
  | "cae"
  | "audit_committee"
  | "auditee_ciso"
  | "cosourced_provider"
  | "ptwg_member"
  | "beta_tester";

// ============================================================
// Core domain types
// ============================================================
export interface Organization {
  id: string;
  name: string;
  slug: string;
  tier: "standard" | "professional" | "enterprise";
  industry?: string;
  country_code?: string;
  settings: Record<string, unknown>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface User {
  id: string;
  org_id: string;
  email: string;
  display_name: string;
  is_active: boolean;
  last_login_at?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  org_id: string;
  name: string;
  slug: string;
  description?: string;
  is_system: boolean;
  created_at: string;
}

export type EngagementStatus =
  | "planning"
  | "fieldwork"
  | "review"
  | "reporting"
  | "completed"
  | "cancelled";

export type FindingSeverity =
  | "critical"
  | "high"
  | "medium"
  | "low"
  | "informational";

export type RiskRating = FindingSeverity;

export interface ControlFramework {
  id: string;
  org_id: string;
  name: string;
  version: string;
  authority: string;
  description?: string;
  is_published: boolean;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Control {
  id: string;
  framework_id: string;
  org_id: string;
  parent_id?: string;
  control_id: string;
  title: string;
  description?: string;
  objective?: string;
  category?: string;
  domain?: string;
  article_ref?: string;
  risk_weight: number;
  tags: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Engagement {
  id: string;
  org_id: string;
  framework_id: string;
  name: string;
  description?: string;
  status: EngagementStatus;
  scope_json: Record<string, unknown>;
  lead_auditor_id?: string;
  target_start_date?: string;
  target_end_date?: string;
  overall_score?: number;
  risk_rating?: RiskRating;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export type EvidenceSourceType =
  | "manual_upload"
  | "api_integration"
  | "automated_scan"
  | "screenshot"
  | "log_export"
  | "configuration_export";

export interface Evidence {
  id: string;
  org_id: string;
  engagement_id: string;
  control_id: string;
  uploaded_by_id?: string;
  title: string;
  description?: string;
  source_type: EvidenceSourceType;
  collection_date: string;
  period_start?: string;
  period_end?: string;
  status: "pending_review" | "accepted" | "rejected" | "archived";
  is_sufficient?: boolean;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Finding {
  id: string;
  org_id: string;
  engagement_id: string;
  control_id: string;
  finding_ref: string;
  title: string;
  description: string;
  root_cause?: string;
  impact?: string;
  severity: FindingSeverity;
  status: "open" | "in_remediation" | "remediated" | "accepted_risk" | "false_positive" | "closed";
  due_date?: string;
  evidence_ids: string[];
  tags: string[];
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Recommendation {
  id: string;
  finding_id: string;
  org_id: string;
  title: string;
  description: string;
  priority: "immediate" | "short_term" | "medium_term" | "long_term";
  effort?: "low" | "medium" | "high";
  status: "proposed" | "accepted" | "rejected" | "in_progress" | "implemented";
  created_at: string;
  updated_at: string;
}

// ============================================================
// API response wrappers
// ============================================================
export interface ApiResponse<T> {
  data: T;
  meta?: {
    total?: number;
    page?: number;
    per_page?: number;
  };
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    total: number;
    page: number;
    per_page: number;
    total_pages: number;
  };
}

// ============================================================
// NIS 2 specific
// ============================================================
export type NIS2Article = "21a" | "21b" | "21c" | "21d" | "21e" | "21f" | "21g" | "21h" | "21i" | "21j";

export interface NIS2ComplianceScore {
  engagement_id: string;
  overall_score: number;
  by_article: Record<NIS2Article, {
    score: number;
    controls_tested: number;
    controls_passed: number;
    findings_count: number;
    critical_findings: number;
  }>;
  computed_at: string;
}

