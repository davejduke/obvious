'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { SeverityBadge } from '@/components/ui/badge';
import { mockEngagements, mockNIS2Score } from '@/lib/mock-data';
import { useState } from 'react';
import { FileText, Download, Eye, Plus, Calendar, Lock } from 'lucide-react';

const reportTemplates = [
  { id: 'nis2-exec', name: 'NIS 2 Executive Summary', description: 'Board-level compliance overview', pages: '~8 pages', access: ['audit_committee', 'cae'] },
  { id: 'nis2-full', name: 'NIS 2 Full Audit Report', description: 'Detailed findings and recommendations', pages: '~45 pages', access: ['internal_auditor', 'cae'] },
  { id: 'findings-register', name: 'Findings Register', description: 'Exportable findings with remediation status', pages: '~12 pages', access: ['internal_auditor', 'cae', 'auditee_ciso'] },
  { id: 'remediation-plan', name: 'Remediation Action Plan', description: 'Prioritised remediation roadmap', pages: '~20 pages', access: ['auditee_ciso', 'cae'] },
  { id: 'evidence-pack', name: 'Evidence Pack', description: 'Compiled evidence with linkages', pages: '~60 pages', access: ['internal_auditor'] },
];

const generatedReports = [
  { id: 'rpt-001', name: 'NIS 2 Executive Summary — Jan 2024', generated: '2024-01-25', engagement: 'NIS 2 Article 21 Audit 2024', format: 'PDF', size: '1.2 MB' },
  { id: 'rpt-002', name: 'Interim Findings Register — Jan 2024', generated: '2024-01-22', engagement: 'NIS 2 Article 21 Audit 2024', format: 'XLSX', size: '245 KB' },
];

export default function ReportsPage() {
  const [generating, setGenerating] = useState<string | null>(null);
  const [selectedEng, setSelectedEng] = useState(mockEngagements[0].id);

  const handleGenerate = (tplId: string) => {
    setGenerating(tplId);
    setTimeout(() => setGenerating(null), 2000);
  };

  return (
    <AppShell title="Reports">
      <div className="p-6 space-y-6">
        {/* Engagement selector */}
        <Card>
          <CardBody className="py-3">
            <div className="flex items-center gap-4">
              <label className="text-sm font-medium text-slate-700">Generate for:</label>
              <select
                value={selectedEng}
                onChange={e => setSelectedEng(e.target.value)}
                className="text-sm border border-slate-200 rounded-md px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {mockEngagements.map(e => (
                  <option key={e.id} value={e.id}>{e.name}</option>
                ))}
              </select>
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <Calendar size={14} />
                <span>Last generated: Jan 25, 2024</span>
              </div>
            </div>
          </CardBody>
        </Card>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Report templates */}
          <div className="space-y-4">
            <h3 className="font-semibold text-slate-900">Report Templates</h3>
            {reportTemplates.map(tpl => (
              <Card key={tpl.id} className="hover:shadow-md transition-shadow">
                <CardBody>
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-start gap-3">
                      <div className="w-10 h-10 bg-blue-50 rounded-lg flex items-center justify-center flex-shrink-0">
                        <FileText size={20} className="text-blue-600" />
                      </div>
                      <div>
                        <p className="font-semibold text-slate-900 text-sm">{tpl.name}</p>
                        <p className="text-xs text-slate-500 mt-0.5">{tpl.description}</p>
                        <p className="text-xs text-slate-400 mt-1">{tpl.pages}</p>
                        <div className="flex gap-1 mt-1.5 flex-wrap">
                          {tpl.access.map(a => (
                            <span key={a} className="text-xs bg-slate-100 text-slate-500 px-1.5 py-0.5 rounded flex items-center gap-1">
                              <Lock size={9} />
                              {a.replace('_', ' ')}
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      variant={generating === tpl.id ? 'ghost' : 'primary'}
                      disabled={generating === tpl.id}
                      onClick={() => handleGenerate(tpl.id)}
                    >
                      {generating === tpl.id ? (
                        <span className="flex items-center gap-1.5">
                          <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin" />
                          Generating...
                        </span>
                      ) : (
                        <><Plus size={14} />Generate</>
                      )}
                    </Button>
                  </div>
                </CardBody>
              </Card>
            ))}
          </div>

          {/* Generated reports */}
          <div className="space-y-4">
            <h3 className="font-semibold text-slate-900">Generated Reports</h3>
            {generatedReports.map(rpt => (
              <Card key={rpt.id} className="hover:shadow-md transition-shadow">
                <CardBody>
                  <div className="flex items-start gap-3">
                    <div className="w-10 h-10 bg-green-50 rounded-lg flex items-center justify-center flex-shrink-0">
                      <FileText size={20} className="text-green-600" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-semibold text-slate-900 text-sm">{rpt.name}</p>
                      <p className="text-xs text-slate-500 mt-0.5">{rpt.engagement}</p>
                      <div className="flex items-center gap-3 mt-1 text-xs text-slate-400">
                        <span>{rpt.format}</span>
                        <span>{rpt.size}</span>
                        <span>Generated: {rpt.generated}</span>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button variant="ghost" size="sm"><Eye size={14} />Preview</Button>
                      <Button variant="secondary" size="sm"><Download size={14} />Download</Button>
                    </div>
                  </div>
                </CardBody>
              </Card>
            ))}

            {/* NIS 2 Score snapshot */}
            <Card>
              <CardHeader>
                <h3 className="font-semibold text-slate-900 text-sm">Compliance Score Snapshot</h3>
              </CardHeader>
              <CardBody className="p-0">
                <table className="w-full text-xs">
                  <thead className="bg-slate-50 text-slate-400 uppercase tracking-wide">
                    <tr>
                      <th className="px-4 py-2 text-left">Article</th>
                      <th className="px-4 py-2 text-left">Score</th>
                      <th className="px-4 py-2 text-left">Findings</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {Object.entries(mockNIS2Score.by_article).map(([art, data]) => (
                      <tr key={art} className="hover:bg-slate-50">
                        <td className="px-4 py-2 font-mono font-medium">Art. {art.toUpperCase()}</td>
                        <td className="px-4 py-2">
                          <div className="flex items-center gap-2">
                            <div className="w-16 h-1.5 bg-slate-100 rounded-full">
                              <div className={`h-full rounded-full ${data.score >= 80 ? 'bg-green-500' : data.score >= 60 ? 'bg-yellow-500' : 'bg-red-500'}`}
                                style={{ width: `${data.score}%` }} />
                            </div>
                            <span className="font-medium">{data.score}%</span>
                          </div>
                        </td>
                        <td className="px-4 py-2">
                          {data.findings_count > 0 ? (
                            <span className={data.critical_findings > 0 ? 'text-red-600 font-medium' : 'text-slate-600'}>
                              {data.findings_count} {data.critical_findings > 0 && `(${data.critical_findings} crit)`}
                            </span>
                          ) : <span className="text-green-600">None</span>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardBody>
            </Card>
          </div>
        </div>
      </div>
    </AppShell>
  );
}
