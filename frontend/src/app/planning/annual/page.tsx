'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ApprovalBanner, type ApprovalWorkflow } from '@/components/ui/approval-banner';
import { mockAnnualPlan, type PlannedEngagement, type EngPlanStatus } from '@/lib/mock-data';
import { useState } from 'react';
import Link from 'next/link';
import { ArrowLeft, CalendarDays, CheckCircle2, Circle, Clock, Ban } from 'lucide-react';
import { clsx } from 'clsx';

const statusIcon: Record<EngPlanStatus, React.ElementType> = {
  completed: CheckCircle2,
  in_progress: Clock,
  planned: Circle,
  deferred: Ban,
};
const statusColor: Record<EngPlanStatus, string> = {
  completed: 'text-green-500',
  in_progress: 'text-blue-500',
  planned: 'text-slate-400',
  deferred: 'text-red-400',
};

const quarterColors = ['bg-blue-50 border-blue-200', 'bg-purple-50 border-purple-200', 'bg-amber-50 border-amber-200', 'bg-teal-50 border-teal-200'];

function QuarterSection({ quarter, engagements }: { quarter: number; engagements: PlannedEngagement[] }) {
  const totalDays = engagements.reduce((s, e) => s + e.budget_days, 0);
  return (
    <div className={clsx('rounded-lg border p-4', quarterColors[quarter - 1])}>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-slate-800">Q{quarter}</h3>
        <span className="text-xs text-slate-500">{engagements.length} engagements · {totalDays}d budget</span>
      </div>
      <div className="space-y-2">
        {engagements.map((eng) => {
          const Icon = statusIcon[eng.status];
          return (
            <div key={eng.id} className="flex items-start gap-2 bg-white rounded-md border border-white/60 p-3 shadow-sm">
              <Icon size={16} className={clsx('flex-shrink-0 mt-0.5', statusColor[eng.status])} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-sm font-medium text-slate-900 truncate">{eng.name}</p>
                  <StatusBadge status={eng.status}>{eng.status.replace('_', ' ')}</StatusBadge>
                </div>
                <p className="text-xs text-slate-500 mt-0.5">{eng.auditable_entity}</p>
                <div className="flex items-center gap-3 mt-1 text-xs text-slate-400">
                  <span>{eng.start_date} → {eng.end_date}</span>
                  <span>{eng.budget_days}d</span>
                  {eng.assigned_team.length > 0 && (
                    <span>{eng.assigned_team.slice(0, 2).join(', ')}{eng.assigned_team.length > 2 ? ` +${eng.assigned_team.length - 2}` : ''}</span>
                  )}
                </div>
              </div>
            </div>
          );
        })}
        {engagements.length === 0 && (
          <p className="text-xs text-slate-400 italic">No engagements scheduled</p>
        )}
      </div>
    </div>
  );
}

export default function AnnualPlanPage() {
  const plan = mockAnnualPlan;
  const [activeView, setActiveView] = useState<'quarterly' | 'list'>('quarterly');

  const workflow: ApprovalWorkflow = {
    id: 'wf-ap-001',
    workflow_type: 'audit_plan',
    status: 'approved',
    submitted_by_email: 'el@example.com',
    submitted_at: '2024-12-15T10:00:00Z',
    decided_by_email: 'cae@example.com',
    decided_at: '2024-12-18T14:00:00Z',
  };

  const completed = plan.engagements.filter(e => e.status === 'completed').length;
  const inProgress = plan.engagements.filter(e => e.status === 'in_progress').length;
  const totalDays = plan.engagements.reduce((s, e) => s + e.budget_days, 0);
  const completedDays = plan.engagements.filter(e => e.status === 'completed').reduce((s, e) => s + e.budget_days, 0);

  return (
    <AppShell title="Annual Audit Plan">
      <div className="p-6 space-y-6">

        <Link href="/planning" className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800">
          <ArrowLeft size={14} /> Back to Planning
        </Link>

        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
              <CalendarDays size={20} className="text-blue-600" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-slate-900">{plan.name}</h1>
              <div className="flex items-center gap-2 mt-0.5">
                <StatusBadge status={plan.status}>{plan.status}</StatusBadge>
                <span className="text-xs text-slate-400">Version {plan.version}</span>
              </div>
            </div>
          </div>
          <Button variant="secondary" size="sm">Edit Plan</Button>
        </div>

        <ApprovalBanner workflow={workflow} userRole="chief_audit_executive" />

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <MetricCard label="Total Engagements" value={plan.engagements.length} accent="border-blue-500" />
          <MetricCard label="Completed" value={completed} accent="border-green-500" />
          <MetricCard label="In Progress" value={inProgress} accent="border-orange-500" />
          <MetricCard label="Budget Days" value={totalDays} subtitle={`${completedDays}d used`} accent="border-purple-500" />
        </div>

        {/* View toggle */}
        <div className="flex gap-2">
          <Button
            variant={activeView === 'quarterly' ? 'primary' : 'secondary'}
            size="sm"
            onClick={() => setActiveView('quarterly')}
          >
            Quarterly View
          </Button>
          <Button
            variant={activeView === 'list' ? 'primary' : 'secondary'}
            size="sm"
            onClick={() => setActiveView('list')}
          >
            List View
          </Button>
        </div>

        {activeView === 'quarterly' ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {[1, 2, 3, 4].map(q => (
              <QuarterSection
                key={q}
                quarter={q}
                engagements={plan.engagements.filter(e => e.quarter === q)}
              />
            ))}
          </div>
        ) : (
          <Card>
            <CardHeader>
              <h2 className="text-sm font-semibold text-slate-900">All Engagements</h2>
            </CardHeader>
            <CardBody className="divide-y divide-slate-100 p-0">
              {plan.engagements.map((eng) => {
                const Icon = statusIcon[eng.status];
                return (
                  <div key={eng.id} className="flex items-start gap-3 px-6 py-4">
                    <Icon size={16} className={clsx('flex-shrink-0 mt-0.5', statusColor[eng.status])} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between gap-2">
                        <p className="text-sm font-medium text-slate-900">{eng.name}</p>
                        <StatusBadge status={eng.status}>{eng.status.replace('_', ' ')}</StatusBadge>
                      </div>
                      <p className="text-xs text-slate-500 mt-0.5">{eng.auditable_entity} · Q{eng.quarter} · {eng.start_date} → {eng.end_date}</p>
                      <p className="text-xs text-slate-400 mt-0.5">{eng.budget_days}d budget · {eng.assigned_team.join(', ')}</p>
                    </div>
                  </div>
                );
              })}
            </CardBody>
          </Card>
        )}

      </div>
    </AppShell>
  );
}
