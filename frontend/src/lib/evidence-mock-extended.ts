/**
 * Extended evidence mock data for Evidence Explorer
 * Includes quality scores, classification levels, chain events, and comparison data
 */
import type { Evidence, EvidenceSourceType } from '@shared/index';

export type ClassificationLevel = 'public' | 'internal' | 'confidential' | 'restricted';
export type QualityDimension = 'completeness' | 'accuracy' | 'timeliness' | 'relevance' | 'reliability';

export interface EvidenceQualityScore {
  overall: number;
  dimensions: Record<QualityDimension, number>;
  flags: string[];
  assessed_at: string;
  assessed_by: 'system' | 'auditor';
}

export interface EvidenceChainEvent {
  id: string;
  stage: 'source' | 'ingestion' | 'classification' | 'quality_assessment' | 'finding';
  timestamp: string;
  actor: string;
  action: string;
  detail: string;
  metadata: Record<string, unknown>;
}

export interface EvidenceTag {
  label: string;
  category: 'control' | 'article' | 'domain' | 'risk';
  color: string;
}

export interface ExtendedEvidence extends Evidence {
  classification: ClassificationLevel;
  quality: EvidenceQualityScore;
  chain: EvidenceChainEvent[];
  tags: EvidenceTag[];
  finding_ids: string[];
  content_preview: string;
  file_type?: string;
  file_size?: string;
}

const makeChain = (
  evidenceId: string,
  sourceType: EvidenceSourceType,
  collectionDate: string,
  findingId?: string,
): EvidenceChainEvent[] => {
  const events: EvidenceChainEvent[] = [
    {
      id: `${evidenceId}-evt-1`,
      stage: 'source',
      timestamp: `${collectionDate}T08:00:00Z`,
      actor: 'System',
      action: 'Source identified',
      detail: `Evidence sourced from ${sourceType.replace(/_/g, ' ')}`,
      metadata: { source_type: sourceType },
    },
    {
      id: `${evidenceId}-evt-2`,
      stage: 'ingestion',
      timestamp: `${collectionDate}T08:15:00Z`,
      actor: 'Pipeline',
      action: 'Ingested & stored',
      detail: 'Evidence file validated, hashed (SHA-256) and stored in secure vault',
      metadata: { hash_algorithm: 'SHA-256', vault_path: `/evidence/${evidenceId}` },
    },
    {
      id: `${evidenceId}-evt-3`,
      stage: 'classification',
      timestamp: `${collectionDate}T08:20:00Z`,
      actor: 'Classifier v2.1',
      action: 'Classified',
      detail: 'NLP classifier applied - content parsed, control mappings identified',
      metadata: { model_version: '2.1', confidence: 0.87 },
    },
    {
      id: `${evidenceId}-evt-4`,
      stage: 'quality_assessment',
      timestamp: `${collectionDate}T08:25:00Z`,
      actor: 'Quality Engine',
      action: 'Quality scored',
      detail: 'Automated quality dimensions assessed: completeness, accuracy, timeliness, relevance, reliability',
      metadata: { engine_version: '1.4' },
    },
  ];
  if (findingId) {
    events.push({
      id: `${evidenceId}-evt-5`,
      stage: 'finding',
      timestamp: `${collectionDate}T09:00:00Z`,
      actor: 'Auditor',
      action: 'Linked to finding',
      detail: `Evidence linked to finding ${findingId} during fieldwork review`,
      metadata: { finding_id: findingId },
    });
  }
  return events;
};

export const extendedMockEvidence: ExtendedEvidence[] = [
  {
    id: 'ev-001',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-001',
    uploaded_by_id: 'usr-002',
    title: 'AD MFA Enrollment Report - Jan 2024',
    description: 'Export of Active Directory MFA enrollment status for all users across 3 domains. Includes privileged account breakdown.',
    source_type: 'api_integration',
    collection_date: '2024-01-18',
    status: 'accepted',
    is_sufficient: true,
    classification: 'confidential',
    content_preview: 'Domain: corp.example.com\nTotal Users: 4,821\nMFA Enrolled: 2,073 (43%)\nPrivileged Accounts: 312\nPrivileged MFA Enrolled: 98 (31%)...',
    file_type: 'CSV',
    file_size: '245KB',
    quality: {
      overall: 88,
      dimensions: { completeness: 92, accuracy: 90, timeliness: 95, relevance: 85, reliability: 78 },
      flags: [],
      assessed_at: '2024-01-18T08:25:00Z',
      assessed_by: 'system',
    },
    chain: makeChain('ev-001', 'api_integration', '2024-01-18', 'fnd-001'),
    tags: [
      { label: 'NIS2-21b', category: 'article', color: 'blue' },
      { label: 'IAM', category: 'domain', color: 'purple' },
      { label: 'MFA', category: 'control', color: 'green' },
    ],
    finding_ids: ['fnd-001'],
    metadata: { file_size: '245KB', format: 'CSV' },
    created_at: '2024-01-18T00:00:00Z',
    updated_at: '2024-01-19T00:00:00Z',
  },
  {
    id: 'ev-002',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-001',
    uploaded_by_id: 'usr-003',
    title: 'MFA Policy Screenshots',
    description: 'Screenshots of current MFA policy configuration in Azure AD Conditional Access.',
    source_type: 'screenshot',
    collection_date: '2024-01-19',
    status: 'pending_review',
    is_sufficient: false,
    classification: 'internal',
    content_preview: 'Screenshot 1: Azure AD Conditional Access policy screen showing MFA requirement for admin roles only. Screenshot 2: Service account exclusion list...',
    file_type: 'PNG',
    file_size: '1.2MB',
    quality: {
      overall: 62,
      dimensions: { completeness: 55, accuracy: 80, timeliness: 90, relevance: 70, reliability: 45 },
      flags: ['Missing privileged service accounts', 'Partial coverage - admin roles only'],
      assessed_at: '2024-01-19T08:25:00Z',
      assessed_by: 'system',
    },
    chain: makeChain('ev-002', 'screenshot', '2024-01-19', 'fnd-001'),
    tags: [
      { label: 'NIS2-21b', category: 'article', color: 'blue' },
      { label: 'IAM', category: 'domain', color: 'purple' },
    ],
    finding_ids: ['fnd-001'],
    metadata: { file_size: '1.2MB', format: 'PNG' },
    created_at: '2024-01-19T00:00:00Z',
    updated_at: '2024-01-19T00:00:00Z',
  },
  {
    id: 'ev-003',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-002',
    uploaded_by_id: 'usr-002',
    title: 'Patch Management Log Export - Q4 2023',
    description: 'SIEM log export showing patch deployment timelines for all critical systems.',
    source_type: 'log_export',
    collection_date: '2024-01-20',
    status: 'accepted',
    is_sufficient: true,
    classification: 'confidential',
    content_preview: '2023-10-01 PATCH_DEPLOY sys=PROD-WEB-001 patch=CVE-2023-4911 days_to_deploy=47\n2023-10-03 PATCH_DEPLOY sys=PROD-DB-001 patch=CVE-2023-38408 days_to_deploy=52...',
    file_type: 'JSON',
    file_size: '8.7MB',
    quality: {
      overall: 91,
      dimensions: { completeness: 95, accuracy: 93, timeliness: 88, relevance: 95, reliability: 85 },
      flags: [],
      assessed_at: '2024-01-20T08:25:00Z',
      assessed_by: 'system',
    },
    chain: makeChain('ev-003', 'log_export', '2024-01-20', 'fnd-002'),
    tags: [
      { label: 'NIS2-21c', category: 'article', color: 'blue' },
      { label: 'Vuln Mgmt', category: 'domain', color: 'orange' },
      { label: 'Patching', category: 'control', color: 'yellow' },
    ],
    finding_ids: ['fnd-002'],
    metadata: { file_size: '8.7MB', format: 'JSON' },
    created_at: '2024-01-20T00:00:00Z',
    updated_at: '2024-01-21T00:00:00Z',
  },
  {
    id: 'ev-004',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-003',
    uploaded_by_id: 'usr-003',
    title: 'IR Playbook v2.1 - Current Version',
    description: 'Current incident response playbook document. Last formally reviewed 18 months prior.',
    source_type: 'manual_upload',
    collection_date: '2024-01-21',
    status: 'accepted',
    is_sufficient: false,
    classification: 'internal',
    content_preview: 'INCIDENT RESPONSE PLAN v2.1\nLast Review: June 2022\nNext Review: June 2023 (OVERDUE)\n\n1. Incident Classification...',
    file_type: 'PDF',
    file_size: '3.4MB',
    quality: {
      overall: 55,
      dimensions: { completeness: 70, accuracy: 60, timeliness: 20, relevance: 80, reliability: 45 },
      flags: ['Stale document - review date overdue by 6 months', 'Missing cloud environment procedures'],
      assessed_at: '2024-01-21T08:25:00Z',
      assessed_by: 'system',
    },
    chain: makeChain('ev-004', 'manual_upload', '2024-01-21', 'fnd-003'),
    tags: [
      { label: 'NIS2-21d', category: 'article', color: 'blue' },
      { label: 'IR', category: 'domain', color: 'red' },
    ],
    finding_ids: ['fnd-003'],
    metadata: { file_size: '3.4MB', format: 'PDF' },
    created_at: '2024-01-21T00:00:00Z',
    updated_at: '2024-01-22T00:00:00Z',
  },
  {
    id: 'ev-005',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-004',
    uploaded_by_id: 'usr-002',
    title: 'Vendor Risk Assessment Matrix 2024',
    description: 'Supplier risk assessment register covering 34 critical vendors. 12 assessments incomplete.',
    source_type: 'manual_upload',
    collection_date: '2024-01-22',
    status: 'pending_review',
    is_sufficient: false,
    classification: 'restricted',
    content_preview: 'VENDOR ID | NAME | RISK TIER | ASSESSMENT STATUS | LAST REVIEW\nVND-001 | Acme Cloud | Tier 1 | COMPLETE | 2023-11-15\nVND-002 | Beta Systems | Tier 1 | INCOMPLETE | N/A...',
    file_type: 'XLSX',
    file_size: '1.1MB',
    quality: {
      overall: 48,
      dimensions: { completeness: 35, accuracy: 72, timeliness: 65, relevance: 90, reliability: 58 },
      flags: ['35% of Tier 1 vendors missing assessments', 'No risk scoring methodology documented'],
      assessed_at: '2024-01-22T08:25:00Z',
      assessed_by: 'system',
    },
    chain: makeChain('ev-005', 'manual_upload', '2024-01-22', 'fnd-004'),
    tags: [
      { label: 'NIS2-21a', category: 'article', color: 'blue' },
      { label: 'Supply Chain', category: 'domain', color: 'teal' },
    ],
    finding_ids: ['fnd-004'],
    metadata: { file_size: '1.1MB', format: 'XLSX' },
    created_at: '2024-01-22T00:00:00Z',
    updated_at: '2024-01-23T00:00:00Z',
  },
  {
    id: 'ev-006',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-005',
    uploaded_by_id: 'usr-002',
    title: 'Network Topology Diagram - OT/IT Boundary',
    description: 'Automated network topology scan showing OT/IT boundary configuration and firewall rules.',
    source_type: 'automated_scan',
    collection_date: '2024-01-23',
    status: 'accepted',
    is_sufficient: true,
    classification: 'restricted',
    content_preview: 'NETWORK TOPOLOGY SCAN REPORT\nScan Date: 2024-01-23\nScanner: NetDiscovery v4.2\n\nOT/IT Boundary Analysis:\n  Junction Point 1 (OT-ZONE-A to IT-DMZ): NO FIREWALL\n  Junction Point 2 (SCADA-NET to CORP-LAN): PARTIAL RULES...',
    file_type: 'JSON',
    file_size: '2.2MB',
    quality: {
      overall: 85,
      dimensions: { completeness: 88, accuracy: 92, timeliness: 98, relevance: 95, reliability: 72 },
      flags: ['3 unprotected junction points identified'],
      assessed_at: '2024-01-23T08:25:00Z',
      assessed_by: 'system',
    },
    chain: makeChain('ev-006', 'automated_scan', '2024-01-23', 'fnd-005'),
    tags: [
      { label: 'NIS2-21e', category: 'article', color: 'blue' },
      { label: 'Network', category: 'domain', color: 'indigo' },
      { label: 'OT Security', category: 'control', color: 'gray' },
    ],
    finding_ids: ['fnd-005'],
    metadata: { file_size: '2.2MB', format: 'JSON' },
    created_at: '2024-01-23T00:00:00Z',
    updated_at: '2024-01-24T00:00:00Z',
  },
  {
    id: 'ev-007',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-005',
    uploaded_by_id: 'usr-003',
    title: 'Firewall Rule Export - OT Segment',
    description: 'Configuration export of firewall rules governing OT network segments.',
    source_type: 'configuration_export',
    collection_date: '2024-01-23',
    status: 'accepted',
    is_sufficient: true,
    classification: 'restricted',
    content_preview: 'FIREWALL CONFIG EXPORT - OT-SEGMENT-FW\nExported: 2024-01-23\n\nACL RULES:\n  RULE 001: ALLOW ANY to OT-ZONE-A (NO FILTER - LEGACY)\n  RULE 002: ALLOW SCADA-NET to CORP-LAN TCP:*...',
    file_type: 'TXT',
    file_size: '512KB',
    quality: {
      overall: 79,
      dimensions: { completeness: 82, accuracy: 88, timeliness: 98, relevance: 90, reliability: 65 },
      flags: ['Legacy permissive rules identified'],
      assessed_at: '2024-01-23T08:25:00Z',
      assessed_by: 'system',
    },
    chain: makeChain('ev-007', 'configuration_export', '2024-01-23', 'fnd-005'),
    tags: [
      { label: 'NIS2-21e', category: 'article', color: 'blue' },
      { label: 'Network', category: 'domain', color: 'indigo' },
    ],
    finding_ids: ['fnd-005'],
    metadata: { file_size: '512KB', format: 'TXT' },
    created_at: '2024-01-23T00:00:00Z',
    updated_at: '2024-01-24T00:00:00Z',
  },
  {
    id: 'ev-008',
    org_id: 'org-001',
    engagement_id: 'eng-001',
    control_id: 'ctrl-001',
    uploaded_by_id: 'usr-004',
    title: 'Privileged Access Review - Service Accounts',
    description: 'Manual review output of service account MFA exemptions with business justifications.',
    source_type: 'manual_upload',
    collection_date: '2024-01-25',
    status: 'rejected',
    is_sufficient: false,
    classification: 'confidential',
    content_preview: 'SERVICE ACCOUNT REVIEW\nDate: 2024-01-25\n\nTotal Service Accounts: 214\nMFA Exempted: 214 (100%)\nJustifications Provided: 18/214\nMissing Justifications: 196...',
    file_type: 'DOCX',
    file_size: '890KB',
    quality: {
      overall: 30,
      dimensions: { completeness: 15, accuracy: 55, timeliness: 90, relevance: 85, reliability: 20 },
      flags: ['91.5% of accounts lack documented justification', 'Reviewer signature missing', 'No approver noted'],
      assessed_at: '2024-01-25T08:25:00Z',
      assessed_by: 'auditor',
    },
    chain: makeChain('ev-008', 'manual_upload', '2024-01-25'),
    tags: [
      { label: 'NIS2-21b', category: 'article', color: 'blue' },
      { label: 'IAM', category: 'domain', color: 'purple' },
    ],
    finding_ids: [],
    metadata: { file_size: '890KB', format: 'DOCX' },
    created_at: '2024-01-25T00:00:00Z',
    updated_at: '2024-01-26T00:00:00Z',
  },
];

export function qualityColor(score: number): string {
  if (score >= 80) return 'text-green-600';
  if (score >= 60) return 'text-yellow-600';
  return 'text-red-600';
}

export function qualityBgColor(score: number): string {
  if (score >= 80) return 'bg-green-500';
  if (score >= 60) return 'bg-yellow-500';
  return 'bg-red-500';
}

export function classificationColor(level: ClassificationLevel): string {
  const map: Record<ClassificationLevel, string> = {
    public: 'bg-green-100 text-green-800',
    internal: 'bg-blue-100 text-blue-800',
    confidential: 'bg-orange-100 text-orange-800',
    restricted: 'bg-red-100 text-red-800',
  };
  return map[level];
}

export const STAGE_LABELS: Record<EvidenceChainEvent['stage'], string> = {
  source: 'Source',
  ingestion: 'Ingestion',
  classification: 'Classification',
  quality_assessment: 'Quality Assessment',
  finding: 'Finding',
};

export const STAGE_COLORS: Record<EvidenceChainEvent['stage'], string> = {
  source: 'bg-slate-400',
  ingestion: 'bg-blue-500',
  classification: 'bg-purple-500',
  quality_assessment: 'bg-teal-500',
  finding: 'bg-orange-500',
};
