'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  mockStrategicPlan,
  mockAnnualPlan,
  mockAssuranceMap,
  mockResourceCalendar,
} from '@/lib/mock-data';
import Link from 'next/link';
import {
  CalendarDays, Map, Users, BarChart3, ArrowRight,
  CheckCircle, Clock, AlertCircle,
} from 'lucide-react';

const modules = [
  {
    href: '/planning/strategic',
    icon: BarChart3,
    label: '3-Year Strategic Plan',
    description: 'Risk-based prioritisation of auditable entities across the planning horizon.',
    color: 'border-purple-500',
    iconBg: 'bg-purple-100',
    iconColor: 'text-purple-600',
    status: mockStrategicPlan.status,
    meta: `${mockStrategicPlan.start_year}–${mockStrategicPlan.end_year} · v${mockStrategicPlan.version} · ${mockStrategicPlan.entities.length} entities`,
  },
  {
    href: '/planning/annual',
    icon: CalendarDays,
    label: 'Annual Audit Plan',
    description: 'Scheduled engagements across the year with assigned teams.',
    color: 'border-blue-500',
    iconBg: 'bg-blue-100',
    iconColor: 'text-blue-600',
    status: mockAnnualPlan.status,
    meta: `${mockAnnualPlan.year} · v${mockAnnualPlan.version} · ${mockAnnualPlan.engagements.length} engagements`,
  },
  {
    href: '/planning/assurance-map',
    icon: Map,
    label: 'Assurance Map',
    description: 'Visual matrix of business units × control domains × coverage level.',
    color: 'border-teal-500',
    iconBg: 'bg-teal-100',
    iconColor: 'text-teal-600',
    status: 'active' as const,
    meta: `${mockAssuranceMap.year} · ${mockAssuranceMap.business_units.length} BUs · ${mockAssuranceMap.control_domains.length} domains`,
  },
  {
    href: '/planning/resource-calendar',
    icon: Users,
    label: 'Resource Calendar',
    description: 'Auditor availability, engagement assignments, and workload balancing.',
    color: 'border-orange-500',
    iconBg: 'bg-orange-100',
    iconColor: 'text-orange-600',
    status: 'active' as const,
    meta: `${mockResourceCalendar.year} · ${mockResourceCalendar.auditors.length} auditors`,
  },
];

function coverageSummary() {
  const matrix = mockAssuranceMap.matrix;
  const full = matrix.filter(c => c.coverage === 'full').length;
  const partial = matrix.filter(c => c.coverage === 'partial').length;
  const none = matrix.filter(c => c.coverage === 'none').length;
  const total = matrix.length;
  const pct = total > 0 ? Math.round((full + partial * 0.5) * 100 / total) : 0;
  return { full, partial, none, total, pct };
}

function workloadSummary() {
  const auditors = mockResourceCalendar.auditors;
  const allocated = auditors.map(a => a.assignments.reduce((s, x) => s + x.allocated_days, 0));
  const available = auditors.map(a => a.available_days);
  const totalAlloc = allocated.reduce((s, v) => s + v, 0);
  const totalAvail = available.reduce((s, v) => s + v, 0);
  const pct = totalAvail > 0 ? Math.round(totalAlloc * 100 / totalAvail) : 0;
  return { totalAlloc, totalAvail, pct };
}

export default function PlanOverviewPage() {
  const coverage = coverageSummary();
  const workload = workloadSummary();
  const completed = mockAnnualPlan.engagements.filter(e => e.status === 'completed').length;
  const inProgress = mockAnnualPlan.engagements.filter(e => e.status === 'in_progress').length;
  const planned = mockAnnualPlan.engagements.filter(e => e.status === 'planned').length;

  return (
    <AppShell title="Audit Planning">
      <div className="p-6 space-y-6">

        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">Audit Planning</h1>
            <p className="mt-1 text-sm text-slate-500">
              Manage your strategic plan, annual schedules, assurance coverage, and resource allocation.
            </p>
          </div>
        </div>

        {/* KPI row */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <MetricCard
            label="Engagements"
            value={mockAnnualPlan.engagements.length}
            subtitle={`${completed} completed · ${inProgress} in progress`}
            accent="border-blue-500"
          />
          <MetricCard
            label="Coverage"
            value={`${coverage.pct}%`}
            subtitle={`${coverage.full} full · ${coverage.partial} partial`}
            accent="border-teal-500"
          />
          <MetricCard
            label="Utilisation"
            value={`${workload.pct}%`}
            subtitle={`${workload.totalAlloc}/${workload.totalAvail} days allocated`}
            accent="border-orange-500"
          />
          <MetricCard
            label="Planned"
            value={planned}
            subtitle="engagements not yet started"
            accent="border-purple-500"
          />
        </div>

        {/* Module cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {modules.map((mod) => {
            const Icon = mod.icon;
            return (
              <Card key={mod.href} className={`border-t-4 ${mod.color} hover:shadow-md transition-shadow`}>
                <CardBody className="p-5">
                  <div className="flex items-start gap-4">
                    <div className={`flex-shrink-0 w-10 h-10 ${mod.iconBg} rounded-lg flex items-center justify-center`}>
                      <Icon size={20} className={mod.iconColor} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <h2 className="text-sm font-semibold text-slate-900">{mod.label}</h2>
                        <StatusBadge status={mod.status}>{mod.status}</StatusBadge>
                      </div>
                      <p className="mt-1 text-xs text-slate-500">{mod.description}</p>
                      <p className="mt-2 text-xs text-slate-400">{mod.meta}</p>
                    </div>
                  </div>
                  <div className="mt-4 flex justify-end">
                    <Link href={mod.href}>
                      <Button variant="ghost" size="sm">
                        Open <ArrowRight size={14} />
                      </Button>
                    </Link>
                  </div>
                </CardBody>
              </Card>
            );
          })}
        </div>

        {/* Annual plan quick view */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-slate-900">Annual Plan Progress — {mockAnnualPlan.year}</h2>
              <Link href="/planning/annual">
                <Button variant="ghost" size="sm">View full plan <ArrowRight size={14} /></Button>
              </Link>
            </div>
          </CardHeader>
          <CardBody className="divide-y divide-slate-100">
            {mockAnnualPlan.engagements.map((eng) => {
              const StatusIcon = eng.status === 'completed' ? CheckCircle : eng.status === 'in_progress' ? Clock : AlertCircle;
              const iconColor = eng.status === 'completed' ? 'text-green-500' : eng.status === 'in_progress' ? 'text-blue-500' : 'text-slate-400';
              return (
                <div key={eng.id} className="flex items-center gap-3 py-3">
                  <StatusIcon size={16} className={`flex-shrink-0 ${iconColor}`} />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-slate-900 truncate">{eng.name}</p>
                    <p className="text-xs text-slate-500">{eng.auditable_entity} · Q{eng.quarter} · {eng.budget_days}d</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <StatusBadge status={eng.status}>{eng.status.replace('_', ' ')}</StatusBadge>
                  </div>
                </div>
              );
            })}
          </CardBody>
        </Card>

      </div>
    </AppShell>
  );
}
