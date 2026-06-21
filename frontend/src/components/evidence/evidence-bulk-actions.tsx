'use client';
import { clsx } from 'clsx';
import { CheckSquare, RefreshCw, Tag, Archive, X } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface EvidenceBulkActionsProps {
  selectedIds: string[];
  onClearSelection: () => void;
  onReassessAll: () => void;
  onReclassifyAll: (status: string) => void;
  onArchiveAll: () => void;
  onToggleBulkMode: () => void;
  bulkMode: boolean;
}

export function EvidenceBulkActions({
  selectedIds,
  onClearSelection,
  onReassessAll,
  onReclassifyAll,
  onArchiveAll,
  onToggleBulkMode,
  bulkMode,
}: EvidenceBulkActionsProps) {
  const count = selectedIds.length;

  return (
    <div className="flex items-center gap-2 px-4 py-2 bg-white border-b border-slate-200">
      {/* Bulk mode toggle */}
      <button
        onClick={onToggleBulkMode}
        className={clsx(
          'flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md border transition-colors',
          bulkMode
            ? 'bg-blue-50 border-blue-300 text-blue-700'
            : 'bg-white border-slate-200 text-slate-600 hover:bg-slate-50',
        )}
      >
        <CheckSquare size={13} />
        {bulkMode ? 'Bulk Select On' : 'Bulk Select'}
      </button>

      {/* Bulk action buttons — only visible when items selected */}
      {count > 0 && (
        <>
          <span className="text-xs text-slate-500">
            {count} selected
          </span>

          <Button
            variant="secondary"
            size="sm"
            onClick={onReassessAll}
            className="flex items-center gap-1"
          >
            <RefreshCw size={12} />
            Reassess Quality
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onClick={() => onReclassifyAll('accepted')}
            className="flex items-center gap-1 text-green-700 border-green-300 hover:bg-green-50"
          >
            <Tag size={12} />
            Accept All
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onClick={() => onReclassifyAll('rejected')}
            className="flex items-center gap-1 text-red-700 border-red-300 hover:bg-red-50"
          >
            <Tag size={12} />
            Reject All
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={onArchiveAll}
            className="flex items-center gap-1 text-slate-500"
          >
            <Archive size={12} />
            Archive
          </Button>

          <button
            onClick={onClearSelection}
            className="ml-auto text-slate-400 hover:text-slate-600 p-1 rounded"
          >
            <X size={14} />
          </button>
        </>
      )}
    </div>
  );
}
