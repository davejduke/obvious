/**
 * Tests for portal routing — validates that route files and data helpers exist
 * and that portal routes are correctly scoped to /portal/*.
 *
 * These are structural/integration tests that run without a browser runtime.
 */
import path from 'path';
import fs from 'fs';

const FRONTEND_SRC = path.resolve(__dirname, '../../');
const APP_DIR = path.join(FRONTEND_SRC, 'app');
const PORTAL_DIR = path.join(APP_DIR, 'portal');

function routeExists(relativePath: string): boolean {
  return fs.existsSync(path.join(PORTAL_DIR, relativePath));
}

describe('portal route files exist', () => {
  it('portal layout.tsx exists', () => {
    expect(routeExists('layout.tsx')).toBe(true);
  });

  it('portal root page.tsx exists', () => {
    expect(routeExists('page.tsx')).toBe(true);
  });

  it('portal/dashboard/page.tsx exists', () => {
    expect(routeExists('dashboard/page.tsx')).toBe(true);
  });

  it('portal/evidence-requests/page.tsx exists', () => {
    expect(routeExists('evidence-requests/page.tsx')).toBe(true);
  });

  it('portal/evidence-requests/[id]/page.tsx exists', () => {
    expect(routeExists('evidence-requests/[id]/page.tsx')).toBe(true);
  });

  it('portal/findings/page.tsx exists', () => {
    expect(routeExists('findings/page.tsx')).toBe(true);
  });

  it('portal/findings/[id]/page.tsx exists', () => {
    expect(routeExists('findings/[id]/page.tsx')).toBe(true);
  });
});

describe('portal component files exist', () => {
  const COMPONENTS_PORTAL = path.join(FRONTEND_SRC, 'components', 'portal');

  it('portal-shell.tsx exists', () => {
    expect(fs.existsSync(path.join(COMPONENTS_PORTAL, 'portal-shell.tsx'))).toBe(true);
  });

  it('rbac-guard.tsx exists', () => {
    expect(fs.existsSync(path.join(COMPONENTS_PORTAL, 'rbac-guard.tsx'))).toBe(true);
  });

  it('notification-bell.tsx exists', () => {
    expect(fs.existsSync(path.join(COMPONENTS_PORTAL, 'notification-bell.tsx'))).toBe(true);
  });
});

describe('portal is isolated from main app routes', () => {
  it('main app has no /portal route in its nav items', () => {
    const sidebarSrc = fs.readFileSync(
      path.join(FRONTEND_SRC, 'components', 'layout', 'sidebar.tsx'),
      'utf8',
    );
    // The main sidebar should not link to /portal routes
    expect(sidebarSrc).not.toMatch(/href.*\/portal\//);
  });

  it('portal layout uses RbacGuard', () => {
    const layoutSrc = fs.readFileSync(
      path.join(PORTAL_DIR, 'layout.tsx'),
      'utf8',
    );
    expect(layoutSrc).toContain('RbacGuard');
  });

  it('portal layout uses PortalShell (not AppShell)', () => {
    const layoutSrc = fs.readFileSync(
      path.join(PORTAL_DIR, 'layout.tsx'),
      'utf8',
    );
    expect(layoutSrc).toContain('PortalShell');
    expect(layoutSrc).not.toContain('AppShell');
  });
});
