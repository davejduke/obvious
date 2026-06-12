'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardBody, MetricCard } from '@/components/ui/card';
import { SeverityBadge, StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { mockFindings } from '@/lib/mock-data';
import { useState } from 'react';
import { Search, SlidersHorizontal, ChevronDown, ChevronUp } from 'lucide-react';
import { clsx } from 'clsx';
import type { FindingSeverity } from '@shared/index';

const severities: Array<FindingSeverity | 'all'> = ['all', 'critical', 'high', 'medium', 'low', 'informational'];
const statuses = ['all', 'open', 'in_remediation', 'remediated', 'accepted_risk', 'closed'];

export default function FindingsPage() {
  const [severityFilter, setSeverityFilter] = useState<FindingSeverity | 'all'>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [search, setSearch] = useState('');
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const displayed = mockFindings.filter(f => {
    if (severityFilter !== 'all' && f.severity !== severityFilter) return false;
    if (statusFilter !== 'all' && f.status !== statusFilter) return false;
    if (search && !f.title.toLowerCase().includes(search.toLowerCase()) && !f.finding_ref.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const criticalCount = mockFindings.filter(f => f.severity === 'critical').length;
  const highCount = mockFindings.filter(f => f.severity === 'high').length;
  const openCount = mockFindings.filter(f => f.status === 'open').length;
  const remCount = mockFindings.filter(f => f.status === 'in_remediation').length;

  return (
    <AppShell title="Findings">
      <div className="p-6 space-y-6">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard label="Total Findings" value={mockFindings.length} accent="border-t-4 border-slate-400" />
          <MetricCard label="Critical" value={criticalCount} accent="border-t-4 border-red-600" />
          <MetricCard label="High" value={highCount} accent="border-t-4 border-orange-500" />
          <MetricCard label="Open / Active" value={openCount + remCount} subtitle={`${openCount} open, ${remCount} in remediation`} accent="border-t-4 border-yellow-500" />
        </div>

        {/* Filters */}
        <div className="bg-white rounded-lg border border-slate-200 p-4 space-y-3">
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input value={search} onChange={e => setSearch(e.target.value)}
                placeholder="Search findings..."
                className="pl-9 pr-4 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 w-56" />
            </div>
            <div className="flex items-center gap-1.5">
              <SlidersHorizontal size={14} className="text-slate-400" />
              <span className="text-xs text-slate-500">Severity:</span>
              {severities.map(s => (
                <button key={s} onClick={() => setSeverityFilter(s)}
                  className={clsx(
                    'px-2.5 py-1 text-xs font-medium rounded transition-colors',
                    severityFilter === s
                      ? s === 'all' ? 'bg-slate-700 text-white' : s === 'critical' ? 'bg-red-600 text-white' : s === 'high' ? 'bg-orange-500 text-white' : s === 'medium' ? 'bg-yellow-500 text-white' : s === 'low' ? 'bg-green-600 text-white' : 'bg-blue-500 text-white'
                      : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                  )}>
                  {s === 'all' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
                </button>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-xs text-slate-500">Status:</span>
            {statuses.map(s => (
              <button key={s} onClick={() => setStatusFilter(s)}
                className={clsx(
                  'px-2.5 py-1 text-xs font-medium rounded transition-colors',
                  statusFilter === s ? 'bg-blue-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                )}>
                {s === 'all' ? 'All' : s.replace('_', ' ').replace(/\b\w/g, c => c.toUpperCase())}
              </button>
            ))}
          </div>
        </div>

        {/* Findings list */}
        <div className="space-y-2">
          {displayed.length === 0 && (
            <div className="text-center py-16 text-slate-400">No findings match the current filters.</div>
          )}
          {displayed.map(f => (
            <Card key={f.id} className={clsx('hover:shadow-md transition-shadow', f.severity === 'critical' && 'border-l-4 border-l-red-600', f.severity === 'high' && 'border-l-4 border-l-orange-500')}>
              <CardBody>
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-mono text-xs text-slate-400">{f.finding_ref}</span>
                      <SeverityBadge severity={f.severity}>{f.severity}</SeverityBadge>
                      <StatusBadge status={f.status}>{f.status.replace('_', ' ')}</StatusBadge>
                    </div>
                    <h4 className="font-semibold text-slate-900">{f.title}</h4>
                    {expandedId === f.id && (
                      <div className="mt-3 space-y-2 text-sm">
                        <p className="text-slate-600">{f.description}</p>
                        {f.root_cause && <div><span className="font-medium text-slate-700">Root Cause:</span><span className="text-slate-500 ml-1">{f.root_cause}</span></div>}
                        {f.impact && <div><span className="font-medium text-slate-700">Impact:</span><span className="text-slate-500 ml-1">{f.impact}</span></div>}
                        <div className="flex gap-1 flex-wrap">
                          {f.tags.map(t => <span key={t} className="text-xs bg-slate-100 text-slate-500 px-1.5 py-0.5 rounded">{t}</span>)}
                        </div>
                      </div>
                    )}
                  </div>
                  <div className="flex-shrink-0 flex flex-col items-end gap-2">
                    {f.due_date && <p className="text-xs text-slate-400">Due: {f.due_date}</p>}
                    <div className="flex gap-2">
                      <Button variant="ghost" size="sm" onClick={() => setExpandedId(expandedId === f.id ? null : f.id)}>
                        {expandedId === f.id ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                        {expandedId === f.id ? 'Less' : 'Details'}
                      </Button>
                      <Button variant="secondary" size="sm">Manage</Button>
                    </div>
                  </div>
                </div>
              </CardBody>
            </Card>
          ))}
        </div>

        <p className="text-xs text-slate-400 text-center">{displayed.length} of {mockFindings.length} findings shown</p>
      </div>
    </AppShell>
  );
}
