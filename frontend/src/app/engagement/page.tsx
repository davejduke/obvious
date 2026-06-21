'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { StatusBadge, SeverityBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ApprovalBanner, type ApprovalWorkflow } from '@/components/ui/approval-banner';
import { SignOffModal, type SignOffDecision, type SignOffPayload } from '@/components/ui/sign-off-modal';
import { mockEngagements, mockFindings } from '@/lib/mock-data';
import { useState } from 'react';
import Link from 'next/link';
import { Briefcase, Calendar, User, ArrowRight, Plus } from 'lucide-react';
import { clsx } from 'clsx';

// Mock current user — in production from JWT.
const CURRENT_USER_ROLE = 'engagement_lead' as const;

/** Build a mock audit plan workflow for a given engagement. */
function buildPlanWorkflow(engId: string): ApprovalWorkflow {
  // Second engagement is in pending_approval for demo.
  if (engId === 'eng-002') {
    return { id: `wf-plan-${engId}`, workflow_type: 'audit_plan', status: 'pending_approval',
      submitted_by_email: 'el@example.com', submitted_at: '2024-03-05T10:00:00Z' };
  }
  return { id: `wf-plan-${engId}`, workflow_type: 'audit_plan', status: 'approved',
    submitted_by_email: 'el@example.com', submitted_at: '2024-01-12T09:00:00Z',
    decided_by_email: 'cae@example.com', decided_at: '2024-01-13T14:00:00Z' };
}

export default function EngagementPage() {
  const [selected, setSelected] = useState(mockEngagements[0]);
  const [planWorkflows, setPlanWorkflows] = useState<Record<string, ApprovalWorkflow>>(() =>
    Object.fromEntries(mockEngagements.map(e => [e.id, buildPlanWorkflow(e.id)]))
  );
  const [signOffEngId, setSignOffEngId] = useState<string | null>(null);
  const [signOffDecision, setSignOffDecision] = useState<SignOffDecision>('approve');

  const findingsForEng = mockFindings.filter(f => f.engagement_id === selected.id);
  const openFindings = findingsForEng.filter(f => f.status === 'open').length;
  const criticals = findingsForEng.filter(f => f.severity === 'critical').length;

  function handleSubmitPlan(engId: string) {
    const wf = planWorkflows[engId];
    if (!wf) return;
    setPlanWorkflows(prev => ({
      ...prev,
      [engId]: { ...wf, status: 'pending_approval', submitted_by_email: 'el@example.com', submitted_at: new Date().toISOString() },
    }));
  }

  function handlePlanSignOff(payload: SignOffPayload) {
    if (!signOffEngId) return;
    const wf = planWorkflows[signOffEngId];
    if (!wf) return;
    const now = new Date().toISOString();
    setPlanWorkflows(prev => ({
      ...prev,
      [signOffEngId]: {
        ...wf,
        status: payload.decision === 'approve' ? 'approved' : 'rejected',
        decided_by_email: payload.actorEmail,
        decided_at: now,
        latest_comment: payload.comment || undefined,
        rejection_reason: payload.rejectionReason,
      },
    }));
    setSignOffEngId(null);
  }

  function handleReturnToDraft(engId: string) {
    const wf = planWorkflows[engId];
    if (!wf) return;
    setPlanWorkflows(prev => ({ ...prev, [engId]: { ...wf, status: 'draft', rejection_reason: undefined } }));
  }

  return (
    <AppShell title="Engagement Workspace">
      <div className="flex h-full">
        {/* Engagement list */}
        <div className="w-80 flex-shrink-0 border-r border-slate-200 bg-white">
          <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100">
            <h3 className="font-semibold text-slate-900 text-sm">Engagements</h3>
            <Button size="sm" variant="ghost"><Plus size={14} />New</Button>
          </div>
          <ul className="divide-y divide-slate-100">
            {mockEngagements.map(eng => (
              <li key={eng.id}>
                <button
                  onClick={() => setSelected(eng)}
                  className={clsx(
                    'w-full text-left px-4 py-4 hover:bg-slate-50 transition-colors',
                    selected.id === eng.id && 'bg-blue-50 border-r-2 border-blue-600'
                  )}
                >
                  <p className="font-medium text-sm text-slate-900 line-clamp-2">{eng.name}</p>
                  <div className="flex items-center gap-2 mt-2">
                    <StatusBadge status={eng.status}>{eng.status}</StatusBadge>
                    {eng.overall_score && <span className="text-xs text-slate-500">{eng.overall_score}%</span>}
                  </div>
                </button>
              </li>
            ))}
          </ul>
        </div>

        {/* Engagement detail */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-xl font-semibold text-slate-900">{selected.name}</h2>
              <p className="text-slate-500 text-sm mt-1">{selected.description}</p>
            </div>
            <div className="flex items-center gap-2">
              <StatusBadge status={selected.status}>{selected.status}</StatusBadge>
              {selected.risk_rating && <SeverityBadge severity={selected.risk_rating}>{selected.risk_rating} risk</SeverityBadge>}
            </div>
          </div>

          {/* Detail cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <MetricCard label="Compliance Score" value={selected.overall_score ? `${selected.overall_score}%` : 'N/A'} accent="border-t-4 border-blue-500" />
            <MetricCard label="Open Findings" value={openFindings} subtitle={`${criticals} critical`} accent="border-t-4 border-red-500" />
            <MetricCard label="Framework" value="NIS 2" subtitle="Article 21(a-j)" accent="border-t-4 border-purple-500" />
            <MetricCard label="Controls Scope" value={Object.keys(selected.scope_json).length > 0 ? `${(selected.scope_json.articles as string[])?.length ?? 0} articles` : 'TBD'} accent="border-t-4 border-green-500" />
          </div>

          {/* Timeline */}
          <Card>
            <CardHeader><h3 className="font-semibold text-slate-900">Timeline</h3></CardHeader>
            <CardBody>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div className="flex items-center gap-2">
                  <Calendar size={16} className="text-slate-400" />
                  <div>
                    <p className="text-xs text-slate-400">Start Date</p>
                    <p className="text-sm font-medium">{selected.target_start_date ?? '—'}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Calendar size={16} className="text-slate-400" />
                  <div>
                    <p className="text-xs text-slate-400">End Date</p>
                    <p className="text-sm font-medium">{selected.target_end_date ?? '—'}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <User size={16} className="text-slate-400" />
                  <div>
                    <p className="text-xs text-slate-400">Lead Auditor</p>
                    <p className="text-sm font-medium">{selected.lead_auditor_id ?? '—'}</p>
                  </div>
                </div>
              </div>
            </CardBody>
          </Card>

          {/* Audit Plan Approval Banner */}
          <ApprovalBanner
            workflow={planWorkflows[selected.id] ?? null}
            userRole={CURRENT_USER_ROLE}
            onSubmit={() => handleSubmitPlan(selected.id)}
            onApprove={() => { setSignOffEngId(selected.id); setSignOffDecision('approve'); }}
            onReject={() => { setSignOffEngId(selected.id); setSignOffDecision('reject'); }}
            onReturnToDraft={() => handleReturnToDraft(selected.id)}
          />

          {/* Scope articles */}
          <Card>
            <CardHeader><h3 className="font-semibold text-slate-900">Scope — NIS 2 Articles</h3></CardHeader>
            <CardBody>
              <div className="flex flex-wrap gap-2">
                {((selected.scope_json.articles ?? []) as string[]).map(art => (
                  <span key={art} className="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-xs font-medium">
                    Article {art.toUpperCase()}
                  </span>
                ))}
              </div>
            </CardBody>
          </Card>

          {/* Quick navigation */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {[
              { href: '/controls', label: 'Browse Controls', icon: ArrowRight },
              { href: '/evidence', label: 'Manage Evidence', icon: ArrowRight },
              { href: '/findings', label: 'View Findings', icon: ArrowRight },
            ].map(({ href, label, icon: Icon }) => (
              <Link key={href} href={href}
                className="flex items-center justify-between p-4 rounded-lg border border-slate-200 hover:border-blue-300 hover:bg-blue-50 transition-colors group">
                <span className="text-sm font-medium text-slate-700 group-hover:text-blue-700">{label}</span>
                <Icon size={16} className="text-slate-400 group-hover:text-blue-500" />
              </Link>
            ))}
          </div>
        </div>
      </div>
      {/* Audit Plan sign-off modal */}
      <SignOffModal
        isOpen={!!signOffEngId}
        decision={signOffDecision}
        onClose={() => setSignOffEngId(null)}
        onConfirm={handlePlanSignOff}
        resourceLabel={signOffEngId ? (mockEngagements.find(e => e.id === signOffEngId)?.name ?? 'Audit Plan') : ''}
        workflowType="audit_plan"
      />
    </AppShell>
  );
}
