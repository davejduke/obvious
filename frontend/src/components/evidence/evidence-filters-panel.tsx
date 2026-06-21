'use client';
import { clsx } from 'clsx';
import { Search, X, SlidersHorizontal } from 'lucide-react';
import type { EvidenceFilters } from '@/lib/evidence-filters';
import {
  SOURCE_TYPE_LABELS,
  STATUS_OPTIONS,
  CLASSIFICATION_OPTIONS,
  ARTICLE_OPTIONS,
  DEFAULT_FILTERS,
} from '@/lib/evidence-filters';
import type { EvidenceSourceType } from '@shared/index';
import type { ClassificationLevel } from '@/lib/evidence-mock-extended';

interface EvidenceFiltersPanelProps {
  filters: EvidenceFilters;
  onChange: (f: EvidenceFilters) => void;
  resultCount: number;
}

function ToggleChip<T extends string>({
  value,
  label,
  selected,
  onToggle,
}: {
  value: T;
  label: string;
  selected: boolean;
  onToggle: (v: T) => void;
}) {
  return (
    <button
      onClick={() => onToggle(value)}
      className={clsx(
        'px-2 py-1 text-xs rounded-md border transition-colors',
        selected
          ? 'bg-blue-600 text-white border-blue-600'
          : 'bg-white text-slate-600 border-slate-200 hover:border-slate-300 hover:bg-slate-50',
      )}
    >
      {label}
    </button>
  );
}

function toggle<T>(arr: T[], val: T): T[] {
  return arr.includes(val) ? arr.filter((x) => x !== val) : [...arr, val];
}

export function EvidenceFiltersPanel({ filters, onChange, resultCount }: EvidenceFiltersPanelProps) {
  const hasActiveFilters =
    filters.search ||
    filters.status.length > 0 ||
    filters.sourceTypes.length > 0 ||
    filters.classifications.length > 0 ||
    filters.qualityMin > 0 ||
    filters.qualityMax < 100 ||
    filters.dateFrom ||
    filters.dateTo ||
    filters.articles.length > 0;

  return (
    <div className="bg-white border-b border-slate-200 px-4 py-3 space-y-3">
      {/* Search + clear */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
          <input
            value={filters.search}
            onChange={(e) => onChange({ ...filters, search: e.target.value })}
            placeholder="Search title, description, control ID, tags..."
            className="w-full pl-9 pr-4 py-2 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          {filters.search && (
            <button
              onClick={() => onChange({ ...filters, search: '' })}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
            >
              <X size={12} />
            </button>
          )}
        </div>
        <div className="flex items-center gap-1 text-xs text-slate-500">
          <SlidersHorizontal size={12} />
          <span>{resultCount} results</span>
        </div>
        {hasActiveFilters && (
          <button
            onClick={() => onChange({ ...DEFAULT_FILTERS })}
            className="text-xs text-blue-600 hover:text-blue-800 underline whitespace-nowrap"
          >
            Clear all
          </button>
        )}
      </div>

      {/* Filter rows */}
      <div className="flex flex-wrap gap-x-6 gap-y-2">
        {/* Status */}
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-medium text-slate-500 mr-1 whitespace-nowrap">Status:</span>
          {STATUS_OPTIONS.map((opt) => (
            <ToggleChip
              key={opt.value}
              value={opt.value}
              label={opt.label}
              selected={filters.status.includes(opt.value)}
              onToggle={(v) => onChange({ ...filters, status: toggle(filters.status, v) })}
            />
          ))}
        </div>

        {/* Source */}
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-medium text-slate-500 mr-1 whitespace-nowrap">Source:</span>
          {(Object.keys(SOURCE_TYPE_LABELS) as EvidenceSourceType[]).map((src) => (
            <ToggleChip
              key={src}
              value={src}
              label={SOURCE_TYPE_LABELS[src]}
              selected={filters.sourceTypes.includes(src)}
              onToggle={(v) =>
                onChange({ ...filters, sourceTypes: toggle(filters.sourceTypes, v) })
              }
            />
          ))}
        </div>

        {/* Classification */}
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-medium text-slate-500 mr-1 whitespace-nowrap">Class:</span>
          {CLASSIFICATION_OPTIONS.map((cls) => (
            <ToggleChip
              key={cls}
              value={cls}
              label={cls.charAt(0).toUpperCase() + cls.slice(1)}
              selected={filters.classifications.includes(cls)}
              onToggle={(v: ClassificationLevel) =>
                onChange({ ...filters, classifications: toggle(filters.classifications, v) })
              }
            />
          ))}
        </div>

        {/* Articles */}
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-medium text-slate-500 mr-1 whitespace-nowrap">Article:</span>
          {ARTICLE_OPTIONS.map((art) => (
            <ToggleChip
              key={art}
              value={art}
              label={art}
              selected={filters.articles.includes(art)}
              onToggle={(v) => onChange({ ...filters, articles: toggle(filters.articles, v) })}
            />
          ))}
        </div>

        {/* Quality range */}
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-slate-500 whitespace-nowrap">Quality:</span>
          <input
            type="number"
            min={0}
            max={100}
            value={filters.qualityMin}
            onChange={(e) => onChange({ ...filters, qualityMin: Number(e.target.value) })}
            className="w-14 text-xs border border-slate-200 rounded px-1.5 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="0"
          />
          <span className="text-xs text-slate-400">-</span>
          <input
            type="number"
            min={0}
            max={100}
            value={filters.qualityMax}
            onChange={(e) => onChange({ ...filters, qualityMax: Number(e.target.value) })}
            className="w-14 text-xs border border-slate-200 rounded px-1.5 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="100"
          />
        </div>

        {/* Date range */}
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-slate-500 whitespace-nowrap">Date:</span>
          <input
            type="date"
            value={filters.dateFrom}
            onChange={(e) => onChange({ ...filters, dateFrom: e.target.value })}
            className="text-xs border border-slate-200 rounded px-1.5 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          <span className="text-xs text-slate-400">to</span>
          <input
            type="date"
            value={filters.dateTo}
            onChange={(e) => onChange({ ...filters, dateTo: e.target.value })}
            className="text-xs border border-slate-200 rounded px-1.5 py-1 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>
    </div>
  );
}
