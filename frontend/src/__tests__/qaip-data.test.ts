/**
 * QAIP data aggregation tests.
 * Pure function tests — no browser/DOM required (node test environment).
 */
import {
  computeEngagementCompletionRate,
  computeOnTimeRate,
  computeFindingSeverityDistribution,
  computeAvgResolutionDays,
  computeIIAConformanceScore,
  buildEngagementMetrics,
  iiaStandards,
} from '../lib/qaip-data';
import type { Engagement, Finding } from '@shared/index';

// ─── Fixtures ───────────────────────────────────────────────────────────────────

const BASE_ENG: Engagement = {
  id: 'eng-t1', org_id: 'org-1', framework_id: 'fw-1',
  name: 'Test Engagement', description: '', status: 'planning',
  scope_json: {}, lead_auditor_id: 'usr-1',
  target_start_date: '2024-01-01', target_end_date: '2025-12-31',
  overall_score: 80, risk_rating: 'medium',
  metadata: {}, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z',
};

const BASE_FINDING: Finding = {
  id: 'fnd-t1', org_id: 'org-1', engagement_id: 'eng-t1', control_id: 'ctrl-1',
  finding_ref: 'F-T-001', title: 'Test Finding', description: 'desc',
  root_cause: 'cause', impact: 'impact', severity: 'high', status: 'open',
  evidence_ids: [], tags: [], metadata: {},
  created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z',
};

// ─── computeEngagementCompletionRate ──────────────────────────────────────────

describe('computeEngagementCompletionRate', () => {
  it('returns 0 for an empty list', () => {
    expect(computeEngagementCompletionRate([])).toBe(0);
  });

  it('returns 0 when all engagements are in-progress', () => {
    const engs: Engagement[] = [
      { ...BASE_ENG, status: 'planning' },
      { ...BASE_ENG, status: 'fieldwork' },
    ];
    expect(computeEngagementCompletionRate(engs)).toBe(0);
  });

  it('returns 100 when all engagements are completed', () => {
    const engs: Engagement[] = [
      { ...BASE_ENG, status: 'completed' },
      { ...BASE_ENG, status: 'completed' },
    ];
    expect(computeEngagementCompletionRate(engs)).toBe(100);
  });

  it('counts reporting status as complete', () => {
    const engs: Engagement[] = [
      { ...BASE_ENG, status: 'reporting' },
      { ...BASE_ENG, status: 'planning' },
    ];
    expect(computeEngagementCompletionRate(engs)).toBe(50);
  });

  it('rounds the percentage to nearest integer', () => {
    const engs: Engagement[] = [
      { ...BASE_ENG, status: 'completed' },
      { ...BASE_ENG, status: 'planning' },
      { ...BASE_ENG, status: 'planning' },
    ];
    // 1/3 = 33.33... → rounds to 33
    expect(computeEngagementCompletionRate(engs)).toBe(33);
  });
});

// ─── computeOnTimeRate ──────────────────────────────────────────────────────

describe('computeOnTimeRate', () => {
  it('returns 0 when no engagements have a target date', () => {
    const engs: Engagement[] = [{ ...BASE_ENG, target_end_date: undefined }];
    expect(computeOnTimeRate(engs)).toBe(0);
  });

  it('treats a future target date as on-time', () => {
    // Use far-future date so this never flips in real runs
    const engs: Engagement[] = [{ ...BASE_ENG, target_end_date: '2099-12-31', status: 'fieldwork' }];
    expect(computeOnTimeRate(engs)).toBe(100);
  });

  it('treats a completed engagement as on-time regardless of date', () => {
    const engs: Engagement[] = [{ ...BASE_ENG, target_end_date: '2020-01-01', status: 'completed' }];
    expect(computeOnTimeRate(engs)).toBe(100);
  });
});

// ─── computeFindingSeverityDistribution ─────────────────────────────────

describe('computeFindingSeverityDistribution', () => {
  it('returns zero counts for all severities when list is empty', () => {
    const dist = computeFindingSeverityDistribution([]);
    dist.forEach(d => expect(d.value).toBe(0));
  });

  it('correctly counts each severity', () => {
    const findings: Finding[] = [
      { ...BASE_FINDING, severity: 'critical' },
      { ...BASE_FINDING, severity: 'critical' },
      { ...BASE_FINDING, severity: 'high' },
      { ...BASE_FINDING, severity: 'medium' },
    ];
    const dist = computeFindingSeverityDistribution(findings);
    const map = Object.fromEntries(dist.map(d => [d.name, d.value]));
    expect(map['Critical']).toBe(2);
    expect(map['High']).toBe(1);
    expect(map['Medium']).toBe(1);
    expect(map['Low']).toBe(0);
  });

  it('returns correct number of severity buckets', () => {
    const dist = computeFindingSeverityDistribution([]);
    expect(dist).toHaveLength(4);
  });
});

// ─── computeAvgResolutionDays ────────────────────────────────────────────

describe('computeAvgResolutionDays', () => {
  it('returns a positive number', () => {
    expect(computeAvgResolutionDays([])).toBeGreaterThan(0);
  });

  it('returns the same value regardless of input (demo implementation)', () => {
    const a = computeAvgResolutionDays([]);
    const b = computeAvgResolutionDays([BASE_FINDING]);
    expect(a).toBe(b);
  });
});

// ─── computeIIAConformanceScore ─────────────────────────────────────────

describe('computeIIAConformanceScore', () => {
  it('returns 0 for an empty list', () => {
    expect(computeIIAConformanceScore([])).toBe(0);
  });

  it('returns exact score when all standards have the same score', () => {
    const stds = [{ id: 's1', code: '1000', name: 'Test', family: 'attribute' as const, conformanceScore: 90 }];
    expect(computeIIAConformanceScore(stds)).toBe(90);
  });

  it('returns the mean of multiple standards', () => {
    const stds = [
      { id: 's1', code: '1000', name: 'A', family: 'attribute' as const, conformanceScore: 80 },
      { id: 's2', code: '2000', name: 'B', family: 'performance' as const, conformanceScore: 100 },
    ];
    expect(computeIIAConformanceScore(stds)).toBe(90);
  });

  it('produces a score in range [0, 100] for the default IIA standards', () => {
    const score = computeIIAConformanceScore(iiaStandards);
    expect(score).toBeGreaterThanOrEqual(0);
    expect(score).toBeLessThanOrEqual(100);
  });
});

// ─── buildEngagementMetrics ───────────────────────────────────────────────

describe('buildEngagementMetrics', () => {
  it('returns one row per engagement', () => {
    const engs = [BASE_ENG, { ...BASE_ENG, id: 'eng-t2', name: 'Eng 2' }];
    const rows = buildEngagementMetrics(engs, []);
    expect(rows).toHaveLength(2);
  });

  it('correctly counts findings per engagement', () => {
    const findings: Finding[] = [
      { ...BASE_FINDING, engagement_id: 'eng-t1' },
      { ...BASE_FINDING, engagement_id: 'eng-t1' },
      { ...BASE_FINDING, engagement_id: 'eng-t2' },
    ];
    const engs = [
      BASE_ENG,
      { ...BASE_ENG, id: 'eng-t2', name: 'Eng 2' },
    ];
    const rows = buildEngagementMetrics(engs, findings);
    const t1 = rows.find(r => r.name === 'Test Engagement')!;
    const t2 = rows.find(r => r.name === 'Eng 2')!;
    expect(t1.findings).toBe(2);
    expect(t2.findings).toBe(1);
  });

  it('correctly counts critical findings', () => {
    const findings: Finding[] = [
      { ...BASE_FINDING, engagement_id: 'eng-t1', severity: 'critical' },
      { ...BASE_FINDING, engagement_id: 'eng-t1', severity: 'high' },
      { ...BASE_FINDING, engagement_id: 'eng-t1', severity: 'critical' },
    ];
    const rows = buildEngagementMetrics([BASE_ENG], findings);
    expect(rows[0].criticalCount).toBe(2);
  });

  it('uses overall_score as completion', () => {
    const engs = [{ ...BASE_ENG, overall_score: 65 }];
    const rows = buildEngagementMetrics(engs, []);
    expect(rows[0].completion).toBe(65);
  });

  it('returns 0 completion for engagements without a score', () => {
    const engs = [{ ...BASE_ENG, overall_score: undefined }];
    const rows = buildEngagementMetrics(engs, []);
    expect(rows[0].completion).toBe(0);
  });
});

