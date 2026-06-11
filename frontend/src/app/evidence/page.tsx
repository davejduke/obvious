'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardBody, MetricCard } from '@/components/ui/card';
import { StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { mockEvidence } from '@/lib/mock-data';
import { useState } from 'react';
import { Upload, FileText, Image, Database, File, Search, Filter } from 'lucide-react';
import { clsx } from 'clsx';
import type { EvidenceSourceType } from '@shared/index';

const sourceIcons: Record<EvidenceSourceType, typeof File> = {
  manual_upload: Upload,
  api_integration: Database,
  automated_scan: Database,
  screenshot: Image,
  log_export: FileText,
  configuration_export: FileText,
};

const filterOptions = ['all', 'pending_review', 'accepted', 'rejected'] as const;

export default function EvidencePage() {
  const [filter, setFilter] = useState<'all' | 'pending_review' | 'accepted' | 'rejected'>('all');
  const [dragOver, setDragOver] = useState(false);

  const displayed = filter === 'all' ? mockEvidence : mockEvidence.filter(e => e.status === filter);
  const pendingCount = mockEvidence.filter(e => e.status === 'pending_review').length;
  const acceptedCount = mockEvidence.filter(e => e.status === 'accepted').length;

  return (
    <AppShell title="Evidence Manager">
      <div className="p-6 space-y-6">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard label="Total Items" value={mockEvidence.length} accent="border-t-4 border-blue-500" />
          <MetricCard label="Pending Review" value={pendingCount} accent="border-t-4 border-yellow-500" />
          <MetricCard label="Accepted" value={acceptedCount} accent="border-t-4 border-green-500" />
          <MetricCard label="Avg Quality Score" value="78%" accent="border-t-4 border-purple-500" />
        </div>

        {/* Upload zone */}
        <div
          onDragOver={e => { e.preventDefault(); setDragOver(true); }}
          onDragLeave={() => setDragOver(false)}
          onDrop={() => setDragOver(false)}
          className={clsx(
            'border-2 border-dashed rounded-lg p-8 text-center transition-colors',
            dragOver ? 'border-blue-500 bg-blue-50' : 'border-slate-200 hover:border-slate-300 bg-slate-50'
          )}
        >
          <Upload size={32} className="mx-auto mb-3 text-slate-400" />
          <p className="text-sm font-medium text-slate-700">Drop evidence files here or</p>
          <p className="text-xs text-slate-400 mt-1 mb-3">Supports CSV, JSON, PNG, PDF, DOCX (max 50MB)</p>
          <Button variant="secondary" size="sm"><Upload size={14} />Choose Files</Button>
        </div>

        {/* Filter bar */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-xs">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input placeholder="Search evidence..." className="w-full pl-9 pr-4 py-2 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div className="flex gap-1">
            {filterOptions.map(opt => (
              <button key={opt} onClick={() => setFilter(opt)}
                className={clsx(
                  'px-3 py-1.5 text-xs font-medium rounded-md transition-colors',
                  filter === opt ? 'bg-blue-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                )}>
                {opt === 'all' ? 'All' : opt.replace('_', ' ').replace(/\b\w/g, c => c.toUpperCase())}
              </button>
            ))}
          </div>
        </div>

        {/* Evidence table */}
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-xs text-slate-500 uppercase tracking-wide">
                <tr>
                  <th className="px-6 py-3 text-left">Evidence Item</th>
                  <th className="px-6 py-3 text-left">Source</th>
                  <th className="px-6 py-3 text-left">Control</th>
                  <th className="px-6 py-3 text-left">Collected</th>
                  <th className="px-6 py-3 text-left">Status</th>
                  <th className="px-6 py-3 text-left">Sufficient</th>
                  <th className="px-6 py-3 text-left">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {displayed.map(ev => {
                  const Icon = sourceIcons[ev.source_type] ?? File;
                  return (
                    <tr key={ev.id} className="hover:bg-slate-50">
                      <td className="px-6 py-4">
                        <div className="flex items-start gap-2">
                          <Icon size={16} className="mt-0.5 text-slate-400 flex-shrink-0" />
                          <div>
                            <p className="font-medium text-slate-900">{ev.title}</p>
                            {ev.description && <p className="text-xs text-slate-400 mt-0.5 line-clamp-1">{ev.description}</p>}
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-slate-500 text-xs">{ev.source_type.replace(/_/g, ' ')}</td>
                      <td className="px-6 py-4 font-mono text-xs text-blue-600">{ev.control_id}</td>
                      <td className="px-6 py-4 text-slate-500 text-xs">{ev.collection_date}</td>
                      <td className="px-6 py-4">
                        <StatusBadge status={ev.status}>{ev.status.replace('_', ' ')}</StatusBadge>
                      </td>
                      <td className="px-6 py-4">
                        {ev.is_sufficient === true ? (
                          <span className="text-green-600 text-xs font-medium">Yes</span>
                        ) : ev.is_sufficient === false ? (
                          <span className="text-red-500 text-xs font-medium">No</span>
                        ) : (
                          <span className="text-slate-400 text-xs">—</span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex gap-2">
                          <Button variant="ghost" size="sm">View</Button>
                          {ev.status === 'pending_review' && <Button variant="secondary" size="sm">Review</Button>}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
    </AppShell>
  );
}
