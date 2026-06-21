'use client';
import { MetricCard, Card, CardHeader, CardBody } from '@/components/ui/card';
import { SeverityBadge, StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { mockFindings, mockEvidence } from '@/lib/mock-data';
import { mockEvidenceRequests, sortRequestsByPriority } from '@/lib/portal-mock-data';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import Link from 'next/link';
import { Upload, FileCheck, Clock, CheckCircle2, ExternalLink, AlertCircle } from 'lucide-react';

const remediationData = [
  { week: 'W1', completed: 1, remaining: 5 },
  { week: 'W2', completed: 2, remaining: 4 },
  { week: 'W3', completed: 2, remaining: 4 },
  { week: 'W4', completed: 3, remaining: 3 },
  { week: 'W5', completed: 3, remaining: 3 },
];

export function AuditeeCISODashboard() {
  const myFindings = mockFindings.filter(f => ['open', 'in_remediation'].includes(f.status));
  const pendingEvidence = mockEvidence.filter(e => e.status === 'pending_review');
  // Portal widget — pending evidence requests with deadlines
  const pendingPortalRequests = sortRequestsByPriority(
    mockEvidenceRequests.filter(r => ['pending', 'in_progress'].includes(r.status))
  );
  const overduePortalRequests = pendingPortalRequests.filter(
    r => new Date(r.due_date) < new Date()
  );

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-900">Auditee / CISO Dashboard</h2>
          <p className="text-sm text-slate-500 mt-1">Remediation tracking &amp; evidence submission</p>
        </div>
        <Link href="/evidence">
          <Button size="sm"><Upload size={14} />Submit Evidence</Button>
        </Link>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard label="Assigned Findings" value={myFindings.length} subtitle="Require remediation" accent="border-t-4 border-red-500" />
        <MetricCard label="Evidence Pending" value={pendingEvidence.length} subtitle="Awaiting review" accent="border-t-4 border-yellow-500" />
        <MetricCard label="Overdue Items" value={2} subtitle="Past due date" accent="border-t-4 border-orange-500" />
        <MetricCard label="Remediated (30d)" value={3} subtitle="Closed this month" accent="border-t-4 border-green-500" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">Remediation Progress (Weekly)</h3>
          </CardHeader>
          <CardBody>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={remediationData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                <XAxis dataKey="week" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} />
                <Tooltip />
                <Bar dataKey="completed" name="Completed" fill="#16A34A" radius={[3,3,0,0]} />
                <Bar dataKey="remaining" name="Remaining" fill="#E2E8F0" radius={[3,3,0,0]} />
              </BarChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>

        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">Evidence Submission Status</h3>
          </CardHeader>
          <CardBody className="p-0">
            <ul className="divide-y divide-slate-100">
              {mockEvidence.map(ev => (
                <li key={ev.id} className="flex items-start gap-3 px-6 py-3">
                  {ev.status === 'accepted' ? <FileCheck size={16} className="mt-0.5 text-green-500 flex-shrink-0" />
                    : ev.status === 'pending_review' ? <Clock size={16} className="mt-0.5 text-yellow-500 flex-shrink-0" />
                    : <CheckCircle2 size={16} className="mt-0.5 text-slate-300 flex-shrink-0" />}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-slate-900 truncate">{ev.title}</p>
                    <p className="text-xs text-slate-500 mt-0.5">{ev.source_type.replace('_', ' ')} · {ev.collection_date}</p>
                  </div>
                  <StatusBadge status={ev.status}>{ev.status.replace('_', ' ')}</StatusBadge>
                </li>
              ))}
            </ul>
          </CardBody>
        </Card>
      </div>

      {/* Auditee Portal widget */}
      <Card className="border-indigo-200 bg-indigo-50/40">
        <CardHeader>
          <div className="flex items-center justify-between">
            <h3 className="font-semibold text-slate-900 flex items-center gap-2">
              <AlertCircle size={16} className="text-indigo-500" />
              Auditee Portal — Evidence Requests
            </h3>
            <Link href="/portal/dashboard">
              <Button variant="secondary" size="sm">
                Open Portal <ExternalLink size={12} />
              </Button>
            </Link>
          </div>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-3 gap-3 mb-4">
            <div className="bg-white rounded-lg border border-slate-200 p-3 text-center">
              <p className="text-2xl font-bold text-slate-900">{pendingPortalRequests.length}</p>
              <p className="text-xs text-slate-500 mt-0.5">Pending Requests</p>
            </div>
            <div className="bg-white rounded-lg border border-slate-200 p-3 text-center">
              <p className="text-2xl font-bold text-red-600">{overduePortalRequests.length}</p>
              <p className="text-xs text-slate-500 mt-0.5">Overdue</p>
            </div>
            <div className="bg-white rounded-lg border border-slate-200 p-3 text-center">
              <p className="text-2xl font-bold text-slate-900">{mockEvidenceRequests.length}</p>
              <p className="text-xs text-slate-500 mt-0.5">Total Requests</p>
            </div>
          </div>
          {pendingPortalRequests.slice(0, 3).map(req => (
            <div key={req.id} className="flex items-center gap-3 py-2 border-b border-indigo-100 last:border-0">
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-slate-900 truncate">{req.title}</p>
                <p className="text-xs text-slate-500 flex items-center gap-1">
                  <Clock size={11} />
                  Due: {req.due_date}
                  {new Date(req.due_date) < new Date() && (
                    <span className="text-red-500 font-semibold ml-1">(OVERDUE)</span>
                  )}
                </p>
              </div>
              <Link href={`/portal/evidence-requests/${req.id}`}>
                <Button variant="ghost" size="sm">Respond</Button>
              </Link>
            </div>
          ))}
        </CardBody>
      </Card>

      {/* Action items */}
      <Card>
        <CardHeader>
          <h3 className="font-semibold text-slate-900">Your Action Items</h3>
        </CardHeader>
        <CardBody className="p-0">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-xs text-slate-500 uppercase tracking-wide">
              <tr>
                <th className="px-6 py-3 text-left">Finding</th>
                <th className="px-6 py-3 text-left">Severity</th>
                <th className="px-6 py-3 text-left">Status</th>
                <th className="px-6 py-3 text-left">Due Date</th>
                <th className="px-6 py-3 text-left">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {myFindings.map(f => (
                <tr key={f.id} className="hover:bg-slate-50">
                  <td className="px-6 py-3 font-medium text-slate-900 max-w-xs">
                    <p className="truncate">{f.title}</p>
                  </td>
                  <td className="px-6 py-3"><SeverityBadge severity={f.severity}>{f.severity}</SeverityBadge></td>
                  <td className="px-6 py-3"><StatusBadge status={f.status}>{f.status.replace('_', ' ')}</StatusBadge></td>
                  <td className="px-6 py-3 text-slate-500">{f.due_date ?? '—'}</td>
                  <td className="px-6 py-3">
                    <Link href="/evidence">
                      <Button variant="secondary" size="sm"><Upload size={12} />Add Evidence</Button>
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardBody>
      </Card>
    </div>
  );
}
