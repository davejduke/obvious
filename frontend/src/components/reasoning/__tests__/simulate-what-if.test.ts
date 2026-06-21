/**
 * simulateWhatIf — deterministic simulation unit tests.
 *
 * Tests cover:
 *  - add_evidence increases score
 *  - remove_evidence decreases score
 *  - score is bounded [0, 100]
 *  - gate_change is correctly detected
 *  - narrative is a non-empty string
 *  - affected_factors contains Quality and Evidence Sufficiency
 */
import { simulateWhatIf } from '@/lib/mock-data';

describe('simulateWhatIf', () => {
  const base = {
    engagement_id: 'eng-001',
    evidence_quality_score: 80,
    count: 2,
  };

  it('adding high-quality evidence increases overall score', () => {
    const result = simulateWhatIf({ ...base, control_id: 'ctrl-003', action: 'add_evidence' });
    expect(result.simulated_score).toBeGreaterThan(result.original_score);
    expect(result.delta).toBeGreaterThan(0);
  });

  it('removing evidence decreases overall score', () => {
    const result = simulateWhatIf({ ...base, control_id: 'ctrl-001', action: 'remove_evidence' });
    expect(result.simulated_score).toBeLessThan(result.original_score);
    expect(result.delta).toBeLessThan(0);
  });

  it('returns original score when quality is 0', () => {
    const result = simulateWhatIf({
      ...base,
      control_id: 'ctrl-001',
      action: 'add_evidence',
      evidence_quality_score: 0,
    });
    expect(result.simulated_score).toBe(result.original_score);
    expect(result.delta).toBe(0);
  });

  it('simulated score is bounded between 0 and 100', () => {
    // Add many high-quality items to try to exceed 100
    const result = simulateWhatIf({
      ...base,
      control_id: 'ctrl-001',
      action: 'add_evidence',
      evidence_quality_score: 100,
      count: 100,
    });
    expect(result.simulated_score).toBeLessThanOrEqual(100);
    expect(result.simulated_score).toBeGreaterThanOrEqual(0);
  });

  it('detects gate now_passes transition', () => {
    // ctrl-003 (Incident Response Plan) is currently blocked with score 48
    // Adding enough quality evidence should push it past threshold 70
    const result = simulateWhatIf({
      ...base,
      control_id: 'ctrl-003',
      action: 'add_evidence',
      evidence_quality_score: 100,
      count: 5,
    });
    expect(result.gate_change).toBe('now_passes');
  });

  it('detects gate now_blocks transition', () => {
    // ctrl-001 (MFA) is currently passed with score 82, threshold 70
    // Removing many items should push it below threshold
    const result = simulateWhatIf({
      ...base,
      control_id: 'ctrl-001',
      action: 'remove_evidence',
      evidence_quality_score: 100,
      count: 5,
    });
    expect(result.gate_change).toBe('now_blocks');
  });

  it('returns non-empty narrative string', () => {
    const result = simulateWhatIf({ ...base, control_id: 'ctrl-003', action: 'add_evidence' });
    expect(result.narrative).toBeTruthy();
    expect(typeof result.narrative).toBe('string');
    expect(result.narrative.length).toBeGreaterThan(10);
  });

  it('returns Quality and Evidence Sufficiency in affected_factors', () => {
    const result = simulateWhatIf({ ...base, control_id: 'ctrl-003', action: 'add_evidence' });
    const factorNames = result.affected_factors.map(f => f.factor);
    expect(factorNames).toContain('Quality');
    expect(factorNames).toContain('Evidence Sufficiency');
  });
});
