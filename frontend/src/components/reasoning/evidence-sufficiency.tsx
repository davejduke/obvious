'use client';
import type { EvidenceSufficiencyItem } from '@/lib/reasoning-types';

const statusConfig = {
  sufficient:   { bar: 'bg-green-500',  badge: 'bg-green-100 text-green-700' },
  partial:      { bar: 'bg-yellow-400', badge: 'bg-yellow-100 text-yellow-700' },
  insufficient: { bar: 'bg-red-500',    badge: 'bg-red-100 text-red-700' },
};

interface Props {
  items: EvidenceSufficiencyItem[];
}

export function EvidenceSufficiency({ items }: Props) {
  return (
    <div className="space-y-3" data-testid="evidence-sufficiency">
      {items.map(item => {
        const cfg = statusConfig[item.status];
        return (
          <div key={item.control_id} className="space-y-1.5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-sm font-medium text-slate-800 truncate">{item.control_title}</span>
                <span className="text-xs font-mono text-slate-400 shrink-0">{item.article_ref}</span>
              </div>
              <div className="flex items-center gap-2 shrink-0 ml-2">
                <span className="text-xs text-slate-500">{item.collected} / {item.required}</span>
                <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${cfg.badge}`}>
                  {item.sufficiency_pct}%
                </span>
              </div>
            </div>
            <div
              className="w-full bg-slate-100 rounded-full h-2 overflow-hidden"
              role="progressbar"
              aria-valuenow={item.sufficiency_pct}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={`${item.control_title} evidence sufficiency`}
            >
              <div
                className={`h-2 rounded-full transition-all duration-500 ${cfg.bar}`}
                style={{ width: `${Math.min(100, item.sufficiency_pct)}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}
