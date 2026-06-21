// Portal mock data — Auditee Portal §4.4
import type {
  EvidenceRequest,
  ManagementResponse,
  PortalNotification,
  Finding,
  Control,
  Engagement,
} from '@shared/index';

// --- Engagement context visible to auditees ---------------------------------
export const portalEngagement: Engagement = {
  id: 'eng-001',
  org_id: 'org-001',
  framework_id: 'nis2-001',
  name: 'NIS 2 Article 21 Audit 2024',
  description:
    'Annual NIS 2 compliance audit covering all Article 21 security measures. ' +
    'Your organisation is the auditee for this engagement.',
  status: 'fieldwork',
  scope_json: { articles: ['21a', '21b', '21c', '21d', '21e'] },
  lead_auditor_id: 'usr-001',
  target_start_date: '2024-01-15',
  target_end_date: '2024-03-31',
  overall_score: 72,
  risk_rating: 'medium',
  metadata: {},
  created_at: '2024-01-10T00:00:00Z',
  updated_at: '2024-01-15T00:00:00Z',
};

// --- Controls visible to auditees (context-before-finding) ------------------
export const portalControls: Control[] = [
  {
    id: 'ctrl-001',
    framework_id: 'nis2-001',
    org_id: 'org-001',
    control_id: 'NIS2-21b-1',
    title: 'Multi-Factor Authentication',
    description:
      'Organisations must implement multi-factor authentication for all privileged ' +
      'accounts and remote access points to prevent unauthorised access.',
    objective: 'Prevent credential-based account compromise.',
    category: 'Access Control',
    domain: 'Identity & Access Management',
    article_ref: '21b',
    risk_weight: 0.85,
    tags: ['iam', 'mfa'],
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'ctrl-002',
    framework_id: 'nis2-001',
    org_id: 'org-001',
    control_id: 'NIS2-21c-1',
    title: 'Vulnerability Management',
    description:
      'Systematic identification, classification, and remediation of vulnerabilities ' +
      'within defined SLA windows (critical: 24h, high: 7d, medium: 30d).',
    objective: 'Reduce exploitable attack surface through timely patching.',
    category: 'Vulnerability Management',
    domain: 'Security Operations',
    article_ref: '21c',
    risk_weight: 0.80,
    tags: ['patching', 'vuln-mgmt'],
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'ctrl-003',
    framework_id: 'nis2-001',
    org_id: 'org-001',
    control_id: 'NIS2-21d-1',
    title: 'Incident Response Plan',
    description:
      'Documented, tested incident response procedures aligned to current operating ' +
      'environment with at least annual review cadence.',
    objective: 'Enable rapid, effective response to security incidents.',
    category: 'Incident Response',
    domain: 'Security Operations',
    article_ref: '21d',
    risk_weight: 0.75,
    tags: ['incident-response'],
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'ctrl-004',
    framework_id: 'nis2-001',
    org_id: 'org-001',
    control_id: 'NIS2-21a-1',
    title: 'Supply Chain Risk Assessment',
    description:
      'All critical suppliers must complete a formal security risk assessment ' +
      'before onboarding and annually thereafter.',
    objective: 'Quantify and manage third-party supply chain risk.',
    category: 'Supply Chain Security',
    domain: 'Third-Party Risk',
    article_ref: '21a',
    risk_weight: 0.78,
    tags: ['supply-chain'],
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

// --- Evidence Requests -------------------------------------------------------
export const mockEvidenceRequests: EvidenceRequest[] = [
  {
    id: 'er-001',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-001',
    finding_id: 'fnd-001',
    requested_by_id: 'usr-001',
    requested_by_email: 'lead.auditor@auditfirm.com',
    assigned_to_id: 'usr-ciso-001',
    title: 'MFA Enrollment Report — All Privileged Accounts',
    description:
      'Please provide an up-to-date export of MFA enrollment status for all ' +
      'privileged accounts in Active Directory, including service accounts.',
    instructions:
      'Export from Azure AD Identity Protection portal. Filter by privileged ' +
      'role. Include columns: UPN, role, MFA status, last sign-in.',
    priority: 'urgent',
    due_date: '2024-02-10',
    status: 'pending',
    evidence_ids: [],
    metadata: {},
    created_at: '2024-01-25T09:00:00Z',
    updated_at: '2024-01-25T09:00:00Z',
  },
  {
    id: 'er-002',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-002',
    finding_id: 'fnd-002',
    requested_by_id: 'usr-001',
    requested_by_email: 'lead.auditor@auditfirm.com',
    assigned_to_id: 'usr-ciso-001',
    title: 'Patch Management Log — Q4 2023 (90-day window)',
    description:
      'Provide SIEM/ITSM log export showing patch deployment dates versus ' +
      'vulnerability disclosure dates for all critical and high-severity CVEs.',
    instructions:
      'Export from ServiceNow Change Management. Date range: Oct 1 to Dec 31 2023. ' +
      'CSV format preferred.',
    priority: 'high',
    due_date: '2024-02-15',
    status: 'in_progress',
    evidence_ids: ['ev-003'],
    metadata: {},
    created_at: '2024-01-24T14:00:00Z',
    updated_at: '2024-01-26T10:00:00Z',
  },
  {
    id: 'er-003',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-003',
    requested_by_id: 'usr-001',
    requested_by_email: 'lead.auditor@auditfirm.com',
    assigned_to_id: 'usr-ciso-001',
    title: 'Incident Response Plan — Current Version',
    description:
      'Provide the current incident response plan document, version history, ' +
      'and evidence of last tabletop exercise.',
    instructions:
      'PDF of IR Plan plus exercise report. Redact third-party vendor names if needed.',
    priority: 'medium',
    due_date: '2024-02-28',
    status: 'submitted',
    evidence_ids: ['ev-004'],
    metadata: {},
    created_at: '2024-01-23T11:00:00Z',
    updated_at: '2024-01-28T16:00:00Z',
  },
  {
    id: 'er-004',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-004',
    requested_by_id: 'usr-001',
    requested_by_email: 'lead.auditor@auditfirm.com',
    assigned_to_id: 'usr-ciso-001',
    title: 'Supplier Risk Assessment Register',
    description:
      'List of all critical suppliers with their risk assessment status, ' +
      'last assessment date, and overall risk rating.',
    instructions:
      'Excel/CSV export from GRC system. Include columns: Supplier, Tier, ' +
      'Risk Rating, Last Assessment Date, Next Review Date.',
    priority: 'high',
    due_date: '2024-03-01',
    status: 'pending',
    evidence_ids: [],
    metadata: {},
    created_at: '2024-01-22T09:00:00Z',
    updated_at: '2024-01-22T09:00:00Z',
  },
  {
    id: 'er-005',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-001',
    requested_by_id: 'usr-001',
    requested_by_email: 'lead.auditor@auditfirm.com',
    assigned_to_id: 'usr-ciso-001',
    title: 'MFA Policy Configuration Screenshot',
    description: 'Screenshot of the current MFA policy settings in Entra ID / Azure AD.',
    instructions:
      'Navigate to Entra ID > Security > Authentication Methods. Screenshot the policy overview.',
    priority: 'low',
    due_date: '2024-02-20',
    status: 'accepted',
    evidence_ids: ['ev-002'],
    metadata: {},
    created_at: '2024-01-20T09:00:00Z',
    updated_at: '2024-01-30T12:00:00Z',
  },
];

// --- Portal-facing findings (auditee view) -----------------------------------
export const portalFindings: Finding[] = [
  {
    id: 'fnd-001',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-001',
    finding_ref: 'F-2024-001',
    title: 'Insufficient MFA coverage on privileged accounts',
    description:
      'Only 43% of privileged accounts have MFA enabled, against the NIS 2 ' +
      'Article 21(b) requirement of 95% coverage. Service accounts are particularly exposed.',
    root_cause:
      'No enforcement policy in Active Directory for service accounts. ' +
      'MFA rollout was scoped to human accounts only during the 2022 initiative.',
    impact:
      'High risk of account compromise and lateral movement. A compromised ' +
      'service account could grant persistent access to critical systems.',
    severity: 'critical',
    status: 'open',
    due_date: '2024-02-15',
    evidence_ids: ['ev-001', 'ev-002'],
    tags: ['iam', 'mfa', '21b'],
    metadata: {},
    created_at: '2024-01-20T00:00:00Z',
    updated_at: '2024-01-20T00:00:00Z',
  },
  {
    id: 'fnd-002',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-002',
    finding_ref: 'F-2024-002',
    title: 'Patch management cycle exceeds 30-day SLA for critical CVEs',
    description:
      'Critical patches averaged 47 days to deploy in 2023, exceeding the ' +
      'NIS 2 Article 21(c) requirement of 30 days for critical vulnerabilities.',
    root_cause:
      'Manual patching process without automated orchestration. ' +
      'Change Advisory Board meeting cadence creates bottlenecks.',
    impact:
      'Extended vulnerability exposure window increases the probability of ' +
      'exploitation. Three CVEs patched beyond 60 days remain in the threat landscape.',
    severity: 'high',
    status: 'in_remediation',
    due_date: '2024-03-01',
    evidence_ids: ['ev-003'],
    tags: ['patching', 'vuln-mgmt', '21c'],
    metadata: {},
    created_at: '2024-01-21T00:00:00Z',
    updated_at: '2024-01-25T00:00:00Z',
  },
  {
    id: 'fnd-003',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-003',
    finding_ref: 'F-2024-003',
    title: 'Incident response playbooks outdated — last reviewed 18 months ago',
    description:
      'IR playbooks have not been reviewed since June 2022, predating the ' +
      'current cloud and OT environment. NIS 2 Article 21(d) requires annual review.',
    root_cause:
      'No scheduled review cadence defined for IR documentation. ' +
      'Ownership transferred post-reorganisation without review obligations.',
    impact:
      'Delayed or ineffective incident response. Playbooks reference decommissioned ' +
      'systems and legacy escalation paths.',
    severity: 'medium',
    status: 'open',
    due_date: '2024-04-01',
    evidence_ids: ['ev-004'],
    tags: ['incident-response', '21d'],
    metadata: {},
    created_at: '2024-01-22T00:00:00Z',
    updated_at: '2024-01-22T00:00:00Z',
  },
  {
    id: 'fnd-004',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-004',
    finding_ref: 'F-2024-004',
    title: 'Supply chain risk assessments incomplete for 12 critical suppliers',
    description:
      '12 of 34 critical suppliers lack a completed risk assessment under ' +
      'NIS 2 Article 21(a). Seven have never been assessed.',
    root_cause:
      'Vendor onboarding checklist does not gate on security assessment completion. ' +
      'Risk team capacity constraint prevented backlog clearance.',
    impact:
      'Unquantified supply chain risk. Two unassessed suppliers have privileged ' +
      'access to production network segments.',
    severity: 'high',
    status: 'open',
    due_date: '2024-03-15',
    evidence_ids: ['ev-005'],
    tags: ['supply-chain', '21a'],
    metadata: {},
    created_at: '2024-01-23T00:00:00Z',
    updated_at: '2024-01-23T00:00:00Z',
  },
];

// --- Management Responses ----------------------------------------------------
export const mockManagementResponses: ManagementResponse[] = [
  {
    id: 'mr-001',
    finding_id: 'fnd-002',
    org_id: 'org-001',
    responder_id: 'usr-ciso-001',
    responder_email: 'ciso@example.com',
    response_text:
      'Management acknowledges the finding. We are implementing automated patch ' +
      'orchestration via Microsoft SCCM plus Defender for Endpoint. An emergency CAB ' +
      'process has been introduced to reduce the critical patch cycle to 14 days.',
    action_plan:
      '1. Deploy SCCM automatic approval rules for critical patches by 2024-02-01.\n' +
      '2. Introduce weekly emergency CAB slot by 2024-01-31.\n' +
      '3. Conduct retrospective on Q4 2023 backlog by 2024-02-15.',
    target_remediation_date: '2024-03-01',
    status: 'submitted',
    submitted_at: '2024-01-28T11:30:00Z',
    created_at: '2024-01-27T10:00:00Z',
    updated_at: '2024-01-28T11:30:00Z',
  },
];

// --- Portal Notifications ----------------------------------------------------
export const mockPortalNotifications: PortalNotification[] = [
  {
    id: 'notif-001',
    type: 'evidence_request',
    title: 'New evidence request: MFA Enrollment Report',
    body: 'Lead auditor has requested MFA enrollment data. Due 10 Feb 2024.',
    reference_id: 'er-001',
    reference_type: 'evidence_request',
    is_read: false,
    created_at: '2024-01-25T09:00:00Z',
  },
  {
    id: 'notif-002',
    type: 'finding_assignment',
    title: 'Finding assigned: F-2024-001 — Insufficient MFA coverage',
    body: 'Critical finding F-2024-001 has been assigned to you for management response.',
    reference_id: 'fnd-001',
    reference_type: 'finding',
    is_read: false,
    created_at: '2024-01-25T09:05:00Z',
  },
  {
    id: 'notif-003',
    type: 'deadline_reminder',
    title: 'Deadline approaching: Patch Management Log request due in 3 days',
    body: 'Evidence request er-002 (Patch Management Log) is due 15 Feb 2024.',
    reference_id: 'er-002',
    reference_type: 'evidence_request',
    is_read: true,
    created_at: '2024-02-12T08:00:00Z',
  },
  {
    id: 'notif-004',
    type: 'response_acknowledged',
    title: 'Management response acknowledged: F-2024-002',
    body: 'The audit team has acknowledged your management response for F-2024-002.',
    reference_id: 'fnd-002',
    reference_type: 'finding',
    is_read: true,
    created_at: '2024-01-29T14:00:00Z',
  },
];

// --- Helpers -----------------------------------------------------------------
export function getControlForFinding(findingControlId: string): Control | undefined {
  return portalControls.find((c) => c.id === findingControlId);
}

export function getRequestsForFinding(findingId: string): EvidenceRequest[] {
  return mockEvidenceRequests.filter((r) => r.finding_id === findingId);
}

export function getManagementResponse(findingId: string): ManagementResponse | undefined {
  return mockManagementResponses.find((mr) => mr.finding_id === findingId);
}

export const PORTAL_PRIORITY_ORDER: Record<EvidenceRequest['priority'], number> = {
  urgent: 0,
  high: 1,
  medium: 2,
  low: 3,
};

export function sortRequestsByPriority(requests: EvidenceRequest[]): EvidenceRequest[] {
  return [...requests].sort(
    (a, b) => PORTAL_PRIORITY_ORDER[a.priority] - PORTAL_PRIORITY_ORDER[b.priority],
  );
}
