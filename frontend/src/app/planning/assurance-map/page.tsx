'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { mockAssuranceMap, type CoverageLevel } from '@/lib/mock-data';
import Link from 'next/link';
import { ArrowLeft, Map } from 'lucide-react';
import { clsx } from 'clsx';

const coverageConfig: Record<CoverageLevel, { bg: string; text: string; label: string; dot: string }> = {
  full: {
    bg: 'bg-green-50',
    text: 'text-green-700',
    label: 'Full',
    dot: 'bg-green-500',
  },
  partial: {
    bg: 'bg-yellow-50',
    text: 'text-yellow-700',
    label: 'Partial',
    dot: 'bg-yellow-400',
  },
  none: {
    bg: 'bg-slate-50',
    text: 'text-slate-400',
    label: 'None',
    dot: 'bg-slate-200',
  },
};

function CoverageCell({ coverage, notes }: { coverage: CoverageLevel; notes?: string }) {
  const cfg = coverageConfig[coverage];
  return (
    <td
      title={notes}
      className={clsx(
        'text-center p-1 border border-white',
        cfg.bg
      )}
    >
      <div className="flex items-center justify-center">
        <div className={clsx('w-3 h-3 rounded-full', cfg.dot)} />
      </div>
    </td>
  );
}

export default function AssuranceMapPage() {
  const am = mockAssuranceMap;

  // Build lookup: [bu][cd] -> coverage
  const lookup: Record<string, Record<string, { coverage: CoverageLevel; notes?: string }>> = {};
  for (const cell of am.matrix) {
    if (!lookup[cell.business_unit]) lookup[cell.business_unit] = {};
    lookup[cell.business_unit][cell.control_domain] = { coverage: cell.coverage, notes: cell.notes };
  }

  const fullCount = am.matrix.filter(c => c.coverage === 'full').length;
  const partialCount = am.matrix.filter(c => c.coverage === 'partial').length;
  const noneCount = am.matrix.filter(c => c.coverage === 'none').length;
  const totalCells = am.matrix.length;
  const coveragePct = totalCells > 0 ? Math.round((fullCount + partialCount * 0.5) * 100 / totalCells) : 0;

  return (
    <AppShell title="Assurance Map">
      <div className="p-6 space-y-6">

        <Link href="/planning" className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800">
          <ArrowLeft size={14} /> Back to Planning
        </Link>

        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-teal-100 rounded-lg flex items-center justify-center">
              <Map size={20} className="text-teal-600" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-slate-900">{am.name}</h1>
              <p className="text-xs text-slate-500 mt-0.5">
                {am.year} · {am.business_units.length} business units · {am.control_domains.length} control domains
              </p>
            </div>
          </div>
          <Button variant="secondary" size="sm">Edit Map</Button>
        </div>

        {/* Coverage summary */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <MetricCard label="Coverage Score" value={`${coveragePct}%`} accent="border-teal-500" />
          <MetricCard label="Full Coverage" value={fullCount} subtitle="cells" accent="border-green-500" />
          <MetricCard label="Partial" value={partialCount} subtitle="cells" accent="border-yellow-500" />
          <MetricCard label="No Coverage" value={noneCount} subtitle="cells" accent="border-slate-400" />
        </div>

        {/* Legend */}
        <div className="flex items-center gap-4 text-xs text-slate-600">
          <span className="font-medium">Coverage:</span>
          {(['full', 'partial', 'none'] as CoverageLevel[]).map(level => (
            <div key={level} className="flex items-center gap-1.5">
              <div className={clsx('w-3 h-3 rounded-full', coverageConfig[level].dot)} />
              <span>{coverageConfig[level].label}</span>
            </div>
          ))}
          <span className="text-slate-400">· Hover cells for notes</span>
        </div>

        {/* Matrix */}
        <Card>
          <CardHeader>
            <h2 className="text-sm font-semibold text-slate-900">Coverage Matrix</h2>
          </CardHeader>
          <div className="overflow-x-auto">
            <table className="w-full text-xs border-collapse">
              <thead>
                <tr>
                  <th className="text-left px-4 py-2 bg-slate-50 text-slate-600 font-semibold border-b border-slate-200 min-w-36">
                    Business Unit
                  </th>
                  {am.control_domains.map(cd => (
                    <th
                      key={cd}
                      className="px-2 py-2 bg-slate-50 text-slate-600 font-medium border-b border-slate-200 text-center whitespace-nowrap"
                    >
                      {cd}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {am.business_units.map((bu, buIdx) => (
                  <tr
                    key={bu}
                    className={clsx(buIdx % 2 === 0 ? 'bg-white' : 'bg-slate-50/50')}
                  >
                    <td className="px-4 py-2.5 font-medium text-slate-700 border-r border-slate-100 whitespace-nowrap">
                      {bu}
                    </td>
                    {am.control_domains.map(cd => {
                      const cell = lookup[bu]?.[cd];
                      const coverage: CoverageLevel = cell?.coverage ?? 'none';
                      return (
                        <CoverageCell
                          key={cd}
                          coverage={coverage}
                          notes={cell?.notes}
                        />
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Coverage breakdown per domain */}
        <Card>
          <CardHeader>
            <h2 className="text-sm font-semibold text-slate-900">Coverage by Control Domain</h2>
          </CardHeader>
          <CardBody className="divide-y divide-slate-100">
            {am.control_domains.map(cd => {
              const cells = am.matrix.filter(c => c.control_domain === cd);
              const fullCt = cells.filter(c => c.coverage === 'full').length;
              const partialCt = cells.filter(c => c.coverage === 'partial').length;
              const pct = cells.length > 0 ? Math.round((fullCt + partialCt * 0.5) * 100 / cells.length) : 0;
              return (
                <div key={cd} className="py-3 flex items-center gap-3">
                  <div className="w-36 text-sm text-slate-700 font-medium">{cd}</div>
                  <div className="flex-1 h-2 bg-slate-100 rounded-full overflow-hidden">
                    <div
                      className="h-full rounded-full bg-teal-500 transition-all"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <div className="w-10 text-xs text-slate-500 text-right">{pct}%</div>
                </div>
              );
            })}
          </CardBody>
        </Card>

      </div>
    </AppShell>
  );
}
