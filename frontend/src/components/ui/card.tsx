import { clsx } from 'clsx';

interface CardProps {
  className?: string;
  children: React.ReactNode;
}

export function Card({ className, children }: CardProps) {
  return (
    <div className={clsx('bg-white rounded-lg border border-slate-200 shadow-sm', className)}>
      {children}
    </div>
  );
}

export function CardHeader({ className, children }: CardProps) {
  return (
    <div className={clsx('px-6 py-4 border-b border-slate-100', className)}>
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

interface MetricCardProps {
  label: string;
  value: string | number;
  subtitle?: string;
  trend?: { value: number; positive: boolean };
  accent?: string;
}

export function MetricCard({ label, value, subtitle, trend, accent = 'border-blue-500' }: MetricCardProps) {
  return (
    <div className={clsx('bg-white rounded-lg border border-slate-200 shadow-sm border-t-4', accent)}>
      <div className="px-6 py-5">
        <p className="text-sm font-medium text-slate-500 uppercase tracking-wide">{label}</p>
        <p className="mt-2 text-3xl font-bold text-slate-900">{value}</p>
        {subtitle && <p className="mt-1 text-sm text-slate-500">{subtitle}</p>}
        {trend && (
          <p className={clsx('mt-1 text-xs font-medium', trend.positive ? 'text-green-600' : 'text-red-600')}>
            {trend.positive ? '+' : ''}{trend.value}% vs last period
          </p>
        )}
      </div>
    </div>
  );
}
