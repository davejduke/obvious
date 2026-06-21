'use client';
import { useState } from 'react';
import { PlayCircle, TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { WhatIfQuery, WhatIfResult, QualityGateControl } from '@/lib/reasoning-types';

interface Props {
  engagementId: string;
  controls: QualityGateControl[];
  onSimulate: (query: WhatIfQuery) => Promise<WhatIfResult>;
}

const GATE_CHANGE_LABELS: Record<WhatIfResult['gate_change'], { label: string; cls: string }> = {
  no_change:  { label: 'No gate change',     cls: 'text-slate-600 bg-slate-100' },
  now_passes: { label: 'Gate now PASSES ✓',  cls: 'text-green-700 bg-green-100' },
  now_blocks: { label: 'Gate now BLOCKS ✗',  cls: 'text-red-700 bg-red-100' },
};

export function WhatIfPanel({ engagementId, controls, onSimulate }: Props) {
  const [controlId, setControlId]         = useState(controls[0]?.control_id ?? '');
  const [action, setAction]               = useState<WhatIfQuery['action']>('add_evidence');
  const [qualityScore, setQualityScore]   = useState(80);
  const [count, setCount]                 = useState(1);
  const [result, setResult]               = useState<WhatIfResult | null>(null);
  const [loading, setLoading]             = useState(false);

  const handleSimulate = async () => {
    setLoading(true);
    try {
      const res = await onSimulate({
        engagement_id: engagementId,
        control_id: controlId,
        action,
        evidence_quality_score: qualityScore,
        count,
      });
      setResult(res);
    } finally {
      setLoading(false);
    }
  };

  const deltaPositive = (result?.delta ?? 0) > 0;
  const deltaZero     = (result?.delta ?? 0) === 0;

  return (
    <div className="space-y-4" data-testid="what-if-panel">
      <p className="text-xs text-slate-500">
        Simulate how adding or removing evidence items affects the overall confidence score.
        All simulations are deterministic — no LLM inference.
      </p>

      {/* Controls */}
      <div className="grid grid-cols-1 gap-3">
        {/* Control selector */}
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">Control</label>
          <select
            className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm text-slate-800 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            value={controlId}
            onChange={e => setControlId(e.target.value)}
            data-testid="what-if-control-select"
          >
            {controls.map(c => (
              <option key={c.control_id} value={c.control_id}>
                {c.control_title} ({c.article_ref})
              </option>
            ))}
          </select>
        </div>

        {/* Action */}
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">Action</label>
          <div className="flex rounded-md border border-slate-200 overflow-hidden">
            {(['add_evidence', 'remove_evidence'] as const).map(a => (
              <button
                key={a}
                className={`flex-1 py-2 text-sm font-medium transition-colors ${
                  action === a
                    ? 'bg-blue-600 text-white'
                    : 'bg-white text-slate-600 hover:bg-slate-50'
                }`}
                onClick={() => setAction(a)}
                data-testid={`action-${a}`}
              >
                {a === 'add_evidence' ? '+ Add evidence' : '− Remove evidence'}
              </button>
            ))}
          </div>
        </div>

        {/* Count + Quality */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1">
              Count
            </label>
            <input
              type="number"
              min={1}
              max={20}
              className="w-full border border-slate-200 rounded-md px-3 py-2 text-sm text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={count}
              onChange={e => setCount(Math.max(1, parseInt(e.target.value) || 1))}
              data-testid="what-if-count"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1">
              Quality score: <span className="font-semibold text-slate-800">{qualityScore}</span>
            </label>
            <input
              type="range"
              min={0}
              max={100}
              className="w-full mt-1"
              value={qualityScore}
              onChange={e => setQualityScore(parseInt(e.target.value))}
              data-testid="what-if-quality"
            />
          </div>
        </div>
      </div>

      <Button
        onClick={handleSimulate}
        disabled={loading || !controlId}
        className="w-full"
        data-testid="simulate-button"
      >
        <PlayCircle size={16} />
        {loading ? 'Simulating…' : 'Run Simulation'}
      </Button>

      {/* Result */}
      {result && (
        <div className="rounded-lg border border-slate-200 overflow-hidden" data-testid="what-if-result">
          {/* Narrative */}
          <div className="px-4 py-3 bg-slate-50 border-b border-slate-200">
            <p className="text-sm text-slate-700">{result.narrative}</p>
          </div>

          {/* Score delta */}
          <div className="px-4 py-3 flex items-center gap-4">
            <div className="text-center">
              <p className="text-xs text-slate-500 mb-0.5">Before</p>
              <p className="text-2xl font-bold text-slate-700">{result.original_score}</p>
            </div>
            <div className="flex-1 flex flex-col items-center">
              {deltaZero ? (
                <Minus size={20} className="text-slate-400" />
              ) : deltaPositive ? (
                <TrendingUp size={20} className="text-green-500" />
              ) : (
                <TrendingDown size={20} className="text-red-500" />
              )}
              <span className={`text-sm font-semibold ${
                deltaZero ? 'text-slate-400' :
                deltaPositive ? 'text-green-600' : 'text-red-600'
              }`}>
                {deltaPositive ? '+' : ''}{result.delta}
              </span>
            </div>
            <div className="text-center">
              <p className="text-xs text-slate-500 mb-0.5">After</p>
              <p className={`text-2xl font-bold ${
                deltaPositive ? 'text-green-600' :
                deltaZero ? 'text-slate-700' : 'text-red-600'
              }`}>{result.simulated_score}</p>
            </div>
          </div>

          {/* Gate change */}
          <div className="px-4 pb-3">
            <span
              data-testid="gate-change-badge"
              data-gate-change={result.gate_change}
              className={`inline-flex items-center text-xs font-medium px-2 py-1 rounded ${
                GATE_CHANGE_LABELS[result.gate_change].cls
              }`}
            >
              {GATE_CHANGE_LABELS[result.gate_change].label}
            </span>
          </div>

          {/* Factor breakdown */}
          <div className="border-t border-slate-100 px-4 py-3">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Affected Factors</p>
            <div className="space-y-2">
              {result.affected_factors.map(f => {
                const fd = f.simulated - f.original;
                return (
                  <div key={f.factor} className="flex items-center justify-between text-xs">
                    <span className="text-slate-600">{f.factor}</span>
                    <div className="flex items-center gap-2">
                      <span className="text-slate-500">{f.original}</span>
                      <span className="text-slate-300">→</span>
                      <span className={fd > 0 ? 'text-green-600 font-medium' : fd < 0 ? 'text-red-600 font-medium' : 'text-slate-600'}>
                        {f.simulated}
                      </span>
                      {fd !== 0 && (
                        <span className={`font-medium ${fd > 0 ? 'text-green-600' : 'text-red-600'}`}>
                          ({fd > 0 ? '+' : ''}{fd})
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
