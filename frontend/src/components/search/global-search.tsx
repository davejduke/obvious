'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { Search, X, FileText, Shield, AlertTriangle, Loader2 } from 'lucide-react';
import { useKeyboardShortcut } from '@/hooks/use-keyboard-shortcuts';

// ─── Types ───────────────────────────────────────────────────────────────────

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

// ─── Constants ───────────────────────────────────────────────────────────────

const SEARCH_URL = process.env.NEXT_PUBLIC_SEARCH_URL ?? 'http://localhost:8089';
const DEBOUNCE_MS = 200;

// ─── Helpers ─────────────────────────────────────────────────────────────────

function isMac(): boolean {
  if (typeof navigator === 'undefined') return false;
  return /Mac|iPhone|iPod|iPad/.test(navigator.platform);
}

function entityIcon(type: EntityType) {
  switch (type) {
    case 'finding':  return <AlertTriangle className="w-4 h-4 text-amber-500" aria-hidden />;
    case 'evidence': return <FileText className="w-4 h-4 text-blue-500" aria-hidden />;
    case 'control':  return <Shield className="w-4 h-4 text-emerald-500" aria-hidden />;
  }
}

function entityLabel(type: EntityType) {
  switch (type) {
    case 'finding':  return 'Finding';
    case 'evidence': return 'Evidence';
    case 'control':  return 'Control';
  }
}

function stripHighlight(html: string): string {
  return html.replace(/<mark>/g, '').replace(/<\/mark>/g, '');
}

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

// ─── Result row ──────────────────────────────────────────────────────────────

function ResultRow({
  result,
  active,
  onSelect,
}: {
  result: SearchResult;
  active: boolean;
  onSelect: (r: SearchResult) => void;
}) {
  const titleHtml = result.highlights?.title?.[0] ?? result.title;
  const snippetHtml = result.highlights?.description?.[0];

  return (
    <button
      type="button"
      className={`w-full flex items-start gap-3 px-4 py-3 focus:outline-none text-left transition-colors ${
        active ? 'bg-[var(--brand-50)]' : 'hover:bg-[var(--bg-muted)]'
      }`}
      onClick={() => onSelect(result)}
      role="option"
      aria-selected={active}
    >
      <span className="mt-0.5 flex-shrink-0">{entityIcon(result.type)}</span>
      <span className="flex-1 min-w-0">
        <span className="block text-sm font-medium text-[var(--text-primary)] truncate">
          <HighlightedTitle html={titleHtml} />
        </span>
        {snippetHtml && (
          <span className="block text-xs text-[var(--text-muted)] mt-0.5 line-clamp-1">
            {stripHighlight(snippetHtml)}
          </span>
        )}
      </span>
      <span className="flex-shrink-0 text-xs text-[var(--text-muted)] mt-0.5">
        {entityLabel(result.type)}
      </span>
    </button>
  );
}

// ─── Section ─────────────────────────────────────────────────────────────────

function Section({
  label,
  results,
  activeIdx,
  globalOffset,
  onSelect,
}: {
  label: string;
  results: SearchResult[];
  activeIdx: number;
  globalOffset: number;
  onSelect: (r: SearchResult) => void;
}) {
  if (results.length === 0) return null;
  return (
    <div>
      <div className="px-4 py-1.5 text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wider bg-[var(--bg-muted)] border-b border-[var(--border-default)]">
        {label}
      </div>
      {results.map((r, i) => (
        <ResultRow
          key={r.id}
          result={r}
          active={activeIdx === globalOffset + i}
          onSelect={onSelect}
        />
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
  const [activeIdx, setActiveIdx] = useState(-1);

  const inputRef  = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Register Cmd+K in global shortcut registry (for help overlay)
  useKeyboardShortcut({
    key: 'k',
    meta: true,
    description: 'Open global search',
    group: 'Navigation',
    handler: () => setOpen(v => !v),
  });

  useKeyboardShortcut({
    key: 'Escape',
    description: 'Close search / panels',
    group: 'Navigation',
    handler: () => setOpen(false),
  });

  // ── Focus/blur ─────────────────────────────────────────────────────────────
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      setQuery('');
      setResults(null);
      setError(null);
      setActiveIdx(-1);
    }
  }, [open]);

  // ── Arrow navigation ──────────────────────────────────────────────────────
  const allResults = results
    ? [...results.findings, ...results.evidence, ...results.controls]
    : [];

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveIdx(i => Math.min(i + 1, allResults.length - 1));
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveIdx(i => Math.max(i - 1, -1));
      }
      if (e.key === 'Enter' && activeIdx >= 0) {
        e.preventDefault();
        const r = allResults[activeIdx];
        if (r) handleSelect(r);
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, allResults, activeIdx]); // eslint-disable-line react-hooks/exhaustive-deps

  // ── Search fetch ──────────────────────────────────────────────────────────
  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) { setResults(null); return; }
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
      setActiveIdx(-1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search unavailable');
      setResults(null);
    } finally {
      setLoading(false);
    }
  }, []);

  const handleQueryChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      setQuery(val);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => doSearch(val), DEBOUNCE_MS);
    },
    [doSearch],
  );

  const handleSelect = useCallback((result: SearchResult) => {
    setOpen(false);
    const paths: Record<EntityType, string> = {
      finding: '/findings', evidence: '/evidence', control: '/controls',
    };
    window.location.href = `${paths[result.type]}?id=${result.id}`;
  }, []);

  const totalResults = results
    ? results.findings.length + results.evidence.length + results.controls.length
    : 0;

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border transition-colors bg-[var(--bg-surface)] border-[var(--border-default)] text-[var(--text-muted)] hover:text-[var(--text-secondary)] hover:border-[var(--border-strong)]"
        aria-label="Open search (Cmd+K)"
        data-testid="search-trigger"
      >
        <Search className="w-4 h-4" />
        <span className="hidden sm:inline">Search</span>
        <kbd className="hidden sm:inline text-xs px-1.5 py-0.5 rounded font-mono bg-[var(--bg-muted)] border border-[var(--border-strong)] text-[var(--text-muted)]">
          {isMac() ? '\u2318K' : 'Ctrl+K'}
        </kbd>
      </button>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-16 px-4 bg-black/40 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) setOpen(false); }}
      role="dialog"
      aria-modal="true"
      aria-label="Global search"
    >
      <div className="w-full max-w-xl rounded-2xl shadow-2xl overflow-hidden border bg-[var(--bg-surface)] border-[var(--border-default)]">
        {/* Input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-default)]">
          {loading ? (
            <Loader2 className="w-5 h-5 text-[var(--text-muted)] animate-spin flex-shrink-0" />
          ) : (
            <Search className="w-5 h-5 text-[var(--text-muted)] flex-shrink-0" />
          )}
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={handleQueryChange}
            placeholder="Search findings, evidence, controls…"
            className="flex-1 bg-transparent text-[var(--text-primary)] placeholder:text-[var(--text-muted)] text-sm focus:outline-none"
            autoComplete="off"
            spellCheck={false}
            role="combobox"
            aria-expanded={totalResults > 0}
            aria-autocomplete="list"
          />
          <button
            type="button"
            onClick={() => setOpen(false)}
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
            aria-label="Close search"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Results */}
        <div className="max-h-[60vh] overflow-y-auto" role="listbox">
          {error && (
            <div className="px-4 py-6 text-center text-sm text-red-500">{error}</div>
          )}
          {!loading && !error && results && totalResults === 0 && (
            <div className="px-4 py-6 text-center text-sm text-[var(--text-muted)]">
              No results for <strong>&ldquo;{query}&rdquo;</strong>
            </div>
          )}
          {results && totalResults > 0 && (
            <>
              <Section label="Findings" results={results.findings} activeIdx={activeIdx} globalOffset={0} onSelect={handleSelect} />
              <Section label="Evidence" results={results.evidence} activeIdx={activeIdx} globalOffset={results.findings.length} onSelect={handleSelect} />
              <Section label="Controls" results={results.controls} activeIdx={activeIdx} globalOffset={results.findings.length + results.evidence.length} onSelect={handleSelect} />
            </>
          )}
          {!query && !results && (
            <div className="px-4 py-8 text-center text-sm text-[var(--text-muted)]">
              Type to search across findings, evidence, and controls
            </div>
          )}
        </div>

        {results && totalResults > 0 && (
          <div className="px-4 py-2 border-t border-[var(--border-default)] flex items-center justify-between text-xs text-[var(--text-muted)]">
            <span>{results.total} result{results.total !== 1 ? 's' : ''}</span>
            <span>↑↓ navigate · ↵ select · Esc close</span>
          </div>
        )}
      </div>
    </div>
  );
}
