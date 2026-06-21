// Report Builder utilities — pure functions for block operations
// Locked decision: contentEditable blocks, no external editor library.

// ─── Types ──────────────────────────────────────────────────────────────────────

export type BlockType =
  | 'heading'
  | 'paragraph'
  | 'finding_block'
  | 'evidence_reference'
  | 'chart'
  | 'table';

export type TemplateId = 'standard' | 'executive' | 'regulatory';

export type ChartKind = 'bar' | 'pie' | 'line';

export interface BlockData {
  findingId?: string;
  evidenceId?: string;
  chartType?: ChartKind;
  tableHeaders?: string[];
  tableRows?: string[][];
}

export interface ReportBlock {
  id: string;
  type: BlockType;
  /** Text content for heading/paragraph blocks */
  content: string;
  /** Heading level (1–3) — only for heading type */
  level?: 1 | 2 | 3;
  /** Structured data for non-text blocks */
  data?: BlockData;
}

// ─── ID generation ────────────────────────────────────────────────────────────

function genId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `block-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
}

// ─── Block creation ───────────────────────────────────────────────────────────

const DEFAULT_CONTENT: Record<BlockType, string> = {
  heading: 'Section Heading',
  paragraph: 'Click to edit paragraph text…',
  finding_block: '',
  evidence_reference: '',
  chart: '',
  table: '',
};

export function createBlock(
  type: BlockType,
  overrides: Partial<Omit<ReportBlock, 'id'>> = {}
): ReportBlock {
  return {
    id: genId(),
    type,
    content: DEFAULT_CONTENT[type],
    ...overrides,
  };
}

// ─── Block mutations (pure — return new array) ─────────────────────────────────

/** Insert a new block immediately after `afterIndex`. */
export function addBlock(
  blocks: ReportBlock[],
  type: BlockType,
  afterIndex: number
): ReportBlock[] {
  const next = createBlock(type);
  const result = [...blocks];
  result.splice(Math.max(0, afterIndex + 1), 0, next);
  return result;
}

/** Remove the block with the given id. */
export function removeBlock(blocks: ReportBlock[], id: string): ReportBlock[] {
  return blocks.filter(b => b.id !== id);
}

/** Replace a block's mutable fields. */
export function updateBlock(
  blocks: ReportBlock[],
  id: string,
  updates: Partial<Omit<ReportBlock, 'id'>>
): ReportBlock[] {
  return blocks.map(b => (b.id === id ? { ...b, ...updates } : b));
}

/** Swap block at `index` with the one above it. */
export function moveBlockUp(blocks: ReportBlock[], id: string): ReportBlock[] {
  const index = blocks.findIndex(b => b.id === id);
  if (index <= 0) return blocks;
  const result = [...blocks];
  [result[index - 1], result[index]] = [result[index], result[index - 1]];
  return result;
}

/** Swap block at `index` with the one below it. */
export function moveBlockDown(blocks: ReportBlock[], id: string): ReportBlock[] {
  const index = blocks.findIndex(b => b.id === id);
  if (index < 0 || index >= blocks.length - 1) return blocks;
  const result = [...blocks];
  [result[index], result[index + 1]] = [result[index + 1], result[index]];
  return result;
}

/** Move a block from `fromIndex` to `toIndex` (drag-and-drop reorder). */
export function reorderBlocks(
  blocks: ReportBlock[],
  fromIndex: number,
  toIndex: number
): ReportBlock[] {
  if (fromIndex === toIndex) return blocks;
  if (fromIndex < 0 || fromIndex >= blocks.length) return blocks;
  if (toIndex < 0 || toIndex >= blocks.length) return blocks;
  const result = [...blocks];
  const [moved] = result.splice(fromIndex, 1);
  result.splice(toIndex, 0, moved);
  return result;
}

// ─── Templates ────────────────────────────────────────────────────────────────────

const TEMPLATES: Record<TemplateId, () => ReportBlock[]> = {
  standard: () => [
    createBlock('heading',            { content: 'Audit Report', level: 1 }),
    createBlock('paragraph',          { content: 'This report summarises the findings and recommendations from the internal audit engagement conducted in accordance with IIA Standards.' }),
    createBlock('heading',            { content: 'Executive Summary', level: 2 }),
    createBlock('paragraph',          { content: 'Describe the overall audit opinion and key themes identified during the engagement.' }),
    createBlock('chart',              { data: { chartType: 'bar' } }),
    createBlock('heading',            { content: 'Key Findings', level: 2 }),
    createBlock('finding_block',      { data: { findingId: 'fnd-001' } }),
    createBlock('finding_block',      { data: { findingId: 'fnd-002' } }),
    createBlock('heading',            { content: 'Supporting Evidence', level: 2 }),
    createBlock('evidence_reference', { data: { evidenceId: 'ev-001' } }),
    createBlock('heading',            { content: 'Recommendations', level: 2 }),
    createBlock('table', {
      data: {
        tableHeaders: ['Priority', 'Recommendation', 'Owner', 'Due Date'],
        tableRows: [
          ['High',   'Implement MFA enforcement policy across all privileged accounts', 'IT Security',  '2024-02-15'],
          ['Medium', 'Update patch management SLA to 21 days for critical patches',    'Operations',   '2024-03-01'],
          ['Medium', 'Revise and test incident response playbooks',                    'Security Ops', '2024-04-01'],
        ],
      },
    }),
  ],

  executive: () => [
    createBlock('heading',       { content: 'Executive Summary Report', level: 1 }),
    createBlock('paragraph',     { content: 'This executive summary presents the key outcomes of the audit engagement for board-level review. The audit assessed compliance with NIS 2 Article 21 requirements.' }),
    createBlock('heading',       { content: 'Overall Audit Opinion', level: 2 }),
    createBlock('paragraph',     { content: 'Insert overall opinion statement and compliance posture here.' }),
    createBlock('chart',         { data: { chartType: 'pie' } }),
    createBlock('heading',       { content: 'Critical Findings', level: 2 }),
    createBlock('finding_block', { data: { findingId: 'fnd-001' } }),
    createBlock('finding_block', { data: { findingId: 'fnd-005' } }),
    createBlock('heading',       { content: 'Management Actions', level: 2 }),
    createBlock('paragraph',     { content: 'Summary of agreed management responses and remediation timelines.' }),
  ],

  regulatory: () => [
    createBlock('heading',            { content: 'Regulatory Compliance Report — NIS 2 Article 21', level: 1 }),
    createBlock('paragraph',          { content: 'NIS 2 Article 21 compliance assessment report prepared for regulatory submission. Engagement period: January–March 2024.' }),
    createBlock('heading',            { content: 'Compliance Posture', level: 2 }),
    createBlock('chart',              { data: { chartType: 'bar' } }),
    createBlock('heading',            { content: 'Non-Conformances', level: 2 }),
    createBlock('finding_block',      { data: { findingId: 'fnd-001' } }),
    createBlock('finding_block',      { data: { findingId: 'fnd-002' } }),
    createBlock('finding_block',      { data: { findingId: 'fnd-004' } }),
    createBlock('heading',            { content: 'Evidence Pack', level: 2 }),
    createBlock('evidence_reference', { data: { evidenceId: 'ev-001' } }),
    createBlock('evidence_reference', { data: { evidenceId: 'ev-003' } }),
    createBlock('heading',            { content: 'Remediation Schedule', level: 2 }),
    createBlock('table', {
      data: {
        tableHeaders: ['Article', 'Finding', 'Status', 'Target Date'],
        tableRows: [
          ['21b', 'Insufficient MFA coverage on privileged accounts', 'In Remediation', '2024-02-15'],
          ['21c', 'Patch management cycle exceeds 30-day SLA',        'Open',           '2024-03-01'],
          ['21a', 'Supply chain risk assessments incomplete',          'Open',           '2024-03-15'],
        ],
      },
    }),
  ],
};

/**
 * Return a fresh set of blocks for the given template.
 * Each call generates new unique IDs so multiple instances don’t share block state.
 */
export function getTemplateBlocks(templateId: TemplateId): ReportBlock[] {
  return TEMPLATES[templateId]();
}

export const TEMPLATE_LABELS: Record<TemplateId, string> = {
  standard:   'Standard Audit Report',
  executive:  'Executive Summary',
  regulatory: 'Regulatory Submission',
};

export const BLOCK_TYPE_LABELS: Record<BlockType, string> = {
  heading:            'Heading',
  paragraph:          'Paragraph',
  finding_block:      'Finding Block',
  evidence_reference: 'Evidence Reference',
  chart:              'Chart',
  table:              'Table',
};

