'use client';
/**
 * PasswordConfirmDialog — critical action gate for report release.
 *
 * Requires the user to re-enter their password before the report
 * is locked and released. This is a second-factor UX pattern —
 * the actual credential verification happens server-side via the
 * approval service's Lock endpoint.
 */

import { useState } from 'react';
import { Lock, Eye, EyeOff, AlertTriangle, X } from 'lucide-react';
import { Button } from './button';
import { clsx } from 'clsx';

export interface PasswordConfirmPayload {
  password: string;
  actorEmail: string;
}

interface PasswordConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (payload: PasswordConfirmPayload) => void;
  reportName: string;
}

export function PasswordConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  reportName,
}: PasswordConfirmDialogProps) {
  const [password, setPassword] = useState('');
  const [actorEmail, setActorEmail] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  if (!isOpen) return null;

  function validate(): boolean {
    const e: Record<string, string> = {};
    if (!actorEmail.trim() || !actorEmail.includes('@')) e.actorEmail = 'Valid email required';
    if (!password) e.password = 'Password is required';
    setErrors(e);
    return Object.keys(e).length === 0;
  }

  function handleConfirm() {
    if (!validate()) return;
    onConfirm({ password, actorEmail: actorEmail.trim() });
    setPassword('');
    setActorEmail('');
    setErrors({});
  }

  function handleClose() {
    setPassword('');
    setActorEmail('');
    setErrors({});
    onClose();
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-sm mx-4 overflow-hidden">
        {/* Header */}
        <div className="bg-blue-900 px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2 text-white">
            <Lock size={18} />
            <div>
              <p className="font-semibold text-sm">Lock & Release Report</p>
              <p className="text-blue-200 text-xs">Password re-entry required</p>
            </div>
          </div>
          <button onClick={handleClose} className="text-blue-200 hover:text-white transition-colors">
            <X size={16} />
          </button>
        </div>

        {/* Warning */}
        <div className="px-6 pt-5 pb-3">
          <div className="flex items-start gap-3 bg-amber-50 border border-amber-200 rounded-lg px-4 py-3">
            <AlertTriangle size={16} className="text-amber-600 flex-shrink-0 mt-0.5" />
            <div className="text-xs text-amber-800">
              <p className="font-semibold">This action is irreversible.</p>
              <p className="mt-1">
                Locking <span className="font-medium">{reportName}</span> will make it read-only and
                permanently seal its contents. This cannot be undone.
              </p>
            </div>
          </div>
        </div>

        {/* Form */}
        <div className="px-6 py-4 space-y-4">
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

          <div>
            <label className="block text-xs font-medium text-slate-700 mb-1">
              Confirm Password *
            </label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={e => setPassword(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleConfirm()}
                placeholder="Enter your password to confirm"
                className={clsx(
                  'w-full text-sm border rounded-md px-3 py-2 pr-10 focus:outline-none focus:ring-2 focus:ring-blue-500',
                  errors.password ? 'border-red-400' : 'border-slate-200',
                )}
              />
              <button
                type="button"
                onClick={() => setShowPassword(v => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
              >
                {showPassword ? <EyeOff size={15} /> : <Eye size={15} />}
              </button>
            </div>
            {errors.password && <p className="text-xs text-red-500 mt-1">{errors.password}</p>}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-slate-100 flex justify-end gap-3">
          <Button variant="ghost" size="sm" onClick={handleClose}>Cancel</Button>
          <Button variant="primary" size="sm" onClick={handleConfirm} className="bg-blue-900 border-blue-900 hover:bg-blue-950">
            <Lock size={13} /> Lock Report
          </Button>
        </div>
      </div>
    </div>
  );
}
