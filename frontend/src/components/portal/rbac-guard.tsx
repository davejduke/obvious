'use client';
import { usePersonaStore } from '@/store/persona';
import { isPortalAuthorized } from '@/store/portal';
import { Shield, Lock } from 'lucide-react';
import Link from 'next/link';

interface RbacGuardProps {
  children: React.ReactNode;
}

/**
 * RbacGuard — restricts /portal/ routes to auditee personas only.
 *
 * In production this enforcement would live at the middleware layer using
 * JWT claims, with this component acting as an additional client-side guard.
 * For the mock UI the persona store is the source of truth.
 */
export function RbacGuard({ children }: RbacGuardProps) {
  const { currentPersona, setPersona } = usePersonaStore();
  const authorized = isPortalAuthorized(currentPersona);

  if (!authorized) {
    return (
      <div className="flex items-center justify-center h-full min-h-[60vh]">
        <div className="max-w-md w-full bg-white rounded-xl border border-slate-200 shadow-sm p-8 text-center">
          <div className="w-14 h-14 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <Lock size={24} className="text-red-500" />
          </div>
          <h2 className="text-lg font-semibold text-slate-900 mb-2">Access Restricted</h2>
          <p className="text-sm text-slate-500 mb-1">
            The Auditee Portal is only accessible to auditee accounts.
          </p>
          <p className="text-xs text-slate-400 mb-6">
            Current role: <span className="font-mono bg-slate-100 px-1 rounded">{currentPersona}</span>
          </p>
          <div className="space-y-3">
            <button
              onClick={() => setPersona('auditee_ciso')}
              className="w-full inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 transition-colors"
            >
              <Shield size={15} />
              Switch to Auditee / CISO view
            </button>
            <Link
              href="/dashboard"
              className="w-full inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-white text-slate-700 text-sm font-medium rounded-lg border border-slate-300 hover:bg-slate-50 transition-colors"
            >
              Back to Main App
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
