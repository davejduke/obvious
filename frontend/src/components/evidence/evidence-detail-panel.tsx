'use client';
import { clsx } from 'clsx';
import { FileText, Calendar, User, Shield, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';
import type { ExtendedEvidence, QualityDimension } from '@/lib/evidence-mock-extended';
import { qualityColor, qualityBgColor, classificationColor } from '@/lib/evidence-mock-extended';
import { StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

interface EvidenceDetailPanelProps {
  evidence: ExtendedEvidence | null;
  onReassess: (id: string) => void;
  onReclassify: (id: string, status: string) => void;
}

const QUALITY_DIMENSIONS: { key: QualityDimension; label: string }[] = [
  { key: 'completeness', label: 'Completeness' },
  { key: 'accuracy', label: 'Accuracy' },
  { key: 'timeliness', label: 'Timeliness' },
  { key: 'relevance', label: 'Relevance' },
  { key: 'reliability', label: 'Reliability' },
];

function QualityBar({ score, label }: { score: number; label: string }) {
  return (
    <div>
      <div className="flex justify-between items-center mb-1">
        <span className="text-xs text-slate-500">{label}</span>
        <span className={clsx('text-xs font-medium', qualityColor(score))}>{score}</span>
      </div>
      <div className="h-1.5 bg-slate-100 rounded-full overflow-hidden">
        <div
          className={clsx('h-full rounded-full transition-all', qualityBgColor(score))}
          style={{ width: `${score}%` }}
        />
      </div>
    </div>
  );
}

export function EvidenceDetailPanel({
  evidence,
  onReassess,
  onReclassify,
}: EvidenceDetailPanelProps) {
  if (!evidence) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-400">
        <FileText size={40} className="mb-3 opacity-30" />
        <p className="text-sm font-medium">Select an evidence item</p>
        <p className="text-xs mt-1">Click any item in the list to view details</p>
      </div>
    );
  }

  const { quality } = evidence;

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="px-5 py-4 border-b border-slate-200 flex-shrink-0 bg-white">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h2 className="text-base font-semibold text-slate-900 leading-snug">{evidence.title}</h2>
            <div className="flex items-center gap-2 mt-1.5 flex-wrap">
              <StatusBadge status={evidence.status}>
                {evidence.status.replace('_', ' ')}
              </StatusBadge>
              <span
                className={clsx(
                  'px-2 py-0.5 text-xs rounded font-medium',
                  classificationColor(evidence.classification),
                )}
              >
                {evidence.classification.toUpperCase()}
              </span>
              <span className="text-xs text-slate-500 font-mono">{evidence.control_id}</span>
            </div>
          </div>
          {/* Overall quality score */}
          <div className="flex-shrink-0 text-center">
            <div
              className={clsx(
                'text-2xl font-bold leading-none',
                qualityColor(quality.overall),
              )}
            >
              {quality.overall}
            </div>
            <div className="text-xs text-slate-400 mt-0.5">Quality</div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 mt-3">
          <Button variant="secondary" size="sm" onClick={() => onReassess(evidence.id)}>
            Reassess Quality
          </Button>
          {evidence.status === 'pending_review' && (
            <>
              <Button
                size="sm"
                onClick={() => onReclassify(evidence.id, 'accepted')}
                className="bg-green-600 hover:bg-green-700 border-green-600 text-white"
              >
                <CheckCircle size={13} /> Accept
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={() => onReclassify(evidence.id, 'rejected')}
              >
                <XCircle size={13} /> Reject
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-5">
        {/* Description */}
        {evidence.description && (
          <div>
            <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Description</h3>
            <p className="text-sm text-slate-700 leading-relaxed">{evidence.description}</p>
          </div>
        )}

        {/* Metadata grid */}
        <div className="grid grid-cols-2 gap-3">
          <div className="flex items-center gap-2">
            <Calendar size={13} className="text-slate-400" />
            <div>
              <p className="text-xs text-slate-400">Collected</p>
              <p className="text-sm text-slate-700">{evidence.collection_date}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <User size={13} className="text-slate-400" />
            <div>
              <p className="text-xs text-slate-400">Uploaded by</p>
              <p className="text-sm text-slate-700 font-mono text-xs">{evidence.uploaded_by_id ?? 'System'}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Shield size={13} className="text-slate-400" />
            <div>
              <p className="text-xs text-slate-400">Source type</p>
              <p className="text-sm text-slate-700">{evidence.source_type.replace(/_/g, ' ')}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <FileText size={13} className="text-slate-400" />
            <div>
              <p className="text-xs text-slate-400">Format / Size</p>
              <p className="text-sm text-slate-700">
                {evidence.file_type ?? 'Unknown'} {evidence.file_size ? `• ${evidence.file_size}` : ''}
              </p>
            </div>
          </div>
        </div>

        {/* Content preview */}
        <div>
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">Content Preview</h3>
          <pre className="text-xs text-slate-600 bg-slate-50 rounded-md p-3 overflow-x-auto whitespace-pre-wrap font-mono leading-relaxed border border-slate-100">
            {evidence.content_preview}
          </pre>
        </div>

        {/* Quality score breakdown */}
        <div>
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3">
            Quality Score Breakdown
          </h3>
          <div className="space-y-2.5">
            {QUALITY_DIMENSIONS.map((dim) => (
              <QualityBar
                key={dim.key}
                score={quality.dimensions[dim.key]}
                label={dim.label}
              />
            ))}
          </div>
          {/* Quality flags */}
          {quality.flags.length > 0 && (
            <div className="mt-3 space-y-1.5">
              {quality.flags.map((flag) => (
                <div key={flag} className="flex items-start gap-2 text-xs text-orange-700 bg-orange-50 rounded p-2">
                  <AlertTriangle size={12} className="mt-0.5 flex-shrink-0" />
                  <span>{flag}</span>
                </div>
              ))}
            </div>
          )}
          <p className="text-xs text-slate-400 mt-2">
            Assessed by {quality.assessed_by} at {new Date(quality.assessed_at).toLocaleString()}
          </p>
        </div>

        {/* Relevance tags */}
        <div>
          <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">
            Relevance Tags
          </h3>
          <div className="flex flex-wrap gap-1.5">
            {evidence.tags.map((tag) => (
              <span
                key={tag.label}
                className={clsx(
                  'px-2 py-1 text-xs rounded-md font-medium',
                  tag.category === 'article' && 'bg-blue-100 text-blue-800',
                  tag.category === 'domain' && 'bg-purple-100 text-purple-800',
                  tag.category === 'control' && 'bg-green-100 text-green-800',
                  tag.category === 'risk' && 'bg-red-100 text-red-800',
                )}
              >
                {tag.label}
              </span>
            ))}
          </div>
        </div>

        {/* Linked findings */}
        {evidence.finding_ids.length > 0 && (
          <div>
            <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">
              Linked Findings
            </h3>
            <div className="space-y-1">
              {evidence.finding_ids.map((fid) => (
                <div key={fid} className="flex items-center gap-2 text-xs text-slate-600 bg-slate-50 rounded px-3 py-2">
                  <AlertTriangle size={11} className="text-orange-500" />
                  <span className="font-mono">{fid}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
