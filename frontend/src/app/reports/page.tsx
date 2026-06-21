'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { SeverityBadge } from '@/components/ui/badge';
import { ApprovalBanner, type ApprovalWorkflow } from '@/components/ui/approval-banner';
import { SignOffModal, type SignOffDecision, type SignOffPayload } from '@/components/ui/sign-off-modal';
import { PasswordConfirmDialog, type PasswordConfirmPayload } from '@/components/ui/password-confirm-dialog';
import { mockEngagements, mockNIS2Score } from '@/lib/mock-data';
import { useState } from 'react';
import { FileText, Download, Eye, Plus, Calendar, Lock, CheckCircle, AlertCircle, Edit3 } from 'lucide-react';
import Link from 'next/link';

// Mock current user — in production from JWT.
const CURRENT_USER_ROLE = 'engagement_lead' as const;

const reportTemplates = [
  { id: 'nis2-exec', name: 'NIS 2 Executive Summary', description: 'Board-level compliance overview', pages: '~8 pages', access: ['audit_committee', 'cae'] },
  { id: 'nis2-full', name: 'NIS 2 Full Audit Report', description: 'Detailed findings and recommendations', pages: '~45 pages', access: ['internal_auditor', 'cae'] },
  { id: 'findings-register', name: 'Findings Register', description: 'Exportable findings with remediation status', pages: '~12 pages', access: ['internal_auditor', 'cae', 'auditee_ciso'] },
  { id: 'remediation-plan', name: 'Remediation Action Plan', description: 'Prioritised remediation roadmap', pages: '~20 pages', access: ['auditee_ciso', 'cae'] },
  { id: 'evidence-pack', name: 'Evidence Pack', description: 'Compiled evidence with linkages', pages: '~60 pages', access: ['internal_auditor'] },
];

interface GeneratedReport {
  id: string;
  name: string;
  generated: string;
  engagement: string;
  format: string;
  size: string;
}

const initialReports: GeneratedReport[] = [
  { id: 'rpt-001', name: 'NIS 2 Executive Summary — Jan 2024', generated: '2024-01-25', engagement: 'NIS 2 Article 21 Audit 2024', format: 'PDF', size: '1.2 MB' },
  { id: 'rpt-002', name: 'Interim Findings Register — Jan 2024', generated: '2024-01-22', engagement: 'NIS 2 Article 21 Audit 2024', format: 'XLSX', size: '245 KB' },
];

/** Build a mock workflow for each report. */
function buildMockReportWorkflow(reportId: string): ApprovalWorkflow {
  if (reportId === 'rpt-001') {
    return {
      id: `wf-${reportId}`,
      workflow_type: 'report_release',
      status: 'approved',
      submitted_by_email: 'el@example.com',
      submitted_at: '2024-01-24T10:00:00Z',
      decided_by_email: 'el@example.com',
      decided_at: '2024-01-24T15:00:00Z',
    };
  }
  return {
    id: `wf-${reportId}`,
    workflow_type: 'report_release',
    status: 'draft',
  };
}

export default function ReportsPage() {
  const [generating, setGenerating] = useState<string | null>(null);
  const [selectedEng, setSelectedEng] = useState(mockEngagements[0].id);
  const [workflows, setWorkflows] = useState<Record<string, ApprovalWorkflow>>(() =>
    Object.fromEntries(initialReports.map(r => [r.id, buildMockReportWorkflow(r.id)]))
  );

  // Approval modal state
  const [approvalReportId, setApprovalReportId] = useState<string | null>(null);
  const [approvalDecision, setApprovalDecision] = useState<SignOffDecision>('approve');
  // Password confirm dialog state
  const [lockReportId, setLockReportId] = useState<string | null>(null);

  const handleGenerate = (tplId: string) => {
    setGenerating(tplId);
    setTimeout(() => setGenerating(null), 2000);
  };

  function handleSubmitForApproval(reportId: string) {
    const wf = workflows[reportId];
    if (!wf) return;
    setWorkflows(prev => ({
      ...prev,
      [reportId]: {
        ...wf,
        status: 'pending_approval',
        submitted_by_email: 'el@example.com',
        submitted_at: new Date().toISOString(),
      },
    }));
  }

  function handleApprovalConfirm(payload: SignOffPayload) {
    if (!approvalReportId) return;
    const wf = workflows[approvalReportId];
    if (!wf) return;
    const now = new Date().toISOString();
    setWorkflows(prev => ({
      ...prev,
      [approvalReportId]: {
        ...wf,
        status: payload.decision === 'approve' ? 'approved' : 'rejected',
        decided_by_email: payload.actorEmail,
        decided_at: now,
        latest_comment: payload.comment || undefined,
        rejection_reason: payload.rejectionReason,
      },
    }));
    setApprovalReportId(null);
  }

  function handleLockConfirm(_payload: PasswordConfirmPayload) {
    if (!lockReportId) return;
    const wf = workflows[lockReportId];
    if (!wf) return;
    // In production: POST /api/v1/approvals/{wf.id}/lock  with password_confirm
    setWorkflows(prev => ({
      ...prev,
      [lockReportId]: {
        ...wf,
        status: 'locked',
        decided_at: new Date().toISOString(),
        latest_comment: 'Report locked and released.',
      },
    }));
    setLockReportId(null);
  }

  function handleReturnToDraft(reportId: string) {
    const wf = workflows[reportId];
    if (!wf) return;
    setWorkflows(prev => ({ ...prev, [reportId]: { ...wf, status: 'draft', rejection_reason: undefined } }));
  }

  const activeReport = approvalReportId ? initialReports.find(r => r.id === approvalReportId) : null;
  const lockReport = lockReportId ? initialReports.find(r => r.id === lockReportId) : null;

  return (
    <AppShell title="Reports">
      <div className="p-6 space-y-6">
        {/* Report Builder CTA */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-900">Reports</h2>
            <p className="text-sm text-slate-500 mt-0.5">Generate and manage audit reports</p>
          </div>
          <Link href="/reports/builder">
            <button className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors">
              <Edit3 size={15} />
              Open Report Builder
            </button>
          </Link>
        </div>

        {/* Engagement selector */}
        <Card>
          <CardBody className="py-3">
            <div className="flex items-center gap-4">
              <label className="text-sm font-medium text-slate-700">Generate for:</label>
              <select
                value={selectedEng}
                onChange={e => setSelectedEng(e.target.value)}
                className="text-sm border border-slate-200 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {mockEngagements.map(e => (
                  <option key={e.id} value={e.id}>{e.name}</option>
                ))}
              </select>
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <Calendar size={14} />
                <span>Last generated: Jan 25, 2024</span>
              </div>
            </div>
          </CardBody>
        </Card>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Report templates */}
          <div className="space-y-4">
            <h3 className="font-semibold text-slate-900">Report Templates</h3>
            {reportTemplates.map(tpl => (
              <Card key={tpl.id} className="hover:shadow-md transition-shadow">
                <CardBody>
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-start gap-3">
                      <div className="w-10 h-10 bg-blue-50 rounded-lg flex items-center justify-center flex-shrink-0">
                        <FileText size={20} className="text-blue-600" />
                      </div>
                      <div>
                        <p className="font-semibold text-slate-900 text-sm">{tpl.name}</p>
                        <p className="text-xs text-slate-500 mt-0.5">{tpl.description}</p>
                        <p className="text-xs text-slate-400 mt-1">{tpl.pages}</p>
                        <div className="flex gap-1 mt-1.5 flex-wrap">
                          {tpl.access.map(a => (
                            <span key={a} className="text-xs bg-slate-100 text-slate-500 px-1.5 py-0.5 rounded flex items-center gap-1">
                              <Lock size={9} />
                              {a.replace('_', ' ')}
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                    <div className="flex flex-col gap-2">
                      <Button size="sm" variant="primary"
                        disabled={!!generating}
                        onClick={() => handleGenerate(tpl.id)}>
                        {generating === tpl.id ? (
                          <span className="animate-pulse">Generating…</span>
                        ) : (
                          <><Plus size={13} /> Generate</>
                        )}
                      </Button>
                    </div>
                  </div>
                </CardBody>
              </Card>
            ))}
          </div>

          {/* Generated reports */}
          <div className="space-y-4">
            <h3 className="font-semibold text-slate-900">Generated Reports</h3>
            {initialReports.map(rpt => {
              const wf = workflows[rpt.id];
              const isLocked = wf?.status === 'locked';
              return (
                <Card key={rpt.id} className="hover:shadow-md transition-shadow">
                  <CardBody>
                    <div className="space-y-3">
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex items-start gap-3">
                          <div className={`w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0 ${isLocked ? 'bg-blue-900' : 'bg-slate-100'}`}>
                            {isLocked
                              ? <Lock size={18} className="text-white" />
                              : <FileText size={18} className="text-slate-600" />}
                          </div>
                          <div>
                            <p className="font-semibold text-slate-900 text-sm">{rpt.name}</p>
                            <p className="text-xs text-slate-500 mt-0.5">{rpt.engagement}</p>
                            <p className="text-xs text-slate-400 mt-1">
                              Generated {rpt.generated} · {rpt.format} · {rpt.size}
                            </p>
                          </div>
                        </div>
                        <div className="flex flex-col gap-2 flex-shrink-0">
                          {wf?.status === 'approved' && (
                            <span className="inline-flex items-center gap-1 text-xs text-green-700 bg-green-50 border border-green-200 px-2 py-0.5 rounded">
                              <CheckCircle size={11} /> Approved
                            </span>
                          )}
                          {isLocked && (
                            <span className="inline-flex items-center gap-1 text-xs text-blue-800 bg-blue-50 border border-blue-200 px-2 py-0.5 rounded">
                              <Lock size={11} /> Locked
                            </span>
                          )}
                          {!isLocked && (
                            <div className="flex gap-2">
                              <Button size="sm" variant="ghost"><Eye size={13} /> Preview</Button>
                              <Button size="sm" variant="secondary"><Download size={13} /> Download</Button>
                            </div>
                          )}
                          {isLocked && (
                            <Button size="sm" variant="secondary"><Download size={13} /> Download</Button>
                          )}
                        </div>
                      </div>

                      {/* Approval banner */}
                      <ApprovalBanner
                        workflow={wf ?? null}
                        userRole={CURRENT_USER_ROLE}
                        onSubmit={() => handleSubmitForApproval(rpt.id)}
                        onApprove={() => { setApprovalReportId(rpt.id); setApprovalDecision('approve'); }}
                        onReject={() => { setApprovalReportId(rpt.id); setApprovalDecision('reject'); }}
                        onLock={() => setLockReportId(rpt.id)}
                        onReturnToDraft={() => handleReturnToDraft(rpt.id)}
                      />
                    </div>
                  </CardBody>
                </Card>
              );
            })}

            {/* Info card about release workflow */}
            <Card className="border-dashed border-blue-200 bg-blue-50/50">
              <CardBody>
                <div className="flex items-start gap-3">
                  <AlertCircle size={16} className="text-blue-500 flex-shrink-0 mt-0.5" />
                  <div className="text-xs text-blue-700">
                    <p className="font-semibold">Report Release Workflow</p>
                    <p className="mt-1 text-blue-600">
                      Reports must be approved before release. Final lock requires password re-entry.
                      Once locked, reports become read-only and are audit-trailed.
                    </p>
                  </div>
                </div>
              </CardBody>
            </Card>
          </div>
        </div>
      </div>

      {/* Sign-off modal (approve / reject) */}
      <SignOffModal
        isOpen={!!approvalReportId}
        decision={approvalDecision}
        onClose={() => setApprovalReportId(null)}
        onConfirm={handleApprovalConfirm}
        resourceLabel={activeReport ? activeReport.name : ''}
        workflowType="report_release"
      />

      {/* Password confirmation dialog (lock) */}
      <PasswordConfirmDialog
        isOpen={!!lockReportId}
        onClose={() => setLockReportId(null)}
        onConfirm={handleLockConfirm}
        reportName={lockReport ? lockReport.name : ''}
      />
    </AppShell>
  );
}
