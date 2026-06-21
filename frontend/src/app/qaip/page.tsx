'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody, MetricCard } from '@/components/ui/card';
import {
  BarChart, Bar, LineChart, Line, AreaChart, Area,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, Legend, ReferenceLine,
} from 'recharts';
import {
  iiaStandards,
  yearOverYearTrend,
  resolutionTimeData,
  computeEngagementCompletionRate,
  computeOnTimeRate,
  computeFindingSeverityDistribution,
  computeAvgResolutionDays,
  computeIIAConformanceScore,
  buildEngagementMetrics,
} from '@/lib/qaip-data';
import { mockEngagements, mockFindings } from '@/lib/mock-data';
import { useState } from 'react';
import { clsx } from 'clsx';
import {
  Award, TrendingUp, TrendingDown,
  CheckCircle, Clock, AlertTriangle,
} from 'lucide-react';

type Tab = 'overview' | 'engagement' | 'findings' | 'conformance' | 'trends';

const TABS: { id: Tab; label: string }[] = [
  { id: 'overview',    label: 'Overview' },
  { id: 'engagement',  label: 'Engagement Metrics' },
  { id: 'findings',    label: 'Finding Quality' },
  { id: 'conformance', label: 'IIA Conformance' },
  { id: 'trends',      label: 'Year-over-Year' },
];

export default function QAIPPage() {
  const [activeTab, setActiveTab] = useState<Tab>('overview');

  // Computed metrics from live mock data
  const completionRate    = computeEngagementCompletionRate(mockEngagements);
  const onTimeRate        = computeOnTimeRate(mockEngagements);
  const avgResolutionDays = computeAvgResolutionDays(mockFindings);
  const conformanceScore  = computeIIAConformanceScore(iiaStandards);
  const severityDist      = computeFindingSeverityDistribution(mockFindings);
  const engagementMetrics = buildEngagementMetrics(mockEngagements, mockFindings);

  return (
    <AppShell title="QAIP Dashboard">
      <div className="p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-900">
              Quality Assurance &amp; Improvement Program
            </h2>
            <p className="text-sm text-slate-500 mt-1">
              IIA Standard 1300 — Internal audit function performance
            </p>
          </div>
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-green-50 border border-green-200 rounded-lg text-xs font-medium text-green-700">
            <Award size={12} />
            IIA Conformance: {conformanceScore}%
          </span>
        </div>

        {/* Tab bar */}
        <div className="border-b border-slate-200">
          <div className="flex gap-1">
            {TABS.map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={clsx(
                  'px-4 py-2.5 text-sm font-medium border-b-2 transition-colors',
                  activeTab === tab.id
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-slate-500 hover:text-slate-900'
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        {/* ── Overview ────────────────────────────────────────────────────────── */}
        {activeTab === 'overview' && (
          <div className="space-y-6">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <MetricCard label="Completion Rate"   value={`${completionRate}%`}    subtitle="Engagements finished"         accent="border-t-4 border-blue-500"   trend={{ value: 5, positive: true }} />
              <MetricCard label="On-Time Delivery"  value={`${onTimeRate}%`}         subtitle="Within target date"          accent="border-t-4 border-green-500"  trend={{ value: 3, positive: true }} />
              <MetricCard label="Avg Resolution"    value={`${avgResolutionDays}d`}  subtitle="Mean finding close time"     accent="border-t-4 border-orange-500" trend={{ value: 4, positive: false }} />
              <MetricCard label="IIA Conformance"   value={`${conformanceScore}%`}   subtitle="Standards 1000–2600"         accent="border-t-4 border-purple-500" trend={{ value: 2, positive: true }} />
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900">Engagement Delivery Trend</h3></CardHeader>
                <CardBody>
                  <ResponsiveContainer width="100%" height={220}>
                    <AreaChart data={yearOverYearTrend}>
                      <defs>
                        <linearGradient id="qCompGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%"  stopColor="#2563EB" stopOpacity={0.15} />
                          <stop offset="95%" stopColor="#2563EB" stopOpacity={0}    />
                        </linearGradient>
                        <linearGradient id="qOnTimeGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%"  stopColor="#16A34A" stopOpacity={0.1} />
                          <stop offset="95%" stopColor="#16A34A" stopOpacity={0}   />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                      <XAxis dataKey="period" tick={{ fontSize: 11 }} />
                      <YAxis domain={[60, 100]} tick={{ fontSize: 11 }} />
                      <Tooltip
                        formatter={(v: unknown, name: string) =>
                          [`${v}%`, name === 'completionRate' ? 'Completion' : 'On-Time']
                        }
                      />
                      <Area type="monotone" dataKey="completionRate" stroke="#2563EB" strokeWidth={2} fill="url(#qCompGrad)"   name="completionRate" />
                      <Area type="monotone" dataKey="onTimeRate"     stroke="#16A34A" strokeWidth={2} fill="url(#qOnTimeGrad)" name="onTimeRate" />
                    </AreaChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>

              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900">Finding Severity Distribution</h3></CardHeader>
                <CardBody className="flex items-center justify-center">
                  <ResponsiveContainer width="100%" height={220}>
                    <PieChart>
                      <Pie data={severityDist} cx="50%" cy="50%" innerRadius={60} outerRadius={90} paddingAngle={3} dataKey="value">
                        {severityDist.map((entry, i) => <Cell key={i} fill={entry.color} />)}
                      </Pie>
                      <Legend iconType="circle" iconSize={10} formatter={value => <span className="text-xs text-slate-600">{value}</span>} />
                      <Tooltip />
                    </PieChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>
            </div>
          </div>
        )}

        {/* ── Engagement Metrics ──────────────────────────────────────────── */}
        {activeTab === 'engagement' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900">Engagement Completion Rates</h3></CardHeader>
                <CardBody>
                  <ResponsiveContainer width="100%" height={240}>
                    <BarChart data={engagementMetrics} layout="vertical">
                      <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                      <XAxis type="number" domain={[0, 100]} tick={{ fontSize: 11 }} />
                      <YAxis type="category" dataKey="name" tick={{ fontSize: 10 }} width={150} />
                      <Tooltip formatter={(v: unknown) => [`${v}%`, 'Completion']} />
                      <Bar dataKey="completion" fill="#2563EB" radius={[0, 4, 4, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>

              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900">On-Time Delivery Rate by Quarter</h3></CardHeader>
                <CardBody>
                  <ResponsiveContainer width="100%" height={240}>
                    <BarChart data={yearOverYearTrend}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                      <XAxis dataKey="period" tick={{ fontSize: 11 }} />
                      <YAxis domain={[0, 100]} tick={{ fontSize: 11 }} />
                      <Tooltip formatter={(v: unknown) => [`${v}%`, 'On-Time Rate']} />
                      <ReferenceLine y={80} stroke="#DC2626" strokeDasharray="4 4"
                        label={{ value: 'Target 80%', fontSize: 10, fill: '#DC2626', position: 'insideTopRight' }}
                      />
                      <Bar dataKey="onTimeRate" fill="#16A34A" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>
            </div>

            <Card>
              <CardHeader><h3 className="font-semibold text-slate-900">Engagement Detail</h3></CardHeader>
              <CardBody className="p-0">
                <table className="w-full text-sm">
                  <thead className="bg-slate-50 text-slate-500 uppercase text-xs tracking-wide">
                    <tr>
                      <th className="px-6 py-3 text-left">Engagement</th>
                      <th className="px-6 py-3 text-left">Completion</th>
                      <th className="px-6 py-3 text-left">On Time</th>
                      <th className="px-6 py-3 text-left">Findings</th>
                      <th className="px-6 py-3 text-left">Critical</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {engagementMetrics.map((row, i) => (
                      <tr key={i} className="hover:bg-slate-50">
                        <td className="px-6 py-3 font-medium text-slate-900">{row.name}</td>
                        <td className="px-6 py-3">
                          <div className="flex items-center gap-2">
                            <div className="flex-1 bg-slate-100 rounded-full h-2 max-w-24">
                              <div className="bg-blue-500 h-2 rounded-full" style={{ width: `${row.completion}%` }} />
                            </div>
                            <span className="text-xs text-slate-600">{row.completion}%</span>
                          </div>
                        </td>
                        <td className="px-6 py-3">
                          {row.onTime
                            ? <span className="inline-flex items-center gap-1 text-xs text-green-700"><CheckCircle size={12} /> On Track</span>
                            : <span className="inline-flex items-center gap-1 text-xs text-red-600"><Clock size={12} /> Overdue</span>
                          }
                        </td>
                        <td className="px-6 py-3 text-slate-700">{row.findings}</td>
                        <td className="px-6 py-3">
                          {row.criticalCount > 0 ? (
                            <span className="inline-flex items-center gap-1 text-xs text-red-700 bg-red-50 border border-red-100 px-2 py-0.5 rounded">
                              <AlertTriangle size={10} />{row.criticalCount} critical
                            </span>
                          ) : <span className="text-xs text-slate-400">—</span>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          </div>
        )}

        {/* ── Finding Quality ─────────────────────────────────────────────── */}
        {activeTab === 'findings' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card>
                <CardHeader><h3 className="font-semibold text-slate-900">Finding Severity Distribution</h3></CardHeader>
                <CardBody>
                  <ResponsiveContainer width="100%" height={240}>
                    <BarChart data={severityDist}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                      <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                      <YAxis tick={{ fontSize: 12 }} allowDecimals={false} />
                      <Tooltip />
                      <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                        {severityDist.map((entry, i) => <Cell key={i} fill={entry.color} />)}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <h3 className="font-semibold text-slate-900">Resolution Time vs Target (Days)</h3>
                  <p className="text-xs text-slate-500 mt-0.5">Actual resolution days compared to SLA target</p>
                </CardHeader>
                <CardBody>
                  <ResponsiveContainer width="100%" height={240}>
                    <BarChart data={resolutionTimeData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                      <XAxis dataKey="severity" tick={{ fontSize: 12 }} />
                      <YAxis tick={{ fontSize: 12 }} />
                      <Tooltip />
                      <Legend />
                      <Bar dataKey="avgDays" name="Actual (days)" fill="#2563EB" radius={[4, 4, 0, 0]} />
                      <Bar dataKey="target"  name="Target (days)" fill="#E2E8F0" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>
            </div>

            <Card>
              <CardHeader><h3 className="font-semibold text-slate-900">Finding Quality Metrics</h3></CardHeader>
              <CardBody>
                <div className="grid grid-cols-3 gap-6">
                  {([
                    { label: 'Finding Acceptance Rate', value: '94%', desc: 'Auditee-accepted findings',     positive: true  },
                    { label: 'Evidence Sufficiency',     value: '87%', desc: 'Findings with sufficient evidence', positive: true  },
                    { label: 'Repeat Findings',          value: '18%', desc: 'Findings from prior periods',   positive: false },
                  ] as const).map(m => (
                    <div key={m.label} className="text-center p-4 bg-slate-50 rounded-lg">
                      <p className="text-2xl font-bold text-slate-900">{m.value}</p>
                      <p className="text-sm font-medium text-slate-700 mt-1">{m.label}</p>
                      <p className="text-xs text-slate-500 mt-0.5">{m.desc}</p>
                      <span className={clsx('text-xs font-medium mt-1 inline-flex items-center gap-1', m.positive ? 'text-green-600' : 'text-orange-600')}>
                        {m.positive
                          ? <><TrendingUp size={12} /> Above target</>
                          : <><TrendingDown size={12} /> Needs improvement</>}
                      </span>
                    </div>
                  ))}
                </div>
              </CardBody>
            </Card>
          </div>
        )}

        {/* ── IIA Conformance ────────────────────────────────────────────── */}
        {activeTab === 'conformance' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-slate-900">IIA Standards Conformance</h3>
                  <span className="text-sm text-slate-500">Overall: {conformanceScore}%</span>
                </div>
              </CardHeader>
              <CardBody>
                <ResponsiveContainer width="100%" height={340}>
                  <BarChart data={iiaStandards} layout="vertical">
                    <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                    <XAxis type="number" domain={[0, 100]} tick={{ fontSize: 11 }} />
                    <YAxis type="category" dataKey="code" tick={{ fontSize: 11 }} width={50} />
                    <Tooltip
                      formatter={(v: unknown) => [`${v}%`, 'Conformance']}
                      labelFormatter={label => {
                        const std = iiaStandards.find(s => s.code === label);
                        return std ? `${std.code} — ${std.name}` : String(label);
                      }}
                    />
                    <ReferenceLine x={80} stroke="#EA580C" strokeDasharray="4 4" />
                    <Bar dataKey="conformanceScore" radius={[0, 4, 4, 0]}>
                      {iiaStandards.map((entry, i) => (
                        <Cell
                          key={i}
                          fill={
                            entry.conformanceScore >= 85 ? '#16A34A'
                              : entry.conformanceScore >= 80 ? '#CA8A04'
                              : '#DC2626'
                          }
                        />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </CardBody>
            </Card>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {(['attribute', 'performance'] as const).map(family => {
                const stds = iiaStandards.filter(s => s.family === family);
                const avg  = Math.round(stds.reduce((sum, s) => sum + s.conformanceScore, 0) / stds.length);
                return (
                  <Card key={family}>
                    <CardHeader>
                      <div className="flex items-center justify-between">
                        <h3 className="font-semibold text-slate-900 capitalize">{family} Standards</h3>
                        <span className={clsx(
                          'text-xs font-medium px-2 py-0.5 rounded',
                          avg >= 85 ? 'bg-green-50 text-green-700'
                            : avg >= 80 ? 'bg-yellow-50 text-yellow-700'
                            : 'bg-red-50 text-red-700'
                        )}>{avg}% avg</span>
                      </div>
                    </CardHeader>
                    <CardBody className="p-0">
                      <table className="w-full text-sm">
                        <thead className="bg-slate-50 text-slate-500 uppercase text-xs">
                          <tr>
                            <th className="px-4 py-2 text-left">Code</th>
                            <th className="px-4 py-2 text-left">Standard</th>
                            <th className="px-4 py-2 text-right">Score</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                          {stds.map(std => (
                            <tr key={std.id} className="hover:bg-slate-50">
                              <td className="px-4 py-2.5 font-mono text-xs text-slate-600">{std.code}</td>
                              <td className="px-4 py-2.5 text-xs text-slate-700">{std.name}</td>
                              <td className="px-4 py-2.5 text-right">
                                <div className="flex items-center justify-end gap-2">
                                  <div className="w-16 bg-slate-100 rounded-full h-1.5">
                                    <div
                                      className={clsx(
                                        'h-1.5 rounded-full',
                                        std.conformanceScore >= 85 ? 'bg-green-500'
                                          : std.conformanceScore >= 80 ? 'bg-yellow-500'
                                          : 'bg-red-500'
                                      )}
                                      style={{ width: `${std.conformanceScore}%` }}
                                    />
                                  </div>
                                  <span className="text-xs font-medium text-slate-700 w-9 text-right">{std.conformanceScore}%</span>
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </CardBody>
                  </Card>
                );
              })}
            </div>
          </div>
        )}

        {/* ── Year-over-Year Trends ────────────────────────────────────────── */}
        {activeTab === 'trends' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card>
                <CardHeader>
                  <h3 className="font-semibold text-slate-900">Completion &amp; On-Time Rate</h3>
                </CardHeader>
                <CardBody>
                  <ResponsiveContainer width="100%" height={240}>
                    <LineChart data={yearOverYearTrend}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                      <XAxis dataKey="period" tick={{ fontSize: 11 }} />
                      <YAxis domain={[60, 100]} tick={{ fontSize: 11 }} />
                      <Tooltip formatter={(v: unknown) => `${v}%`} />
                      <Legend />
                      <Line type="monotone" dataKey="completionRate" stroke="#2563EB" strokeWidth={2} dot={{ r: 4 }} name="Completion Rate" />
                      <Line type="monotone" dataKey="onTimeRate"     stroke="#16A34A" strokeWidth={2} dot={{ r: 4 }} name="On-Time Rate"   />
                    </LineChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <h3 className="font-semibold text-slate-900">IIA Conformance Score Trend</h3>
                </CardHeader>
                <CardBody>
                  <ResponsiveContainer width="100%" height={240}>
                    <AreaChart data={yearOverYearTrend}>
                      <defs>
                        <linearGradient id="confGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%"  stopColor="#7C3AED" stopOpacity={0.15} />
                          <stop offset="95%" stopColor="#7C3AED" stopOpacity={0}    />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                      <XAxis dataKey="period" tick={{ fontSize: 11 }} />
                      <YAxis domain={[60, 100]} tick={{ fontSize: 11 }} />
                      <Tooltip formatter={(v: unknown) => `${v}%`} />
                      <Area type="monotone" dataKey="conformanceScore" stroke="#7C3AED" strokeWidth={2} fill="url(#confGrad)" name="Conformance" />
                    </AreaChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <h3 className="font-semibold text-slate-900">Finding Count Trend</h3>
                <p className="text-xs text-slate-500 mt-0.5">Quarterly finding volume — lower is better</p>
              </CardHeader>
              <CardBody>
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart data={yearOverYearTrend}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                    <XAxis dataKey="period" tick={{ fontSize: 11 }} />
                    <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                    <Tooltip />
                    <Bar dataKey="findingCount" fill="#EA580C" radius={[4, 4, 0, 0]} name="Findings" />
                  </BarChart>
                </ResponsiveContainer>
              </CardBody>
            </Card>
          </div>
        )}
      </div>
    </AppShell>
  );
}

