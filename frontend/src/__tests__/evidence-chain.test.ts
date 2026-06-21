/**
 * Tests for Evidence Chain Navigator data
 * Tests: chain event structure, stage ordering, stage completeness
 */
import { extendedMockEvidence, STAGE_LABELS, STAGE_COLORS } from '../lib/evidence-mock-extended';

describe('Evidence Chain Navigator', () => {
  describe('chain event structure', () => {
    it('every evidence item has at least 4 chain events (source through quality_assessment)', () => {
      extendedMockEvidence.forEach((ev) => {
        expect(ev.chain.length).toBeGreaterThanOrEqual(4);
      });
    });

    it('every chain event has required fields', () => {
      extendedMockEvidence.forEach((ev) => {
        ev.chain.forEach((event) => {
          expect(event.id).toBeTruthy();
          expect(event.stage).toBeTruthy();
          expect(event.timestamp).toBeTruthy();
          expect(event.actor).toBeTruthy();
          expect(event.action).toBeTruthy();
          expect(event.detail).toBeTruthy();
          expect(typeof event.metadata).toBe('object');
        });
      });
    });

    it('chain events appear in expected stage order: source, ingestion, classification, quality_assessment', () => {
      const expectedOrder = ['source', 'ingestion', 'classification', 'quality_assessment'];
      extendedMockEvidence.forEach((ev) => {
        const stages = ev.chain.slice(0, 4).map((e) => e.stage);
        expect(stages).toEqual(expectedOrder);
      });
    });

    it('evidence with finding_ids gets a finding chain event', () => {
      const withFindings = extendedMockEvidence.filter((ev) => ev.finding_ids.length > 0);
      withFindings.forEach((ev) => {
        const findingEvent = ev.chain.find((e) => e.stage === 'finding');
        expect(findingEvent).toBeDefined();
      });
    });

    it('evidence without finding_ids does not get a finding chain event', () => {
      const withoutFindings = extendedMockEvidence.filter((ev) => ev.finding_ids.length === 0);
      withoutFindings.forEach((ev) => {
        const findingEvent = ev.chain.find((e) => e.stage === 'finding');
        expect(findingEvent).toBeUndefined();
      });
    });

    it('chain event IDs are unique within each evidence item', () => {
      extendedMockEvidence.forEach((ev) => {
        const ids = ev.chain.map((e) => e.id);
        const unique = new Set(ids);
        expect(unique.size).toBe(ids.length);
      });
    });
  });

  describe('stage labels and colors', () => {
    it('STAGE_LABELS covers all required stages', () => {
      const requiredStages = ['source', 'ingestion', 'classification', 'quality_assessment', 'finding'];
      requiredStages.forEach((stage) => {
        expect(STAGE_LABELS[stage as keyof typeof STAGE_LABELS]).toBeTruthy();
      });
    });

    it('STAGE_COLORS covers all required stages', () => {
      const requiredStages = ['source', 'ingestion', 'classification', 'quality_assessment', 'finding'];
      requiredStages.forEach((stage) => {
        expect(STAGE_COLORS[stage as keyof typeof STAGE_COLORS]).toBeTruthy();
      });
    });
  });

  describe('quality scores', () => {
    it('quality overall score is between 0 and 100', () => {
      extendedMockEvidence.forEach((ev) => {
        expect(ev.quality.overall).toBeGreaterThanOrEqual(0);
        expect(ev.quality.overall).toBeLessThanOrEqual(100);
      });
    });

    it('quality dimensions all exist', () => {
      const dims = ['completeness', 'accuracy', 'timeliness', 'relevance', 'reliability'] as const;
      extendedMockEvidence.forEach((ev) => {
        dims.forEach((dim) => {
          expect(typeof ev.quality.dimensions[dim]).toBe('number');
          expect(ev.quality.dimensions[dim]).toBeGreaterThanOrEqual(0);
          expect(ev.quality.dimensions[dim]).toBeLessThanOrEqual(100);
        });
      });
    });

    it('quality flags is an array', () => {
      extendedMockEvidence.forEach((ev) => {
        expect(Array.isArray(ev.quality.flags)).toBe(true);
      });
    });

    it('assessed_by is either system or auditor', () => {
      extendedMockEvidence.forEach((ev) => {
        expect(['system', 'auditor']).toContain(ev.quality.assessed_by);
      });
    });
  });
});
