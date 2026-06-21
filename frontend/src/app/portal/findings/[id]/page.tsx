'use client';
import { use, useState } from 'react';
import { notFound } from 'next/navigation';
import {
  portalFindings,
  getControlForFinding,
  getManagementResponse,
  getRequestsForFinding,
  portalEngagement,
} from '@/lib/portal-mock-data';
import type { ManagementResponse } from '@shared/index';
import { Card, CardHeader, CardBody } from '@/components/ui/card';
import { SeverityBadge, StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  ChevronLeft,
  ChevronRight,
  Info,
  AlertTriangle,
  CheckCircle2,
  Loader2,
  FileSearch,
  Shield,
  Briefcase,
  MessageSquare,
  Clock,
} from 'lucide-react';
import Link from 'next/link';
import { clsx } from 'clsx';

// ─── Context-before-finding stepper ──────────────────────────────────────────
// Step 0: Engagement context
// Step 1: Control details
// Step 2: Finding details
// Step 3: Management response form / view
const STEP_LABELS = [
  { label: 'Engagement', icon: Briefcase },
  { label: 'Control', icon: Shield },
  { label: 'Finding', icon: AlertTriangle },
  { label: 'Your Response', icon: MessageSquare },
];

function Stepper({ step, total }: { step: number; total: number }) {
  return (
    <div className="flex items-center gap-0 mb-6">
      {STEP_LABELS.slice(0, total).map((s, i) => {
        const Icon = s.icon;
        const done = i < step;
        const active = i === step;
        return (
          <div key={i} className="flex items-center flex-1 last:flex-none">
            <div className="flex items-center gap-2">
              <div
                className={clsx(
                  'w-8 h-8 rounded-full flex items-center justify-center transition-colors',
                  done ? 'bg-green-500 text-white' : active ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-400',
                )}
              >
                {done ? <CheckCircle2 size={16} /> : <Icon size={15} />}
              </div>
              <span
                className={clsx(
                  'text-xs font-medium hidden sm:block',
                  active ? 'text-indigo-700' : done ? 'text-slate-500' : 'text-slate-400',
                )}
              >
                {s.label}
              </span>
            </div>
            {i < total - 1 && (
              <div
                className={clsx(
                  'flex-1 h-0.5 mx-3',
                  i < step ? 'bg-green-400' : 'bg-slate-200',
                )}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

// ─── Management response form ─────────────────────────────────────────────────
interface ResponseFormProps {
  findingId: string;
  existingResponse?: ManagementResponse;
  onSubmitted: (response: ManagementResponse) => void;
}

function ManagementResponseForm({ findingId, existingResponse, onSubmitted }: ResponseFormProps) {
  const [responseText, setResponseText] = useState(existingResponse?.response_text ?? '');
  const [actionPlan, setActionPlan] = useState(existingResponse?.action_plan ?? '');
  const [targetDate, setTargetDate] = useState(existingResponse?.target_remediation_date ?? '');
  const [submitting, setSubmitting] = useState(false);

  const isViewOnly = !!existingResponse && existingResponse.status !== 'draft';

  async function handleSubmit() {
    if (!responseText.trim()) return;
    setSubmitting(true);
    // Simulate API call (production: POST /api/v1/portal/findings/:id/management-response)
    await new Promise((resolve) => setTimeout(resolve, 900));
    const now = new Date().toISOString();
    const newResponse: ManagementResponse = {
      id: existingResponse?.id ?? `mr-${Date.now()}`,
      finding_id: findingId,
      org_id: 'org-001',
      responder_id: 'usr-ciso-001',
      responder_email: 'ciso@example.com',
      response_text: responseText,
      action_plan: actionPlan || undefined,
      target_remediation_date: targetDate || undefined,
      status: 'submitted',
      submitted_at: now,
      created_at: existingResponse?.created_at ?? now,
      updated_at: now,
    };
    setSubmitting(false);
    onSubmitted(newResponse);
  }

  return (
    <div className="space-y-5">
      {isViewOnly && existingResponse && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 flex items-center gap-3">
          <CheckCircle2 size={20} className="text-green-500 flex-shrink-0" />
          <div>
            <p className="text-sm font-semibold text-green-900">Response submitted</p>
            <p className="text-xs text-green-700">
              Submitted {existingResponse.submitted_at
                ? new Date(existingResponse.submitted_at).toLocaleString()
                : 'recently'} by {existingResponse.responder_email}
            </p>
          </div>
          <StatusBadge status={existingResponse.status} className="ml-auto">
            {existingResponse.status}
          </StatusBadge>
        </div>
      )}

      <div>
        <label className="block text-sm font-semibold text-slate-800 mb-1.5">
          Management Response <span className="text-red-500">*</span>
        </label>
        <p className="text-xs text-slate-500 mb-2">
          Describe management's position on this finding, including whether you accept,
          dispute, or partially dispute it.
        </p>
        <textarea
          value={responseText}
          onChange={(e) => setResponseText(e.target.value)}
          disabled={isViewOnly}
          rows={5}
          placeholder="Management acknowledges this finding and commits to remediation..."
          className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none disabled:bg-slate-50 disabled:text-slate-600"
        />
      </div>

      <div>
        <label className="block text-sm font-semibold text-slate-800 mb-1.5">
          Action Plan <span className="text-slate-400 font-normal">(optional)</span>
        </label>
        <p className="text-xs text-slate-500 mb-2">
          List the specific steps you will take to remediate this finding.
        </p>
        <textarea
          value={actionPlan}
          onChange={(e) => setActionPlan(e.target.value)}
          disabled={isViewOnly}
          rows={4}
          placeholder="1. Step one by date...\n2. Step two by date..."
          className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none disabled:bg-slate-50 disabled:text-slate-600 font-mono"
        />
      </div>

      <div>
        <label className="block text-sm font-semibold text-slate-800 mb-1.5">
          Target Remediation Date <span className="text-slate-400 font-normal">(optional)</span>
        </label>
        <input
          type="date"
          value={targetDate}
          onChange={(e) => setTargetDate(e.target.value)}
          disabled={isViewOnly}
          className="px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50"
        />
      </div>

      {!isViewOnly && (
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100">
          <Button
            variant="primary"
            size="md"
            onClick={handleSubmit}
            disabled={submitting || !responseText.trim()}
          >
            {submitting ? (
              <>
                <Loader2 size={14} className="animate-spin" /> Submitting...
              </>
            ) : (
              <>
                <MessageSquare size={14} /> Submit Management Response
              </>
            )}
          </Button>
        </div>
      )}
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export default function PortalFindingDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const finding = portalFindings.find((f) => f.id === id);
  if (!finding) notFound();

  const control = getControlForFinding(finding.control_id);
  const initialResponse = getManagementResponse(finding.id);
  const relatedRequests = getRequestsForFinding(finding.id);

  const [step, setStep] = useState(0);
  const [response, setResponse] = useState<ManagementResponse | undefined>(initialResponse);
  const [responseSubmitted, setResponseSubmitted] = useState(
    !!initialResponse && initialResponse.status !== 'draft',
  );

  function handleResponseSubmitted(r: ManagementResponse) {
    setResponse(r);
    setResponseSubmitted(true);
  }

  const totalSteps = STEP_LABELS.length;

  return (
    <div className="p-6 space-y-6 max-w-3xl">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-slate-500">
        <Link href="/portal/findings" className="hover:text-indigo-600 flex items-center gap-1">
          <ChevronLeft size={14} />
          Findings
        </Link>
        <span>/</span>
        <span className="text-slate-900 font-mono text-xs">{finding.finding_ref}</span>
      </div>

      {/* Stepper */}
      <Stepper step={step} total={totalSteps} />

      {/* Step content */}

      {/* Step 0: Engagement context */}
      {step === 0 && (
        <div className="space-y-4">
          <div className="bg-indigo-50 border border-indigo-100 rounded-lg p-4 flex items-start gap-3">
            <Info size={18} className="text-indigo-500 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-semibold text-indigo-900">Review engagement context first</p>
              <p className="text-sm text-indigo-700 mt-1">
                Before viewing the finding, please review the engagement and control context.
                This helps ensure you have full background before preparing your management response.
              </p>
            </div>
          </div>

          <Card>
            <CardHeader>
              <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                <Briefcase size={16} className="text-indigo-500" />
                Engagement Details
              </h3>
            </CardHeader>
            <CardBody>
              <div className="space-y-3">
                <div>
                  <p className="text-xs text-slate-500 uppercase tracking-wide mb-1">Engagement</p>
                  <p className="font-semibold text-slate-900">{portalEngagement.name}</p>
                </div>
                {portalEngagement.description && (
                  <p className="text-sm text-slate-500">{portalEngagement.description}</p>
                )}
                <div className="grid grid-cols-3 gap-4 text-sm pt-2">
                  <div>
                    <p className="text-xs text-slate-400">Status</p>
                    <StatusBadge status={portalEngagement.status}>{portalEngagement.status}</StatusBadge>
                  </div>
                  <div>
                    <p className="text-xs text-slate-400">Start</p>
                    <p className="text-slate-700 font-medium">{portalEngagement.target_start_date}</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-400">End</p>
                    <p className="text-slate-700 font-medium">{portalEngagement.target_end_date}</p>
                  </div>
                </div>
              </div>
            </CardBody>
          </Card>
        </div>
      )}

      {/* Step 1: Control context */}
      {step === 1 && (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                <Shield size={16} className="text-indigo-500" />
                Related Control
              </h3>
            </CardHeader>
            {control ? (
              <CardBody>
                <div className="space-y-3">
                  <div className="flex items-start gap-3">
                    <span className="font-mono text-xs bg-indigo-50 text-indigo-700 px-2 py-1 rounded flex-shrink-0">
                      {control.control_id}
                    </span>
                    <div>
                      <p className="font-semibold text-slate-900">{control.title}</p>
                      <p className="text-xs text-slate-500 mt-0.5">{control.domain} — {control.category}</p>
                    </div>
                  </div>
                  {control.description && (
                    <div>
                      <p className="text-xs font-semibold text-slate-600 uppercase tracking-wide mb-1">Description</p>
                      <p className="text-sm text-slate-600">{control.description}</p>
                    </div>
                  )}
                  {control.objective && (
                    <div>
                      <p className="text-xs font-semibold text-slate-600 uppercase tracking-wide mb-1">Control Objective</p>
                      <p className="text-sm text-slate-600 italic">{control.objective}</p>
                    </div>
                  )}
                  <div className="flex gap-2 flex-wrap">
                    {control.tags.map((t) => (
                      <span key={t} className="text-xs bg-slate-100 text-slate-500 px-2 py-0.5 rounded">
                        {t}
                      </span>
                    ))}
                  </div>
                </div>
              </CardBody>
            ) : (
              <CardBody>
                <p className="text-sm text-slate-500">Control details not available.</p>
              </CardBody>
            )}
          </Card>

          {/* Related evidence requests for this finding */}
          {relatedRequests.length > 0 && (
            <Card>
              <CardHeader>
                <h3 className="text-sm font-semibold text-slate-700 flex items-center gap-2">
                  <FileSearch size={14} className="text-blue-500" />
                  Evidence Requests for this Finding
                </h3>
              </CardHeader>
              <CardBody className="p-0">
                <ul className="divide-y divide-slate-100">
                  {relatedRequests.map((req) => (
                    <li key={req.id} className="flex items-center gap-3 px-6 py-3">
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-slate-900 truncate">{req.title}</p>
                        <div className="flex items-center gap-2 mt-0.5">
                          <StatusBadge status={req.status}>{req.status.replace('_', ' ')}</StatusBadge>
                          <span className="text-xs text-slate-400 flex items-center gap-1">
                            <Clock size={10} /> Due: {req.due_date}
                          </span>
                        </div>
                      </div>
                      <Link href={`/portal/evidence-requests/${req.id}`}>
                        <Button variant="ghost" size="sm">View</Button>
                      </Link>
                    </li>
                  ))}
                </ul>
              </CardBody>
            </Card>
          )}
        </div>
      )}

      {/* Step 2: Finding details */}
      {step === 2 && (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-start gap-3">
                <AlertTriangle
                  size={18}
                  className={clsx(
                    finding.severity === 'critical' ? 'text-red-500' :
                    finding.severity === 'high' ? 'text-orange-500' :
                    'text-yellow-500',
                  )}
                />
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-mono text-xs text-slate-400">{finding.finding_ref}</span>
                    <SeverityBadge severity={finding.severity}>{finding.severity}</SeverityBadge>
                    <StatusBadge status={finding.status}>{finding.status.replace('_', ' ')}</StatusBadge>
                  </div>
                  <h2 className="text-lg font-semibold text-slate-900">{finding.title}</h2>
                </div>
              </div>
            </CardHeader>
            <CardBody>
              <div className="space-y-4">
                <div>
                  <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1.5">Description</p>
                  <p className="text-sm text-slate-700">{finding.description}</p>
                </div>
                {finding.root_cause && (
                  <div>
                    <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1.5">Root Cause</p>
                    <p className="text-sm text-slate-700">{finding.root_cause}</p>
                  </div>
                )}
                {finding.impact && (
                  <div>
                    <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1.5">Impact</p>
                    <p className="text-sm text-slate-700">{finding.impact}</p>
                  </div>
                )}
                {finding.due_date && (
                  <div className="pt-2 border-t border-slate-100">
                    <p className="text-xs text-slate-400 flex items-center gap-1">
                      <Clock size={11} />
                      Management response due: <span className="font-medium text-slate-700">{finding.due_date}</span>
                    </p>
                  </div>
                )}
                <div className="flex gap-1.5 flex-wrap pt-2">
                  {finding.tags.map((t) => (
                    <span key={t} className="text-xs bg-slate-100 text-slate-500 px-2 py-0.5 rounded">{t}</span>
                  ))}
                </div>
              </div>
            </CardBody>
          </Card>
        </div>
      )}

      {/* Step 3: Management response */}
      {step === 3 && (
        <div className="space-y-4">
          {responseSubmitted && response ? (
            <div className="bg-green-50 border border-green-200 rounded-lg p-5 flex items-center gap-4 mb-2">
              <CheckCircle2 size={28} className="text-green-500 flex-shrink-0" />
              <div>
                <p className="font-semibold text-green-900">Management response submitted</p>
                <p className="text-sm text-green-700 mt-0.5">
                  Submitted {response.submitted_at
                    ? new Date(response.submitted_at).toLocaleString()
                    : ''} by {response.responder_email}
                </p>
              </div>
            </div>
          ) : null}

          <Card>
            <CardHeader>
              <h3 className="font-semibold text-slate-900 flex items-center gap-2">
                <MessageSquare size={16} className="text-indigo-500" />
                Management Response for {finding.finding_ref}
              </h3>
            </CardHeader>
            <CardBody>
              <ManagementResponseForm
                findingId={finding.id}
                existingResponse={response}
                onSubmitted={handleResponseSubmitted}
              />
            </CardBody>
          </Card>
        </div>
      )}

      {/* Step navigation */}
      <div className="flex items-center justify-between pt-2">
        <Button
          variant="secondary"
          size="md"
          onClick={() => setStep((s) => Math.max(0, s - 1))}
          disabled={step === 0}
        >
          <ChevronLeft size={15} /> Previous
        </Button>

        <span className="text-xs text-slate-400">
          Step {step + 1} of {totalSteps}
        </span>

        {step < totalSteps - 1 ? (
          <Button
            variant="primary"
            size="md"
            onClick={() => setStep((s) => Math.min(totalSteps - 1, s + 1))}
          >
            Next <ChevronRight size={15} />
          </Button>
        ) : (
          <Link href="/portal/findings">
            <Button variant="secondary" size="md">
              Back to Findings
            </Button>
          </Link>
        )}
      </div>
    </div>
  );
}
