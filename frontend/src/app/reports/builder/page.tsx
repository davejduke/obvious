'use client';
import { useState, useCallback } from 'react';
import { AppShell } from '@/components/layout/app-shell';
import { Button } from '@/components/ui/button';
import { BlockEditor } from '@/components/reports/block-editor';
import {
  getTemplateBlocks,
  TEMPLATE_LABELS,
  type ReportBlock,
  type TemplateId,
} from '@/lib/report-builder';
import {
  FileText, Download, Save, LayoutTemplate,
  ChevronDown, CheckCircle, Loader2, ArrowLeft,
} from 'lucide-react';
import { clsx } from 'clsx';
import Link from 'next/link';

const TEMPLATE_IDS: TemplateId[] = ['standard', 'executive', 'regulatory'];

type ExportState = 'idle' | 'exporting' | 'done';

export default function ReportBuilderPage() {
  const [reportTitle, setReportTitle] = useState('Untitled Report');
  const [blocks, setBlocks]           = useState<ReportBlock[]>([]);
  const [exportState, setExportState] = useState<ExportState>('idle');
  const [saved, setSaved]             = useState(false);
  const [templateOpen, setTemplateOpen] = useState(false);

  const handleBlocksChange = useCallback((next: ReportBlock[]) => {
    setBlocks(next);
    setSaved(false);
  }, []);

  function loadTemplate(id: TemplateId) {
    setBlocks(getTemplateBlocks(id));
    setReportTitle(TEMPLATE_LABELS[id]);
    setSaved(false);
    setTemplateOpen(false);
  }

  function handleSave() {
    // In production: POST /api/v1/reports/drafts with blocks + title
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  }

  function handleExportPDF() {
    if (exportState !== 'idle') return;
    setExportState('exporting');
    // In production: POST /api/v1/reports/render with blocks JSON
    // Returns a PDF blob for download.
    setTimeout(() => {
      setExportState('done');
      setTimeout(() => setExportState('idle'), 3000);
    }, 2000);
  }

  const blockCount    = blocks.length;
  const findingCount  = blocks.filter(b => b.type === 'finding_block').length;
  const chartCount    = blocks.filter(b => b.type === 'chart').length;

  return (
    <AppShell title="Report Builder">
      <div className="flex flex-col h-full">
        {/* Top bar */}
        <div className="flex items-center gap-3 px-6 py-3 border-b border-slate-200 bg-white flex-shrink-0">
          <Link href="/reports" className="text-slate-400 hover:text-slate-700 transition-colors">
            <ArrowLeft size={16} />
          </Link>

          {/* Title input */}
          <input
            type="text"
            value={reportTitle}
            onChange={e => { setReportTitle(e.target.value); setSaved(false); }}
            className="flex-1 max-w-xs text-sm font-semibold text-slate-900 bg-transparent border-b border-transparent hover:border-slate-300 focus:border-blue-500 focus:outline-none px-1 py-0.5 transition-colors"
            placeholder="Report title…"
          />

          {/* Block stats */}
          <div className="flex items-center gap-3 text-xs text-slate-400 mx-4">
            <span>{blockCount} block{blockCount !== 1 ? 's' : ''}</span>
            {findingCount > 0 && <span>{findingCount} finding{findingCount !== 1 ? 's' : ''}</span>}
            {chartCount   > 0 && <span>{chartCount} chart{chartCount !== 1 ? 's' : ''}</span>}
          </div>

          {/* Template picker */}
          <div className="relative">
            <Button variant="secondary" size="sm" onClick={() => setTemplateOpen(o => !o)}>
              <LayoutTemplate size={14} />
              Template
              <ChevronDown size={12} className={clsx('transition-transform', templateOpen && 'rotate-180')} />
            </Button>
            {templateOpen && (
              <div className="absolute right-0 top-full mt-1 z-10 bg-white border border-slate-200 rounded-lg shadow-lg py-1 min-w-[200px]">
                {TEMPLATE_IDS.map(id => (
                  <button
                    key={id}
                    onClick={() => loadTemplate(id)}
                    className="flex items-start gap-2 w-full px-4 py-2 text-left hover:bg-blue-50 transition-colors"
                  >
                    <FileText size={14} className="text-blue-500 flex-shrink-0 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-slate-800">{TEMPLATE_LABELS[id]}</p>
                      <p className="text-xs text-slate-400">
                        {id === 'standard'   && 'Full audit report with findings & recommendations'}
                        {id === 'executive'  && 'Board-level summary with key metrics'}
                        {id === 'regulatory' && 'NIS 2 compliance submission format'}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Save */}
          <Button
            variant={saved ? 'secondary' : 'ghost'}
            size="sm"
            onClick={handleSave}
          >
            {saved
              ? <><CheckCircle size={14} className="text-green-600" /> Saved</>
              : <><Save size={14} /> Save</>}
          </Button>

          {/* PDF Export */}
          <Button
            variant="primary"
            size="sm"
            onClick={handleExportPDF}
            disabled={exportState !== 'idle' || blocks.length === 0}
          >
            {exportState === 'idle'      && <><Download size={14} /> Export PDF</>}
            {exportState === 'exporting' && <><Loader2 size={14} className="animate-spin" /> Exporting…</>}
            {exportState === 'done'      && <><CheckCircle size={14} className="text-green-300" /> Ready to download</>}
          </Button>
        </div>

        {/* Export success banner */}
        {exportState === 'done' && (
          <div className="px-6 py-2 bg-green-50 border-b border-green-200 flex items-center gap-2 text-sm text-green-700">
            <CheckCircle size={14} />
            PDF generated by reporting service — download ready.
            <button
              className="ml-auto text-xs font-medium underline hover:no-underline"
              onClick={() => setExportState('idle')}
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Canvas */}
        <div className="flex-1 overflow-y-auto bg-slate-50">
          <div className="max-w-4xl mx-auto py-8 px-6">
            {/* Page title preview */}
            <div className="mb-6 text-center border-b border-slate-200 pb-6">
              <p className="text-xs text-slate-400 uppercase tracking-wide mb-2">Report Preview</p>
              <h1 className="text-3xl font-bold text-slate-900">{reportTitle}</h1>
              <p className="text-xs text-slate-400 mt-2">
                {blockCount > 0
                  ? `${blockCount} blocks — ${findingCount} live finding references — ${chartCount} embedded charts`
                  : 'Select a template to get started, or add blocks manually.'}
              </p>
            </div>

            {/* Block editor */}
            <BlockEditor blocks={blocks} onChange={handleBlocksChange} />

            {/* Empty state add button */}
            {blocks.length === 0 && (
              <div className="mt-6 grid grid-cols-3 gap-3">
                {TEMPLATE_IDS.map(id => (
                  <button
                    key={id}
                    onClick={() => loadTemplate(id)}
                    className="p-4 bg-white border border-slate-200 rounded-lg hover:border-blue-400 hover:shadow-sm text-left transition-all"
                  >
                    <LayoutTemplate size={20} className="text-blue-500 mb-2" />
                    <p className="text-sm font-semibold text-slate-800">{TEMPLATE_LABELS[id]}</p>
                    <p className="text-xs text-slate-500 mt-1">
                      {id === 'standard'   && 'Full audit report structure'}
                      {id === 'executive'  && 'Board-level summary'}
                      {id === 'regulatory' && 'Regulatory submission'}
                    </p>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Bottom status bar */}
        <div className="flex items-center justify-between px-6 py-2 border-t border-slate-200 bg-white text-xs text-slate-400 flex-shrink-0">
          <span>Report Builder WYSIWYG — contentEditable blocks, native drag-and-drop</span>
          <span className="flex items-center gap-2">
            {findingCount > 0 && (
              <span className="px-2 py-0.5 bg-orange-50 text-orange-600 rounded border border-orange-100">
                {findingCount} live finding{findingCount !== 1 ? 's' : ''}
              </span>
            )}
            {chartCount > 0 && (
              <span className="px-2 py-0.5 bg-purple-50 text-purple-600 rounded border border-purple-100">
                {chartCount} chart{chartCount !== 1 ? 's' : ''}
              </span>
            )}
            <span>{blockCount} block{blockCount !== 1 ? 's' : ''} total</span>
          </span>
        </div>
      </div>
    </AppShell>
  );
}

