// Mock data for all screens - uses shared types
import type {
  Engagement, Finding, Evidence, Control, NIS2ComplianceScore, Persona
} from '@shared/index';

export const mockEngagements: Engagement[] = [
  {
    id: 'eng-001', org_id: 'org-001', framework_id: 'nis2-001',
    name: 'NIS 2 Article 21 Audit 2024', description: 'Annual NIS 2 compliance audit',
    status: 'fieldwork', scope_json: { articles: ['21a','21b','21c','21d','21e'] },
    lead_auditor_id: 'usr-001', target_start_date: '2024-01-15',
    target_end_date: '2024-03-31', overall_score: 72, risk_rating: 'medium',
    metadata: {}, created_at: '2024-01-10T00:00:00Z', updated_at: '2024-01-15T00:00:00Z',
  },
  {
    id: 'eng-002', org_id: 'org-001', framework_id: 'nis2-001',
    name: 'Q2 Continuous Monitoring', description: 'Quarterly control monitoring',
    status: 'planning', scope_json: { articles: ['21f','21g','21h'] },
    lead_auditor_id: 'usr-001', target_start_date: '2024-04-01',
    target_end_date: '2024-06-30', overall_score: undefined, risk_rating: undefined,
    metadata: {}, created_at: '2024-03-01T00:00:00Z', updated_at: '2024-03-01T00:00:00Z',
  },
];

export const mockFindings: Finding[] = [
  {
    id: 'fnd-001', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-001',
    finding_ref: 'F-2024-001', title: 'Insufficient MFA coverage on privileged accounts',
    description: 'Only 43% of privileged accounts have MFA enabled, below the 95% threshold.',
    root_cause: 'No enforcement policy in Active Directory for service accounts.',
    impact: 'High risk of account compromise and unauthorized access.',
    severity: 'critical', status: 'open', due_date: '2024-02-15',
    evidence_ids: ['ev-001', 'ev-002'], tags: ['iam', 'mfa', '21b'],
    metadata: {}, created_at: '2024-01-20T00:00:00Z', updated_at: '2024-01-20T00:00:00Z',
  },
  {
    id: 'fnd-002', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-002',
    finding_ref: 'F-2024-002', title: 'Patch management cycle exceeds 30-day SLA',
    description: 'Critical patches averaged 47 days to deploy in 2023.',
    root_cause: 'Manual patching process without automated orchestration.',
    impact: 'Exposure window extends vulnerability exploitation risk.',
    severity: 'high', status: 'in_remediation', due_date: '2024-03-01',
    evidence_ids: ['ev-003'], tags: ['patching', 'vuln-mgmt', '21c'],
    metadata: {}, created_at: '2024-01-21T00:00:00Z', updated_at: '2024-01-25T00:00:00Z',
  },
  {
    id: 'fnd-003', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-003',
    finding_ref: 'F-2024-003', title: 'Incident response playbooks outdated',
    description: 'IR playbooks last reviewed 18 months ago, not aligned with current environment.',
    root_cause: 'No scheduled review cadence for IR documentation.',
    impact: 'Delayed or ineffective response during security incidents.',
    severity: 'medium', status: 'open', due_date: '2024-04-01',
    evidence_ids: ['ev-004'], tags: ['incident-response', '21d'],
    metadata: {}, created_at: '2024-01-22T00:00:00Z', updated_at: '2024-01-22T00:00:00Z',
  },
  {
    id: 'fnd-004', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-004',
    finding_ref: 'F-2024-004', title: 'Supply chain risk assessments incomplete',
    description: '12 of 34 critical suppliers lack completed risk assessments.',
    root_cause: 'Vendor onboarding checklist does not include security assessment gate.',
    impact: 'Unquantified supply chain risk exposure.',
    severity: 'high', status: 'open', due_date: '2024-03-15',
    evidence_ids: ['ev-005'], tags: ['supply-chain', '21a'],
    metadata: {}, created_at: '2024-01-23T00:00:00Z', updated_at: '2024-01-23T00:00:00Z',
  },
  {
    id: 'fnd-005', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-005',
    finding_ref: 'F-2024-005', title: 'Network segmentation gaps in OT environment',
    description: 'OT/IT network boundary lacks firewall enforcement at 3 junction points.',
    root_cause: 'Legacy network architecture not updated post-merger.',
    impact: 'Lateral movement risk from IT to OT systems.',
    severity: 'critical', status: 'open', due_date: '2024-02-28',
    evidence_ids: ['ev-006', 'ev-007'], tags: ['network', 'ot', '21e'],
    metadata: {}, created_at: '2024-01-24T00:00:00Z', updated_at: '2024-01-24T00:00:00Z',
  },
];

export const mockEvidence: Evidence[] = [
  {
    id: 'ev-001', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-001',
    uploaded_by_id: 'usr-002', title: 'AD MFA Enrollment Report - Jan 2024',
    description: 'Export of Active Directory MFA enrollment status for all users.',
    source_type: 'api_integration', collection_date: '2024-01-18',
    status: 'accepted', is_sufficient: true,
    metadata: { file_size: '245KB', format: 'CSV' },
    created_at: '2024-01-18T00:00:00Z', updated_at: '2024-01-19T00:00:00Z',
  },
  {
    id: 'ev-002', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-001',
    uploaded_by_id: 'usr-003', title: 'MFA Policy Screenshots',
    description: 'Screenshots of current MFA policy configuration.',
    source_type: 'screenshot', collection_date: '2024-01-19',
    status: 'pending_review', is_sufficient: false,
    metadata: { file_size: '1.2MB', format: 'PNG' },
    created_at: '2024-01-19T00:00:00Z', updated_at: '2024-01-19T00:00:00Z',
  },
  {
    id: 'ev-003', org_id: 'org-001', engagement_id: 'eng-001', control_id: 'ctrl-002',
    uploaded_by_id: 'usr-002', title: 'Patch Management Log Export - Q4 2023',
    description: 'SIEM log export showing patch deployment timelines.',
    source_type: 'log_export', collection_date: '2024-01-20',
    status: 'accepted', is_sufficient: true,
    metadata: { file_size: '4.8MB', format: 'JSON' },
    created_at: '2024-01-20T00:00:00Z', updated_at: '2024-01-21T00:00:00Z',
  },
];

export const mockNIS2Score: NIS2ComplianceScore = {
  engagement_id: 'eng-001',
  overall_score: 72,
  by_article: {
    '21a': { score: 68, controls_tested: 8, controls_passed: 5, findings_count: 2, critical_findings: 0 },
    '21b': { score: 45, controls_tested: 12, controls_passed: 5, findings_count: 4, critical_findings: 2 },
    '21c': { score: 61, controls_tested: 6, controls_passed: 4, findings_count: 2, critical_findings: 0 },
    '21d': { score: 78, controls_tested: 5, controls_passed: 4, findings_count: 1, critical_findings: 0 },
    '21e': { score: 52, controls_tested: 9, controls_passed: 5, findings_count: 3, critical_findings: 1 },
    '21f': { score: 88, controls_tested: 4, controls_passed: 4, findings_count: 0, critical_findings: 0 },
    '21g': { score: 91, controls_tested: 3, controls_passed: 3, findings_count: 0, critical_findings: 0 },
    '21h': { score: 75, controls_tested: 7, controls_passed: 5, findings_count: 1, critical_findings: 0 },
    '21i': { score: 82, controls_tested: 5, controls_passed: 4, findings_count: 1, critical_findings: 0 },
    '21j': { score: 70, controls_tested: 6, controls_passed: 4, findings_count: 2, critical_findings: 0 },
  },
  computed_at: '2024-01-25T10:00:00Z',
};

export const mockDAGNodes = [
  { id: 'start', label: 'Audit Initiation', type: 'start', status: 'completed', x: 400, y: 40 },
  { id: 'scope', label: 'Scope Analysis\n(NIS 2 Art. 21)', type: 'process', status: 'completed', x: 400, y: 120 },
  { id: 'evidence-collect', label: 'Evidence Collection', type: 'process', status: 'completed', x: 200, y: 220 },
  { id: 'control-map', label: 'Control Mapping', type: 'process', status: 'completed', x: 600, y: 220 },
  { id: 'risk-assess', label: 'Risk Assessment\nEngine', type: 'decision', status: 'in_progress', x: 400, y: 320 },
  { id: 'finding-gen', label: 'Finding Generation', type: 'process', status: 'in_progress', x: 200, y: 420 },
  { id: 'scoring', label: 'Compliance Scoring', type: 'process', status: 'pending', x: 600, y: 420 },
  { id: 'report-draft', label: 'Report Drafting', type: 'process', status: 'pending', x: 400, y: 520 },
  { id: 'end', label: 'Audit Complete', type: 'end', status: 'pending', x: 400, y: 600 },
];

export const mockDAGEdges = [
  { source: 'start', target: 'scope' },
  { source: 'scope', target: 'evidence-collect' },
  { source: 'scope', target: 'control-map' },
  { source: 'evidence-collect', target: 'risk-assess' },
  { source: 'control-map', target: 'risk-assess' },
  { source: 'risk-assess', target: 'finding-gen' },
  { source: 'risk-assess', target: 'scoring' },
  { source: 'finding-gen', target: 'report-draft' },
  { source: 'scoring', target: 'report-draft' },
  { source: 'report-draft', target: 'end' },
];

export const mockControls: Control[] = [
  {
    id: 'ctrl-001', framework_id: 'nis2-001', org_id: 'org-001',
    control_id: 'NIS2-21b-1', title: 'Multi-Factor Authentication',
    description: 'Implement MFA for all privileged and remote access.',
    objective: 'Prevent unauthorized access via credential compromise.',
    category: 'Access Control', domain: 'Identity & Access Management',
    article_ref: '21b', risk_weight: 0.85, tags: ['iam', 'mfa'],
    is_active: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'ctrl-002', framework_id: 'nis2-001', org_id: 'org-001',
    control_id: 'NIS2-21c-1', title: 'Vulnerability Management',
    description: 'Systematic identification, classification, and remediation of vulnerabilities.',
    objective: 'Reduce exploitable attack surface through timely patching.',
    category: 'Vulnerability Management', domain: 'Security Operations',
    article_ref: '21c', risk_weight: 0.80, tags: ['patching', 'vuln-mgmt'],
    is_active: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'ctrl-003', framework_id: 'nis2-001', org_id: 'org-001',
    control_id: 'NIS2-21d-1', title: 'Incident Response Plan',
    description: 'Documented and tested incident response procedures.',
    objective: 'Enable rapid, effective response to security incidents.',
    category: 'Incident Response', domain: 'Security Operations',
    article_ref: '21d', risk_weight: 0.75, tags: ['incident-response'],
    is_active: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z',
  },
];

export const personas: Array<{ id: Persona; label: string; description: string }> = [
  { id: 'internal_auditor', label: 'Internal Auditor', description: 'Fieldwork and testing focus' },
  { id: 'cae', label: 'CAE', description: 'Portfolio oversight and metrics' },
  { id: 'audit_committee', label: 'Audit Committee', description: 'Board-level risk summary' },
  { id: 'auditee_ciso', label: 'Auditee / CISO', description: 'Remediation and evidence submission' },
  { id: 'cosourced_provider', label: 'Co-sourced Provider', description: 'External audit support' },
  { id: 'ptwg_member', label: 'PTWG Member', description: 'Policy and framework guidance' },
  { id: 'beta_tester', label: 'Beta Tester', description: 'Platform testing access' },
];
