/**
 * Tests for Evidence Comparison helpers
 * Tests: quality color, classification color, comparison dimension delta logic
 */
import {
  qualityColor,
  qualityBgColor,
  classificationColor,
  extendedMockEvidence,
  type ClassificationLevel,
} from '../lib/evidence-mock-extended';

describe('qualityColor', () => {
  it('returns green for score >= 80', () => {
    expect(qualityColor(80)).toBe('text-green-600');
    expect(qualityColor(95)).toBe('text-green-600');
    expect(qualityColor(100)).toBe('text-green-600');
  });

  it('returns yellow for 60 <= score < 80', () => {
    expect(qualityColor(60)).toBe('text-yellow-600');
    expect(qualityColor(70)).toBe('text-yellow-600');
    expect(qualityColor(79)).toBe('text-yellow-600');
  });

  it('returns red for score < 60', () => {
    expect(qualityColor(0)).toBe('text-red-600');
    expect(qualityColor(30)).toBe('text-red-600');
    expect(qualityColor(59)).toBe('text-red-600');
  });
});

describe('qualityBgColor', () => {
  it('returns green background for score >= 80', () => {
    expect(qualityBgColor(80)).toBe('bg-green-500');
    expect(qualityBgColor(100)).toBe('bg-green-500');
  });

  it('returns yellow background for 60 <= score < 80', () => {
    expect(qualityBgColor(60)).toBe('bg-yellow-500');
    expect(qualityBgColor(79)).toBe('bg-yellow-500');
  });

  it('returns red background for score < 60', () => {
    expect(qualityBgColor(0)).toBe('bg-red-500');
    expect(qualityBgColor(59)).toBe('bg-red-500');
  });
});

describe('classificationColor', () => {
  const levels: ClassificationLevel[] = ['public', 'internal', 'confidential', 'restricted'];

  it('returns a non-empty string for all classification levels', () => {
    levels.forEach((level) => {
      const cls = classificationColor(level);
      expect(cls).toBeTruthy();
      expect(typeof cls).toBe('string');
    });
  });

  it('returns different colors for different levels', () => {
    const colors = levels.map(classificationColor);
    const unique = new Set(colors);
    expect(unique.size).toBe(4);
  });

  it('restricted gets red styling', () => {
    expect(classificationColor('restricted')).toContain('red');
  });

  it('public gets green styling', () => {
    expect(classificationColor('public')).toContain('green');
  });
});

describe('evidence comparison data integrity', () => {
  it('ev-001 and ev-002 share the same control_id (ctrl-001) for cross-validation', () => {
    const ev1 = extendedMockEvidence.find((e) => e.id === 'ev-001');
    const ev2 = extendedMockEvidence.find((e) => e.id === 'ev-002');
    expect(ev1).toBeDefined();
    expect(ev2).toBeDefined();
    expect(ev1!.control_id).toBe(ev2!.control_id);
  });

  it('quality overall score differences can be computed for comparison', () => {
    const ev1 = extendedMockEvidence.find((e) => e.id === 'ev-001')!;
    const ev2 = extendedMockEvidence.find((e) => e.id === 'ev-002')!;
    const diff = ev1.quality.overall - ev2.quality.overall;
    // ev-001 (88) - ev-002 (62) = 26
    expect(diff).toBe(26);
  });

  it('all evidence items have unique IDs', () => {
    const ids = extendedMockEvidence.map((e) => e.id);
    const unique = new Set(ids);
    expect(unique.size).toBe(ids.length);
  });

  it('evidence tags have valid category values', () => {
    const validCategories = ['control', 'article', 'domain', 'risk'];
    extendedMockEvidence.forEach((ev) => {
      ev.tags.forEach((tag) => {
        expect(validCategories).toContain(tag.category);
      });
    });
  });
});

describe('bulk actions data logic', () => {
  it('can select a subset of evidence IDs', () => {
    const allIds = extendedMockEvidence.map((e) => e.id);
    const subset = allIds.slice(0, 3);
    expect(subset).toHaveLength(3);
    subset.forEach((id) => expect(allIds).toContain(id));
  });

  it('simulates batch quality reassessment (increment overall by 5, cap 100)', () => {
    const items = extendedMockEvidence.slice(0, 3);
    const reassessed = items.map((ev) => ({
      ...ev,
      quality: {
        ...ev.quality,
        overall: Math.min(100, ev.quality.overall + 5),
      },
    }));
    reassessed.forEach((ev, i) => {
      const original = items[i].quality.overall;
      expect(ev.quality.overall).toBe(Math.min(100, original + 5));
    });
  });

  it('simulates batch reclassification to accepted', () => {
    const targetIds = ['ev-002', 'ev-005'];
    const updated = extendedMockEvidence.map((ev) =>
      targetIds.includes(ev.id) ? { ...ev, status: 'accepted' as const } : ev,
    );
    targetIds.forEach((id) => {
      const ev = updated.find((e) => e.id === id);
      expect(ev?.status).toBe('accepted');
    });
    // Items not in target retain original status
    const ev001 = updated.find((e) => e.id === 'ev-001');
    expect(ev001?.status).toBe('accepted'); // was already accepted
  });

  it('simulates batch archive', () => {
    const targetIds = ['ev-003', 'ev-004'];
    const updated = extendedMockEvidence.map((ev) =>
      targetIds.includes(ev.id) ? { ...ev, status: 'archived' as const } : ev,
    );
    targetIds.forEach((id) => {
      const ev = updated.find((e) => e.id === id);
      expect(ev?.status).toBe('archived');
    });
  });
});
