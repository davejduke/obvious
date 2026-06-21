'use client';
/**
 * BlockEditor — contentEditable-based WYSIWYG report block editor.
 * Locked decision: NO external editor library. Uses native contentEditable + HTML5 drag API.
 */
import { useRef, useState, useCallback, useEffect } from 'react';
import { clsx } from 'clsx';
import {
  GripVertical, Plus, Trash2, ChevronUp, ChevronDown,
  Type, AlignLeft, Search, BarChart2, Table, FileText,
} from 'lucide-react';
import {
  BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Legend,
} from 'recharts';
import type {
  ReportBlock, BlockType,
} from '@/lib/report-builder';
import {
  addBlock, removeBlock, updateBlock,
  moveBlockUp, moveBlockDown, reorderBlocks,
} from '@/lib/report-builder';
import { mockFindings, mockEvidence, mockNIS2Score } from '@/lib/mock-data';
import { SeverityBadge } from '@/components/ui/badge';

// ─── Types ───────────────────────────────────────────────────────────────────

interface BlockEditorProps {
  blocks: ReportBlock[];
  onChange: (blocks: ReportBlock[]) => void;
}

const BLOCK_TYPE_ICONS: Record<BlockType, React.ElementType> = {
  heading:            Type,
  paragraph:          AlignLeft,
  finding_block:      Search,
  evidence_reference: FileText,
  chart:              BarChart2,
  table:              Table,
};

const ADD_OPTIONS: { type: BlockType; label: string }[] = [
  { type: 'heading',            label: 'Heading' },
  { type: 'paragraph',          label: 'Paragraph' },
  { type: 'finding_block',      label: 'Finding Block' },
  { type: 'evidence_reference', label: 'Evidence Reference' },
  { type: 'chart',              label: 'Chart' },
  { type: 'table',              label: 'Table' },
];

// ─── ContentEditable primitive ──────────────────────────────────────────────
/**
 * Uncontrolled contentEditable element.
 * Uses a key-based remount strategy: changing `resetKey` forces remount and
 * re-initialises content, preventing cursor jumps on React state updates.
 */
function ContentEditable({
  initialContent,
  resetKey,
  onInput,
  className,
  tag: Tag = 'div',
  placeholder,
}: {
  initialContent: string;
  resetKey: string;
  onInput: (text: string) => void;
  className?: string;
  tag?: 'div' | 'h1' | 'h2' | 'h3';
  placeholder?: string;
}) {
  const ref = useRef<HTMLElement>(null);

  // Set initial text only on mount (resetKey triggers remount)
  useEffect(() => {
    if (ref.current) {
      ref.current.innerText = initialContent;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Tag
      key={resetKey}
      ref={ref as React.RefObject<HTMLDivElement & HTMLHeadingElement>}
      contentEditable
      suppressContentEditableWarning
      data-placeholder={placeholder}
      className={clsx(className, 'outline-none focus:ring-2 focus:ring-blue-200 focus:ring-inset rounded-sm')}
      onInput={e => onInput(e.currentTarget.innerText)}
    />
  );
}

// ─── Individual block renderers ─────────────────────────────────────────────

const HEADING_CLASS: Record<number, string> = {
  1: 'text-2xl font-bold text-slate-900',
  2: 'text-xl font-semibold text-slate-800',
  3: 'text-lg font-semibold text-slate-700',
};

const HEADING_TAGS: Record<number, 'h1' | 'h2' | 'h3'> = { 1: 'h1', 2: 'h2', 3: 'h3' };

function HeadingBlockRenderer({
  block,
  onUpdate,
}: {
  block: ReportBlock;
  onUpdate: (updates: Partial<ReportBlock>) => void;
}) {
  const level = block.level ?? 2;
  const Tag = HEADING_TAGS[level];
  return (
    <div>
      <div className="flex items-center gap-2 mb-1">
        {([1, 2, 3] as const).map(l => (
          <button
            key={l}
            onClick={() => onUpdate({ level: l })}
            className={clsx(
              'px-1.5 py-0.5 rounded text-xs font-mono border',
              level === l
                ? 'bg-blue-600 text-white border-blue-600'
                : 'bg-white text-slate-500 border-slate-300 hover:bg-slate-50'
            )}
          >
            H{l}
          </button>
        ))}
      </div>
      <ContentEditable
        key={block.id}
        resetKey={block.id}
        initialContent={block.content}
        onInput={content => onUpdate({ content })}
        tag={Tag}
        className={HEADING_CLASS[level]}
        placeholder="Section Heading"
      />
    </div>
  );
}

function ParagraphBlockRenderer({
  block,
  onUpdate,
}: {
  block: ReportBlock;
  onUpdate: (updates: Partial<ReportBlock>) => void;
}) {
  return (
    <ContentEditable
      key={block.id}
      resetKey={block.id}
      initialContent={block.content}
      onInput={content => onUpdate({ content })}
      tag="div"
      className="text-sm text-slate-700 leading-relaxed min-h-[3em]"
      placeholder="Click to edit paragraph…"
    />
  );
}

/** Live-bound finding block — pulls real data from mockFindings. */
function FindingBlockRenderer({
  block,
  onUpdate,
}: {
  block: ReportBlock;
  onUpdate: (updates: Partial<ReportBlock>) => void;
}) {
  const findingId = block.data?.findingId ?? '';
  const finding   = mockFindings.find(f => f.id === findingId);

  return (
    <div className="rounded-lg border border-orange-200 bg-orange-50/40 p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-medium text-orange-700 uppercase tracking-wide">Finding Block</span>
        <select
          value={findingId}
          onChange={e => onUpdate({ data: { ...block.data, findingId: e.target.value } })}
          className="text-xs border border-slate-200 rounded px-2 py-1 bg-white text-slate-700 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          <option value="">Select finding…</option>
          {mockFindings.map(f => (
            <option key={f.id} value={f.id}>{f.finding_ref} — {f.title.slice(0, 50)}…</option>
          ))}
        </select>
      </div>

      {finding ? (
        <div className="space-y-2">
          <div className="flex items-start gap-2">
            <SeverityBadge severity={finding.severity}>{finding.severity}</SeverityBadge>
            <p className="font-semibold text-sm text-slate-900">{finding.title}</p>
          </div>
          <p className="text-xs text-slate-600">{finding.description}</p>
          <div className="flex gap-3 text-xs text-slate-500">
            <span>Ref: <span className="font-mono">{finding.finding_ref}</span></span>
            <span>Status: {finding.status.replace(/_/g, ' ')}</span>
            {finding.due_date && <span>Due: {finding.due_date}</span>}
          </div>
        </div>
      ) : (
        <p className="text-xs text-slate-400 italic">Select a finding to populate live data.</p>
      )}
    </div>
  );
}

/** Evidence reference block — live data binding from mockEvidence. */
function EvidenceBlockRenderer({
  block,
  onUpdate,
}: {
  block: ReportBlock;
  onUpdate: (updates: Partial<ReportBlock>) => void;
}) {
  const evidenceId = block.data?.evidenceId ?? '';
  const evidence   = mockEvidence.find(e => e.id === evidenceId);

  return (
    <div className="rounded-lg border border-blue-200 bg-blue-50/40 p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-medium text-blue-700 uppercase tracking-wide">Evidence Reference</span>
        <select
          value={evidenceId}
          onChange={e => onUpdate({ data: { ...block.data, evidenceId: e.target.value } })}
          className="text-xs border border-slate-200 rounded px-2 py-1 bg-white text-slate-700 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          <option value="">Select evidence…</option>
          {mockEvidence.map(ev => (
            <option key={ev.id} value={ev.id}>{ev.title}</option>
          ))}
        </select>
      </div>

      {evidence ? (
        <div className="space-y-1">
          <p className="font-semibold text-sm text-slate-900">{evidence.title}</p>
          <p className="text-xs text-slate-600">{evidence.description}</p>
          <div className="flex gap-3 text-xs text-slate-500">
            <span>Source: {evidence.source_type.replace(/_/g, ' ')}</span>
            <span>Collected: {evidence.collection_date}</span>
            <span className={clsx(
              'px-1.5 py-0.5 rounded font-medium',
              evidence.status === 'accepted' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'
            )}>{evidence.status.replace(/_/g, ' ')}</span>
          </div>
        </div>
      ) : (
        <p className="text-xs text-slate-400 italic">Select evidence to populate live data.</p>
      )}
    </div>
  );
}

const CHART_COLORS = ['#DC2626', '#EA580C', '#CA8A04', '#16A34A', '#2563EB'];

/** Inline chart block using Recharts. */
function ChartBlockRenderer({
  block,
  onUpdate,
}: {
  block: ReportBlock;
  onUpdate: (updates: Partial<ReportBlock>) => void;
}) {
  const chartType = block.data?.chartType ?? 'bar';

  // Build chart data from NIS2 compliance score
  const barData = Object.entries(mockNIS2Score.by_article).map(([article, metrics]) => ({
    article: article.toUpperCase(),
    score: metrics.score,
  }));

  const pieData = [
    { name: 'Critical', value: mockFindings.filter(f => f.severity === 'critical').length, color: '#DC2626' },
    { name: 'High',     value: mockFindings.filter(f => f.severity === 'high').length,     color: '#EA580C' },
    { name: 'Medium',   value: mockFindings.filter(f => f.severity === 'medium').length,   color: '#CA8A04' },
    { name: 'Low',      value: mockFindings.filter(f => f.severity === 'low').length,      color: '#16A34A' },
  ];

  return (
    <div className="rounded-lg border border-purple-200 bg-purple-50/30 p-4">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-medium text-purple-700 uppercase tracking-wide">Chart Block</span>
        <div className="flex gap-1">
          {(['bar', 'pie'] as const).map(t => (
            <button
              key={t}
              onClick={() => onUpdate({ data: { ...block.data, chartType: t } })}
              className={clsx(
                'px-2 py-1 text-xs rounded border transition-colors',
                chartType === t
                  ? 'bg-purple-600 text-white border-purple-600'
                  : 'bg-white text-slate-600 border-slate-300 hover:bg-slate-50'
              )}
            >
              {t === 'bar' ? 'Bar' : 'Pie'}
            </button>
          ))}
        </div>
      </div>

      <ResponsiveContainer width="100%" height={200}>
        {chartType === 'pie' ? (
          <PieChart>
            <Pie data={pieData} cx="50%" cy="50%" outerRadius={80} dataKey="value" label={({ name, value }) => `${name}: ${value}`}>
              {pieData.map((entry, i) => <Cell key={i} fill={entry.color} />)}
            </Pie>
            <Tooltip />
            <Legend iconSize={10} />
          </PieChart>
        ) : (
          <BarChart data={barData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
            <XAxis dataKey="article" tick={{ fontSize: 10 }} />
            <YAxis domain={[0, 100]} tick={{ fontSize: 10 }} />
            <Tooltip formatter={(v: unknown) => [`${v}%`, 'Score']} />
            <Bar dataKey="score" radius={[3, 3, 0, 0]}>
              {barData.map((entry, i) => (
                <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
              ))}
            </Bar>
          </BarChart>
        )}
      </ResponsiveContainer>
    </div>
  );
}

/** Table block renderer. */
function TableBlockRenderer({ block }: { block: ReportBlock }) {
  const headers = block.data?.tableHeaders ?? ['Column 1', 'Column 2', 'Column 3'];
  const rows    = block.data?.tableRows ?? [['Row 1 data', 'Row 1 data', 'Row 1 data']];

  return (
    <div className="rounded-lg border border-slate-200 overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slate-100 text-slate-600">
          <tr>
            {headers.map((h, i) => (
              <th key={i} className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {rows.map((row, ri) => (
            <tr key={ri} className="hover:bg-slate-50">
              {row.map((cell, ci) => (
                <td key={ci} className="px-4 py-2 text-slate-700 text-xs">{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ─── Block shell (drag + controls) ──────────────────────────────────────────

function BlockShell({
  block,
  index,
  total,
  isDragOver,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
  onMoveUp,
  onMoveDown,
  onDelete,
  onAddAfter,
  children,
}: {
  block: ReportBlock;
  index: number;
  total: number;
  isDragOver: boolean;
  onDragStart: () => void;
  onDragOver: (e: React.DragEvent) => void;
  onDragLeave: () => void;
  onDrop: (e: React.DragEvent) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onDelete: () => void;
  onAddAfter: (type: BlockType) => void;
  children: React.ReactNode;
}) {
  const [addOpen, setAddOpen] = useState(false);
  const IconComponent = BLOCK_TYPE_ICONS[block.type];

  return (
    <div
      draggable
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
      className={clsx(
        'group relative flex gap-2 rounded-lg border bg-white p-4 transition-all',
        isDragOver
          ? 'border-blue-400 shadow-md ring-2 ring-blue-200'
          : 'border-slate-200 hover:border-slate-300 hover:shadow-sm'
      )}
    >
      {/* Drag handle */}
      <div className="flex flex-col items-center gap-0.5 pt-0.5 cursor-grab active:cursor-grabbing flex-shrink-0">
        <GripVertical size={16} className="text-slate-300 group-hover:text-slate-400" />
        <IconComponent size={12} className="text-slate-300 group-hover:text-slate-400" />
      </div>

      {/* Block content */}
      <div className="flex-1 min-w-0">
        {children}
      </div>

      {/* Controls (visible on hover) */}
      <div className="flex flex-col gap-1 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          onClick={onMoveUp}
          disabled={index === 0}
          title="Move up"
          className="p-1 rounded hover:bg-slate-100 text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ChevronUp size={14} />
        </button>
        <button
          onClick={onMoveDown}
          disabled={index === total - 1}
          title="Move down"
          className="p-1 rounded hover:bg-slate-100 text-slate-400 hover:text-slate-600 disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ChevronDown size={14} />
        </button>
        <div className="relative">
          <button
            onClick={() => setAddOpen(o => !o)}
            title="Add block below"
            className="p-1 rounded hover:bg-blue-50 text-slate-400 hover:text-blue-600"
          >
            <Plus size={14} />
          </button>
          {addOpen && (
            <div className="absolute right-0 top-full mt-1 z-10 bg-white border border-slate-200 rounded-lg shadow-lg py-1 min-w-[160px]">
              {ADD_OPTIONS.map(opt => (
                <button
                  key={opt.type}
                  onClick={() => { onAddAfter(opt.type); setAddOpen(false); }}
                  className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-slate-700 hover:bg-blue-50 hover:text-blue-700"
                >
                  {(() => { const Ic = BLOCK_TYPE_ICONS[opt.type]; return <Ic size={12} />; })()}
                  {opt.label}
                </button>
              ))}
            </div>
          )}
        </div>
        <button
          onClick={onDelete}
          title="Delete block"
          className="p-1 rounded hover:bg-red-50 text-slate-400 hover:text-red-600"
        >
          <Trash2 size={14} />
        </button>
      </div>
    </div>
  );
}

// ─── Main BlockEditor ─────────────────────────────────────────────────────────

export function BlockEditor({ blocks, onChange }: BlockEditorProps) {
  const dragIndexRef  = useRef<number | null>(null);
  const [dragOver, setDragOver] = useState<number | null>(null);

  const handleUpdate = useCallback(
    (id: string, updates: Partial<ReportBlock>) =>
      onChange(updateBlock(blocks, id, updates)),
    [blocks, onChange]
  );

  const handleMoveUp   = (id: string) => onChange(moveBlockUp(blocks, id));
  const handleMoveDown = (id: string) => onChange(moveBlockDown(blocks, id));
  const handleDelete   = (id: string) => onChange(removeBlock(blocks, id));
  const handleAdd      = (index: number, type: BlockType) =>
    onChange(addBlock(blocks, type, index));

  const handleDragStart = (index: number) => {
    dragIndexRef.current = index;
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setDragOver(index);
  };

  const handleDragLeave = () => setDragOver(null);

  const handleDrop = (e: React.DragEvent, toIndex: number) => {
    e.preventDefault();
    setDragOver(null);
    if (dragIndexRef.current !== null && dragIndexRef.current !== toIndex) {
      onChange(reorderBlocks(blocks, dragIndexRef.current, toIndex));
    }
    dragIndexRef.current = null;
  };

  function renderBlock(block: ReportBlock) {
    const onUpdate = (updates: Partial<ReportBlock>) => handleUpdate(block.id, updates);
    switch (block.type) {
      case 'heading':            return <HeadingBlockRenderer   block={block} onUpdate={onUpdate} />;
      case 'paragraph':          return <ParagraphBlockRenderer  block={block} onUpdate={onUpdate} />;
      case 'finding_block':      return <FindingBlockRenderer    block={block} onUpdate={onUpdate} />;
      case 'evidence_reference': return <EvidenceBlockRenderer   block={block} onUpdate={onUpdate} />;
      case 'chart':              return <ChartBlockRenderer      block={block} onUpdate={onUpdate} />;
      case 'table':              return <TableBlockRenderer      block={block} />;
      default:                   return null;
    }
  }

  if (blocks.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-slate-400">
        <FileText size={48} className="mb-4 opacity-30" />
        <p className="text-sm">No blocks yet. Select a template above or add a block.</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {blocks.map((block, index) => (
        <BlockShell
          key={block.id}
          block={block}
          index={index}
          total={blocks.length}
          isDragOver={dragOver === index}
          onDragStart={() => handleDragStart(index)}
          onDragOver={e => handleDragOver(e, index)}
          onDragLeave={handleDragLeave}
          onDrop={e => handleDrop(e, index)}
          onMoveUp={() => handleMoveUp(block.id)}
          onMoveDown={() => handleMoveDown(block.id)}
          onDelete={() => handleDelete(block.id)}
          onAddAfter={type => handleAdd(index, type)}
        >
          {renderBlock(block)}
        </BlockShell>
      ))}
    </div>
  );
}

export type { BlockEditorProps };

