/**
 * Tests for Auditee Portal RBAC enforcement.
 *
 * These tests validate the RBAC logic in isolation (pure function tests)
 * so they have no Next.js rendering dependencies.
 */
import { isPortalAuthorized, PORTAL_ALLOWED_PERSONAS } from '@/store/portal';
import type { Persona } from '@shared/index';

describe('isPortalAuthorized', () => {
  it('returns true for auditee_ciso', () => {
    expect(isPortalAuthorized('auditee_ciso')).toBe(true);
  });

  it('returns false for internal_auditor', () => {
    expect(isPortalAuthorized('internal_auditor')).toBe(false);
  });

  it('returns false for cae', () => {
    expect(isPortalAuthorized('cae')).toBe(false);
  });

  it('returns false for audit_committee', () => {
    expect(isPortalAuthorized('audit_committee')).toBe(false);
  });

  it('returns false for cosourced_provider', () => {
    expect(isPortalAuthorized('cosourced_provider')).toBe(false);
  });

  it('returns false for ptwg_member', () => {
    expect(isPortalAuthorized('ptwg_member')).toBe(false);
  });

  it('returns false for beta_tester', () => {
    expect(isPortalAuthorized('beta_tester')).toBe(false);
  });

  it('PORTAL_ALLOWED_PERSONAS contains auditee_ciso', () => {
    expect(PORTAL_ALLOWED_PERSONAS).toContain('auditee_ciso');
  });

  it('non-auditee personas are not in PORTAL_ALLOWED_PERSONAS', () => {
    const nonAuditee: Persona[] = [
      'internal_auditor',
      'cae',
      'audit_committee',
      'cosourced_provider',
      'ptwg_member',
      'beta_tester',
    ];
    for (const p of nonAuditee) {
      expect(PORTAL_ALLOWED_PERSONAS).not.toContain(p);
    }
  });
});
