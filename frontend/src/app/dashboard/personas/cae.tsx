'use client';
import { MetricCard, Card, CardHeader, CardBody } from '@/components/ui/card';
import { SeverityBadge } from '@/components/ui/badge';
import { mockEngagements, mockFindings, mockNIS2Score } from '@/lib/mock-data';
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend } from 'recharts';
import { colors } from '@/lib/tokens';
import Link from 'next/link';

const trendData = [
  { month: 'Aug', score: 58 }, { month: 'Sep', score: 62 },
  { month: 'Oct', score: 65 }, { month: 'Nov', score: 68 },
  { month: 'Dec', score: 71 }, { month: 'Jan', score: 72 },
];

const riskDistribution = [
  { name: 'Critical', value: 2, color: colors.risk.critical },
  { name: 'High', value: 4, color: colors.risk.high },
  { name: 'Medium', value: 5, color: colors.risk.medium },
  { name: 'Low', value: 8, color: colors.risk.low },
];

export function CAEDashboard() {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-slate-900">Chief Audit Executive Dashboard</h2>
        <p className="text-sm text-slate-500 mt-1">Portfolio overview — 2 active engagements</p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard label="Portfolio Score" value="72%" subtitle="NIS 2 compliance avg" accent="border-t-4 border-blue-500" trend={{ value: 4, positive: true }} />
        <MetricCard label="Active Engagements" value={mockEngagements.length} subtitle="1 in fieldwork, 1 planning" accent="border-t-4 border-purple-500" />
        <MetricCard label="Open Findings" value={mockFindings.filter(f => f.status === 'open').length} subtitle="2 critical, 2 high" accent="border-t-4 border-red-500" />
        <MetricCard label="Remediation Rate" value="38%" subtitle="30-day window" accent="border-t-4 border-green-500" trend={{ value: 8, positive: true }} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">Compliance Score Trend</h3>
          </CardHeader>
          <CardBody>
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={trendData}>
                <defs>
                  <linearGradient id="scoreGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#2563EB" stopOpacity={0.15}/>
                    <stop offset="95%" stopColor="#2563EB" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                <XAxis dataKey="month" tick={{ fontSize: 12 }} />
                <YAxis domain={[50, 100]} tick={{ fontSize: 12 }} />
                <Tooltip formatter={(v: number) => [`${v}%`, 'Score']} />
                <Area type="monotone" dataKey="score" stroke="#2563EB" strokeWidth={2} fill="url(#scoreGrad)" />
              </AreaChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>

        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">Finding Risk Distribution</h3>
          </CardHeader>
          <CardBody className="flex items-center justify-center">
            <ResponsiveContainer width="100%" height={200}>
              <PieChart>
                <Pie data={riskDistribution} cx="50%" cy="50%" innerRadius={55} outerRadius={80} paddingAngle={3} dataKey="value">
                  {riskDistribution.map((entry, i) => <Cell key={i} fill={entry.color} />)}
                </Pie>
                <Legend iconType="circle" iconSize={10} />
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>
      </div>

      {/* Engagement portfolio */}
      <Card>
        <CardHeader>
          <h3 className="font-semibold text-slate-900">Engagement Portfolio</h3>
        </CardHeader>
        <CardBody className="p-0">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-slate-500 uppercase text-xs tracking-wide">
              <tr>
                <th className="px-6 py-3 text-left">Engagement</th>
                <th className="px-6 py-3 text-left">Status</th>
                <th className="px-6 py-3 text-left">Score</th>
                <th className="px-6 py-3 text-left">Risk</th>
                <th className="px-6 py-3 text-left">Target End</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {mockEngagements.map(eng => (
                <tr key={eng.id} className="hover:bg-slate-50">
                  <td className="px-6 py-3 font-medium text-slate-900">
                    <Link href="/engagement" className="hover:text-blue-600">{eng.name}</Link>
                  </td>
                  <td className="px-6 py-3">
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">{eng.status}</span>
                  </td>
                  <td className="px-6 py-3">{eng.overall_score ? `${eng.overall_score}%` : '—'}</td>
                  <td className="px-6 py-3">{eng.risk_rating ? <SeverityBadge severity={eng.risk_rating}>{eng.risk_rating}</SeverityBadge> : '—'}</td>
                  <td className="px-6 py-3 text-slate-500">{eng.target_end_date ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardBody>
      </Card>
    </div>
  );
}
