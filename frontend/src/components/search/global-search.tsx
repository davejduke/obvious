'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { Search, X, FileText, Shield, AlertTriangle, Loader2 } from 'lucide-react';

// ─── Types ────────────────────────────────────────────────────────────────────

type EntityType = 'finding' | 'evidence' | 'control';

interface SearchResult {
  id: string;
  type: EntityType;
  title: string;
  description: string;
  org_id: string;
  score: number;
  highlights: Record<string, string[]>;
  meta: Record<string, unknown>;
}

interface SearchResponse {
  query: string;
  total: number;
  findings: SearchResult[];
  evidence: SearchResult[];
  controls: SearchResult[];
}

// ─── Constants ────────────────────────────────────────────────────────────────

const SEARCH_URL = process.env.NEXT_PUBLIC_SEARCH_URL ?? 'http://localhost:8089';
const DEBOUNCE_MS = 200;

// ─── Helpers ─────────────────────────────────────────────────────────────────

function isMac(): boolean {
  if (typeof navigator === 'undefined') return false;
  return /Mac|iPhone|iPod|iPad/.test(navigator.platform);
}

function entityIcon(type: EntityType) {
  switch (type) {
    case 'finding': return <AlertTriangle className="w-4 h-4 text-amber-500" aria-hidden />;
    case 'evidence': return <FileText className="w-4 h-4 text-blue-500" aria-hidden />;
    case 'control': return <Shield className="w-4 h-4 text-emerald-500" aria-hidden />;
  }
}

function entityLabel(type: EntityType) {
  switch (type) {
    case 'finding': return 'Finding';
    case 'evidence': return 'Evidence';
    case 'control': return 'Control';
  }
}

// Strips HTML tags from highlight snippets for safe display.
function stripHighlight(html: string): string {
  return html.replace(/<mark>/g, '').replace(/<\/mark>/g, '');
}

// Renders a title string with <mark> tags highlighted.
function HighlightedTitle({ html }: { html: string }) {
  return (
    <span
      dangerouslySetInnerHTML={{
        __html: html
          .replace(/</g, '&lt;')
          .replace(/>/g, '&gt;')
          .replace(/&lt;mark&gt;/g, '<mark class="bg-yellow-200 text-yellow-900 rounded px-0.5">')
          .replace(/&lt;\/mark&gt;/g, '</mark>'),
      }}
    />
  );
}

// ─── Result row ───────────────────────────────────────────────────────────────

function ResultRow({ result, onSelect }: { result: SearchResult; onSelect: (r: SearchResult) => void }) {
  const titleHtml = result.highlights?.title?.[0] ?? result.title;
  const snippetHtml = result.highlights?.description?.[0];

  return (
    <button
      type="button"
      className="w-full flex items-start gap-3 px-4 py-3 hover:bg-slate-50 focus:bg-slate-50 focus:outline-none text-left group"
      onClick={() => onSelect(result)}
    >
      <span className="mt-0.5 flex-shrink-0">{entityIcon(result.type)}</span>
      <span className="flex-1 min-w-0">
        <span className="block text-sm font-medium text-slate-900 truncate">
          <HighlightedTitle html={titleHtml} />
        </span>
        {snippetHtml && (
          <span className="block text-xs text-slate-500 mt-0.5 line-clamp-1">
            {stripHighlight(snippetHtml)}
          </span>
        )}
      </span>
      <span className="flex-shrink-0 text-xs text-slate-400 mt-0.5">
        {entityLabel(result.type)}
      </span>
    </button>
  );
}

// ─── Section ─────────────────────────────────────────────────────────────────

function Section({
  label,
  results,
  onSelect,
}: {
  label: string;
  results: SearchResult[];
  onSelect: (r: SearchResult) => void;
}) {
  if (results.length === 0) return null;
  return (
    <div>
      <div className="px-4 py-1.5 text-xs font-semibold text-slate-400 uppercase tracking-wider bg-slate-50 border-b border-slate-100">
        {label}
      </div>
      {results.map((r) => (
        <ResultRow key={r.id} result={r} onSelect={onSelect} />
      ))}
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export function GlobalSearch() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ── Open/close keyboard shortcut ──────────────────────────────────────────
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const mod = isMac() ? e.metaKey : e.ctrlKey;
      if (mod && e.key === 'k') {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
      if (e.key === 'Escape') {
        setOpen(false);
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  // ── Focus input when dialog opens ─────────────────────────────────────────
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      setQuery('');
      setResults(null);
      setError(null);
    }
  }, [open]);

  // ── Search fetch ──────────────────────────────────────────────────────────
  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(
        `${SEARCH_URL}/api/v1/search?q=${encodeURIComponent(q)}&size=10`,
        { signal: AbortSignal.timeout(5000) },
      );
      if (!res.ok) throw new Error(`search ${res.status}`);
      const data: SearchResponse = await res.json();
      setResults(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search unavailable');
      setResults(null);
    } finally {
      setLoading(false);
    }
  }, []);

  // ── Debounced query change ─────────────────────────────────────────────────
  const handleQueryChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      setQuery(val);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => doSearch(val), DEBOUNCE_MS);
    },
    [doSearch],
  );

  // ── Result selection (navigate to entity page) ────────────────────────────
  const handleSelect = useCallback((result: SearchResult) => {
    setOpen(false);
    // Navigation is shallow — real routing depends on app router wiring.
    const paths: Record<EntityType, string> = {
      finding: '/findings',
      evidence: '/evidence',
      control: '/controls',
    };
    window.location.href = `${paths[result.type]}?id=${result.id}`;
  }, []);

  const totalResults = results
    ? results.findings.length + results.evidence.length + results.controls.length
    : 0;

  // ── Backdrop click to close ───────────────────────────────────────────────
  function handleBackdropClick(e: React.MouseEvent<HTMLDivElement>) {
    if (e.target === e.currentTarget) setOpen(false);
  }

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="flex items-center gap-2 px-3 py-1.5 text-sm text-slate-500 bg-white border border-slate-200 rounded-lg hover:border-slate-300 hover:text-slate-700 transition-colors"
        aria-label="Open search (Cmd+K)"
      >
        <Search className="w-4 h-4" />
        <span className="hidden sm:inline">Search</span>
        <kbd className="hidden sm:inline text-xs bg-slate-100 text-slate-400 px-1.5 py-0.5 rounded border border-slate-200 font-mono">
          {isMac() ? '⌘K' : 'Ctrl+K'}
        </kbd>
      </button>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-16 px-4 bg-black/40 backdrop-blur-sm"
      onClick={handleBackdropClick}
      role="dialog"
      aria-modal="true"
      aria-label="Global search"
    >
      <div className="w-full max-w-xl bg-white rounded-2xl shadow-2xl overflow-hidden border border-slate-200">
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-100">
          {loading ? (
            <Loader2 className="w-5 h-5 text-slate-400 animate-spin flex-shrink-0" />
          ) : (
            <Search className="w-5 h-5 text-slate-400 flex-shrink-0" />
          )}
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={handleQueryChange}
            placeholder="Search findings, evidence, controls…"
            className="flex-1 bg-transparent text-slate-900 placeholder-slate-400 text-sm focus:outline-none"
            autoComplete="off"
            spellCheck={false}
          />
          <button
            type="button"
            onClick={() => setOpen(false)}
            className="text-slate-400 hover:text-slate-600 transition-colors"
            aria-label="Close search"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Results */}
        <div className="max-h-[60vh] overflow-y-auto">
          {error && (
            <div className="px-4 py-6 text-center text-sm text-red-500">
              {error}
            </div>
          )}

          {!loading && !error && results && totalResults === 0 && (
            <div className="px-4 py-6 text-center text-sm text-slate-500">
              No results for <strong>&ldquo;{query}&rdquo;</strong>
            </div>
          )}

          {results && totalResults > 0 && (
            <>
              <Section label="Findings" results={results.findings} onSelect={handleSelect} />
              <Section label="Evidence" results={results.evidence} onSelect={handleSelect} />
              <Section label="Controls" results={results.controls} onSelect={handleSelect} />
            </>
          )}

          {!query && !results && (
            <div className="px-4 py-8 text-center text-sm text-slate-400">
              Type to search across findings, evidence, and controls
            </div>
          )}
        </div>

        {/* Footer */}
        {results && totalResults > 0 && (
          <div className="px-4 py-2 border-t border-slate-100 flex items-center justify-between text-xs text-slate-400">
            <span>{results.total} result{results.total !== 1 ? 's' : ''}</span>
            <span>↵ to select · Esc to close</span>
          </div>
        )}
      </div>
    </div>
  );
}

