import type { Metadata } from 'next';
import { PortalShell } from '@/components/portal/portal-shell';
import { RbacGuard } from '@/components/portal/rbac-guard';

export const metadata: Metadata = {
  title: 'AIAUDITOR — Auditee Portal',
  description: 'Auditee portal for evidence submission and finding response',
};

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  return (
    <PortalShell>
      <RbacGuard>
        {children}
      </RbacGuard>
    </PortalShell>
  );
}
