'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { mockResourceCalendar, type AuditorAllocation } from '@/lib/mock-data';
import Link from 'next/link';
import { ArrowLeft, Users, AlertTriangle } from 'lucide-react';
import { clsx } from 'clsx';

const QUARTER_MONTHS: Record<number, string> = { 1: 'Jan–Mar', 2: 'Apr–Jun', 3: 'Jul–Sep', 4: 'Oct–Dec' };

function utilisationColor(pct: number): string {
  if (pct >= 90) return 'bg-red-500';
  if (pct >= 75) return 'bg-orange-500';
  if (pct >= 50) return 'bg-yellow-400';
  return 'bg-green-500';
}

function utilisationTextColor(pct: number): string {
  if (pct >= 90) return 'text-red-700';
  if (pct >= 75) return 'text-orange-700';
  return 'text-slate-700';
}

function AuditorCard({ auditor }: { auditor: AuditorAllocation }) {
  const allocatedDays = auditor.assignments.reduce((s, a) => s + a.allocated_days, 0);
  const pct = auditor.available_days > 0
    ? Math.min(100, Math.round(allocatedDays * 100 / auditor.available_days))
    : 0;
  const isOverloaded = allocatedDays > auditor.available_days;

  return (
    <div className="bg-white rounded-lg border border-slate-200 shadow-sm p-4">
      {/* Auditor header */}
      <div className="flex items-start justify-between gap-2 mb-3">
        <div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-slate-700 rounded-full flex items-center justify-center text-white text-xs font-bold">
              {auditor.auditor_name.split(' ').map(n => n[0]).join('')}
            </div>
            <div>
              <p className="text-sm font-semibold text-slate-900">{auditor.auditor_name}</p>
              <p className="text-xs text-slate-500">{auditor.role}</p>
            </div>
          </div>
        </div>
        <div className="text-right">
          <p className={clsx('text-sm font-bold', utilisationTextColor(pct))}>
            {pct}%
            {isOverloaded && <AlertTriangle size={12} className="inline ml-1 text-red-500" />}
          </p>
          <p className="text-xs text-slate-400">{allocatedDays}/{auditor.available_days}d</p>
        </div>
      </div>

      {/* Utilisation bar */}
      <div className="h-2 bg-slate-100 rounded-full overflow-hidden mb-3">
        <div
          className={clsx('h-full rounded-full transition-all', utilisationColor(pct))}
          style={{ width: `${pct}%` }}
        />
      </div>

      {/* Assignments */}
      {auditor.assignments.length > 0 ? (
        <div className="space-y-1.5">
          {auditor.assignments.map((asgn) => (
            <div key={asgn.id} className="flex items-center justify-between gap-2 text-xs bg-slate-50 rounded px-2 py-1.5">
              <div className="flex-1 min-w-0">
                <p className="font-medium text-slate-700 truncate">{asgn.engagement_name}</p>
                <p className="text-slate-400">Q{asgn.quarter} · {asgn.start_date} → {asgn.end_date}</p>
              </div>
              <span className="text-slate-500 font-medium flex-shrink-0">{asgn.allocated_days}d</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-xs text-slate-400 italic">No assignments</p>
      )}
    </div>
  );
}

function QuarterSummary({ quarter, auditors }: { quarter: number; auditors: AuditorAllocation[] }) {
  const assignments = auditors.flatMap(a => a.assignments.filter(x => x.quarter === quarter));
  const totalDays = assignments.reduce((s, a) => s + a.allocated_days, 0);
  const engagements = Array.from(new Set(assignments.map(a => a.engagement_name)));
  return (
    <div className="bg-white rounded-lg border border-slate-200 p-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold text-slate-800">Q{quarter} <span className="font-normal text-slate-500">({QUARTER_MONTHS[quarter]})</span></h3>
        <span className="text-xs text-slate-500">{totalDays}d</span>
      </div>
      {engagements.length > 0 ? (
        <ul className="space-y-1">
          {engagements.map(e => (
            <li key={e} className="text-xs text-slate-600 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-400 flex-shrink-0" />
              {e}
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-xs text-slate-400 italic">No engagements</p>
      )}
    </div>
  );
}

export default function ResourceCalendarPage() {
  const rc = mockResourceCalendar;

  const totalAllocated = rc.auditors.flatMap(a => a.assignments).reduce((s, a) => s + a.allocated_days, 0);
  const totalAvailable = rc.auditors.reduce((s, a) => s + a.available_days, 0);
  const teamUtilPct = totalAvailable > 0 ? Math.round(totalAllocated * 100 / totalAvailable) : 0;
  const overloaded = rc.auditors.filter(a => {
    const alloc = a.assignments.reduce((s, x) => s + x.allocated_days, 0);
    return alloc > a.available_days;
  }).length;

  return (
    <AppShell title="Resource Calendar">
      <div className="p-6 space-y-6">

        <Link href="/planning" className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800">
          <ArrowLeft size={14} /> Back to Planning
        </Link>

        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-orange-100 rounded-lg flex items-center justify-center">
              <Users size={20} className="text-orange-600" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-slate-900">{rc.name}</h1>
              <p className="text-xs text-slate-500 mt-0.5">
                {rc.year} · {rc.auditors.length} auditors
              </p>
            </div>
          </div>
          <Button variant="secondary" size="sm">Edit Calendar</Button>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <MetricCard label="Team Utilisation" value={`${teamUtilPct}%`} subtitle="of available days" accent="border-orange-500" />
          <MetricCard label="Total Auditors" value={rc.auditors.length} accent="border-blue-500" />
          <MetricCard label="Days Allocated" value={totalAllocated} subtitle={`of ${totalAvailable} available`} accent="border-purple-500" />
          <MetricCard label="Overloaded" value={overloaded} subtitle="auditors over capacity" accent={overloaded > 0 ? 'border-red-500' : 'border-green-500'} />
        </div>

        {/* Quarterly summary */}
        <div>
          <h2 className="text-sm font-semibold text-slate-900 mb-3">Quarterly Overview</h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {[1, 2, 3, 4].map(q => (
              <QuarterSummary key={q} quarter={q} auditors={rc.auditors} />
            ))}
          </div>
        </div>

        {/* Auditor cards */}
        <div>
          <h2 className="text-sm font-semibold text-slate-900 mb-3">Auditor Workload</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {rc.auditors.map(auditor => (
              <AuditorCard key={auditor.auditor_id} auditor={auditor} />
            ))}
          </div>
        </div>

      </div>
    </AppShell>
  );
}
