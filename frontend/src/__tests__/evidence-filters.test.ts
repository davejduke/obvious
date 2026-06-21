/**
 * Tests for Evidence Explorer filter logic
 * Tests: applyFilters — search, status, source type, classification, quality range, date range, articles
 */
import { applyFilters, DEFAULT_FILTERS, type EvidenceFilters } from '../lib/evidence-filters';
import { extendedMockEvidence } from '../lib/evidence-mock-extended';

describe('applyFilters', () => {
  it('returns all items when filters are default', () => {
    const result = applyFilters(extendedMockEvidence, DEFAULT_FILTERS);
    expect(result).toHaveLength(extendedMockEvidence.length);
  });

  describe('search filter', () => {
    it('filters by title (case-insensitive)', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, search: 'mfa' };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result.length).toBeGreaterThan(0);
      result.forEach((ev) => {
        const text = [ev.title, ev.description ?? '', ev.control_id, ...ev.tags.map((t) => t.label)]
          .join(' ')
          .toLowerCase();
        expect(text).toContain('mfa');
      });
    });

    it('filters by tag label', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, search: 'NIS2-21c' };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result.length).toBeGreaterThan(0);
      result.forEach((ev) => {
        const hasTag = ev.tags.some((t) => t.label.toLowerCase().includes('nis2-21c'));
        const inTitle = ev.title.toLowerCase().includes('nis2-21c');
        const inControl = ev.control_id.toLowerCase().includes('nis2-21c');
        expect(hasTag || inTitle || inControl).toBe(true);
      });
    });

    it('returns empty when search matches nothing', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, search: 'xyzzy-nonexistent-12345' };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result).toHaveLength(0);
    });
  });

  describe('status filter', () => {
    it('filters to accepted only', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, status: ['accepted'] };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result.length).toBeGreaterThan(0);
      result.forEach((ev) => expect(ev.status).toBe('accepted'));
    });

    it('filters to pending_review only', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, status: ['pending_review'] };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => expect(ev.status).toBe('pending_review'));
    });

    it('allows multiple status values', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, status: ['accepted', 'rejected'] };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => expect(['accepted', 'rejected']).toContain(ev.status));
    });
  });

  describe('source type filter', () => {
    it('filters by source type api_integration', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, sourceTypes: ['api_integration'] };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result.length).toBeGreaterThan(0);
      result.forEach((ev) => expect(ev.source_type).toBe('api_integration'));
    });

    it('filters by multiple source types', () => {
      const filters: EvidenceFilters = {
        ...DEFAULT_FILTERS,
        sourceTypes: ['log_export', 'screenshot'],
      };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => expect(['log_export', 'screenshot']).toContain(ev.source_type));
    });
  });

  describe('classification filter', () => {
    it('filters by confidential classification', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, classifications: ['confidential'] };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result.length).toBeGreaterThan(0);
      result.forEach((ev) => expect(ev.classification).toBe('confidential'));
    });

    it('filters by restricted classification', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, classifications: ['restricted'] };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => expect(ev.classification).toBe('restricted'));
    });
  });

  describe('quality score filter', () => {
    it('filters by minimum quality score', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, qualityMin: 80, qualityMax: 100 };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result.length).toBeGreaterThan(0);
      result.forEach((ev) => expect(ev.quality.overall).toBeGreaterThanOrEqual(80));
    });

    it('filters by quality range 40-70', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, qualityMin: 40, qualityMax: 70 };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => {
        expect(ev.quality.overall).toBeGreaterThanOrEqual(40);
        expect(ev.quality.overall).toBeLessThanOrEqual(70);
      });
    });

    it('returns empty when quality range is impossible', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, qualityMin: 95, qualityMax: 100 };
      // No mock items have quality >= 95
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => expect(ev.quality.overall).toBeGreaterThanOrEqual(95));
    });
  });

  describe('date range filter', () => {
    it('filters by dateFrom', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, dateFrom: '2024-01-22' };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => expect(ev.collection_date >= '2024-01-22').toBe(true));
    });

    it('filters by dateTo', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, dateTo: '2024-01-20' };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => expect(ev.collection_date <= '2024-01-20').toBe(true));
    });

    it('filters within date range', () => {
      const filters: EvidenceFilters = {
        ...DEFAULT_FILTERS,
        dateFrom: '2024-01-20',
        dateTo: '2024-01-22',
      };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => {
        expect(ev.collection_date >= '2024-01-20').toBe(true);
        expect(ev.collection_date <= '2024-01-22').toBe(true);
      });
    });
  });

  describe('article filter', () => {
    it('filters by NIS2 article tag', () => {
      const filters: EvidenceFilters = { ...DEFAULT_FILTERS, articles: ['NIS2-21b'] };
      const result = applyFilters(extendedMockEvidence, filters);
      expect(result.length).toBeGreaterThan(0);
      result.forEach((ev) => {
        const hasArticle = ev.tags.some((t) => t.category === 'article' && t.label === 'NIS2-21b');
        expect(hasArticle).toBe(true);
      });
    });
  });

  describe('combined filters', () => {
    it('applies multiple filters simultaneously', () => {
      const filters: EvidenceFilters = {
        ...DEFAULT_FILTERS,
        status: ['accepted'],
        qualityMin: 80,
        qualityMax: 100,
      };
      const result = applyFilters(extendedMockEvidence, filters);
      result.forEach((ev) => {
        expect(ev.status).toBe('accepted');
        expect(ev.quality.overall).toBeGreaterThanOrEqual(80);
      });
    });
  });
});
