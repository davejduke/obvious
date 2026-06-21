'use client';
import { CheckCircle2, XCircle, Clock } from 'lucide-react';
import type { QualityGateControl, GateStatus } from '@/lib/reasoning-types';

const statusConfig: Record<GateStatus, {
  icon: React.ReactNode;
  badge: string;
  row: string;
}> = {
  passed:  {
    icon: <CheckCircle2 size={16} className="text-green-500" />,
    badge: 'bg-green-100 text-green-700',
    row: 'hover:bg-green-50',
  },
  blocked: {
    icon: <XCircle size={16} className="text-red-500" />,
    badge: 'bg-red-100 text-red-700',
    row: 'hover:bg-red-50 bg-red-50/40',
  },
  pending: {
    icon: <Clock size={16} className="text-slate-400" />,
    badge: 'bg-slate-100 text-slate-500',
    row: 'hover:bg-slate-50',
  },
};

const blockReasonLabels: Record<string, string> = {
  floor_not_met:          'Hard floor not met',
  insufficient_evidence:  'Insufficient evidence',
  unresolved_conflicts:   'Unresolved conflicts',
};

interface Props {
  gates: QualityGateControl[];
}

export function QualityGateStatus({ gates }: Props) {
  const passed  = gates.filter(g => g.status === 'passed').length;
  const blocked = gates.filter(g => g.status === 'blocked').length;

  return (
    <div data-testid="quality-gate-status">
      {/* Summary bar */}
      <div className="flex items-center gap-4 mb-4 text-sm">
        <div className="flex items-center gap-1.5 text-green-600">
          <CheckCircle2 size={14} />
          <span className="font-semibold">{passed}</span>
          <span className="text-slate-500">passed</span>
        </div>
        <div className="flex items-center gap-1.5 text-red-600">
          <XCircle size={14} />
          <span className="font-semibold">{blocked}</span>
          <span className="text-slate-500">blocked</span>
        </div>
        <div className="flex-1 h-2 bg-slate-100 rounded-full overflow-hidden">
          <div
            className="h-2 bg-green-500 rounded-full"
            style={{ width: `${(passed / gates.length) * 100}%` }}
          />
        </div>
        <span className="text-xs text-slate-500">
          {Math.round((passed / gates.length) * 100)}%
        </span>
      </div>

      {/* Gate table */}
      <div className="overflow-hidden rounded-lg border border-slate-200">
        <table className="w-full text-sm" data-testid="gate-table">
          <thead className="bg-slate-50 text-xs text-slate-500 uppercase tracking-wide">
            <tr>
              <th className="px-4 py-2.5 text-left">Control</th>
              <th className="px-4 py-2.5 text-left">Article</th>
              <th className="px-4 py-2.5 text-center">Status</th>
              <th className="px-4 py-2.5 text-right">Score / Threshold</th>
              <th className="px-4 py-2.5 text-right">Evidence</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {gates.map(gate => {
              const cfg = statusConfig[gate.status];
              return (
                <tr key={gate.control_id} className={`${cfg.row} transition-colors`}>
                  <td className="px-4 py-3">
                    <p className="font-medium text-slate-900">{gate.control_title}</p>
                    {gate.block_reasons.length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {gate.block_reasons.map(r => (
                          <span key={r} className="text-xs bg-red-100 text-red-600 px-1.5 py-0.5 rounded">
                            {blockReasonLabels[r] ?? r}
                          </span>
                        ))}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-500 text-xs font-mono uppercase">
                    {gate.article_ref}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${cfg.badge}`}>
                      {cfg.icon}
                      {gate.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <div className="w-16 bg-slate-100 rounded-full h-1.5 overflow-hidden">
                        <div
                          className={`h-1.5 rounded-full ${gate.score >= gate.threshold ? 'bg-green-500' : 'bg-red-500'}`}
                          style={{ width: `${gate.score}%` }}
                        />
                      </div>
                      <span className="text-slate-700 font-medium">{gate.score}</span>
                      <span className="text-slate-400">/ {gate.threshold}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right text-slate-600">
                    <span className={gate.evidence_count >= gate.required_evidence ? 'text-green-600 font-medium' : 'text-red-600 font-medium'}>
                      {gate.evidence_count}
                    </span>
                    <span className="text-slate-400"> / {gate.required_evidence}</span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
