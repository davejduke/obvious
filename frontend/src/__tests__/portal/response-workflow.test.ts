/**
 * Tests for the finding response workflow — validates the context-before-finding
 * pattern, management response data model, and helper functions.
 */
import {
  portalFindings,
  portalControls,
  mockManagementResponses,
  portalEngagement,
  getControlForFinding,
  getManagementResponse,
  getRequestsForFinding,
  mockEvidenceRequests,
} from '@/lib/portal-mock-data';

describe('context-before-finding — data availability', () => {
  it('portalEngagement is defined with name and status', () => {
    expect(portalEngagement.id).toBeDefined();
    expect(portalEngagement.name).toBeTruthy();
    expect(portalEngagement.status).toBeTruthy();
  });

  it('portalControls has at least one control per finding control_id', () => {
    const findingControlIds = new Set(portalFindings.map((f) => f.control_id));
    for (const controlId of findingControlIds) {
      const found = portalControls.some((c) => c.id === controlId);
      expect(found).toBe(true);
    }
  });

  it('each portal finding has a matching control via getControlForFinding', () => {
    for (const finding of portalFindings) {
      const control = getControlForFinding(finding.control_id);
      expect(control).toBeDefined();
      expect(control?.id).toBe(finding.control_id);
    }
  });
});

describe('management response workflow', () => {
  it('mockManagementResponses contains at least one submitted response', () => {
    const submitted = mockManagementResponses.filter((r) => r.status === 'submitted');
    expect(submitted.length).toBeGreaterThan(0);
  });

  it('getManagementResponse returns the correct response for a finding', () => {
    const mr = mockManagementResponses[0];
    const result = getManagementResponse(mr.finding_id);
    expect(result).toBeDefined();
    expect(result?.id).toBe(mr.id);
    expect(result?.finding_id).toBe(mr.finding_id);
  });

  it('getManagementResponse returns undefined for a finding with no response', () => {
    // Find a finding not in mockManagementResponses
    const respondedIds = new Set(mockManagementResponses.map((r) => r.finding_id));
    const unrespondedFinding = portalFindings.find((f) => !respondedIds.has(f.id));
    if (unrespondedFinding) {
      expect(getManagementResponse(unrespondedFinding.id)).toBeUndefined();
    }
  });

  it('management response has required fields', () => {
    for (const mr of mockManagementResponses) {
      expect(mr.id).toBeTruthy();
      expect(mr.finding_id).toBeTruthy();
      expect(mr.responder_email).toBeTruthy();
      expect(mr.response_text.length).toBeGreaterThan(0);
      expect(['draft', 'submitted', 'acknowledged']).toContain(mr.status);
    }
  });

  it('submitted responses have a submitted_at timestamp', () => {
    const submitted = mockManagementResponses.filter((r) => r.status === 'submitted');
    for (const mr of submitted) {
      expect(mr.submitted_at).toBeDefined();
      expect(new Date(mr.submitted_at!).toISOString()).toBe(mr.submitted_at);
    }
  });
});

describe('evidence request — finding linkage', () => {
  it('getRequestsForFinding returns only requests for that finding', () => {
    for (const finding of portalFindings) {
      const requests = getRequestsForFinding(finding.id);
      for (const req of requests) {
        expect(req.finding_id).toBe(finding.id);
      }
    }
  });

  it('evidence requests that link to a finding reference a real finding', () => {
    const findingIds = new Set(portalFindings.map((f) => f.id));
    const linkedRequests = mockEvidenceRequests.filter((r) => r.finding_id);
    for (const req of linkedRequests) {
      expect(findingIds.has(req.finding_id!)).toBe(true);
    }
  });
});

describe('finding status values for auditee perspective', () => {
  it('all portal findings have a status recognised by the portal', () => {
    const validStatuses = [
      'open',
      'in_remediation',
      'remediated',
      'accepted_risk',
      'false_positive',
      'closed',
    ];
    for (const f of portalFindings) {
      expect(validStatuses).toContain(f.status);
    }
  });

  it('portalFindings includes at least one critical finding', () => {
    const critical = portalFindings.filter((f) => f.severity === 'critical');
    expect(critical.length).toBeGreaterThan(0);
  });

  it('portalFindings includes at least one open finding', () => {
    const open = portalFindings.filter((f) => f.status === 'open');
    expect(open.length).toBeGreaterThan(0);
  });
});
