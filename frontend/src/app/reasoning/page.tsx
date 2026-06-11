'use client';
import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardBody } from '@/components/ui/card';
import { ReasoningDAG } from '@/components/dag/reasoning-dag';
import { mockDAGNodes } from '@/lib/mock-data';
import { Network, CheckCircle2, Clock, Circle } from 'lucide-react';

const completedCount = mockDAGNodes.filter(n => n.status === 'completed').length;
const inProgressCount = mockDAGNodes.filter(n => n.status === 'in_progress').length;
const pendingCount = mockDAGNodes.filter(n => n.status === 'pending').length;

export default function ReasoningPage() {
  return (
    <AppShell title="Reasoning Visualizer">
      <div className="p-6 space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-slate-500 text-sm">D3.js DAG visualization of the AIAUDITOR reasoning engine execution graph</p>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <span className="flex items-center gap-1.5 text-green-600">
              <CheckCircle2 size={14} /> <span>{completedCount} completed</span>
            </span>
            <span className="flex items-center gap-1.5 text-blue-600">
              <Clock size={14} /> <span>{inProgressCount} in progress</span>
            </span>
            <span className="flex items-center gap-1.5 text-slate-400">
              <Circle size={14} /> <span>{pendingCount} pending</span>
            </span>
          </div>
        </div>

        {/* DAG Visualizer */}
        <ReasoningDAG />

        {/* Execution log */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Network size={16} className="text-slate-500" />
              <h3 className="font-semibold text-slate-900">Execution Log</h3>
            </div>
          </CardHeader>
          <CardBody>
            <div className="font-mono text-xs space-y-1 text-slate-600 bg-slate-900 rounded-md p-4 max-h-48 overflow-y-auto">
              <p className="text-green-400">[2024-01-25 09:00:01] ENGINE START: Audit engagement eng-001</p>
              <p className="text-green-400">[2024-01-25 09:00:02] COMPLETED: Audit Initiation</p>
              <p className="text-green-400">[2024-01-25 09:00:03] COMPLETED: Scope Analysis (NIS 2 Art. 21) — 10 articles in scope</p>
              <p className="text-green-400">[2024-01-25 09:01:15] COMPLETED: Evidence Collection — 47 items collected</p>
              <p className="text-green-400">[2024-01-25 09:02:30] COMPLETED: Control Mapping — 65/85 controls mapped</p>
              <p className="text-blue-400">[2024-01-25 09:03:00] IN_PROGRESS: Risk Assessment Engine — processing 65 controls</p>
              <p className="text-blue-400">[2024-01-25 09:03:12] IN_PROGRESS: Finding Generation — 3 findings generated so far</p>
              <p className="text-slate-400">[2024-01-25 09:??:??] PENDING: Compliance Scoring</p>
              <p className="text-slate-400">[2024-01-25 09:??:??] PENDING: Report Drafting</p>
              <p className="text-slate-400">[2024-01-25 09:??:??] PENDING: Audit Complete</p>
            </div>
          </CardBody>
        </Card>

        {/* Engine node table */}
        <Card>
          <CardHeader>
            <h3 className="font-semibold text-slate-900">Reasoning Engine Nodes</h3>
          </CardHeader>
          <CardBody className="p-0">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-xs text-slate-500 uppercase tracking-wide">
                <tr>
                  <th className="px-6 py-3 text-left">Node</th>
                  <th className="px-6 py-3 text-left">Type</th>
                  <th className="px-6 py-3 text-left">Status</th>
                  <th className="px-6 py-3 text-left">Dependencies</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {mockDAGNodes.map(node => (
                  <tr key={node.id} className="hover:bg-slate-50">
                    <td className="px-6 py-3 font-medium text-slate-900">{node.label.replace('\n', ' ')}</td>
                    <td className="px-6 py-3 capitalize text-slate-500">{node.type}</td>
                    <td className="px-6 py-3">
                      <span className={`inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded ${
                        node.status === 'completed' ? 'bg-green-100 text-green-700' :
                        node.status === 'in_progress' ? 'bg-blue-100 text-blue-700' :
                        'bg-slate-100 text-slate-500'
                      }`}>
                        {node.status.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="px-6 py-3 text-slate-400 text-xs font-mono">
                      {node.id}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardBody>
        </Card>
      </div>
    </AppShell>
  );
}
