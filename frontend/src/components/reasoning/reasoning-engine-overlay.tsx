'use client';
import { useState, useEffect, useCallback } from 'react';
import { X, Brain, ChevronDown } from 'lucide-react';
import { clsx } from 'clsx';
import { ConfidenceFactorBreakdown } from './confidence-factor-breakdown';
import { RiskHeatMap } from './risk-heat-map';
import { QualityGateStatus } from './quality-gate-status';
import { EvidenceSufficiency } from './evidence-sufficiency';
import { ScopeDagViewer } from './scope-dag-viewer';
import { WhatIfPanel } from './what-if-panel';
import type { ReasoningEngineState, WhatIfQuery, WhatIfResult } from '@/lib/reasoning-types';
import { simulateWhatIf } from '@/lib/mock-data';

type Tab = 'confidence' | 'heatmap' | 'quality' | 'sufficiency' | 'dag' | 'whatif';

const TABS: { id: Tab; label: string }[] = [
  { id: 'confidence',  label: 'Confidence' },
  { id: 'heatmap',     label: 'Risk Heat Map' },
  { id: 'quality',     label: 'Quality Gates' },
  { id: 'sufficiency', label: 'Evidence' },
  { id: 'dag',         label: 'Scope DAG' },
  { id: 'whatif',      label: 'What If?' },
];

interface Props {
  state: ReasoningEngineState;
  open: boolean;
  onClose: () => void;
}

export function ReasoningEngineOverlay({ state, open, onClose }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>('confidence');

  // Escape key to close
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, onClose]);

  const handleSimulate = useCallback(
    async (query: WhatIfQuery): Promise<WhatIfResult> => {
      // Deterministic simulation — no async needed; wrapped for API shape compatibility
      return Promise.resolve(simulateWhatIf(query));
    },
    [],
  );

  if (!open) return null;

  const score = state.overall_confidence.score;
  const scoreColor =
    score >= 75 ? 'text-green-600' :
    score >= 55 ? 'text-yellow-600' : 'text-red-600';

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col bg-slate-900/60 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label="Reasoning Engine Overlay"
      data-testid="reasoning-engine-overlay"
    >
      {/* Panel */}
      <div className="flex flex-col bg-white h-full max-h-screen overflow-hidden shadow-2xl">

        {/* Header */}
        <div className="flex items-center gap-4 px-6 py-4 border-b border-slate-200 bg-slate-50 shrink-0">
          <div className="flex items-center gap-2">
            <Brain size={20} className="text-blue-600" />
            <div>
              <h2 className="text-base font-semibold text-slate-900">Reasoning Engine</h2>
              <p className="text-xs text-slate-500">{state.engagement_name}</p>
            </div>
          </div>

          {/* Overall confidence badge */}
          <div className="flex items-center gap-2 ml-4">
            <span className="text-xs text-slate-500">Confidence:</span>
            <span className={`text-xl font-bold ${scoreColor}`}>{score}</span>
            <span className="text-xs text-slate-400">/ 100</span>
          </div>

          {/* Spacer */}
          <div className="flex-1" />

          {/* Close */}
          <button
            onClick={onClose}
            className="p-1.5 rounded-md text-slate-400 hover:text-slate-700 hover:bg-slate-200 transition-colors"
            aria-label="Close overlay"
            data-testid="overlay-close"
          >
            <X size={18} />
          </button>
        </div>

        {/* Tab bar */}
        <div className="flex items-center gap-1 px-6 border-b border-slate-200 bg-white shrink-0 overflow-x-auto">
          {TABS.map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={clsx(
                'px-4 py-3 text-sm font-medium whitespace-nowrap border-b-2 transition-colors',
                activeTab === tab.id
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300',
              )}
              data-testid={`tab-${tab.id}`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-6">
          {activeTab === 'confidence' && (
            <ConfidenceFactorBreakdown confidence={state.overall_confidence} />
          )}

          {activeTab === 'heatmap' && (
            <div>
              <p className="text-sm text-slate-500 mb-4">
                Impact × Likelihood matrix. Each bubble represents a control placed
                by deterministic risk-weight mapping — no LLM inference. Hover for details.
              </p>
              <RiskHeatMap data={state.heat_map} />
            </div>
          )}

          {activeTab === 'quality' && (
            <QualityGateStatus gates={state.quality_gates} />
          )}

          {activeTab === 'sufficiency' && (
            <div>
              <p className="text-sm text-slate-500 mb-4">
                Evidence collected vs. required per control objective.
              </p>
              <EvidenceSufficiency items={state.evidence_sufficiency} />
            </div>
          )}

          {activeTab === 'dag' && (
            <div>
              <p className="text-sm text-slate-500 mb-4">
                Scope execution DAG. Badge shows evidence count per node.
                Click nodes with ▼ to expand children. Scroll to zoom.
              </p>
              <ScopeDagViewer nodes={state.dag_nodes} edges={state.dag_edges} />
            </div>
          )}

          {activeTab === 'whatif' && (
            <WhatIfPanel
              engagementId={state.engagement_id}
              controls={state.quality_gates}
              onSimulate={handleSimulate}
            />
          )}
        </div>
      </div>
    </div>
  );
}

// ---- Trigger button -----
interface TriggerProps {
  state: ReasoningEngineState;
}

export function ReasoningEngineTrigger({ state }: TriggerProps) {
  const [open, setOpen] = useState(false);
  const score = state.overall_confidence.score;
  const scoreColor =
    score >= 75 ? 'bg-green-100 text-green-700 border-green-200' :
    score >= 55 ? 'bg-yellow-100 text-yellow-700 border-yellow-200' :
                  'bg-red-100 text-red-700 border-red-200';

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className={clsx(
          'inline-flex items-center gap-2 px-3 py-1.5 rounded-md border text-sm font-medium transition-colors hover:opacity-90',
          scoreColor,
        )}
        data-testid="reasoning-engine-trigger"
      >
        <Brain size={14} />
        Reasoning Engine
        <span className="font-bold">{score}</span>
        <ChevronDown size={12} />
      </button>

      <ReasoningEngineOverlay
        state={state}
        open={open}
        onClose={() => setOpen(false)}
      />
    </>
  );
}
