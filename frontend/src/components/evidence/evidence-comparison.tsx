'use client';
import { clsx } from 'clsx';
import { X, ArrowLeftRight } from 'lucide-react';
import type { ExtendedEvidence, QualityDimension } from '@/lib/evidence-mock-extended';
import { qualityColor, qualityBgColor, classificationColor } from '@/lib/evidence-mock-extended';
import { StatusBadge } from '@/components/ui/badge';

const QUALITY_DIMENSIONS: { key: QualityDimension; label: string }[] = [
  { key: 'completeness', label: 'Completeness' },
  { key: 'accuracy', label: 'Accuracy' },
  { key: 'timeliness', label: 'Timeliness' },
  { key: 'relevance', label: 'Relevance' },
  { key: 'reliability', label: 'Reliability' },
];

interface EvidenceComparisonProps {
  items: ExtendedEvidence[];
  onRemove: (id: string) => void;
  onClose: () => void;
}

function DimensionRow({
  label,
  scoreA,
  scoreB,
}: {
  label: string;
  scoreA: number;
  scoreB: number;
}) {
  const diff = scoreA - scoreB;
  return (
    <tr className="border-b border-slate-100">
      <td className="py-2 text-xs text-slate-500 pr-3 whitespace-nowrap">{label}</td>
      <td className="py-2 text-center">
        <div className="flex items-center gap-1.5">
          <div className="flex-1 h-1.5 bg-slate-100 rounded-full overflow-hidden">
            <div
              className={clsx('h-full rounded-full', qualityBgColor(scoreA))}
              style={{ width: `${scoreA}%` }}
            />
          </div>
          <span className={clsx('text-xs font-medium w-7 text-right', qualityColor(scoreA))}>
            {scoreA}
          </span>
        </div>
      </td>
      <td className="py-2 px-3 text-center">
        <span
          className={clsx(
            'text-xs font-medium',
            diff > 0 ? 'text-green-600' : diff < 0 ? 'text-red-500' : 'text-slate-400',
          )}
        >
          {diff > 0 ? `+${diff}` : diff === 0 ? '=' : diff}
        </span>
      </td>
      <td className="py-2 text-center">
        <div className="flex items-center gap-1.5">
          <span className={clsx('text-xs font-medium w-7', qualityColor(scoreB))}>
            {scoreB}
          </span>
          <div className="flex-1 h-1.5 bg-slate-100 rounded-full overflow-hidden">
            <div
              className={clsx('h-full rounded-full', qualityBgColor(scoreB))}
              style={{ width: `${scoreB}%` }}
            />
          </div>
        </div>
      </td>
    </tr>
  );
}

export function EvidenceComparison({ items, onRemove, onClose }: EvidenceComparisonProps) {
  if (items.length < 2) {
    return (
      <div className="border-t border-slate-200 bg-blue-50 px-6 py-4 text-center">
        <div className="flex items-center justify-center gap-2 text-blue-700 text-sm">
          <ArrowLeftRight size={16} />
          <span>
            Select {2 - items.length} more evidence item{items.length === 0 ? 's' : ''} to compare
            (click the <strong>checkbox icon</strong> next to items in the list)
          </span>
        </div>
      </div>
    );
  }

  const [a, b] = items;

  return (
    <div className="border-t-2 border-blue-200 bg-white shadow-lg">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-3 bg-blue-50 border-b border-blue-200">
        <div className="flex items-center gap-2 text-sm font-semibold text-blue-800">
          <ArrowLeftRight size={16} />
          Evidence Comparison
        </div>
        <button
          onClick={onClose}
          className="text-slate-400 hover:text-slate-600 p-1 rounded transition-colors"
        >
          <X size={16} />
        </button>
      </div>

      {/* Comparison body */}
      <div className="overflow-x-auto">
        <div className="min-w-[640px] px-6 py-4">
          {/* Evidence headers */}
          <div className="grid grid-cols-2 gap-6 mb-4">
            {[a, b].map((ev) => (
              <div key={ev.id} className="relative bg-slate-50 rounded-lg p-3">
                <button
                  onClick={() => onRemove(ev.id)}
                  className="absolute top-2 right-2 text-slate-300 hover:text-slate-500"
                >
                  <X size={12} />
                </button>
                <p className="text-sm font-semibold text-slate-800 pr-5 leading-snug">{ev.title}</p>
                <div className="flex items-center gap-2 mt-1.5 flex-wrap">
                  <StatusBadge status={ev.status}>{ev.status.replace('_', ' ')}</StatusBadge>
                  <span
                    className={clsx(
                      'px-1.5 py-0.5 text-xs rounded font-medium',
                      classificationColor(ev.classification),
                    )}
                  >
                    {ev.classification.toUpperCase()}
                  </span>
                  <span className={clsx('text-sm font-bold', qualityColor(ev.quality.overall))}>
                    Q:{ev.quality.overall}
                  </span>
                </div>
                <p className="text-xs text-slate-400 mt-1">{ev.source_type.replace(/_/g, ' ')} &bull; {ev.collection_date}</p>
              </div>
            ))}
          </div>

          {/* Quality score comparison table */}
          <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">
            Quality Dimensions
          </h4>
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-200">
                <th className="text-left text-xs text-slate-400 py-1 pr-3 font-normal">Dimension</th>
                <th className="text-left text-xs text-slate-500 py-1 font-medium">{a.id}</th>
                <th className="text-center text-xs text-slate-400 py-1 px-3 font-normal">Diff</th>
                <th className="text-left text-xs text-slate-500 py-1 font-medium">{b.id}</th>
              </tr>
            </thead>
            <tbody>
              {QUALITY_DIMENSIONS.map((dim) => (
                <DimensionRow
                  key={dim.key}
                  label={dim.label}
                  scoreA={a.quality.dimensions[dim.key]}
                  scoreB={b.quality.dimensions[dim.key]}
                />
              ))}
              <DimensionRow
                label="Overall"
                scoreA={a.quality.overall}
                scoreB={b.quality.overall}
              />
            </tbody>
          </table>

          {/* Tag comparison */}
          <div className="grid grid-cols-2 gap-6 mt-4">
            {[a, b].map((ev) => (
              <div key={ev.id}>
                <p className="text-xs font-medium text-slate-500 mb-1.5">Tags — {ev.id}</p>
                <div className="flex flex-wrap gap-1">
                  {ev.tags.map((t) => (
                    <span
                      key={t.label}
                      className={clsx(
                        'px-1.5 py-0.5 text-xs rounded font-medium',
                        t.category === 'article' && 'bg-blue-100 text-blue-800',
                        t.category === 'domain' && 'bg-purple-100 text-purple-800',
                        t.category === 'control' && 'bg-green-100 text-green-800',
                      )}
                    >
                      {t.label}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
