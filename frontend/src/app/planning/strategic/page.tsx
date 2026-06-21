'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import { StatusBadge, SeverityBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ApprovalBanner, type ApprovalWorkflow } from '@/components/ui/approval-banner';
import { mockStrategicPlan, type AuditableEntity, type RiskRating } from '@/lib/mock-data';
import { useState } from 'react';
import Link from 'next/link';
import { ArrowLeft, BarChart3, ChevronDown, ChevronUp } from 'lucide-react';
import { clsx } from 'clsx';

// Map risk rating to severity badge type
const riskToSeverity: Record<RiskRating, string> = {
  critical: 'critical',
  high: 'high',
  medium: 'medium',
  low: 'low',
};

function YearSection({ year, entities }: { year: number; entities: AuditableEntity[] }) {
  const [expanded, setExpanded] = useState(true);
  return (
    <div className="mb-4">
      <button
        onClick={() => setExpanded(v => !v)}
        className="w-full flex items-center justify-between px-4 py-2 bg-slate-50 hover:bg-slate-100 rounded-md text-sm font-semibold text-slate-700 transition-colors"
      >
        <span>Year {year}</span>
        <div className="flex items-center gap-2 text-slate-500">
          <span className="text-xs font-normal">{entities.length} entities</span>
          {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </div>
      </button>
      {expanded && (
        <div className="mt-2 space-y-2">
          {entities.sort((a, b) => a.priority - b.priority).map((entity) => (
            <div key={entity.id} className="flex items-start gap-3 px-4 py-3 bg-white rounded-md border border-slate-200 hover:border-blue-300 transition-colors">
              <div className="flex-shrink-0 w-7 h-7 rounded-full bg-slate-100 flex items-center justify-center text-xs font-bold text-slate-600">
                {entity.priority}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-sm font-medium text-slate-900">{entity.name}</span>
                  <SeverityBadge severity={riskToSeverity[entity.risk_rating]}>
                    {entity.risk_rating}
                  </SeverityBadge>
                </div>
                <p className="text-xs text-slate-500 mt-0.5">{entity.business_unit}</p>
                {entity.control_domains.length > 0 && (
                  <div className="mt-1.5 flex flex-wrap gap-1">
                    {entity.control_domains.map(cd => (
                      <span key={cd} className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-slate-100 text-slate-600">
                        {cd}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function StrategicPlanPage() {
  const plan = mockStrategicPlan;
  const years = Array.from(new Set(plan.entities.map(e => e.planned_year))).sort();
  const criticals = plan.entities.filter(e => e.risk_rating === 'critical').length;
  const highs = plan.entities.filter(e => e.risk_rating === 'high').length;

  const workflow: ApprovalWorkflow = {
    id: 'wf-sp-001',
    workflow_type: 'audit_plan',
    status: 'approved',
    submitted_by_email: 'el@example.com',
    submitted_at: '2024-12-08T10:00:00Z',
    decided_by_email: plan.approved_by,
    decided_at: plan.approved_at,
  };

  return (
    <AppShell title="Strategic Plan">
      <div className="p-6 space-y-6">

        {/* Back nav */}
        <Link href="/planning" className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800">
          <ArrowLeft size={14} /> Back to Planning
        </Link>

        {/* Header */}
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-purple-100 rounded-lg flex items-center justify-center">
              <BarChart3 size={20} className="text-purple-600" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-slate-900">{plan.name}</h1>
              <div className="flex items-center gap-2 mt-0.5">
                <StatusBadge status={plan.status}>{plan.status}</StatusBadge>
                <span className="text-xs text-slate-400">Version {plan.version}</span>
                <span className="text-xs text-slate-400">{plan.start_year}–{plan.end_year}</span>
              </div>
            </div>
          </div>
          <Button variant="secondary" size="sm">Edit Plan</Button>
        </div>

        {/* Approval banner */}
        <ApprovalBanner
          workflow={workflow}
          userRole="chief_audit_executive"
        />

        {/* Description */}
        {plan.description && (
          <p className="text-sm text-slate-600">{plan.description}</p>
        )}

        {/* Metrics */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <MetricCard label="Total Entities" value={plan.entities.length} accent="border-purple-500" />
          <MetricCard label="Critical Risk" value={criticals} accent="border-red-500" />
          <MetricCard label="High Risk" value={highs} accent="border-orange-500" />
          <MetricCard label="Plan Years" value={plan.end_year - plan.start_year + 1} subtitle={`${plan.start_year}–${plan.end_year}`} accent="border-blue-500" />
        </div>

        {/* Risk distribution bar */}
        <Card>
          <CardHeader>
            <h2 className="text-sm font-semibold text-slate-900">Risk Distribution</h2>
          </CardHeader>
          <CardBody>
            {(['critical', 'high', 'medium', 'low'] as RiskRating[]).map(rating => {
              const count = plan.entities.filter(e => e.risk_rating === rating).length;
              const pct = plan.entities.length > 0 ? count * 100 / plan.entities.length : 0;
              const barColor: Record<RiskRating, string> = {
                critical: 'bg-red-500',
                high: 'bg-orange-500',
                medium: 'bg-yellow-500',
                low: 'bg-green-500',
              };
              return (
                <div key={rating} className="flex items-center gap-3 mb-3">
                  <div className="w-16 text-xs font-medium text-slate-600 capitalize">{rating}</div>
                  <div className="flex-1 h-2 bg-slate-100 rounded-full overflow-hidden">
                    <div
                      className={clsx('h-full rounded-full transition-all', barColor[rating])}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <div className="w-6 text-xs text-slate-500 text-right">{count}</div>
                </div>
              );
            })}
          </CardBody>
        </Card>

        {/* Entities by year */}
        <Card>
          <CardHeader>
            <h2 className="text-sm font-semibold text-slate-900">Auditable Entities by Year</h2>
          </CardHeader>
          <CardBody>
            {years.map(year => (
              <YearSection
                key={year}
                year={year}
                entities={plan.entities.filter(e => e.planned_year === year)}
              />
            ))}
          </CardBody>
        </Card>

      </div>
    </AppShell>
  );
}
