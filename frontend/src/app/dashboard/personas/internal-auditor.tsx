'use client';
import { MetricCard, Card, CardHeader, CardBody } from '@/components/ui/card';
import { SeverityBadge, StatusBadge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { mockEngagements, mockFindings, mockNIS2Score } from '@/lib/mock-data';
import { RadialBarChart, RadialBar, ResponsiveContainer, Tooltip, BarChart, Bar, XAxis, YAxis, CartesianGrid } from 'recharts';
import { colors } from '@/lib/tokens';
import Link from 'next/link';
import { ArrowRight, Clock, AlertTriangle, CheckCircle2 } from 'lucide-react';

const nis2Data = Object.entries(mockNIS2Score.by_article).map(([art, data]) => ({
  article: art.toUpperCase(), score: data.score, fill: data.score >= 80 ? colors.risk.low : data.score >= 60 ? colors.risk.medium : colors.risk.high,
}));

export function InternalAuditorDashboard() {
  const activeEng = mockEngagements[0];
  const openFindings = mockFindings.filter(f => f.status === 'open');
  const criticalCount = mockFindings.filter(f => f.severity === 'critical').length;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-900">Internal Auditor Dashboard</h2>
          <p className="text-sm text-slate-500 mt-1">Active engagement: {activeEng.name}</p>
        </div>
        <Link href="/engagement">
          <Button size="sm"><ArrowRight size={14} />Open Engagement</Button>
        </Link>
      </div>

      {/* Key metrics */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard label="Overall Score" value={`${mockNIS2Score.overall_score}%`} subtitle="NIS 2 Compliance" accent="border-t-4 border-blue-500" />
        <MetricCard label="Open Findings" value={openFindings.length} subtitle={`${criticalCount} critical`} accent="border-t-4 border-red-500" />
        <MetricCard label="Evidence Items" value={47} subtitle="12 pending review" accent="border-t-4 border-yellow-500" />
        <MetricCard label="Controls Tested" value={65} subtitle="of 85 total" accent="border-t-4 border-green-500" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* NIS 2 article scores */}
        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">NIS 2 Article 21 Scores</h3>
          </CardHeader>
          <CardBody>
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={nis2Data} margin={{ top: 0, right: 0, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                <XAxis dataKey="article" tick={{ fontSize: 11 }} />
                <YAxis domain={[0, 100]} tick={{ fontSize: 11 }} />
                <Tooltip formatter={(v: number) => [`${v}%`, 'Score']} />
                <Bar dataKey="score" radius={[3, 3, 0, 0]}>
                  {nis2Data.map((entry, i) => (
                    <rect key={i} fill={entry.fill} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>

        {/* Recent findings */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-slate-900">Recent Findings</h3>
              <Link href="/findings" className="text-sm text-blue-600 hover:text-blue-700">View all</Link>
            </div>
          </CardHeader>
          <CardBody className="p-0">
            <ul className="divide-y divide-slate-100">
              {mockFindings.slice(0, 4).map(f => (
                <li key={f.id} className="flex items-start gap-3 px-6 py-3">
                  <AlertTriangle size={16} className="mt-0.5 flex-shrink-0 text-slate-400" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-slate-900 truncate">{f.title}</p>
                    <p className="text-xs text-slate-500 mt-0.5">{f.finding_ref} · Due {f.due_date}</p>
                  </div>
                  <SeverityBadge severity={f.severity}>{f.severity}</SeverityBadge>
                </li>
              ))}
            </ul>
          </CardBody>
        </Card>
      </div>

      {/* Engagement timeline */}
      <Card>
        <CardHeader>
          <h3 className="font-semibold text-slate-900">Engagement Progress</h3>
        </CardHeader>
        <CardBody>
          <div className="flex items-center gap-4">
            {(['planning', 'fieldwork', 'review', 'reporting', 'completed'] as const).map((phase) => {
              const isActive = activeEng.status === phase;
              const isPast = ['planning'].indexOf(activeEng.status) < ['planning', 'fieldwork', 'review', 'reporting', 'completed'].indexOf(phase);
              return (
                <div key={phase} className="flex items-center gap-2">
                  <div className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${
                    isActive ? 'bg-blue-600 text-white' : isPast ? 'bg-slate-200 text-slate-400' : 'bg-slate-100 text-slate-600'
                  }`}>
                    {isActive && <Clock size={12} />}
                    {!isPast && !isActive && <CheckCircle2 size={12} />}
                    {phase.charAt(0).toUpperCase() + phase.slice(1)}
                  </div>
                  {phase !== 'completed' && <div className="w-8 h-0.5 bg-slate-200" />}
                </div>
              );
            })}
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
