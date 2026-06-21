'use client';
import { useState } from 'react';
import { Card, CardBody, MetricCard } from '@/components/ui/card';
import { SeverityBadge, StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  portalFindings,
  mockManagementResponses,
  getControlForFinding,
} from '@/lib/portal-mock-data';
import type { FindingSeverity } from '@shared/index';
import { Search, ArrowRight, CheckCircle2, MessageSquare } from 'lucide-react';
import Link from 'next/link';
import { clsx } from 'clsx';

const SEVERITY_FILTERS: Array<FindingSeverity | 'all'> = [
  'all', 'critical', 'high', 'medium', 'low', 'informational',
];

export default function PortalFindingsPage() {
  const [search, setSearch] = useState('');
  const [severityFilter, setSeverityFilter] = useState<FindingSeverity | 'all'>('all');

  const responseMap = new Set(mockManagementResponses.map((r) => r.finding_id));

  const filtered = portalFindings.filter((f) => {
    if (severityFilter !== 'all' && f.severity !== severityFilter) return false;
    if (
      search &&
      !f.title.toLowerCase().includes(search.toLowerCase()) &&
      !f.finding_ref.toLowerCase().includes(search.toLowerCase())
    )
      return false;
    return true;
  });

  const criticalCount = portalFindings.filter((f) => f.severity === 'critical').length;
  const highCount = portalFindings.filter((f) => f.severity === 'high').length;
  const openCount = portalFindings.filter((f) => f.status === 'open').length;
  const responseCount = responseMap.size;

  return (
    <div className="p-6 space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-slate-900">Findings</h2>
        <p className="text-sm text-slate-500 mt-1">
          Audit findings assigned to your organisation. Click a finding to review context and submit
          your management response.
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          label="Total Findings"
          value={portalFindings.length}
          accent="border-t-4 border-slate-400"
        />
        <MetricCard
          label="Critical"
          value={criticalCount}
          accent="border-t-4 border-red-600"
        />
        <MetricCard
          label="High"
          value={highCount}
          accent="border-t-4 border-orange-500"
        />
        <MetricCard
          label="Responses Submitted"
          value={responseCount}
          subtitle={`of ${portalFindings.length} findings`}
          accent="border-t-4 border-green-500"
        />
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg border border-slate-200 p-4 flex flex-wrap items-center gap-3">
        <div className="relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search findings..."
            className="pl-9 pr-4 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 w-52"
          />
        </div>
        <div className="flex gap-1 flex-wrap">
          {SEVERITY_FILTERS.map((s) => (
            <button
              key={s}
              onClick={() => setSeverityFilter(s)}
              className={clsx(
                'px-2.5 py-1 text-xs font-medium rounded transition-colors',
                severityFilter === s
                  ? s === 'all'
                    ? 'bg-slate-700 text-white'
                    : s === 'critical'
                    ? 'bg-red-600 text-white'
                    : s === 'high'
                    ? 'bg-orange-500 text-white'
                    : s === 'medium'
                    ? 'bg-yellow-500 text-white'
                    : 'bg-green-600 text-white'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200',
              )}
            >
              {s === 'all' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Findings list */}
      <div className="space-y-3">
        {filtered.length === 0 && (
          <div className="text-center py-12 text-slate-400">No findings match your filters.</div>
        )}
        {filtered.map((f) => {
          const hasResponse = responseMap.has(f.id);
          const control = getControlForFinding(f.control_id);
          return (
            <Card
              key={f.id}
              className={clsx(
                'hover:shadow-md transition-shadow',
                f.severity === 'critical' && 'border-l-4 border-l-red-600',
                f.severity === 'high' && 'border-l-4 border-l-orange-500',
              )}
            >
              <CardBody>
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1.5">
                      <span className="font-mono text-xs text-slate-400">{f.finding_ref}</span>
                      <SeverityBadge severity={f.severity}>{f.severity}</SeverityBadge>
                      <StatusBadge status={f.status}>{f.status.replace('_', ' ')}</StatusBadge>
                      {hasResponse && (
                        <span className="inline-flex items-center gap-1 text-xs text-green-700 bg-green-50 border border-green-200 px-1.5 py-0.5 rounded">
                          <CheckCircle2 size={11} /> Response submitted
                        </span>
                      )}
                    </div>
                    <h4 className="font-semibold text-slate-900">{f.title}</h4>
                    <p className="text-sm text-slate-500 mt-1 line-clamp-2">{f.description}</p>
                    {control && (
                      <p className="text-xs text-indigo-600 mt-1.5 flex items-center gap-1">
                        <span className="font-mono bg-indigo-50 px-1 rounded">{control.control_id}</span>
                        {control.title}
                      </p>
                    )}
                  </div>
                  <div className="flex-shrink-0 flex flex-col items-end gap-2">
                    {f.due_date && (
                      <p className="text-xs text-slate-400">Due: {f.due_date}</p>
                    )}
                    <Link href={`/portal/findings/${f.id}`}>
                      <Button variant={hasResponse ? 'secondary' : 'primary'} size="sm">
                        {hasResponse ? (
                          <>
                            <MessageSquare size={13} /> View Response
                          </>
                        ) : (
                          <>
                            Respond <ArrowRight size={13} />
                          </>
                        )}
                      </Button>
                    </Link>
                  </div>
                </div>
              </CardBody>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
