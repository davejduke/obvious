/**
 * Management response lifecycle tests
 */
import { useManagementResponseStore } from '../store/management-responses';
import type { ManagementResponseStatus } from '@shared/index';

// Reset store before each test by re-creating state
beforeEach(() => {
  // Re-initialise from module defaults
  useManagementResponseStore.setState(
    useManagementResponseStore.getInitialState?.() ?? useManagementResponseStore.getState()
  );
});

describe('Management response store', () => {
  it('initialises with responses for all mock findings', () => {
    const { responses } = useManagementResponseStore.getState();
    expect(responses.length).toBeGreaterThan(0);
  });

  it('each response has a finding_id and finding_severity', () => {
    const { responses } = useManagementResponseStore.getState();
    for (const r of responses) {
      expect(r.finding_id).toBeDefined();
      expect(['critical', 'high', 'medium', 'low', 'informational']).toContain(r.finding_severity);
    }
  });

  it('contains all 5 status stages across initial data', () => {
    const { responses } = useManagementResponseStore.getState();
    const statuses = new Set(responses.map(r => r.status));
    const expected: ManagementResponseStatus[] = [
      'pending', 'accepted', 'implementation_planned', 'implemented', 'verified'
    ];
    for (const s of expected) {
      expect(statuses.has(s)).toBe(true);
    }
  });

  it('transitionStatus moves pending → accepted', () => {
    const { responses, transitionStatus } = useManagementResponseStore.getState();
    const pending = responses.find(r => r.status === 'pending');
    expect(pending).toBeDefined();
    if (!pending) return;
    transitionStatus(pending.id, 'accepted', { accepted_by: 'CISO', acceptance_notes: 'Accepted' });
    const updated = useManagementResponseStore.getState().responses.find(r => r.id === pending.id)!;
    expect(updated.status).toBe('accepted');
    expect(updated.accepted_by).toBe('CISO');
    expect(updated.accepted_at).toBeDefined();
  });

  it('transitionStatus moves accepted → implementation_planned with plan data', () => {
    const { responses, transitionStatus } = useManagementResponseStore.getState();
    const accepted = responses.find(r => r.status === 'accepted');
    expect(accepted).toBeDefined();
    if (!accepted) return;
    transitionStatus(accepted.id, 'implementation_planned', {
      implementation_plan: 'Deploy patch X',
      implementation_owner: 'IT Team',
      implementation_due_date: '2024-06-01',
    });
    const updated = useManagementResponseStore.getState().responses.find(r => r.id === accepted.id)!;
    expect(updated.status).toBe('implementation_planned');
    expect(updated.implementation_plan).toBe('Deploy patch X');
    expect(updated.planned_at).toBeDefined();
  });

  it('transitionStatus moves implemented → verified', () => {
    const { responses, transitionStatus } = useManagementResponseStore.getState();
    const impl = responses.find(r => r.status === 'implemented');
    expect(impl).toBeDefined();
    if (!impl) return;
    transitionStatus(impl.id, 'verified', { verified_by: 'Lead Auditor', verification_notes: 'All good' });
    const updated = useManagementResponseStore.getState().responses.find(r => r.id === impl.id)!;
    expect(updated.status).toBe('verified');
    expect(updated.verified_by).toBe('Lead Auditor');
    expect(updated.verified_at).toBeDefined();
  });

  it('updateResponse patches arbitrary fields', () => {
    const { responses, updateResponse } = useManagementResponseStore.getState();
    const r = responses[0];
    updateResponse(r.id, { acceptance_notes: 'New note' });
    const updated = useManagementResponseStore.getState().responses.find(x => x.id === r.id)!;
    expect(updated.acceptance_notes).toBe('New note');
    expect(updated.updated_at).toBeDefined();
  });

  it('does not affect other responses when updating one', () => {
    const { responses, transitionStatus } = useManagementResponseStore.getState();
    const pending = responses.find(r => r.status === 'pending');
    if (!pending || responses.length < 2) return;
    const otherBefore = responses.filter(r => r.id !== pending.id).map(r => r.status);
    transitionStatus(pending.id, 'accepted');
    const otherAfter = useManagementResponseStore.getState().responses
      .filter(r => r.id !== pending.id).map(r => r.status);
    expect(otherAfter).toEqual(otherBefore);
  });
});
