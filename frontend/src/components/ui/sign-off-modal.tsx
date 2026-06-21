'use client';
/**
 * SignOffModal — captures approver name, role, and an optional comment
 * before confirming an approval or rejection decision.
 *
 * Used by the Finding Sign-Off and Audit Plan Approval workflows.
 */

import { useState } from 'react';
import { CheckCircle, XCircle, X } from 'lucide-react';
import { Button } from './button';
import { clsx } from 'clsx';

export type SignOffDecision = 'approve' | 'reject';

export interface SignOffPayload {
  decision: SignOffDecision;
  actorEmail: string;
  actorRole: 'engagement_lead' | 'chief_audit_executive' | 'reviewer';
  comment: string;
  rejectionReason?: string;
}

interface SignOffModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (payload: SignOffPayload) => void;
  decision: SignOffDecision;
  /** e.g. "Finding F-2024-001" or "Audit Plan — NIS 2 Article 21 Audit 2024" */
  resourceLabel: string;
  workflowType: 'audit_plan' | 'finding_signoff' | 'report_release';
}

const roleOptions: Array<{ value: SignOffPayload['actorRole']; label: string }> = [
  { value: 'engagement_lead', label: 'Engagement Lead' },
  { value: 'chief_audit_executive', label: 'Chief Audit Executive' },
  { value: 'reviewer', label: 'Reviewer' },
];

export function SignOffModal({
  isOpen,
  onClose,
  onConfirm,
  decision,
  resourceLabel,
  workflowType,
}: SignOffModalProps) {
  const [actorEmail, setActorEmail] = useState('');
  const [actorRole, setActorRole] = useState<SignOffPayload['actorRole']>('engagement_lead');
  const [comment, setComment] = useState('');
  const [rejectionReason, setRejectionReason] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  if (!isOpen) return null;

  const isApprove = decision === 'approve';

  function validate(): boolean {
    const e: Record<string, string> = {};
    if (!actorEmail.trim()) e.actorEmail = 'Email is required';
    if (!actorEmail.includes('@')) e.actorEmail = 'Enter a valid email';
    if (decision === 'reject' && !rejectionReason.trim()) {
      e.rejectionReason = 'Rejection reason is required';
    }
    setErrors(e);
    return Object.keys(e).length === 0;
  }

  function handleConfirm() {
    if (!validate()) return;
    onConfirm({
      decision,
      actorEmail: actorEmail.trim(),
      actorRole,
      comment: comment.trim(),
      rejectionReason: decision === 'reject' ? rejectionReason.trim() : undefined,
    });
    resetForm();
  }

  function resetForm() {
    setActorEmail('');
    setComment('');
    setRejectionReason('');
    setErrors({});
  }

  function handleClose() {
    resetForm();
    onClose();
  }

  const workflowLabels: Record<typeof workflowType, string> = {
    audit_plan: 'Audit Plan Approval',
    finding_signoff: 'Finding Sign-Off',
    report_release: 'Report Release Approval',
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className={clsx(
          'px-6 py-4 flex items-center justify-between',
          isApprove ? 'bg-green-50 border-b border-green-200' : 'bg-red-50 border-b border-red-200',
        )}>
          <div className="flex items-center gap-2">
            {isApprove
              ? <CheckCircle size={20} className="text-green-600" />
              : <XCircle size={20} className="text-red-600" />}
            <div>
              <p className="font-semibold text-slate-900">
                {isApprove ? 'Confirm Approval' : 'Confirm Rejection'}
              </p>
              <p className="text-xs text-slate-500">{workflowLabels[workflowType]}</p>
            </div>
          </div>
          <button onClick={handleClose} className="text-slate-400 hover:text-slate-600 transition-colors">
            <X size={18} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4">
          <p className="text-sm text-slate-600">
            You are {isApprove ? 'approving' : 'rejecting'}{' '}
            <span className="font-medium text-slate-900">{resourceLabel}</span>.
          </p>

          {/* Actor email */}
          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1">Your Email *</label>
            <input
              type="email"
              value={actorEmail}
              onChange={e => setActorEmail(e.target.value)}
              placeholder="auditor@example.com"
              className={clsx(
                'w-full text-sm border rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500',
                errors.actorEmail ? 'border-red-400' : 'border-slate-200',
              )}
            />
            {errors.actorEmail && <p className="text-xs text-red-500 mt-1">{errors.actorEmail}</p>}
          </div>

          {/* Role */}
          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1">Your Role *</label>
            <select
              value={actorRole}
              onChange={e => setActorRole(e.target.value as SignOffPayload['actorRole'])}
              className="w-full text-sm border border-slate-200 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {roleOptions.map(r => (
                <option key={r.value} value={r.value}>{r.label}</option>
              ))}
            </select>
          </div>

          {/* Rejection reason — only for reject decisions */}
          {!isApprove && (
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">Rejection Reason *</label>
              <textarea
                value={rejectionReason}
                onChange={e => setRejectionReason(e.target.value)}
                placeholder="Explain why the submission is being rejected..."
                rows={3}
                className={clsx(
                  'w-full text-sm border rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none',
                  errors.rejectionReason ? 'border-red-400' : 'border-slate-200',
                )}
              />
              {errors.rejectionReason && <p className="text-xs text-red-500 mt-1">{errors.rejectionReason}</p>}
            </div>
          )}

          {/* Optional comment */}
          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1">Comment (optional)</label>
            <textarea
              value={comment}
              onChange={e => setComment(e.target.value)}
              placeholder={isApprove ? 'Add any notes for the record...' : 'Additional context...'}
              rows={2}
              className="w-full text-sm border border-slate-200 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-slate-100 flex justify-end gap-3">
          <Button variant="ghost" size="sm" onClick={handleClose}>Cancel</Button>
          <Button
            variant={isApprove ? 'primary' : 'danger'}
            size="sm"
            onClick={handleConfirm}
          >
            {isApprove ? <CheckCircle size={13} /> : <XCircle size={13} />}
            {isApprove ? 'Confirm Approval' : 'Confirm Rejection'}
          </Button>
        </div>
      </div>
    </div>
  );
}
