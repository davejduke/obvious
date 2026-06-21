// QAIP Data Layer — Quality Assurance Improvement Program metrics
// IIA Standard 1300: Internal audit function performance measurement

import { mockEngagements, mockFindings } from './mock-data';
import type { Finding, Engagement } from '@shared/index';

// ─── IIA Standards ────────────────────────────────────────────────────────────

export interface IIAStandard {
  id: string;
  code: string;
  name: string;
  family: 'attribute' | 'performance' | 'implementation';
  conformanceScore: number; // 0-100
}

export const iiaStandards: IIAStandard[] = [
  { id: 'std-1000', code: '1000', name: 'Purpose, Authority, and Responsibility', family: 'attribute', conformanceScore: 92 },
  { id: 'std-1100', code: '1100', name: 'Independence and Objectivity', family: 'attribute', conformanceScore: 88 },
  { id: 'std-1200', code: '1200', name: 'Proficiency and Due Professional Care', family: 'attribute', conformanceScore: 85 },
  { id: 'std-1300', code: '1300', name: 'Quality Assurance and Improvement', family: 'attribute', conformanceScore: 78 },
  { id: 'std-2000', code: '2000', name: 'Managing the Internal Audit Activity', family: 'performance', conformanceScore: 90 },
  { id: 'std-2100', code: '2100', name: 'Nature of Work', family: 'performance', conformanceScore: 83 },
  { id: 'std-2200', code: '2200', name: 'Engagement Planning', family: 'performance', conformanceScore: 87 },
  { id: 'std-2300', code: '2300', name: 'Performing the Engagement', family: 'performance', conformanceScore: 81 },
  { id: 'std-2400', code: '2400', name: 'Communicating Results', family: 'performance', conformanceScore: 89 },
  { id: 'std-2500', code: '2500', name: 'Monitoring Progress', family: 'performance', conformanceScore: 76 },
  { id: 'std-2600', code: '2600', name: 'Communicating the Acceptance of Risks', family: 'performance', conformanceScore: 80 },
];

// ─── Metric helpers (pure functions — fully testable) ─────────────────────────

/** Percentage of engagements in completed or reporting status */
export function computeEngagementCompletionRate(engagements: Engagement[]): number {
  if (engagements.length === 0) return 0;
  const done = engagements.filter(
    e => e.status === 'completed' || e.status === 'reporting'
  ).length;
  return Math.round((done / engagements.length) * 100);
}

/** Percentage of engagements on-track vs their target end date */
export function computeOnTimeRate(engagements: Engagement[]): number {
  const withTarget = engagements.filter(e => e.target_end_date);
  if (withTarget.length === 0) return 0;
  const now = Date.now();
  const onTime = withTarget.filter(e => {
    const target = new Date(e.target_end_date!).getTime();
    return e.status === 'completed' || now <= target;
  });
  return Math.round((onTime.length / withTarget.length) * 100);
}

/** Finding severity distribution for charts */
export function computeFindingSeverityDistribution(findings: Finding[]) {
  const dist: Record<string, number> = { critical: 0, high: 0, medium: 0, low: 0 };
  for (const f of findings) {
    const sev = f.severity.toLowerCase();
    if (sev in dist) dist[sev]++;
  }
  return [
    { name: 'Critical', value: dist.critical, color: '#DC2626' },
    { name: 'High',     value: dist.high,     color: '#EA580C' },
    { name: 'Medium',   value: dist.medium,   color: '#CA8A04' },
    { name: 'Low',      value: dist.low,      color: '#16A34A' },
  ];
}

/** Average resolution days across a sample set (demo values for open findings) */
export function computeAvgResolutionDays(_findings: Finding[]): number {
  // In production: derived from finding created_at + resolved_at.
  // Demo: static representative values.
  const samples = [12, 28, 45, 7];
  return Math.round(samples.reduce((a, b) => a + b, 0) / samples.length);
}

/** Mean conformance score across all standards */
export function computeIIAConformanceScore(standards: IIAStandard[]): number {
  if (standards.length === 0) return 0;
  return Math.round(
    standards.reduce((sum, s) => sum + s.conformanceScore, 0) / standards.length
  );
}

// ─── Trend data ───────────────────────────────────────────────────────────────

export interface TrendDataPoint {
  period: string;
  completionRate: number;
  onTimeRate: number;
  findingCount: number;
  conformanceScore: number;
}

export const yearOverYearTrend: TrendDataPoint[] = [
  { period: 'Q1 2023', completionRate: 72, onTimeRate: 65, findingCount: 18, conformanceScore: 74 },
  { period: 'Q2 2023', completionRate: 75, onTimeRate: 68, findingCount: 22, conformanceScore: 76 },
  { period: 'Q3 2023', completionRate: 78, onTimeRate: 71, findingCount: 19, conformanceScore: 79 },
  { period: 'Q4 2023', completionRate: 80, onTimeRate: 74, findingCount: 16, conformanceScore: 81 },
  { period: 'Q1 2024', completionRate: 83, onTimeRate: 77, findingCount: 14, conformanceScore: 83 },
  { period: 'Q2 2024', completionRate: 85, onTimeRate: 80, findingCount: 12, conformanceScore: 85 },
];

export interface ResolutionTimeBySeverity {
  severity: string;
  avgDays: number;
  target: number;
}

export const resolutionTimeData: ResolutionTimeBySeverity[] = [
  { severity: 'Critical', avgDays: 12, target: 10 },
  { severity: 'High',     avgDays: 28, target: 30 },
  { severity: 'Medium',   avgDays: 45, target: 60 },
  { severity: 'Low',      avgDays: 72, target: 90 },
];

export interface EngagementMetricRow {
  name: string;
  completion: number;
  onTime: boolean;
  findings: number;
  criticalCount: number;
}

export function buildEngagementMetrics(
  engagements: Engagement[],
  findings: Finding[]
): EngagementMetricRow[] {
  const now = Date.now();
  return engagements.map(eng => {
    const engFindings = findings.filter(f => f.engagement_id === eng.id);
    const criticalCount = engFindings.filter(f => f.severity === 'critical').length;
    const target = eng.target_end_date ? new Date(eng.target_end_date).getTime() : null;
    const onTime = target !== null ? now <= target : true;
    return {
      name: eng.name,
      completion: eng.overall_score ?? 0,
      onTime,
      findings: engFindings.length,
      criticalCount,
    };
  });
}

// Re-export computed defaults for convenience
export const defaultCompletionRate = computeEngagementCompletionRate(mockEngagements);
export const defaultOnTimeRate = computeOnTimeRate(mockEngagements);
export const defaultAvgResolutionDays = computeAvgResolutionDays(mockFindings);
export const defaultConformanceScore = computeIIAConformanceScore(iiaStandards);

