'use client';
import { clsx } from 'clsx';
import { Upload, FileText, Image, Database, File, CheckSquare } from 'lucide-react';
import type { ExtendedEvidence } from '@/lib/evidence-mock-extended';
import { qualityColor, classificationColor } from '@/lib/evidence-mock-extended';
import { StatusBadge } from '@/components/ui/badge';
import type { EvidenceSourceType } from '@shared/index';

const SOURCE_ICONS: Record<EvidenceSourceType, typeof File> = {
  manual_upload: Upload,
  api_integration: Database,
  automated_scan: Database,
  screenshot: Image,
  log_export: FileText,
  configuration_export: FileText,
};

interface EvidenceListPanelProps {
  items: ExtendedEvidence[];
  selectedId: string | null;
  compareIds: string[];
  onSelect: (id: string) => void;
  onToggleCompare: (id: string) => void;
  bulkSelected: string[];
  onToggleBulk: (id: string) => void;
  onSelectAllBulk: () => void;
  showBulkSelect: boolean;
}

export function EvidenceListPanel({
  items,
  selectedId,
  compareIds,
  onSelect,
  onToggleCompare,
  bulkSelected,
  onToggleBulk,
  onSelectAllBulk,
  showBulkSelect,
}: EvidenceListPanelProps) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* List header */}
      <div className="flex items-center justify-between px-4 py-2 bg-slate-50 border-b border-slate-200 flex-shrink-0">
        <span className="text-xs font-semibold text-slate-600 uppercase tracking-wide">
          Evidence Items ({items.length})
        </span>
        {showBulkSelect && items.length > 0 && (
          <button
            onClick={onSelectAllBulk}
            className="text-xs text-blue-600 hover:text-blue-800"
          >
            {bulkSelected.length === items.length ? 'Deselect all' : 'Select all'}
          </button>
        )}
      </div>

      {/* Scrollable list */}
      <div className="flex-1 overflow-y-auto">
        {items.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-slate-400 py-16">
            <File size={32} className="mb-2 opacity-40" />
            <p className="text-sm">No evidence matches filters</p>
          </div>
        ) : (
          items.map((ev) => {
            const Icon = SOURCE_ICONS[ev.source_type] ?? File;
            const isSelected = ev.id === selectedId;
            const isInCompare = compareIds.includes(ev.id);
            const isBulkChecked = bulkSelected.includes(ev.id);

            return (
              <div
                key={ev.id}
                onClick={() => onSelect(ev.id)}
                className={clsx(
                  'px-4 py-3 border-b border-slate-100 cursor-pointer transition-colors',
                  isSelected ? 'bg-blue-50 border-l-2 border-l-blue-500' : 'hover:bg-slate-50',
                )}
              >
                <div className="flex items-start gap-2">
                  {/* Bulk checkbox */}
                  {showBulkSelect && (
                    <input
                      type="checkbox"
                      checked={isBulkChecked}
                      onClick={(e) => e.stopPropagation()}
                      onChange={() => onToggleBulk(ev.id)}
                      className="mt-1 h-3.5 w-3.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500 flex-shrink-0"
                    />
                  )}

                  <Icon size={15} className="mt-0.5 text-slate-400 flex-shrink-0" />

                  <div className="flex-1 min-w-0">
                    {/* Title */}
                    <p className="text-sm font-medium text-slate-800 truncate">{ev.title}</p>

                    {/* Meta row */}
                    <div className="flex items-center gap-2 mt-1 flex-wrap">
                      <StatusBadge status={ev.status}>
                        {ev.status.replace('_', ' ')}
                      </StatusBadge>

                      <span
                        className={clsx(
                          'px-1.5 py-0.5 text-xs rounded font-medium',
                          classificationColor(ev.classification),
                        )}
                      >
                        {ev.classification.toUpperCase()}
                      </span>

                      <span className={clsx('text-xs font-semibold', qualityColor(ev.quality.overall))}>
                        Q: {ev.quality.overall}
                      </span>

                      <span className="text-xs text-slate-400">{ev.collection_date}</span>
                    </div>

                    {/* Tags */}
                    {ev.tags.length > 0 && (
                      <div className="flex items-center gap-1 mt-1.5 flex-wrap">
                        {ev.tags.slice(0, 3).map((t) => (
                          <span
                            key={t.label}
                            className="px-1.5 py-0.5 text-xs bg-slate-100 text-slate-600 rounded"
                          >
                            {t.label}
                          </span>
                        ))}
                        {ev.tags.length > 3 && (
                          <span className="text-xs text-slate-400">+{ev.tags.length - 3}</span>
                        )}
                      </div>
                    )}
                  </div>

                  {/* Compare toggle */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onToggleCompare(ev.id);
                    }}
                    title={isInCompare ? 'Remove from comparison' : 'Add to comparison'}
                    className={clsx(
                      'flex-shrink-0 p-1 rounded transition-colors',
                      isInCompare
                        ? 'text-blue-600 bg-blue-100'
                        : 'text-slate-300 hover:text-slate-500',
                    )}
                  >
                    <CheckSquare size={14} />
                  </button>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
