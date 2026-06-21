'use client';
import { MetricCard, Card, CardHeader, CardBody } from '@/components/ui/card';
import { StatusBadge, SeverityBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  mockEvidenceRequests,
  portalFindings,
  sortRequestsByPriority,
  portalEngagement,
} from '@/lib/portal-mock-data';
import {
  Clock,
  FileSearch,
  AlertTriangle,
  CheckCircle2,
  ArrowRight,
  Calendar,
} from 'lucide-react';
import Link from 'next/link';
import { clsx } from 'clsx';

const PRIORITY_BADGE: Record<string, string> = {
  urgent: 'bg-red-100 text-red-800 border border-red-200',
  high: 'bg-orange-100 text-orange-800 border border-orange-200',
  medium: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
  low: 'bg-slate-100 text-slate-600 border border-slate-200',
};

export default function PortalDashboardPage() {
  const pending = mockEvidenceRequests.filter((r) => ['pending', 'in_progress'].includes(r.status));
  const overdue = pending.filter((r) => new Date(r.due_date) < new Date());
  const activeFindings = portalFindings.filter((f) =>
    ['open', 'in_remediation'].includes(f.status),
  );
  const criticalFindings = activeFindings.filter((f) => f.severity === 'critical');
  const topRequests = sortRequestsByPriority(pending).slice(0, 3);

  return (
    <div className="p-6 space-y-6">
      {/* Page header */}
      <div>
        <h2 className="text-xl font-semibold text-slate-900">Your Dashboard</h2>
        <p className="text-sm text-slate-500 mt-1">
          Engagement: <span className="font-medium text-slate-700">{portalEngagement.name}</span>
        </p>
      </div>

      {/* Metric cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          label="Pending Requests"
          value={pending.length}
          subtitle="Awaiting your response"
          accent="border-t-4 border-blue-500"
        />
        <MetricCard
          label="Overdue"
          value={overdue.length}
          subtitle="Past due date"
          accent="border-t-4 border-red-500"
        />
        <MetricCard
          label="Active Findings"
          value={activeFindings.length}
          subtitle="Require management response"
          accent="border-t-4 border-orange-500"
        />
        <MetricCard
          label="Critical Findings"
          value={criticalFindings.length}
          subtitle="Immediate action required"
          accent="border-t-4 border-red-600"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Pending evidence requests */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                <FileSearch size={16} className="text-blue-500" />
                Pending Evidence Requests
              </h3>
              <Link href="/portal/evidence-requests">
                <Button variant="ghost" size="sm">
                  View all <ArrowRight size={13} />
                </Button>
              </Link>
            </div>
          </CardHeader>
          <CardBody className="p-0">
            {topRequests.length === 0 ? (
              <div className="px-6 py-8 text-center">
                <CheckCircle2 size={28} className="mx-auto mb-2 text-green-400" />
                <p className="text-sm text-slate-500">All requests fulfilled!</p>
              </div>
            ) : (
              <ul className="divide-y divide-slate-100">
                {topRequests.map((req) => (
                  <li key={req.id}>
                    <Link
                      href={`/portal/evidence-requests/${req.id}`}
                      className="flex items-start gap-3 px-6 py-3 hover:bg-slate-50 transition-colors"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className={clsx('text-[10px] font-semibold px-1.5 py-0.5 rounded', PRIORITY_BADGE[req.priority])}>
                            {req.priority.toUpperCase()}
                          </span>
                          <StatusBadge status={req.status}>{req.status.replace('_', ' ')}</StatusBadge>
                        </div>
                        <p className="text-sm font-medium text-slate-900 truncate">{req.title}</p>
                        <p className="text-xs text-slate-400 mt-0.5 flex items-center gap-1">
                          <Calendar size={11} />
                          Due: {req.due_date}
                          {new Date(req.due_date) < new Date() && (
                            <span className="text-red-500 font-medium ml-1">(OVERDUE)</span>
                          )}
                        </p>
                      </div>
                      <ArrowRight size={14} className="text-slate-300 mt-1 flex-shrink-0" />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>

        {/* Active findings requiring response */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                <AlertTriangle size={16} className="text-orange-500" />
                Findings Requiring Response
              </h3>
              <Link href="/portal/findings">
                <Button variant="ghost" size="sm">
                  View all <ArrowRight size={13} />
                </Button>
              </Link>
            </div>
          </CardHeader>
          <CardBody className="p-0">
            {activeFindings.length === 0 ? (
              <div className="px-6 py-8 text-center">
                <CheckCircle2 size={28} className="mx-auto mb-2 text-green-400" />
                <p className="text-sm text-slate-500">No active findings!</p>
              </div>
            ) : (
              <ul className="divide-y divide-slate-100">
                {activeFindings.slice(0, 3).map((f) => (
                  <li key={f.id}>
                    <Link
                      href={`/portal/findings/${f.id}`}
                      className="flex items-start gap-3 px-6 py-3 hover:bg-slate-50 transition-colors"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <SeverityBadge severity={f.severity}>{f.severity}</SeverityBadge>
                          <span className="font-mono text-xs text-slate-400">{f.finding_ref}</span>
                        </div>
                        <p className="text-sm font-medium text-slate-900 line-clamp-1">{f.title}</p>
                        {f.due_date && (
                          <p className="text-xs text-slate-400 mt-0.5 flex items-center gap-1">
                            <Clock size={11} />
                            Due: {f.due_date}
                          </p>
                        )}
                      </div>
                      <ArrowRight size={14} className="text-slate-300 mt-1 flex-shrink-0" />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>
      </div>

      {/* Engagement context */}
      <Card>
        <CardHeader>
          <h3 className="font-semibold text-slate-900">Engagement Context</h3>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wide mb-1">Status</p>
              <StatusBadge status={portalEngagement.status}>
                {portalEngagement.status}
              </StatusBadge>
            </div>
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wide mb-1">Start Date</p>
              <p className="font-medium text-slate-900">{portalEngagement.target_start_date}</p>
            </div>
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wide mb-1">End Date</p>
              <p className="font-medium text-slate-900">{portalEngagement.target_end_date}</p>
            </div>
            <div>
              <p className="text-xs text-slate-500 uppercase tracking-wide mb-1">Overall Score</p>
              <p className="font-medium text-slate-900">{portalEngagement.overall_score ?? 'Pending'}</p>
            </div>
          </div>
          {portalEngagement.description && (
            <p className="text-sm text-slate-500 mt-4 border-t border-slate-100 pt-4">
              {portalEngagement.description}
            </p>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
