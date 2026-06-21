'use client';
import { useState, useCallback, useMemo } from 'react';
import { AppShell } from '@/components/layout/app-shell';
import { ResizableThreePanel } from '@/components/evidence/resizable-three-panel';
import { EvidenceFiltersPanel } from '@/components/evidence/evidence-filters-panel';
import { EvidenceListPanel } from '@/components/evidence/evidence-list-panel';
import { EvidenceDetailPanel } from '@/components/evidence/evidence-detail-panel';
import { EvidenceChainPanel } from '@/components/evidence/evidence-chain-panel';
import { EvidenceComparison } from '@/components/evidence/evidence-comparison';
import { EvidenceBulkActions } from '@/components/evidence/evidence-bulk-actions';
import { extendedMockEvidence, type ExtendedEvidence } from '@/lib/evidence-mock-extended';
import { applyFilters, DEFAULT_FILTERS, type EvidenceFilters } from '@/lib/evidence-filters';
import { Search } from 'lucide-react';

export default function EvidenceExplorerPage() {
  // Filters
  const [filters, setFilters] = useState<EvidenceFilters>(DEFAULT_FILTERS);

  // Evidence state (supports local updates for status changes)
  const [evidenceData, setEvidenceData] = useState<ExtendedEvidence[]>(extendedMockEvidence);

  // Selection & comparison
  const [selectedId, setSelectedId] = useState<string | null>(extendedMockEvidence[0]?.id ?? null);
  const [compareIds, setCompareIds] = useState<string[]>([]);
  const [showComparison, setShowComparison] = useState(false);

  // Bulk actions
  const [bulkMode, setBulkMode] = useState(false);
  const [bulkSelected, setBulkSelected] = useState<string[]>([]);

  // OpenSearch integration stub (Phase 2)
  const [openSearchActive] = useState(false);

  // Filtered list
  const filteredEvidence = useMemo(
    () => applyFilters(evidenceData, filters),
    [evidenceData, filters],
  );

  const selectedEvidence = evidenceData.find((e) => e.id === selectedId) ?? null;
  const compareEvidence = compareIds.map((id) => evidenceData.find((e) => e.id === id)).filter(Boolean) as ExtendedEvidence[];

  // Handlers
  const handleSelect = useCallback((id: string) => {
    setSelectedId(id);
  }, []);

  const handleToggleCompare = useCallback((id: string) => {
    setCompareIds((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id);
      if (prev.length >= 2) return [prev[1], id]; // keep last 2
      return [...prev, id];
    });
    setShowComparison(true);
  }, []);

  const handleToggleBulk = useCallback((id: string) => {
    setBulkSelected((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
  }, []);

  const handleSelectAllBulk = useCallback(() => {
    setBulkSelected((prev) =>
      prev.length === filteredEvidence.length ? [] : filteredEvidence.map((e) => e.id),
    );
  }, [filteredEvidence]);

  const handleReassess = useCallback((id: string) => {
    // Stub: simulate quality reassessment
    setEvidenceData((prev) =>
      prev.map((ev) =>
        ev.id === id
          ? {
              ...ev,
              quality: {
                ...ev.quality,
                overall: Math.min(100, ev.quality.overall + 5),
                assessed_at: new Date().toISOString(),
                assessed_by: 'auditor' as const,
              },
            }
          : ev,
      ),
    );
  }, []);

  const handleReclassify = useCallback((id: string, status: string) => {
    setEvidenceData((prev) =>
      prev.map((ev) =>
        ev.id === id
          ? { ...ev, status: status as ExtendedEvidence['status'] }
          : ev,
      ),
    );
  }, []);

  // Bulk actions
  const handleBulkReassess = useCallback(() => {
    setEvidenceData((prev) =>
      prev.map((ev) =>
        bulkSelected.includes(ev.id)
          ? {
              ...ev,
              quality: {
                ...ev.quality,
                overall: Math.min(100, ev.quality.overall + 5),
                assessed_at: new Date().toISOString(),
                assessed_by: 'auditor' as const,
              },
            }
          : ev,
      ),
    );
    setBulkSelected([]);
  }, [bulkSelected]);

  const handleBulkReclassify = useCallback((status: string) => {
    setEvidenceData((prev) =>
      prev.map((ev) =>
        bulkSelected.includes(ev.id)
          ? { ...ev, status: status as ExtendedEvidence['status'] }
          : ev,
      ),
    );
    setBulkSelected([]);
  }, [bulkSelected]);

  const handleBulkArchive = useCallback(() => {
    setEvidenceData((prev) =>
      prev.map((ev) =>
        bulkSelected.includes(ev.id) ? { ...ev, status: 'archived' as const } : ev,
      ),
    );
    setBulkSelected([]);
  }, [bulkSelected]);

  const handleToggleBulkMode = useCallback(() => {
    setBulkMode((prev) => !prev);
    setBulkSelected([]);
  }, []);

  return (
    <AppShell title="Evidence Explorer">
      <div className="flex flex-col h-full overflow-hidden" style={{ height: 'calc(100vh - 56px)' }}>
        {/* OpenSearch Phase 2 stub banner */}
        {openSearchActive && (
          <div className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-xs">
            <Search size={13} />
            OpenSearch full-text search active
          </div>
        )}
        {!openSearchActive && (
          <div className="flex items-center gap-2 px-4 py-1.5 bg-amber-50 border-b border-amber-200 text-xs text-amber-700">
            <Search size={12} />
            OpenSearch integration (Phase 2) — not yet connected. Using local filter engine.
          </div>
        )}

        {/* Filters bar */}
        <EvidenceFiltersPanel
          filters={filters}
          onChange={setFilters}
          resultCount={filteredEvidence.length}
        />

        {/* Bulk actions toolbar */}
        <EvidenceBulkActions
          bulkMode={bulkMode}
          selectedIds={bulkSelected}
          onClearSelection={() => setBulkSelected([])}
          onReassessAll={handleBulkReassess}
          onReclassifyAll={handleBulkReclassify}
          onArchiveAll={handleBulkArchive}
          onToggleBulkMode={handleToggleBulkMode}
        />

        {/* Three-panel layout */}
        <div className="flex-1 overflow-hidden">
          <ResizableThreePanel
            defaultLeftWidth={26}
            defaultRightWidth={28}
            left={
              <EvidenceListPanel
                items={filteredEvidence}
                selectedId={selectedId}
                compareIds={compareIds}
                onSelect={handleSelect}
                onToggleCompare={handleToggleCompare}
                bulkSelected={bulkSelected}
                onToggleBulk={handleToggleBulk}
                onSelectAllBulk={handleSelectAllBulk}
                showBulkSelect={bulkMode}
              />
            }
            center={
              <EvidenceDetailPanel
                evidence={selectedEvidence}
                onReassess={handleReassess}
                onReclassify={handleReclassify}
              />
            }
            right={
              <EvidenceChainPanel evidence={selectedEvidence} />
            }
          />
        </div>

        {/* Comparison tray */}
        {showComparison && (
          <EvidenceComparison
            items={compareEvidence}
            onRemove={(id) => setCompareIds((prev) => prev.filter((x) => x !== id))}
            onClose={() => {
              setShowComparison(false);
              setCompareIds([]);
            }}
          />
        )}
      </div>
    </AppShell>
  );
}
