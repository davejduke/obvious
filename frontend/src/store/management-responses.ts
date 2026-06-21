import { create } from 'zustand';
import type { ManagementResponse, ManagementResponseStatus } from '@shared/index';
import { mockFindings } from '@/lib/mock-data';

// Build initial mock responses from existing findings
function buildInitialResponses(): ManagementResponse[] {
  return mockFindings.map((f, i) => {
    const statusMap: ManagementResponseStatus[] = [
      'pending', 'accepted', 'implementation_planned', 'implemented', 'verified'
    ];
    const status = statusMap[i % statusMap.length];
    const base: ManagementResponse = {
      id: `mr-${f.id}`,
      finding_id: f.id,
      finding_ref: f.finding_ref,
      finding_title: f.title,
      finding_severity: f.severity,
      status,
      created_at: f.created_at,
      updated_at: f.updated_at,
    };

    if (['accepted', 'implementation_planned', 'implemented', 'verified'].includes(status)) {
      base.accepted_by = 'CISO';
      base.accepted_at = '2024-02-01T09:00:00Z';
      base.acceptance_notes = 'Risk accepted. Remediation plan required.';
    }
    if (['implementation_planned', 'implemented', 'verified'].includes(status)) {
      base.implementation_plan = 'Deploy MFA enforcement policy to all privileged accounts via AD group policy.';
      base.implementation_owner = 'IT Security Team';
      base.implementation_due_date = '2024-03-31';
      base.planned_at = '2024-02-05T10:00:00Z';
    }
    if (['implemented', 'verified'].includes(status)) {
      base.implemented_at = '2024-03-20T14:00:00Z';
      base.implementation_notes = 'MFA deployed to 98% of privileged accounts. Policy enforced via AD.';
    }
    if (status === 'verified') {
      base.verified_by = 'Lead Auditor';
      base.verified_at = '2024-03-28T11:00:00Z';
      base.verification_notes = 'Verified via Active Directory report. Compliant.';
    }

    return base;
  });
}

interface ManagementResponseState {
  responses: ManagementResponse[];
  updateResponse: (id: string, patch: Partial<ManagementResponse>) => void;
  transitionStatus: (id: string, newStatus: ManagementResponseStatus, meta?: Partial<ManagementResponse>) => void;
}

export const useManagementResponseStore = create<ManagementResponseState>((set) => ({
  responses: buildInitialResponses(),

  updateResponse: (id, patch) =>
    set((s) => ({
      responses: s.responses.map(r =>
        r.id === id ? { ...r, ...patch, updated_at: new Date().toISOString() } : r
      ),
    })),

  transitionStatus: (id, newStatus, meta = {}) =>
    set((s) => ({
      responses: s.responses.map(r => {
        if (r.id !== id) return r;
        const now = new Date().toISOString();
        const updates: Partial<ManagementResponse> = { ...meta, status: newStatus, updated_at: now };

        if (newStatus === 'accepted' && !r.accepted_at) {
          updates.accepted_at = now;
        }
        if (newStatus === 'implementation_planned' && !r.planned_at) {
          updates.planned_at = now;
        }
        if (newStatus === 'implemented' && !r.implemented_at) {
          updates.implemented_at = now;
        }
        if (newStatus === 'verified' && !r.verified_at) {
          updates.verified_at = now;
        }

        return { ...r, ...updates };
      }),
    })),
}));
