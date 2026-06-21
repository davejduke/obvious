// Mock data for all screens - uses shared types
import type {
  Engagement, Finding, Evidence, Control, NIS2ComplianceScore, Persona
} from '@shared/index';
import type {
  ReasoningEngineState, WhatIfQuery, WhatIfResult
} from '@/lib/reasoning-types';

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

// ---- Reasoning Engine mock data ----
export const mockReasoningState: ReasoningEngineState = {
  engagement_id: 'eng-001',
  engagement_name: 'NIS 2 Article 21 Audit 2024',
  overall_confidence: {
    score: 72,
    engagement_id: 'eng-001',
    computed_at: '2024-01-25T10:00:00Z',
    factors: [
      {
        name: 'scope',
        label: 'Scope Coverage',
        score: 85,
        weight: 0.25,
        contribution: 21.25,
        breakdown: [
          { label: 'Articles in scope', value: 100, detail: '10 / 10 articles mapped' },
          { label: 'Control mapping rate', value: 76, detail: '65 / 85 controls mapped' },
          { label: 'DAG completeness', value: 80, detail: '4 of 5 nodes completed' },
        ],
      },
      {
        name: 'risk',
        label: 'Risk Assessment',
        score: 68,
        weight: 0.30,
        contribution: 20.40,
        breakdown: [
          { label: 'Controls assessed', value: 76, detail: '65 / 85 controls assessed' },
          { label: 'Critical zone controls', value: 40, detail: '3 of 7 critical resolved' },
          { label: 'Residual risk score', value: 62, detail: 'Avg residual 9.4 / 25' },
        ],
      },
      {
        name: 'quality',
        label: 'Evidence Quality',
        score: 70,
        weight: 0.30,
        contribution: 21.00,
        breakdown: [
          { label: 'Evidence accepted rate', value: 78, detail: '42 / 54 items accepted' },
          { label: 'Quality gate pass rate', value: 60, detail: '3 / 5 gates passed' },
          { label: 'Sufficiency rate', value: 72, detail: '13 / 18 controls sufficient' },
        ],
      },
      {
        name: 'economy',
        label: 'Work Economy',
        score: 82,
        weight: 0.15,
        contribution: 12.30,
        breakdown: [
          { label: 'Budget utilisation', value: 80, detail: '320 / 400 hours used' },
          { label: 'Milestone on-track', value: 90, detail: '9 / 10 milestones on time' },
          { label: 'Resource allocation', value: 75, detail: '3 auditors assigned' },
        ],
      },
    ],
  },
  quality_gates: [
    {
      control_id: 'ctrl-001',
      control_title: 'Multi-Factor Authentication',
      article_ref: '21b',
      status: 'passed',
      score: 82,
      threshold: 70,
      evidence_count: 8,
      required_evidence: 6,
      block_reasons: [],
    },
    {
      control_id: 'ctrl-002',
      control_title: 'Vulnerability Management',
      article_ref: '21c',
      status: 'passed',
      score: 74,
      threshold: 70,
      evidence_count: 5,
      required_evidence: 5,
      block_reasons: [],
    },
    {
      control_id: 'ctrl-003',
      control_title: 'Incident Response Plan',
      article_ref: '21d',
      status: 'blocked',
      score: 48,
      threshold: 70,
      evidence_count: 2,
      required_evidence: 6,
      block_reasons: ['floor_not_met', 'insufficient_evidence'],
    },
    {
      control_id: 'ctrl-004',
      control_title: 'Supply Chain Risk Assessment',
      article_ref: '21a',
      status: 'blocked',
      score: 55,
      threshold: 70,
      evidence_count: 3,
      required_evidence: 5,
      block_reasons: ['floor_not_met'],
    },
    {
      control_id: 'ctrl-005',
      control_title: 'Network Segmentation',
      article_ref: '21e',
      status: 'passed',
      score: 77,
      threshold: 70,
      evidence_count: 7,
      required_evidence: 5,
      block_reasons: [],
    },
  ],
  evidence_sufficiency: [
    { control_id: 'ctrl-001', control_title: 'Multi-Factor Authentication',   article_ref: '21b', collected: 8,  required: 6,  sufficiency_pct: 100, status: 'sufficient' },
    { control_id: 'ctrl-002', control_title: 'Vulnerability Management',       article_ref: '21c', collected: 5,  required: 5,  sufficiency_pct: 100, status: 'sufficient' },
    { control_id: 'ctrl-003', control_title: 'Incident Response Plan',         article_ref: '21d', collected: 2,  required: 6,  sufficiency_pct: 33,  status: 'insufficient' },
    { control_id: 'ctrl-004', control_title: 'Supply Chain Risk Assessment',   article_ref: '21a', collected: 3,  required: 5,  sufficiency_pct: 60,  status: 'partial' },
    { control_id: 'ctrl-005', control_title: 'Network Segmentation',           article_ref: '21e', collected: 7,  required: 5,  sufficiency_pct: 100, status: 'sufficient' },
    { control_id: 'ctrl-006', control_title: 'Logging & Monitoring',           article_ref: '21f', collected: 4,  required: 5,  sufficiency_pct: 80,  status: 'partial' },
    { control_id: 'ctrl-007', control_title: 'Access Control Policy',          article_ref: '21b', collected: 6,  required: 4,  sufficiency_pct: 100, status: 'sufficient' },
    { control_id: 'ctrl-008', control_title: 'Business Continuity Plan',       article_ref: '21h', collected: 1,  required: 5,  sufficiency_pct: 20,  status: 'insufficient' },
  ],
  heat_map: {
    controls: [
      { control_id: 'ctrl-001', control_title: 'Multi-Factor Authentication',    impact: 4, likelihood: 2, zone: 'high',     article_ref: '21b' },
      { control_id: 'ctrl-002', control_title: 'Vulnerability Management',        impact: 4, likelihood: 3, zone: 'high',     article_ref: '21c' },
      { control_id: 'ctrl-003', control_title: 'Incident Response Plan',          impact: 3, likelihood: 4, zone: 'high',     article_ref: '21d' },
      { control_id: 'ctrl-004', control_title: 'Supply Chain Risk Assessment',    impact: 5, likelihood: 3, zone: 'critical', article_ref: '21a' },
      { control_id: 'ctrl-005', control_title: 'Network Segmentation',            impact: 5, likelihood: 4, zone: 'critical', article_ref: '21e' },
      { control_id: 'ctrl-006', control_title: 'Logging & Monitoring',            impact: 2, likelihood: 3, zone: 'medium',   article_ref: '21f' },
      { control_id: 'ctrl-007', control_title: 'Access Control Policy',           impact: 3, likelihood: 2, zone: 'medium',   article_ref: '21b' },
      { control_id: 'ctrl-008', control_title: 'Business Continuity Plan',        impact: 4, likelihood: 4, zone: 'critical', article_ref: '21h' },
    ],
    zone_summary: { critical: 3, high: 3, medium: 2, low: 0 },
  },
  dag_nodes: [
    { id: 'start',          label: 'Audit Initiation',              type: 'start',    status: 'completed',  x: 400, y: 40,  evidence_count: 2 },
    { id: 'scope',          label: 'Scope Analysis\n(NIS 2 Art. 21)', type: 'process', status: 'completed',  x: 400, y: 120, evidence_count: 10 },
    { id: 'evidence-collect', label: 'Evidence Collection',          type: 'process', status: 'completed',  x: 200, y: 220, evidence_count: 47, children_ids: ['risk-assess'] },
    { id: 'control-map',    label: 'Control Mapping',                type: 'process', status: 'completed',  x: 600, y: 220, evidence_count: 65, children_ids: ['risk-assess'] },
    { id: 'risk-assess',    label: 'Risk Assessment\nEngine',        type: 'decision', status: 'in_progress', x: 400, y: 320, evidence_count: 65 },
    { id: 'finding-gen',    label: 'Finding Generation',             type: 'process', status: 'in_progress', x: 200, y: 420, evidence_count: 5 },
    { id: 'scoring',        label: 'Compliance Scoring',             type: 'process', status: 'pending',    x: 600, y: 420, evidence_count: 0 },
    { id: 'report-draft',   label: 'Report Drafting',                type: 'process', status: 'pending',    x: 400, y: 520, evidence_count: 0 },
    { id: 'end',            label: 'Audit Complete',                 type: 'end',     status: 'pending',    x: 400, y: 600, evidence_count: 0 },
  ],
  dag_edges: [
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
  ],
};

/** Deterministic What-If simulation (no LLM). */
export function simulateWhatIf(query: WhatIfQuery): WhatIfResult {
  const state = mockReasoningState;
  const originalScore = state.overall_confidence.score;

  const suffItem = state.evidence_sufficiency.find(e => e.control_id === query.control_id);
  const gateItem = state.quality_gates.find(g => g.control_id === query.control_id);

  const multiplier = query.action === 'add_evidence' ? 1 : -1;
  const qualityWeight = query.evidence_quality_score / 100;
  const evidenceDelta = multiplier * query.count * qualityWeight;

  // Deterministic quality score impact: each high-quality evidence item shifts score
  const scoreDelta = Math.round(evidenceDelta * 3.5);
  const simulatedScore = Math.max(0, Math.min(100, originalScore + scoreDelta));

  const gateOriginal = gateItem?.score ?? 50;
  const gateSimulated = Math.max(0, Math.min(100, gateOriginal + Math.round(evidenceDelta * 5)));
  const threshold = gateItem?.threshold ?? 70;

  let gate_change: WhatIfResult['gate_change'] = 'no_change';
  if (gateOriginal < threshold && gateSimulated >= threshold) gate_change = 'now_passes';
  else if (gateOriginal >= threshold && gateSimulated < threshold) gate_change = 'now_blocks';

  const sufficiencyOriginal = suffItem?.sufficiency_pct ?? 50;
  const sufficiencySimulated = Math.max(0, Math.min(100,
    sufficiencyOriginal + multiplier * query.count * 12));

  const narrative = query.action === 'add_evidence'
    ? `Adding ${query.count} evidence item${query.count !== 1 ? 's' : ''} at ${query.evidence_quality_score}% quality to "${gateItem?.control_title ?? query.control_id}" increases overall confidence by ${Math.abs(scoreDelta)} point${Math.abs(scoreDelta) !== 1 ? 's' : ''} (${
        gate_change === 'now_passes' ? 'quality gate now PASSES' :
        gate_change === 'now_blocks' ? 'quality gate now BLOCKS' :
        'no gate change'
      }).`
    : `Removing ${query.count} evidence item${query.count !== 1 ? 's' : ''} from "${gateItem?.control_title ?? query.control_id}" decreases overall confidence by ${Math.abs(scoreDelta)} point${Math.abs(scoreDelta) !== 1 ? 's' : ''} (${
        gate_change === 'now_passes' ? 'quality gate now PASSES' :
        gate_change === 'now_blocks' ? 'quality gate now BLOCKS' :
        'no gate change'
      }).`;

  return {
    original_score: originalScore,
    simulated_score: simulatedScore,
    delta: scoreDelta,
    affected_factors: [
      { factor: 'Quality', original: gateOriginal, simulated: gateSimulated },
      { factor: 'Evidence Sufficiency', original: sufficiencyOriginal, simulated: sufficiencySimulated },
    ],
    gate_change,
    narrative,
  };
}

export const personas: Array<{ id: Persona; label: string; description: string }> = [
  { id: 'internal_auditor', label: 'Internal Auditor', description: 'Fieldwork and testing focus' },
  { id: 'cae', label: 'CAE', description: 'Portfolio oversight and metrics' },
  { id: 'audit_committee', label: 'Audit Committee', description: 'Board-level risk summary' },
  { id: 'auditee_ciso', label: 'Auditee / CISO', description: 'Remediation and evidence submission' },
  { id: 'cosourced_provider', label: 'Co-sourced Provider', description: 'External audit support' },
  { id: 'ptwg_member', label: 'PTWG Member', description: 'Policy and framework guidance' },
  { id: 'beta_tester', label: 'Beta Tester', description: 'Platform testing access' },
];

// ---------------------------------------------------------------------------
// Audit Planning Mock Data
// ---------------------------------------------------------------------------

export type RiskRating = 'critical' | 'high' | 'medium' | 'low';
export type PlanStatus = 'draft' | 'approved' | 'active' | 'archived';
export type CoverageLevel = 'none' | 'partial' | 'full';
export type EngPlanStatus = 'planned' | 'in_progress' | 'completed' | 'deferred';

export interface AuditableEntity {
  id: string;
  name: string;
  business_unit: string;
  risk_rating: RiskRating;
  priority: number;
  planned_year: number;
  control_domains: string[];
  notes?: string;
}

export interface StrategicPlan {
  id: string;
  org_id: string;
  name: string;
  description?: string;
  start_year: number;
  end_year: number;
  status: PlanStatus;
  version: number;
  entities: AuditableEntity[];
  approved_at?: string;
  approved_by?: string;
}

export interface PlannedEngagement {
  id: string;
  name: string;
  auditable_entity: string;
  assigned_team: string[];
  lead_auditor_id?: string;
  quarter: number;
  start_date: string;
  end_date: string;
  budget_days: number;
  status: EngPlanStatus;
}

export interface AnnualPlan {
  id: string;
  org_id: string;
  strategic_plan_id?: string;
  year: number;
  name: string;
  status: PlanStatus;
  version: number;
  engagements: PlannedEngagement[];
}

export interface AssuranceCell {
  business_unit: string;
  control_domain: string;
  coverage: CoverageLevel;
  notes?: string;
}

export interface AssuranceMap {
  id: string;
  org_id: string;
  year: number;
  name: string;
  business_units: string[];
  control_domains: string[];
  matrix: AssuranceCell[];
}

export interface Assignment {
  id: string;
  engagement_name: string;
  start_date: string;
  end_date: string;
  allocated_days: number;
  quarter: number;
}

export interface AuditorAllocation {
  auditor_id: string;
  auditor_name: string;
  auditor_email: string;
  role: string;
  available_days: number;
  assignments: Assignment[];
}

export interface ResourceCalendar {
  id: string;
  org_id: string;
  year: number;
  name: string;
  auditors: AuditorAllocation[];
}

// ---- Strategic Plan ----
export const mockStrategicPlan: StrategicPlan = {
  id: 'sp-001', org_id: 'org-001',
  name: '3-Year Cyber Assurance Plan 2025-2027',
  description: 'Risk-based rolling plan covering NIS 2 Article 21 obligations, supply chain, and operational resilience.',
  start_year: 2025, end_year: 2027,
  status: 'active', version: 2,
  approved_at: '2024-12-10T09:00:00Z',
  approved_by: 'cae@example.com',
  entities: [
    { id: 'ae-001', name: 'Identity & Access Management', business_unit: 'IT Security', risk_rating: 'critical', priority: 1, planned_year: 2025, control_domains: ['IAM', 'Privileged Access'] },
    { id: 'ae-002', name: 'Supply Chain Risk', business_unit: 'Procurement', risk_rating: 'critical', priority: 2, planned_year: 2025, control_domains: ['Vendor Management', 'Third-Party Risk'] },
    { id: 'ae-003', name: 'Incident Response', business_unit: 'Security Operations', risk_rating: 'high', priority: 3, planned_year: 2025, control_domains: ['Incident Management', 'BCDR'] },
    { id: 'ae-004', name: 'Patch Management', business_unit: 'IT Operations', risk_rating: 'high', priority: 4, planned_year: 2025, control_domains: ['Vulnerability Management'] },
    { id: 'ae-005', name: 'Network Segmentation', business_unit: 'Network Ops', risk_rating: 'high', priority: 5, planned_year: 2026, control_domains: ['Network Security', 'OT/IT Boundary'] },
    { id: 'ae-006', name: 'Data Classification', business_unit: 'Data Governance', risk_rating: 'medium', priority: 6, planned_year: 2026, control_domains: ['Data Protection', 'Privacy'] },
    { id: 'ae-007', name: 'Cloud Security Posture', business_unit: 'Cloud Ops', risk_rating: 'high', priority: 7, planned_year: 2026, control_domains: ['Cloud Security', 'Configuration Management'] },
    { id: 'ae-008', name: 'HR Security', business_unit: 'People', risk_rating: 'medium', priority: 8, planned_year: 2027, control_domains: ['Personnel Security', 'Awareness'] },
    { id: 'ae-009', name: 'Physical Security', business_unit: 'Facilities', risk_rating: 'medium', priority: 9, planned_year: 2027, control_domains: ['Physical Access', 'Environmental'] },
    { id: 'ae-010', name: 'Cryptography Controls', business_unit: 'IT Security', risk_rating: 'medium', priority: 10, planned_year: 2027, control_domains: ['Cryptography', 'Key Management'] },
  ],
};

// ---- Annual Plan ----
export const mockAnnualPlan: AnnualPlan = {
  id: 'ap-001', org_id: 'org-001', strategic_plan_id: 'sp-001',
  year: 2025, name: 'Annual Audit Plan 2025',
  status: 'active', version: 3,
  engagements: [
    { id: 'pe-001', name: 'IAM Audit Q1', auditable_entity: 'Identity & Access Management', assigned_team: ['Alice Smith', 'Bob Lee'], lead_auditor_id: 'usr-001', quarter: 1, start_date: '2025-01-13', end_date: '2025-02-07', budget_days: 20, status: 'completed' },
    { id: 'pe-002', name: 'Supply Chain Risk Review', auditable_entity: 'Supply Chain Risk', assigned_team: ['Alice Smith', 'Carol White'], lead_auditor_id: 'usr-001', quarter: 1, start_date: '2025-02-10', end_date: '2025-03-07', budget_days: 18, status: 'completed' },
    { id: 'pe-003', name: 'Incident Response Readiness', auditable_entity: 'Incident Response', assigned_team: ['Bob Lee', 'Dan Green'], lead_auditor_id: 'usr-002', quarter: 2, start_date: '2025-04-07', end_date: '2025-05-02', budget_days: 15, status: 'in_progress' },
    { id: 'pe-004', name: 'Patch Management Assessment', auditable_entity: 'Patch Management', assigned_team: ['Carol White'], lead_auditor_id: 'usr-002', quarter: 2, start_date: '2025-05-05', end_date: '2025-05-30', budget_days: 12, status: 'planned' },
    { id: 'pe-005', name: 'Network Security Review', auditable_entity: 'Network Segmentation', assigned_team: ['Alice Smith', 'Dan Green'], lead_auditor_id: 'usr-001', quarter: 3, start_date: '2025-07-07', end_date: '2025-08-01', budget_days: 15, status: 'planned' },
    { id: 'pe-006', name: 'Cloud CSPM Audit', auditable_entity: 'Cloud Security Posture', assigned_team: ['Bob Lee', 'Eve Chen'], lead_auditor_id: 'usr-003', quarter: 4, start_date: '2025-10-06', end_date: '2025-10-31', budget_days: 18, status: 'planned' },
  ],
};

// ---- Assurance Map ----
const BUS = ['IT Security', 'Network Ops', 'Procurement', 'Security Ops', 'Data Governance', 'Cloud Ops'];
const CDS = ['IAM', 'Network Security', 'Vendor Management', 'Incident Management', 'Data Protection', 'Cloud Security'];

function buildCell(bu: string, cd: string, coverage: CoverageLevel, notes?: string): AssuranceCell {
  return { business_unit: bu, control_domain: cd, coverage, notes };
}

export const mockAssuranceMap: AssuranceMap = {
  id: 'am-001', org_id: 'org-001', year: 2025,
  name: 'Assurance Map 2025',
  business_units: BUS,
  control_domains: CDS,
  matrix: [
    buildCell('IT Security', 'IAM', 'full', 'IAM Audit Q1 complete'),
    buildCell('IT Security', 'Network Security', 'partial'),
    buildCell('IT Security', 'Vendor Management', 'none'),
    buildCell('IT Security', 'Incident Management', 'partial'),
    buildCell('IT Security', 'Data Protection', 'partial'),
    buildCell('IT Security', 'Cloud Security', 'none'),
    buildCell('Network Ops', 'IAM', 'partial'),
    buildCell('Network Ops', 'Network Security', 'full', 'Network Review Q3 planned'),
    buildCell('Network Ops', 'Vendor Management', 'none'),
    buildCell('Network Ops', 'Incident Management', 'none'),
    buildCell('Network Ops', 'Data Protection', 'none'),
    buildCell('Network Ops', 'Cloud Security', 'none'),
    buildCell('Procurement', 'IAM', 'none'),
    buildCell('Procurement', 'Network Security', 'none'),
    buildCell('Procurement', 'Vendor Management', 'full', 'Supply Chain Review complete'),
    buildCell('Procurement', 'Incident Management', 'none'),
    buildCell('Procurement', 'Data Protection', 'partial'),
    buildCell('Procurement', 'Cloud Security', 'none'),
    buildCell('Security Ops', 'IAM', 'partial'),
    buildCell('Security Ops', 'Network Security', 'partial'),
    buildCell('Security Ops', 'Vendor Management', 'none'),
    buildCell('Security Ops', 'Incident Management', 'partial', 'IR Readiness in progress'),
    buildCell('Security Ops', 'Data Protection', 'partial'),
    buildCell('Security Ops', 'Cloud Security', 'partial'),
    buildCell('Data Governance', 'IAM', 'partial'),
    buildCell('Data Governance', 'Network Security', 'none'),
    buildCell('Data Governance', 'Vendor Management', 'none'),
    buildCell('Data Governance', 'Incident Management', 'none'),
    buildCell('Data Governance', 'Data Protection', 'full'),
    buildCell('Data Governance', 'Cloud Security', 'none'),
    buildCell('Cloud Ops', 'IAM', 'partial'),
    buildCell('Cloud Ops', 'Network Security', 'partial'),
    buildCell('Cloud Ops', 'Vendor Management', 'none'),
    buildCell('Cloud Ops', 'Incident Management', 'partial'),
    buildCell('Cloud Ops', 'Data Protection', 'partial'),
    buildCell('Cloud Ops', 'Cloud Security', 'partial', 'CSPM Audit Q4 planned'),
  ],
};

// ---- Resource Calendar ----
export const mockResourceCalendar: ResourceCalendar = {
  id: 'rc-001', org_id: 'org-001', year: 2025,
  name: 'Audit Team Calendar 2025',
  auditors: [
    {
      auditor_id: 'usr-001', auditor_name: 'Alice Smith', auditor_email: 'alice@example.com',
      role: 'Senior Auditor', available_days: 220,
      assignments: [
        { id: 'asgn-001', engagement_name: 'IAM Audit Q1', start_date: '2025-01-13', end_date: '2025-02-07', allocated_days: 20, quarter: 1 },
        { id: 'asgn-002', engagement_name: 'Supply Chain Risk Review', start_date: '2025-02-10', end_date: '2025-03-07', allocated_days: 18, quarter: 1 },
        { id: 'asgn-003', engagement_name: 'Network Security Review', start_date: '2025-07-07', end_date: '2025-08-01', allocated_days: 15, quarter: 3 },
      ],
    },
    {
      auditor_id: 'usr-002', auditor_name: 'Bob Lee', auditor_email: 'bob@example.com',
      role: 'Auditor', available_days: 200,
      assignments: [
        { id: 'asgn-004', engagement_name: 'IAM Audit Q1', start_date: '2025-01-13', end_date: '2025-02-07', allocated_days: 20, quarter: 1 },
        { id: 'asgn-005', engagement_name: 'Incident Response Readiness', start_date: '2025-04-07', end_date: '2025-05-02', allocated_days: 15, quarter: 2 },
        { id: 'asgn-006', engagement_name: 'Cloud CSPM Audit', start_date: '2025-10-06', end_date: '2025-10-31', allocated_days: 18, quarter: 4 },
      ],
    },
    {
      auditor_id: 'usr-003', auditor_name: 'Carol White', auditor_email: 'carol@example.com',
      role: 'Senior Auditor', available_days: 220,
      assignments: [
        { id: 'asgn-007', engagement_name: 'Supply Chain Risk Review', start_date: '2025-02-10', end_date: '2025-03-07', allocated_days: 18, quarter: 1 },
        { id: 'asgn-008', engagement_name: 'Patch Management Assessment', start_date: '2025-05-05', end_date: '2025-05-30', allocated_days: 12, quarter: 2 },
      ],
    },
    {
      auditor_id: 'usr-004', auditor_name: 'Dan Green', auditor_email: 'dan@example.com',
      role: 'Auditor', available_days: 200,
      assignments: [
        { id: 'asgn-009', engagement_name: 'Incident Response Readiness', start_date: '2025-04-07', end_date: '2025-05-02', allocated_days: 15, quarter: 2 },
        { id: 'asgn-010', engagement_name: 'Network Security Review', start_date: '2025-07-07', end_date: '2025-08-01', allocated_days: 15, quarter: 3 },
      ],
    },
    {
      auditor_id: 'usr-005', auditor_name: 'Eve Chen', auditor_email: 'eve@example.com',
      role: 'Manager', available_days: 180,
      assignments: [
        { id: 'asgn-011', engagement_name: 'Cloud CSPM Audit', start_date: '2025-10-06', end_date: '2025-10-31', allocated_days: 18, quarter: 4 },
      ],
    },
  ],
};
