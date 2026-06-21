'use client';
import { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import type { ScopeDAGNode } from '@/lib/reasoning-types';

type NodeStatus = 'completed' | 'in_progress' | 'pending';

const statusColors: Record<NodeStatus, string> = {
  completed:   '#16A34A',
  in_progress: '#2563EB',
  pending:     '#94A3B8',
};

const statusBg: Record<NodeStatus, string> = {
  completed:   '#DCFCE7',
  in_progress: '#DBEAFE',
  pending:     '#F1F5F9',
};

interface Props {
  nodes: ScopeDAGNode[];
  edges: { source: string; target: string }[];
}

export function ScopeDagViewer({ nodes, edges }: Props) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [selectedNode, setSelectedNode] = useState<ScopeDAGNode | null>(null);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  const toggleExpand = (nodeId: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  };

  useEffect(() => {
    if (!svgRef.current) return;
    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();

    const width  = svgRef.current.clientWidth || 800;
    const height = 660;
    svg.attr('width', width).attr('height', height);

    // Arrow marker
    svg.append('defs').append('marker')
      .attr('id', 'dag-arrow')
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 28).attr('refY', 0)
      .attr('markerWidth', 8).attr('markerHeight', 8)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', '#CBD5E1');

    const g = svg.append('g').attr('transform', `translate(${(width - 800) / 2 + 50}, 20)`);

    // Draw edges
    const edgeGroup = g.append('g');
    edges.forEach(edge => {
      const src = nodes.find(n => n.id === edge.source);
      const tgt = nodes.find(n => n.id === edge.target);
      if (!src || !tgt) return;
      edgeGroup.append('line')
        .attr('x1', src.x).attr('y1', src.y + 25)
        .attr('x2', tgt.x).attr('y2', tgt.y - 25)
        .attr('stroke', '#CBD5E1')
        .attr('stroke-width', 1.5)
        .attr('marker-end', 'url(#dag-arrow)');
    });

    // Draw nodes
    nodes.forEach(node => {
      const status = node.status as NodeStatus;
      const color = statusColors[status] ?? '#94A3B8';
      const bg    = statusBg[status]    ?? '#F1F5F9';
      const isExpanded = expandedNodes.has(node.id);
      const hasChildren = (node.children_ids?.length ?? 0) > 0;
      const evidenceCount = node.evidence_count ?? 0;

      const ng = g.append('g')
        .attr('transform', `translate(${node.x - 75}, ${node.y - 25})`)
        .attr('cursor', hasChildren ? 'pointer' : 'default')
        .on('click', () => {
          setSelectedNode(node);
          if (hasChildren) toggleExpand(node.id);
        });

      // Highlight selected
      if (selectedNode?.id === node.id) {
        ng.append('rect')
          .attr('x', -3).attr('y', -3)
          .attr('width', 156).attr('height', 56)
          .attr('rx', node.type === 'start' || node.type === 'end' ? 28 : 8)
          .attr('fill', 'none')
          .attr('stroke', color)
          .attr('stroke-width', 3)
          .attr('opacity', 0.4);
      }

      if (node.type === 'decision') {
        ng.append('polygon')
          .attr('points', '75,0 150,25 75,50 0,25')
          .attr('fill', bg).attr('stroke', color).attr('stroke-width', 2);
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

      // Evidence count badge
      if (evidenceCount > 0) {
        ng.append('rect')
          .attr('x', 2).attr('y', 2)
          .attr('width', 28).attr('height', 14)
          .attr('rx', 7)
          .attr('fill', '#1E293B')
          .attr('opacity', 0.75);
        ng.append('text')
          .attr('x', 16).attr('y', 9)
          .attr('text-anchor', 'middle')
          .attr('dominant-baseline', 'middle')
          .attr('font-size', '8px')
          .attr('font-weight', '700')
          .attr('fill', 'white')
          .text(`${evidenceCount}e`);
      }

      // Expand indicator for nodes with children
      if (hasChildren) {
        ng.append('text')
          .attr('x', 148).attr('y', 46)
          .attr('text-anchor', 'end')
          .attr('font-size', '10px')
          .attr('fill', color)
          .text(isExpanded ? '▲' : '▼');
      }

      // Label
      const lines = node.label.split('\n');
      lines.forEach((line, i) => {
        ng.append('text')
          .attr('x', 75)
          .attr('y', 22 + (i - (lines.length - 1) / 2) * 14)
          .attr('text-anchor', 'middle')
          .attr('dominant-baseline', 'middle')
          .attr('font-size', '11px')
          .attr('font-family', 'Inter, sans-serif')
          .attr('font-weight', '500')
          .attr('fill', '#1E293B')
          .text(line);
      });
    });

    // Zoom
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.5, 2])
      .on('zoom', event => g.attr('transform', event.transform.toString()));
    svg.call(zoom);

  }, [nodes, edges, selectedNode, expandedNodes]);

  return (
    <div className="relative w-full" data-testid="scope-dag-viewer">
      <svg
        ref={svgRef}
        className="w-full bg-slate-50 rounded-lg border border-slate-200"
        style={{ height: 660 }}
      />

      {/* Legend */}
      <div className="absolute bottom-4 left-4 bg-white border border-slate-200 rounded-lg p-3 shadow-sm">
        <p className="text-xs font-semibold text-slate-500 mb-2 uppercase tracking-wide">Status</p>
        {(Object.entries(statusColors) as [NodeStatus, string][]).map(([status, color]) => (
          <div key={status} className="flex items-center gap-2 mb-1">
            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: color }} />
            <span className="text-xs text-slate-600 capitalize">{status.replace('_', ' ')}</span>
          </div>
        ))}
        <div className="flex items-center gap-2 mt-2 border-t border-slate-100 pt-2">
          <div className="w-6 h-3 rounded bg-slate-800 opacity-75 flex items-center justify-center">
            <span className="text-white text-[6px] font-bold">Ne</span>
          </div>
          <span className="text-xs text-slate-500">Evidence count</span>
        </div>
        <p className="text-xs text-slate-400 mt-1">Scroll to zoom · Click to expand</p>
      </div>

      {/* Node detail panel */}
      {selectedNode && (
        <div className="absolute top-4 right-4 bg-white border border-slate-200 rounded-lg p-4 shadow-md w-60">
          <div className="flex items-center justify-between mb-2">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Node Detail</p>
            <button
              onClick={() => setSelectedNode(null)}
              className="text-slate-400 hover:text-slate-600 text-xs"
            >
              ✕
            </button>
          </div>
          <p className="font-semibold text-slate-900 text-sm">{selectedNode.label.replace('\n', ' ')}</p>
          <div className="mt-2 space-y-1 text-xs">
            <div className="flex justify-between">
              <span className="text-slate-500">Type</span>
              <span className="font-medium capitalize">{selectedNode.type}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">Status</span>
              <span className="font-medium" style={{ color: statusColors[selectedNode.status as NodeStatus] }}>
                {selectedNode.status.replace('_', ' ')}
              </span>
            </div>
            {(selectedNode.evidence_count ?? 0) > 0 && (
              <div className="flex justify-between">
                <span className="text-slate-500">Evidence items</span>
                <span className="font-medium text-slate-700">{selectedNode.evidence_count}</span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
