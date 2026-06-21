'use client';
import { useState } from 'react';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  mockEvidenceRequests,
  sortRequestsByPriority,
  PORTAL_PRIORITY_ORDER,
} from '@/lib/portal-mock-data';
import type { EvidenceRequest } from '@shared/index';
import { Search, Calendar, User, ArrowRight, CheckCircle2 } from 'lucide-react';
import Link from 'next/link';
import { clsx } from 'clsx';

const PRIORITY_STYLES: Record<EvidenceRequest['priority'], string> = {
  urgent: 'bg-red-100 text-red-800 border border-red-200',
  high: 'bg-orange-100 text-orange-800 border border-orange-200',
  medium: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
  low: 'bg-slate-100 text-slate-600 border border-slate-200',
};

const STATUS_FILTERS = ['all', 'pending', 'in_progress', 'submitted', 'accepted', 'rejected'] as const;
type StatusFilter = (typeof STATUS_FILTERS)[number];

export default function EvidenceRequestsPage() {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');

  const pendingCount = mockEvidenceRequests.filter((r) =>
    ['pending', 'in_progress'].includes(r.status),
  ).length;
  const submittedCount = mockEvidenceRequests.filter((r) => r.status === 'submitted').length;
  const acceptedCount = mockEvidenceRequests.filter((r) => r.status === 'accepted').length;
  const overdueCount = mockEvidenceRequests.filter(
    (r) => ['pending', 'in_progress'].includes(r.status) && new Date(r.due_date) < new Date(),
  ).length;

  const filtered = sortRequestsByPriority(
    mockEvidenceRequests.filter((r) => {
      if (statusFilter !== 'all' && r.status !== statusFilter) return false;
      if (search && !r.title.toLowerCase().includes(search.toLowerCase())) return false;
      return true;
    }),
  );

  return (
    <div className="p-6 space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-slate-900">Evidence Requests</h2>
        <p className="text-sm text-slate-500 mt-1">
          Requests from the audit team — sorted by priority.
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          label="Pending"
          value={pendingCount}
          subtitle="Awaiting your action"
          accent="border-t-4 border-blue-500"
        />
        <MetricCard
          label="Overdue"
          value={overdueCount}
          subtitle="Past due date"
          accent="border-t-4 border-red-500"
        />
        <MetricCard
          label="Submitted"
          value={submittedCount}
          subtitle="Under review"
          accent="border-t-4 border-yellow-500"
        />
        <MetricCard
          label="Accepted"
          value={acceptedCount}
          subtitle="Fulfilled"
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
            placeholder="Search requests..."
            className="pl-9 pr-4 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 w-52"
          />
        </div>
        <div className="flex gap-1">
          {STATUS_FILTERS.map((s) => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={clsx(
                'px-2.5 py-1 text-xs font-medium rounded transition-colors',
                statusFilter === s
                  ? 'bg-indigo-600 text-white'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200',
              )}
            >
              {s === 'all' ? 'All' : s.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
            </button>
          ))}
        </div>
      </div>

      {/* Request list */}
      <div className="space-y-3">
        {filtered.length === 0 && (
          <div className="text-center py-12 text-slate-400">
            <CheckCircle2 size={32} className="mx-auto mb-2 text-slate-300" />
            <p>No requests match your filters.</p>
          </div>
        )}
        {filtered.map((req) => {
          const isOverdue =
            ['pending', 'in_progress'].includes(req.status) &&
            new Date(req.due_date) < new Date();
          return (
            <Card
              key={req.id}
              className={clsx(
                'hover:shadow-md transition-shadow',
                req.priority === 'urgent' && 'border-l-4 border-l-red-500',
                req.priority === 'high' && 'border-l-4 border-l-orange-500',
              )}
            >
              <CardBody>
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1.5">
                      <span
                        className={clsx(
                          'text-[10px] font-semibold px-1.5 py-0.5 rounded uppercase tracking-wide',
                          PRIORITY_STYLES[req.priority],
                        )}
                      >
                        {req.priority}
                      </span>
                      <StatusBadge status={req.status}>
                        {req.status.replace('_', ' ')}
                      </StatusBadge>
                      {isOverdue && (
                        <span className="text-[10px] font-bold text-red-600 bg-red-50 border border-red-200 px-1.5 py-0.5 rounded">
                          OVERDUE
                        </span>
                      )}
                    </div>
                    <h4 className="font-semibold text-slate-900">{req.title}</h4>
                    <p className="text-sm text-slate-500 mt-1 line-clamp-2">{req.description}</p>
                    <div className="flex items-center gap-4 mt-2 text-xs text-slate-400">
                      <span className="flex items-center gap-1">
                        <Calendar size={11} />
                        Due: {req.due_date}
                      </span>
                      <span className="flex items-center gap-1">
                        <User size={11} />
                        {req.requested_by_email}
                      </span>
                      {req.evidence_ids.length > 0 && (
                        <span className="text-green-600">
                          {req.evidence_ids.length} file(s) attached
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex-shrink-0">
                    <Link href={`/portal/evidence-requests/${req.id}`}>
                      <Button
                        variant={
                          req.status === 'pending'
                            ? 'primary'
                            : req.status === 'in_progress'
                            ? 'primary'
                            : 'secondary'
                        }
                        size="sm"
                      >
                        {req.status === 'pending'
                          ? 'Respond'
                          : req.status === 'in_progress'
                          ? 'Continue'
                          : 'View'}
                        <ArrowRight size={13} />
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
