'use client';
import { clsx } from 'clsx';
import { Link, ArrowDown } from 'lucide-react';
import type { ExtendedEvidence, EvidenceChainEvent } from '@/lib/evidence-mock-extended';
import { STAGE_LABELS, STAGE_COLORS, qualityColor, classificationColor } from '@/lib/evidence-mock-extended';

interface EvidenceChainPanelProps {
  evidence: ExtendedEvidence | null;
}

function ChainEventNode({ event, isLast }: { event: EvidenceChainEvent; isLast: boolean }) {
  return (
    <div className="flex gap-3">
      {/* Timeline track */}
      <div className="flex flex-col items-center flex-shrink-0">
        <div className={clsx('w-3 h-3 rounded-full flex-shrink-0 mt-0.5', STAGE_COLORS[event.stage])} />
        {!isLast && <div className="w-0.5 flex-1 bg-slate-200 mt-1 mb-1" />}
      </div>

      {/* Content */}
      <div className={clsx('pb-4 flex-1', isLast && 'pb-0')}>
        <div className="flex items-start justify-between gap-2">
          <div>
            <span
              className={clsx(
                'inline-block px-1.5 py-0.5 text-xs rounded font-medium text-white mb-1',
                STAGE_COLORS[event.stage],
              )}
            >
              {STAGE_LABELS[event.stage]}
            </span>
            <p className="text-xs font-medium text-slate-800">{event.action}</p>
            <p className="text-xs text-slate-500 mt-0.5">{event.detail}</p>
          </div>
        </div>

        {/* Meta */}
        <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-400">
          <span>{event.actor}</span>
          <span>{new Date(event.timestamp).toLocaleString()}</span>
        </div>

        {/* Metadata chips */}
        {Object.keys(event.metadata).length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1.5">
            {Object.entries(event.metadata).map(([k, v]) => (
              <span key={k} className="px-1.5 py-0.5 text-xs bg-slate-100 text-slate-500 rounded font-mono">
                {k}: {String(v)}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export function EvidenceChainPanel({ evidence }: EvidenceChainPanelProps) {
  if (!evidence) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-400">
        <Link size={32} className="mb-2 opacity-30" />
        <p className="text-sm font-medium">Chain Navigator</p>
        <p className="text-xs mt-1">Select evidence to trace its lifecycle</p>
      </div>
    );
  }

  const stages = [
    'source',
    'ingestion',
    'classification',
    'quality_assessment',
    'finding',
  ] as const;

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-200 bg-slate-50 flex-shrink-0">
        <h3 className="text-xs font-semibold text-slate-600 uppercase tracking-wide">
          Evidence Chain Navigator
        </h3>
        <p className="text-xs text-slate-400 mt-0.5 truncate">{evidence.title}</p>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto px-4 py-4">

        {/* Stage progress tracker */}
        <div className="flex items-center gap-1 mb-5 overflow-x-auto pb-1">
          {stages.map((stage, i) => {
            const event = evidence.chain.find((e) => e.stage === stage);
            const reached = !!event;
            return (
              <div key={stage} className="flex items-center gap-1 flex-shrink-0">
                <div
                  className={clsx(
                    'px-2 py-1 rounded text-xs font-medium whitespace-nowrap',
                    reached
                      ? `${STAGE_COLORS[stage]} text-white`
                      : 'bg-slate-100 text-slate-400',
                  )}
                >
                  {STAGE_LABELS[stage]}
                </div>
                {i < stages.length - 1 && (
                  <ArrowDown
                    size={10}
                    className={clsx(
                      'flex-shrink-0 rotate-[-90deg]',
                      reached ? 'text-slate-400' : 'text-slate-200',
                    )}
                  />
                )}
              </div>
            );
          })}
        </div>

        {/* Chain events */}
        <div className="space-y-0">
          {evidence.chain.map((event, i) => (
            <ChainEventNode
              key={event.id}
              event={event}
              isLast={i === evidence.chain.length - 1}
            />
          ))}
        </div>

        {/* Summary section */}
        <div className="mt-5 pt-4 border-t border-slate-100 space-y-3">
          <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
            Evidence Summary
          </h4>

          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500">Quality Score</span>
              <span className={clsx('font-semibold', qualityColor(evidence.quality.overall))}>
                {evidence.quality.overall}/100
              </span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500">Classification</span>
              <span
                className={clsx(
                  'px-1.5 py-0.5 rounded text-xs font-medium',
                  classificationColor(evidence.classification),
                )}
              >
                {evidence.classification.toUpperCase()}
              </span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500">Chain stages</span>
              <span className="text-slate-700 font-medium">{evidence.chain.length} / {stages.length}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500">Linked findings</span>
              <span className="text-slate-700 font-medium">{evidence.finding_ids.length}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500">Sufficient</span>
              <span
                className={clsx(
                  'font-medium',
                  evidence.is_sufficient === true
                    ? 'text-green-600'
                    : evidence.is_sufficient === false
                    ? 'text-red-500'
                    : 'text-slate-400',
                )}
              >
                {evidence.is_sufficient === true
                  ? 'Yes'
                  : evidence.is_sufficient === false
                  ? 'No'
                  : 'Not assessed'}
              </span>
            </div>
          </div>

          {/* Quality flags */}
          {evidence.quality.flags.length > 0 && (
            <div>
              <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">
                Quality Flags
              </h4>
              <ul className="space-y-1">
                {evidence.quality.flags.map((f) => (
                  <li key={f} className="text-xs text-orange-700 flex gap-1.5 items-start">
                    <span className="text-orange-400 mt-0.5">&#9679;</span>
                    {f}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
