'use client';
/**
 * ApprovalBanner — shows the current approval state for a resource
 * and provides quick-action buttons (submit / approve / reject / lock).
 *
 * Designed to sit at the top of a detail view (engagement plan, finding, report).
 */

import { clsx } from 'clsx';
import { CheckCircle, Clock, Lock, XCircle, AlertTriangle, Send } from 'lucide-react';
import { Button } from './button';

export type WorkflowStatus = 'draft' | 'pending_approval' | 'approved' | 'locked' | 'rejected';
export type WorkflowType = 'audit_plan' | 'finding_signoff' | 'report_release';

export interface ApprovalWorkflow {
  id: string;
  workflow_type: WorkflowType;
  status: WorkflowStatus;
  submitted_by_email?: string;
  submitted_at?: string;
  decided_by_email?: string;
  decided_at?: string;
  latest_comment?: string;
  rejection_reason?: string;
}

interface ApprovalBannerProps {
  workflow: ApprovalWorkflow | null;
  /** Current user's role slug */
  userRole: 'engagement_lead' | 'chief_audit_executive' | 'reviewer';
  onSubmit?: () => void;
  onApprove?: () => void;
  onReject?: () => void;
  onLock?: () => void;
  onReturnToDraft?: () => void;
}

const statusConfig: Record<WorkflowStatus, {
  bg: string;
  border: string;
  icon: React.ElementType;
  iconColor: string;
  label: string;
}> = {
  draft: {
    bg: 'bg-slate-50',
    border: 'border-slate-200',
    icon: Clock,
    iconColor: 'text-slate-400',
    label: 'Draft — not submitted for approval',
  },
  pending_approval: {
    bg: 'bg-yellow-50',
    border: 'border-yellow-200',
    icon: Clock,
    iconColor: 'text-yellow-500',
    label: 'Pending Approval',
  },
  approved: {
    bg: 'bg-green-50',
    border: 'border-green-200',
    icon: CheckCircle,
    iconColor: 'text-green-600',
    label: 'Approved',
  },
  locked: {
    bg: 'bg-blue-50',
    border: 'border-blue-200',
    icon: Lock,
    iconColor: 'text-blue-600',
    label: 'Locked — read only',
  },
  rejected: {
    bg: 'bg-red-50',
    border: 'border-red-200',
    icon: XCircle,
    iconColor: 'text-red-600',
    label: 'Rejected',
  },
};

export function ApprovalBanner({
  workflow,
  userRole,
  onSubmit,
  onApprove,
  onReject,
  onLock,
  onReturnToDraft,
}: ApprovalBannerProps) {
  // No workflow exists yet — show a neutral prompt if EL
  if (!workflow) {
    if (userRole !== 'engagement_lead') return null;
    return (
      <div className="flex items-center justify-between px-4 py-3 rounded-lg border border-slate-200 bg-slate-50 text-sm">
        <div className="flex items-center gap-2 text-slate-600">
          <AlertTriangle size={15} className="text-slate-400" />
          No approval workflow started yet.
        </div>
        {onSubmit && (
          <Button size="sm" onClick={onSubmit} variant="secondary">
            <Send size={13} /> Start Approval
          </Button>
        )}
      </div>
    );
  }

  const cfg = statusConfig[workflow.status];
  const StatusIcon = cfg.icon;
  const canApprove = canUserApprove(workflow.workflow_type, userRole) && workflow.status === 'pending_approval';
  const canSubmit = userRole === 'engagement_lead' && (workflow.status === 'draft' || workflow.status === 'rejected');
  const canLock =
    workflow.workflow_type === 'report_release' &&
    workflow.status === 'approved' &&
    canUserApprove(workflow.workflow_type, userRole);
  const canReturnToDraft = userRole === 'engagement_lead' && workflow.status === 'rejected';

  return (
    <div className={clsx('rounded-lg border px-4 py-3 space-y-2', cfg.bg, cfg.border)}>
      {/* Status row */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <StatusIcon size={16} className={cfg.iconColor} />
          <span className="text-sm font-medium text-slate-800">{cfg.label}</span>
          {workflow.submitted_by_email && workflow.status !== 'draft' && (
            <span className="text-xs text-slate-500">
              · submitted by {workflow.submitted_by_email}
              {workflow.submitted_at && ` on ${new Date(workflow.submitted_at).toLocaleDateString()}`}
            </span>
          )}
          {workflow.decided_by_email && (workflow.status === 'approved' || workflow.status === 'locked' || workflow.status === 'rejected') && (
            <span className="text-xs text-slate-500">
              · decided by {workflow.decided_by_email}
              {workflow.decided_at && ` on ${new Date(workflow.decided_at).toLocaleDateString()}`}
            </span>
          )}
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2">
          {canSubmit && onSubmit && (
            <Button size="sm" onClick={onSubmit} variant="primary">
              <Send size={13} />
              {workflow.status === 'rejected' ? 'Resubmit' : 'Submit for Approval'}
            </Button>
          )}
          {canReturnToDraft && onReturnToDraft && (
            <Button size="sm" onClick={onReturnToDraft} variant="secondary">
              Return to Draft
            </Button>
          )}
          {canApprove && (
            <>
              {onApprove && (
                <Button size="sm" onClick={onApprove} variant="primary">
                  <CheckCircle size={13} /> Approve
                </Button>
              )}
              {onReject && (
                <Button size="sm" onClick={onReject} variant="danger">
                  <XCircle size={13} /> Reject
                </Button>
              )}
            </>
          )}
          {canLock && onLock && (
            <Button size="sm" onClick={onLock} variant="primary">
              <Lock size={13} /> Lock & Release
            </Button>
          )}
        </div>
      </div>

      {/* Rejection reason */}
      {workflow.status === 'rejected' && workflow.rejection_reason && (
        <p className="text-xs text-red-700 bg-red-100 rounded px-3 py-1.5">
          <span className="font-medium">Rejection reason:</span> {workflow.rejection_reason}
        </p>
      )}

      {/* Latest comment */}
      {workflow.latest_comment && workflow.status !== 'rejected' && (
        <p className="text-xs text-slate-600 italic">&ldquo;{workflow.latest_comment}&rdquo;</p>
      )}
    </div>
  );
}

/** Returns true when role is allowed to approve this workflow type. */
function canUserApprove(wt: WorkflowType, role: ApprovalBannerProps['userRole']): boolean {
  if (wt === 'audit_plan') return role === 'chief_audit_executive';
  // finding_signoff and report_release: both EL and CAE can approve
  return role === 'engagement_lead' || role === 'chief_audit_executive';
}
