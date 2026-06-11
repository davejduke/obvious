'use client';
import { MetricCard, Card, CardHeader, CardBody } from '@/components/ui/card';
import { SeverityBadge } from '@/components/ui/badge';
import { mockFindings, mockNIS2Score } from '@/lib/mock-data';
import { RadarChart, Radar, PolarGrid, PolarAngleAxis, ResponsiveContainer, Tooltip } from 'recharts';
import { colors } from '@/lib/tokens';
import { ShieldAlert, TrendingUp, CheckCircle2, AlertOctagon } from 'lucide-react';

const radarData = Object.entries(mockNIS2Score.by_article).map(([art, data]) => ({
  article: art.toUpperCase(), score: data.score,
}));

export function AuditCommitteeDashboard() {
  const criticalFindings = mockFindings.filter(f => f.severity === 'critical');
  const highFindings = mockFindings.filter(f => f.severity === 'high');

  return (
    <div className="p-6 space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-slate-900">Audit Committee Dashboard</h2>
        <p className="text-sm text-slate-500 mt-1">Board-level NIS 2 risk summary — as of January 2024</p>
      </div>

      {/* Executive risk indicators */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg border-2 border-red-200 p-5">
          <div className="flex items-center gap-2 mb-2">
            <AlertOctagon size={20} className="text-red-600" />
            <span className="text-sm font-semibold text-red-700">Critical Risks</span>
          </div>
          <p className="text-3xl font-bold text-red-600">{criticalFindings.length}</p>
          <p className="text-xs text-slate-500 mt-1">Require board attention</p>
        </div>
        <div className="bg-white rounded-lg border-2 border-orange-200 p-5">
          <div className="flex items-center gap-2 mb-2">
            <ShieldAlert size={20} className="text-orange-600" />
            <span className="text-sm font-semibold text-orange-700">High Risks</span>
          </div>
          <p className="text-3xl font-bold text-orange-600">{highFindings.length}</p>
          <p className="text-xs text-slate-500 mt-1">Active remediation</p>
        </div>
        <div className="bg-white rounded-lg border-2 border-blue-200 p-5">
          <div className="flex items-center gap-2 mb-2">
            <TrendingUp size={20} className="text-blue-600" />
            <span className="text-sm font-semibold text-blue-700">Compliance Score</span>
          </div>
          <p className="text-3xl font-bold text-blue-600">{mockNIS2Score.overall_score}%</p>
          <p className="text-xs text-slate-500 mt-1">+4% vs last quarter</p>
        </div>
        <div className="bg-white rounded-lg border-2 border-green-200 p-5">
          <div className="flex items-center gap-2 mb-2">
            <CheckCircle2 size={20} className="text-green-600" />
            <span className="text-sm font-semibold text-green-700">Articles Passing</span>
          </div>
          <p className="text-3xl font-bold text-green-600">
            {Object.values(mockNIS2Score.by_article).filter(a => a.score >= 75).length}/10
          </p>
          <p className="text-xs text-slate-500 mt-1">NIS 2 Article 21</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Radar chart */}
        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">NIS 2 Compliance Coverage</h3>
          </CardHeader>
          <CardBody>
            <ResponsiveContainer width="100%" height={260}>
              <RadarChart data={radarData}>
                <PolarGrid stroke="#e2e8f0" />
                <PolarAngleAxis dataKey="article" tick={{ fontSize: 11 }} />
                <Radar name="Score" dataKey="score" stroke="#2563EB" fill="#2563EB" fillOpacity={0.2} />
                <Tooltip formatter={(v: number) => [`${v}%`, 'Score']} />
              </RadarChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>

        {/* Critical findings summary */}
        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">Critical & High Findings Requiring Action</h3>
          </CardHeader>
          <CardBody className="p-0">
            <ul className="divide-y divide-slate-100">
              {mockFindings.filter(f => f.severity === 'critical' || f.severity === 'high').map(f => (
                <li key={f.id} className="px-6 py-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1">
                      <p className="text-sm font-semibold text-slate-900">{f.title}</p>
                      <p className="text-xs text-slate-500 mt-0.5 line-clamp-2">{f.impact}</p>
                    </div>
                    <SeverityBadge severity={f.severity}>{f.severity}</SeverityBadge>
                  </div>
                  {f.due_date && (
                    <p className="text-xs text-slate-400 mt-1">Due: {f.due_date}</p>
                  )}
                </li>
              ))}
            </ul>
          </CardBody>
        </Card>
      </div>

      {/* Governance note */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <p className="text-sm font-semibold text-blue-900 mb-1">Board Governance Note</p>
        <p className="text-sm text-blue-700">
          The organisation's NIS 2 Article 21 compliance score of <strong>72%</strong> indicates material gaps in 
          identity & access management (Art. 21b: 45%) and network security (Art. 21e: 52%). 
          Management has committed to remediation by Q1 2024. Next board report: 15 March 2024.
        </p>
      </div>
    </div>
  );
}
