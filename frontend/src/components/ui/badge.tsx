import { severityBgClass } from '@/lib/tokens';
import { clsx } from 'clsx';

interface BadgeProps {
  severity?: string;
  children: React.ReactNode;
  className?: string;
}

export function SeverityBadge({ severity = 'informational', children, className }: BadgeProps) {
  return (
    <span className={clsx(
      'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
      severityBgClass[severity] ?? severityBgClass.informational,
      className
    )}>
      {children}
    </span>
  );
}

interface StatusBadgeProps {
  status: string;
  children: React.ReactNode;
  className?: string;
}

const statusColors: Record<string, string> = {
  open: 'bg-red-100 text-red-800 border border-red-200',
  in_remediation: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
  remediated: 'bg-green-100 text-green-800 border border-green-200',
  accepted_risk: 'bg-gray-100 text-gray-700 border border-gray-200',
  false_positive: 'bg-gray-100 text-gray-500 border border-gray-200',
  closed: 'bg-gray-100 text-gray-400 border border-gray-200',
  pending_review: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
  accepted: 'bg-green-100 text-green-800 border border-green-200',
  rejected: 'bg-red-100 text-red-800 border border-red-200',
  archived: 'bg-gray-100 text-gray-500 border border-gray-200',
  planning: 'bg-blue-100 text-blue-800 border border-blue-200',
  fieldwork: 'bg-purple-100 text-purple-800 border border-purple-200',
  review: 'bg-orange-100 text-orange-800 border border-orange-200',
  reporting: 'bg-teal-100 text-teal-800 border border-teal-200',
  completed: 'bg-green-100 text-green-800 border border-green-200',
  cancelled: 'bg-gray-100 text-gray-500 border border-gray-200',
  // Portal / evidence request statuses
  pending: 'bg-blue-100 text-blue-800 border border-blue-200',
  in_progress: 'bg-purple-100 text-purple-800 border border-purple-200',
  submitted: 'bg-yellow-100 text-yellow-800 border border-yellow-200',
  // Portal finding statuses
  in_review: 'bg-indigo-100 text-indigo-800 border border-indigo-200',
  management_response_required: 'bg-orange-100 text-orange-800 border border-orange-200',
  // Management response statuses
  draft: 'bg-slate-100 text-slate-500 border border-slate-200',
  acknowledged: 'bg-teal-100 text-teal-800 border border-teal-200',
};

export function StatusBadge({ status, children, className }: StatusBadgeProps) {
  return (
    <span className={clsx(
      'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
      statusColors[status] ?? 'bg-gray-100 text-gray-600',
      className
    )}>
      {children}
    </span>
  );
}
