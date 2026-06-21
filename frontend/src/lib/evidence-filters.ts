/**
 * Evidence Explorer filter types and filter logic
 */
import type { ExtendedEvidence, ClassificationLevel } from './evidence-mock-extended';
import type { EvidenceSourceType } from '@shared/index';

export interface EvidenceFilters {
  search: string;
  status: string[];
  sourceTypes: EvidenceSourceType[];
  classifications: ClassificationLevel[];
  qualityMin: number;
  qualityMax: number;
  dateFrom: string;
  dateTo: string;
  articles: string[];
}

export const DEFAULT_FILTERS: EvidenceFilters = {
  search: '',
  status: [],
  sourceTypes: [],
  classifications: [],
  qualityMin: 0,
  qualityMax: 100,
  dateFrom: '',
  dateTo: '',
  articles: [],
};

export function applyFilters(
  evidence: ExtendedEvidence[],
  filters: EvidenceFilters,
): ExtendedEvidence[] {
  return evidence.filter((ev) => {
    // Search
    if (filters.search) {
      const q = filters.search.toLowerCase();
      const match =
        ev.title.toLowerCase().includes(q) ||
        (ev.description ?? '').toLowerCase().includes(q) ||
        ev.control_id.toLowerCase().includes(q) ||
        ev.tags.some((t) => t.label.toLowerCase().includes(q));
      if (!match) return false;
    }

    // Status
    if (filters.status.length > 0 && !filters.status.includes(ev.status)) return false;

    // Source types
    if (filters.sourceTypes.length > 0 && !filters.sourceTypes.includes(ev.source_type)) return false;

    // Classification
    if (
      filters.classifications.length > 0 &&
      !filters.classifications.includes(ev.classification)
    )
      return false;

    // Quality score
    if (ev.quality.overall < filters.qualityMin || ev.quality.overall > filters.qualityMax)
      return false;

    // Date range
    if (filters.dateFrom && ev.collection_date < filters.dateFrom) return false;
    if (filters.dateTo && ev.collection_date > filters.dateTo) return false;

    // Articles
    if (filters.articles.length > 0) {
      const hasArticle = ev.tags.some(
        (t) => t.category === 'article' && filters.articles.includes(t.label),
      );
      if (!hasArticle) return false;
    }

    return true;
  });
}

export const SOURCE_TYPE_LABELS: Record<EvidenceSourceType, string> = {
  manual_upload: 'Manual Upload',
  api_integration: 'API Integration',
  automated_scan: 'Automated Scan',
  screenshot: 'Screenshot',
  log_export: 'Log Export',
  configuration_export: 'Config Export',
};

export const STATUS_OPTIONS = [
  { value: 'pending_review', label: 'Pending Review' },
  { value: 'accepted', label: 'Accepted' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'archived', label: 'Archived' },
] as const;

export const CLASSIFICATION_OPTIONS: ClassificationLevel[] = [
  'public',
  'internal',
  'confidential',
  'restricted',
];

export const ARTICLE_OPTIONS = ['NIS2-21a', 'NIS2-21b', 'NIS2-21c', 'NIS2-21d', 'NIS2-21e', 'NIS2-21f', 'NIS2-21g', 'NIS2-21h'];
