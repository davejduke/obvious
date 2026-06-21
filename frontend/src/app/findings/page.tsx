'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardBody, MetricCard } from '@/components/ui/card';
import { SeverityBadge, StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ApprovalBanner, type ApprovalWorkflow } from '@/components/ui/approval-banner';
import { SignOffModal, type SignOffDecision, type SignOffPayload } from '@/components/ui/sign-off-modal';
import { mockFindings } from '@/lib/mock-data';
import { useState } from 'react';
import { Search, SlidersHorizontal, ChevronDown, ChevronUp, CheckCircle2 } from 'lucide-react';
import { clsx } from 'clsx';
import type { FindingSeverity } from '@shared/index';

const severities: Array<FindingSeverity | 'all'> = ['all', 'critical', 'high', 'medium', 'low', 'informational'];
const statuses = ['all', 'open', 'in_remediation', 'remediated', 'accepted_risk', 'closed'];

// Mock the current user context — in production this comes from JWT / auth provider.
const CURRENT_USER_ROLE = 'chief_audit_executive' as const;

/** Build a minimal workflow shape for each finding (mock state). */
function buildMockWorkflow(findingId: string): ApprovalWorkflow {
  // For demo purposes: first finding is pending, second is approved, rest are draft.
  if (findingId === 'fnd-001') {
    return {
      id: `wf-${findingId}`,
      workflow_type: 'finding_signoff',
      status: 'pending_approval',
      submitted_by_email: 'auditor@example.com',
      submitted_at: '2024-01-25T09:00:00Z',
    };
  }
  if (findingId === 'fnd-002') {
    return {
      id: `wf-${findingId}`,
      workflow_type: 'finding_signoff',
      status: 'approved',
      submitted_by_email: 'auditor@example.com',
      submitted_at: '2024-01-22T08:00:00Z',
      decided_by_email: 'cae@example.com',
      decided_at: '2024-01-22T14:00:00Z',
    };
  }
  return {
    id: `wf-${findingId}`,
    workflow_type: 'finding_signoff',
    status: 'draft',
  };
}

export default function FindingsPage() {
  const [severityFilter, setSeverityFilter] = useState<FindingSeverity | 'all'>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [search, setSearch] = useState('');
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [workflows, setWorkflows] = useState<Record<string, ApprovalWorkflow>>(() =>
    Object.fromEntries(mockFindings.map(f => [f.id, buildMockWorkflow(f.id)]))
  );

  // Sign-off modal state
  const [signOffFindingId, setSignOffFindingId] = useState<string | null>(null);
  const [signOffDecision, setSignOffDecision] = useState<SignOffDecision>('approve');

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
  const pendingSignOff = Object.values(workflows).filter(w => w.status === 'pending_approval').length;

  function openSignOff(findingId: string, decision: SignOffDecision) {
    setSignOffFindingId(findingId);
    setSignOffDecision(decision);
  }

  function handleSignOffConfirm(payload: SignOffPayload) {
    if (!signOffFindingId) return;
    const wf = workflows[signOffFindingId];
    if (!wf) return;

    // In production: POST /api/v1/approvals/{wf.id}/{approve|reject}
    const now = new Date().toISOString();
    const updated: ApprovalWorkflow = {
      ...wf,
      status: payload.decision === 'approve' ? 'approved' : 'rejected',
      decided_by_email: payload.actorEmail,
      decided_at: now,
      latest_comment: payload.comment || undefined,
      rejection_reason: payload.rejectionReason,
    };
    setWorkflows(prev => ({ ...prev, [signOffFindingId]: updated }));
    setSignOffFindingId(null);
  }

  function handleSubmit(findingId: string) {
    const wf = workflows[findingId];
    if (!wf) return;
    // In production: POST /api/v1/approvals/{wf.id}/submit
    setWorkflows(prev => ({
      ...prev,
      [findingId]: {
        ...wf,
        status: 'pending_approval',
        submitted_by_email: 'el@example.com',
        submitted_at: new Date().toISOString(),
      },
    }));
  }

  function handleReturnToDraft(findingId: string) {
    const wf = workflows[findingId];
    if (!wf) return;
    setWorkflows(prev => ({
      ...prev,
      [findingId]: { ...wf, status: 'draft', rejection_reason: undefined },
    }));
  }

  const activeSignOffFinding = signOffFindingId ? mockFindings.find(f => f.id === signOffFindingId) : null;

  return (
    <AppShell title="Findings">
      <div className="p-6 space-y-6">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          <MetricCard label="Total Findings" value={mockFindings.length} accent="border-t-4 border-slate-400" />
          <MetricCard label="Critical" value={criticalCount} accent="border-t-4 border-red-600" />
          <MetricCard label="High" value={highCount} accent="border-t-4 border-orange-500" />
          <MetricCard label="Open / Active" value={openCount + remCount} subtitle={`${openCount} open, ${remCount} in remediation`} accent="border-t-4 border-yellow-500" />
          <MetricCard label="Pending Sign-Off" value={pendingSignOff} accent="border-t-4 border-blue-500" />
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
          {displayed.map(f => {
            const wf = workflows[f.id];
            return (
              <Card key={f.id} className={clsx(
                'hover:shadow-md transition-shadow',
                f.severity === 'critical' && 'border-l-4 border-l-red-600',
                f.severity === 'high' && 'border-l-4 border-l-orange-500',
              )}>
                <CardBody>
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-mono text-xs text-slate-400">{f.finding_ref}</span>
                        <SeverityBadge severity={f.severity}>{f.severity}</SeverityBadge>
                        <StatusBadge status={f.status}>{f.status.replace('_', ' ')}</StatusBadge>
                        {wf?.status === 'approved' && (
                          <span className="inline-flex items-center gap-1 text-xs text-green-700 bg-green-50 border border-green-200 px-2 py-0.5 rounded">
                            <CheckCircle2 size={11} /> Signed Off
                          </span>
                        )}
                      </div>
                      <h4 className="font-semibold text-slate-900">{f.title}</h4>
                      {expandedId === f.id && (
                        <div className="mt-3 space-y-3 text-sm">
                          <p className="text-slate-600">{f.description}</p>
                          {f.root_cause && <div><span className="font-medium text-slate-700">Root Cause:</span><span className="text-slate-500 ml-1">{f.root_cause}</span></div>}
                          {f.impact && <div><span className="font-medium text-slate-700">Impact:</span><span className="text-slate-500 ml-1">{f.impact}</span></div>}
                          <div className="flex gap-1 flex-wrap">
                            {f.tags.map(t => <span key={t} className="text-xs bg-slate-100 text-slate-500 px-1.5 py-0.5 rounded">{t}</span>)}
                          </div>
                          {/* Approval banner inside expanded view */}
                          <div className="mt-2">
                            <ApprovalBanner
                              workflow={wf ?? null}
                              userRole={CURRENT_USER_ROLE}
                              onSubmit={() => handleSubmit(f.id)}
                              onApprove={() => openSignOff(f.id, 'approve')}
                              onReject={() => openSignOff(f.id, 'reject')}
                              onReturnToDraft={() => handleReturnToDraft(f.id)}
                            />
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
            );
          })}
        </div>

        <p className="text-xs text-slate-400 text-center">{displayed.length} of {mockFindings.length} findings shown</p>
      </div>

      {/* Sign-off modal */}
      <SignOffModal
        isOpen={!!signOffFindingId}
        decision={signOffDecision}
        onClose={() => setSignOffFindingId(null)}
        onConfirm={handleSignOffConfirm}
        resourceLabel={activeSignOffFinding ? `Finding ${activeSignOffFinding.finding_ref}` : ''}
        workflowType="finding_signoff"
      />
    </AppShell>
  );
}
