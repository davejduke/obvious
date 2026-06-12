'use client';
import { AppShell } from '@/components/layout/app-shell';
import { usePersonaStore } from '@/store/persona';
import { InternalAuditorDashboard } from './personas/internal-auditor';
import { CAEDashboard } from './personas/cae';
import { AuditCommitteeDashboard } from './personas/audit-committee';
import { AuditeeCISODashboard } from './personas/auditee-ciso';
import { DefaultDashboard } from './personas/default';

export default function DashboardPage() {
  const { currentPersona } = usePersonaStore();

  const renderDashboard = () => {
    switch (currentPersona) {
      case 'internal_auditor': return <InternalAuditorDashboard />;
      case 'cae': return <CAEDashboard />;
      case 'audit_committee': return <AuditCommitteeDashboard />;
      case 'auditee_ciso': return <AuditeeCISODashboard />;
      default: return <DefaultDashboard />;
    }
  };

  return (
    <AppShell title="Dashboard">
      {renderDashboard()}
    </AppShell>
  );
}
