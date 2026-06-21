import { clsx } from 'clsx';
import type { ReactNode } from 'react';

interface CardProps {
  className?: string;
  children: ReactNode;
}

export function Card({ className, children }: CardProps) {
  return (
    <div className={clsx('rounded-lg border shadow-sm bg-[var(--bg-surface)] border-[var(--border-default)]', className)}>
      {children}
    </div>
  );
}

export function CardHeader({ className, children }: CardProps) {
  return (
    <div className={clsx('px-6 py-4 border-b border-[var(--border-default)]', className)}>
      {children}
    </div>
  );
}

export function CardBody({ className, children }: CardProps) {
  return (
    <div className={clsx('px-6 py-4', className)}>
      {children}
    </div>
  );
}

const COLOR_MAP: Record<string, string> = {
  blue:   'border-t-blue-500',
  green:  'border-t-green-500',
  red:    'border-t-red-500',
  amber:  'border-t-amber-500',
  purple: 'border-t-purple-500',
  slate:  'border-t-slate-400',
};

interface MetricCardProps {
  label: string;
  value: string | number;
  subtitle?: string;
  trend?: { value: number; positive: boolean };
  /** Legacy prop — kept for back-compat */
  accent?: string;
  /** New: colour key from COLOR_MAP */
  color?: string;
  icon?: ReactNode;
}

export function MetricCard({ label, value, subtitle, trend, accent, color, icon }: MetricCardProps) {
  const topBorder = color ? (COLOR_MAP[color] ?? 'border-t-blue-500') : (accent ?? 'border-blue-500');
  return (
    <div className={clsx(
      'rounded-lg border shadow-sm border-t-4 bg-[var(--bg-surface)] border-[var(--border-default)]',
      topBorder
    )}>
      <div className="px-5 py-4">
        <div className="flex items-center justify-between mb-1">
          <p className="text-xs font-medium uppercase tracking-wide text-[var(--text-muted)]">{label}</p>
          {icon && <span className="text-[var(--text-muted)]">{icon}</span>}
        </div>
        <p className="text-2xl font-bold text-[var(--text-primary)]">{value}</p>
        {subtitle && <p className="mt-0.5 text-xs text-[var(--text-secondary)]">{subtitle}</p>}
        {trend && (
          <p className={clsx('mt-1 text-xs font-medium', trend.positive ? 'text-green-600' : 'text-red-600')}>
            {trend.positive ? '+' : ''}{trend.value}% vs last period
          </p>
        )}
      </div>
    </div>
  );
}
