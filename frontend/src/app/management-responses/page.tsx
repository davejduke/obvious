'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { SeverityBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useManagementResponseStore } from '@/store/management-responses';
import type { ManagementResponse, ManagementResponseStatus } from '@shared/index';
import { useState } from 'react';
import { clsx } from 'clsx';
import {
  CheckCircle2, Clock, AlertTriangle, TrendingUp, ChevronDown, ChevronUp,
  ClipboardList, ShieldCheck, Wrench, BadgeCheck
} from 'lucide-react';

/* ---------- Status helpers ---------- */

const STATUS_META: Record<ManagementResponseStatus, { label: string; color: string; icon: React.ElementType }> = {
  pending:                { label: 'Pending',             color: 'bg-slate-100 text-slate-700',       icon: Clock },
  accepted:               { label: 'Accepted',            color: 'bg-blue-100 text-blue-700',         icon: CheckCircle2 },
  implementation_planned: { label: 'Plan Created',        color: 'bg-amber-100 text-amber-700',       icon: ClipboardList },
  implemented:            { label: 'Implemented',         color: 'bg-purple-100 text-purple-700',     icon: Wrench },
  verified:               { label: 'Verified',            color: 'bg-green-100 text-green-700',       icon: BadgeCheck },
};

const STATUS_ORDER: ManagementResponseStatus[] = [
  'pending', 'accepted', 'implementation_planned', 'implemented', 'verified'
];

function StatusBadge({ status }: { status: ManagementResponseStatus }) {
  const meta = STATUS_META[status];
  const Icon = meta.icon;
  return (
    <span className={clsx('inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium', meta.color)}>
      <Icon size={11} />
      {meta.label}
    </span>
  );
}

/* ---------- Timeline ---------- */

function StatusTimeline({ response }: { response: ManagementResponse }) {
  const current = STATUS_ORDER.indexOf(response.status);

  const milestones = [
    { status: 'pending',                label: 'Finding raised',        date: response.created_at },
    { status: 'accepted',               label: 'Accepted by management', date: response.accepted_at },
    { status: 'implementation_planned', label: 'Plan created',           date: response.planned_at },
    { status: 'implemented',            label: 'Implemented',            date: response.implemented_at },
    { status: 'verified',               label: 'Verified',               date: response.verified_at },
  ] as const;

  return (
    <ol className="relative ml-3 border-l border-[var(--border-default)] space-y-3 mt-3">
      {milestones.map((m, i) => {
        const done = i <= current;
        return (
          <li key={m.status} className="pl-4">
            <div className={clsx(
              'absolute -left-1.5 w-3 h-3 rounded-full border-2',
              done
                ? 'bg-[var(--brand-600)] border-[var(--brand-600)]'
                : 'bg-[var(--bg-surface)] border-[var(--border-strong)]'
            )} />
            <p className={clsx('text-xs font-medium', done ? 'text-[var(--text-primary)]' : 'text-[var(--text-muted)]')}>
              {m.label}
            </p>
            {m.date && (
              <p className="text-xs text-[var(--text-muted)]">{new Date(m.date).toLocaleDateString()}</p>
            )}
          </li>
        );
      })}
    </ol>
  );
}

/* ---------- Transition modal ---------- */

interface TransitionModalProps {
  response: ManagementResponse;
  targetStatus: ManagementResponseStatus;
  onConfirm: (notes: string, extra: Partial<ManagementResponse>) => void;
  onCancel: () => void;
}

function TransitionModal({ response, targetStatus, onConfirm, onCancel }: TransitionModalProps) {
  const [notes, setNotes] = useState('');
  const [owner, setOwner] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [plan, setPlan] = useState('');

  function handleConfirm() {
    const extra: Partial<ManagementResponse> = {};
    if (targetStatus === 'accepted') extra.acceptance_notes = notes;
    if (targetStatus === 'implementation_planned') {
      extra.implementation_plan = plan;
      extra.implementation_owner = owner;
      extra.implementation_due_date = dueDate;
    }
    if (targetStatus === 'implemented') extra.implementation_notes = notes;
    if (targetStatus === 'verified') extra.verification_notes = notes;
    onConfirm(notes, extra);
  }

  const meta = STATUS_META[targetStatus];

  return (
    <div className="fixed inset-0 z-[150] flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="w-full max-w-md mx-4 rounded-xl shadow-xl bg-[var(--bg-surface)] border border-[var(--border-default)]">
        <div className="px-5 py-4 border-b border-[var(--border-default)]">
          <h3 className="text-sm font-semibold text-[var(--text-primary)]">
            Transition to: {meta.label}
          </h3>
          <p className="text-xs text-[var(--text-muted)] mt-0.5">{response.finding_title}</p>
        </div>

        <div className="px-5 py-4 space-y-3">
          {targetStatus === 'implementation_planned' && (
            <>
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1">Implementation plan *</label>
                <textarea
                  value={plan}
                  onChange={e => setPlan(e.target.value)}
                  rows={3}
                  className="w-full px-3 py-2 text-sm rounded-lg border bg-[var(--bg-surface)] border-[var(--border-default)] text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]"
                  placeholder="Describe the remediation plan…"
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1">Owner</label>
                  <input type="text" value={owner} onChange={e => setOwner(e.target.value)}
                    className="w-full px-2 py-1.5 text-sm rounded-lg border bg-[var(--bg-surface)] border-[var(--border-default)] text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]"
                    placeholder="Team or name"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1">Due date</label>
                  <input type="date" value={dueDate} onChange={e => setDueDate(e.target.value)}
                    className="w-full px-2 py-1.5 text-sm rounded-lg border bg-[var(--bg-surface)] border-[var(--border-default)] text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]"
                  />
                </div>
              </div>
            </>
          )}

          {targetStatus !== 'implementation_planned' && (
            <div>
              <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1">Notes</label>
              <textarea
                value={notes}
                onChange={e => setNotes(e.target.value)}
                rows={3}
                className="w-full px-3 py-2 text-sm rounded-lg border bg-[var(--bg-surface)] border-[var(--border-default)] text-[var(--text-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--ring-focus)]"
                placeholder={`Notes for this transition…`}
              />
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 px-5 py-3 border-t border-[var(--border-default)]">
          <Button variant="secondary" size="sm" onClick={onCancel}>Cancel</Button>
          <Button size="sm" onClick={handleConfirm}>Confirm transition</Button>
        </div>
      </div>
    </div>
  );
}

/* ---------- Row ---------- */

function ResponseRow({ response }: { response: ManagementResponse }) {
  const [expanded, setExpanded] = useState(false);
  const [transitioning, setTransitioning] = useState<ManagementResponseStatus | null>(null);
  const { transitionStatus } = useManagementResponseStore();

  const currentIdx = STATUS_ORDER.indexOf(response.status);
  const nextStatus = currentIdx < STATUS_ORDER.length - 1 ? STATUS_ORDER[currentIdx + 1] : null;

  const isOverdue = response.implementation_due_date && response.status !== 'verified' && response.status !== 'implemented'
    && new Date(response.implementation_due_date) < new Date();

  function handleTransition(notes: string, extra: Partial<ManagementResponse>) {
    if (!transitioning) return;
    transitionStatus(response.id, transitioning, extra);
    setTransitioning(null);
  }

  return (
    <>
      <div
        className={clsx(
          'border border-[var(--border-default)] rounded-lg overflow-hidden',
          isOverdue && 'border-orange-300 dark:border-orange-700'
        )}
      >
        <div
          className="flex items-center gap-3 p-3 cursor-pointer hover:bg-[var(--bg-hover)] transition-colors"
          onClick={() => setExpanded(!expanded)}
          data-testid={`response-row-${response.id}`}
        >
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-xs font-mono text-[var(--text-muted)]">{response.finding_ref}</span>
              <SeverityBadge severity={response.finding_severity}>{response.finding_severity}</SeverityBadge>
              {isOverdue && (
                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-orange-100 text-orange-700">
                  <AlertTriangle size={10} /> Overdue
                </span>
              )}
            </div>
            <p className="text-sm font-medium text-[var(--text-primary)] truncate mt-0.5">{response.finding_title}</p>
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            <StatusBadge status={response.status} />
            {expanded ? <ChevronUp size={14} className="text-[var(--text-muted)]" /> : <ChevronDown size={14} className="text-[var(--text-muted)]" />}
          </div>
        </div>

        {expanded && (
          <div className="border-t border-[var(--border-default)] p-4 bg-[var(--bg-muted)] space-y-4">
            <StatusTimeline response={response} />

            {/* Details by status */}
            {response.acceptance_notes && (
              <div>
                <p className="text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-1">Management acceptance</p>
                <p className="text-sm text-[var(--text-secondary)]">{response.acceptance_notes}</p>
                {response.accepted_by && <p className="text-xs text-[var(--text-muted)] mt-0.5">By {response.accepted_by} · {response.accepted_at ? new Date(response.accepted_at).toLocaleDateString() : ''}</p>}
              </div>
            )}
            {response.implementation_plan && (
              <div>
                <p className="text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-1">Implementation plan</p>
                <p className="text-sm text-[var(--text-secondary)]">{response.implementation_plan}</p>
                <div className="flex gap-4 mt-1 text-xs text-[var(--text-muted)]">
                  {response.implementation_owner && <span>Owner: {response.implementation_owner}</span>}
                  {response.implementation_due_date && <span>Due: {new Date(response.implementation_due_date).toLocaleDateString()}</span>}
                </div>
              </div>
            )}
            {response.implementation_notes && (
              <div>
                <p className="text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-1">Implementation notes</p>
                <p className="text-sm text-[var(--text-secondary)]">{response.implementation_notes}</p>
              </div>
            )}
            {response.verification_notes && (
              <div>
                <p className="text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-1">Verification notes</p>
                <p className="text-sm text-[var(--text-secondary)]">{response.verification_notes}</p>
                {response.verified_by && <p className="text-xs text-[var(--text-muted)] mt-0.5">By {response.verified_by} · {response.verified_at ? new Date(response.verified_at).toLocaleDateString() : ''}</p>}
              </div>
            )}

            {/* Next action */}
            {nextStatus && (
              <div className="flex justify-end">
                <Button
                  size="sm"
                  onClick={(e) => { e.stopPropagation(); setTransitioning(nextStatus); }}
                  data-testid={`transition-btn-${response.id}`}
                >
                  <TrendingUp size={13} className="mr-1" />
                  Move to: {STATUS_META[nextStatus].label}
                </Button>
              </div>
            )}
          </div>
        )}
      </div>

      {transitioning && (
        <TransitionModal
          response={response}
          targetStatus={transitioning}
          onConfirm={handleTransition}
          onCancel={() => setTransitioning(null)}
        />
      )}
    </>
  );
}

/* ---------- Page ---------- */

export default function ManagementResponsesPage() {
  const { responses } = useManagementResponseStore();
  const [statusFilter, setStatusFilter] = useState<ManagementResponseStatus | 'all'>('all');

  const displayed = responses.filter(r =>
    statusFilter === 'all' ? true : r.status === statusFilter
  );

  const openCount      = responses.filter(r => r.status === 'pending').length;
  const acceptedCount  = responses.filter(r => r.status === 'accepted').length;
  const plannedCount   = responses.filter(r => r.status === 'implementation_planned').length;
  const implCount      = responses.filter(r => r.status === 'implemented').length;
  const verifiedCount  = responses.filter(r => r.status === 'verified').length;
  const overdueCount   = responses.filter(r =>
    r.implementation_due_date && r.status !== 'verified' && r.status !== 'implemented'
    && new Date(r.implementation_due_date) < new Date()
  ).length;

  return (
    <AppShell title="Management Responses">
      <div className="p-6 space-y-6">
        {/* Metrics */}
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-4">
          <MetricCard label="Pending" value={openCount}      color="slate"  icon={<Clock size={16} />} />
          <MetricCard label="Accepted" value={acceptedCount}  color="blue"   icon={<CheckCircle2 size={16} />} />
          <MetricCard label="Plan created" value={plannedCount}  color="amber"  icon={<ClipboardList size={16} />} />
          <MetricCard label="Implemented" value={implCount}    color="purple" icon={<Wrench size={16} />} />
          <MetricCard label="Verified" value={verifiedCount}  color="green"  icon={<ShieldCheck size={16} />} />
          <MetricCard label="Overdue" value={overdueCount}   color="red"    icon={<AlertTriangle size={16} />} />
        </div>

        {/* Filter */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-[var(--text-primary)]">Response Tracker</h2>
              <div className="flex gap-1 flex-wrap">
                {(['all', ...STATUS_ORDER] as const).map(s => (
                  <button
                    key={s}
                    onClick={() => setStatusFilter(s)}
                    className={clsx(
                      'px-2.5 py-1 rounded text-xs font-medium transition-colors',
                      statusFilter === s
                        ? 'bg-[var(--brand-600)] text-white'
                        : 'text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]'
                    )}
                  >
                    {s === 'all' ? 'All' : STATUS_META[s].label}
                  </button>
                ))}
              </div>
            </div>
          </CardHeader>
          <CardBody className="space-y-2">
            {displayed.length === 0 && (
              <p className="text-sm text-[var(--text-muted)] text-center py-6">No responses matching this filter.</p>
            )}
            {displayed.map(r => <ResponseRow key={r.id} response={r} />)}
          </CardBody>
        </Card>
      </div>
    </AppShell>
  );
}
