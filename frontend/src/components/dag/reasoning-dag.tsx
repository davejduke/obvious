'use client';
import { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import { mockDAGNodes, mockDAGEdges } from '@/lib/mock-data';

type NodeStatus = 'completed' | 'in_progress' | 'pending';

const statusColors: Record<NodeStatus, string> = {
  completed: '#16A34A',
  in_progress: '#2563EB',
  pending: '#94A3B8',
};

const statusBg: Record<NodeStatus, string> = {
  completed: '#DCFCE7',
  in_progress: '#DBEAFE',
  pending: '#F1F5F9',
};

const nodeTypeShape: Record<string, string> = {
  start: 'rounded',
  end: 'rounded',
  process: 'rect',
  decision: 'diamond',
};

interface DAGNode {
  id: string;
  label: string;
  type: string;
  status: NodeStatus;
  x: number;
  y: number;
}

interface DAGEdge {
  source: string;
  target: string;
}

export function ReasoningDAG() {
  const svgRef = useRef<SVGSVGElement>(null);
  const [selectedNode, setSelectedNode] = useState<DAGNode | null>(null);

  useEffect(() => {
    if (!svgRef.current) return;
    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();

    const width = svgRef.current.clientWidth || 800;
    const height = svgRef.current.clientHeight || 660;

    svg.attr('width', width).attr('height', height);

    // Arrow marker
    svg.append('defs').append('marker')
      .attr('id', 'arrow')
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 28).attr('refY', 0)
      .attr('markerWidth', 8).attr('markerHeight', 8)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', '#CBD5E1');

    const g = svg.append('g').attr('transform', `translate(${(width - 800) / 2 + 50}, 20)`);

    // Draw edges
    const typedNodes = mockDAGNodes.map(n => ({ ...n, status: n.status as NodeStatus }));
    const edgeGroup = g.append('g').attr('class', 'edges');
    mockDAGEdges.forEach((edge) => {
      const src = typedNodes.find((n) => n.id === edge.source);
      const tgt = typedNodes.find((n) => n.id === edge.target);
      if (!src || !tgt) return;

      edgeGroup.append('line')
        .attr('x1', src.x).attr('y1', src.y + 25)
        .attr('x2', tgt.x).attr('y2', tgt.y - 25)
        .attr('stroke', '#CBD5E1').attr('stroke-width', 1.5)
        .attr('marker-end', 'url(#arrow)');
    });

    // Draw nodes
    const nodeGroup = g.append('g').attr('class', 'nodes');
    typedNodes.forEach((node: DAGNode) => {
      const ng = nodeGroup.append('g')
        .attr('transform', `translate(${node.x - 75}, ${node.y - 25})`)

        .attr('cursor', 'pointer')
        .on('click', () => setSelectedNode(node));

      const color = statusColors[node.status] ?? '#94A3B8';
      const bg = statusBg[node.status] ?? '#F1F5F9';

      if (node.type === 'decision') {
        // Diamond shape for decisions
        const pts = '75,0 150,25 75,50 0,25';
        ng.append('polygon')
          .attr('points', pts)
          .attr('fill', bg).attr('stroke', color).attr('stroke-width', 2)
          .attr('rx', 0);
      } else {
        ng.append('rect')
          .attr('width', 150).attr('height', 50)
          .attr('rx', node.type === 'start' || node.type === 'end' ? 25 : 6)
          .attr('fill', bg).attr('stroke', color).attr('stroke-width', 2);
      }

      // Status dot
      ng.append('circle')
        .attr('cx', 135).attr('cy', 8)
        .attr('r', 5)
        .attr('fill', color);

      // Label
      const lines = node.label.split('\n');
      lines.forEach((line, i) => {
        ng.append('text')
          .attr('x', 75).attr('y', 22 + (i - (lines.length - 1) / 2) * 14)
          .attr('text-anchor', 'middle')
          .attr('dominant-baseline', 'middle')
          .attr('font-size', '11px')
          .attr('font-family', 'Inter, sans-serif')
          .attr('font-weight', '500')
          .attr('fill', '#1E293B')
          .text(line);
      });
    });

    // Zoom support
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.5, 2])
      .on('zoom', (event) => g.attr('transform', event.transform.toString()));
    svg.call(zoom);

  }, []);

  return (
    <div className="relative w-full">
      <svg
        ref={svgRef}
        className="w-full bg-slate-50 rounded-lg border border-slate-200"
        style={{ height: 680 }}
      />

      {/* Legend */}
      <div className="absolute bottom-4 left-4 bg-white border border-slate-200 rounded-lg p-3 shadow-sm">
        <p className="text-xs font-semibold text-slate-500 mb-2 uppercase tracking-wide">Node Status</p>
        {(Object.entries(statusColors) as [NodeStatus, string][]).map(([status, color]) => (
          <div key={status} className="flex items-center gap-2 mb-1">
            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: color }} />
            <span className="text-xs text-slate-600 capitalize">{status.replace('_', ' ')}</span>
          </div>
        ))}
        <p className="text-xs text-slate-400 mt-2">Scroll to zoom · Click node for details</p>
      </div>

      {/* Node detail panel */}
      {selectedNode && (
        <div className="absolute top-4 right-4 bg-white border border-slate-200 rounded-lg p-4 shadow-md w-56">
          <div className="flex items-center justify-between mb-2">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Node Detail</p>
            <button onClick={() => setSelectedNode(null)} className="text-slate-400 hover:text-slate-600 text-xs">✕</button>
          </div>
          <p className="font-semibold text-slate-900 text-sm">{selectedNode.label.replace('\n', ' ')}</p>
          <div className="mt-2 space-y-1">
            <div className="flex justify-between text-xs">
              <span className="text-slate-500">Type</span>
              <span className="font-medium capitalize">{selectedNode.type}</span>
            </div>
            <div className="flex justify-between text-xs">
              <span className="text-slate-500">Status</span>
              <span className="font-medium" style={{ color: statusColors[selectedNode.status] }}>
                {selectedNode.status.replace('_', ' ')}
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
