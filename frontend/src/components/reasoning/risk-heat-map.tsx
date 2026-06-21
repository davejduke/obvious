'use client';
import { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import type { RiskHeatMapData, HeatMapControlPoint, RiskZone } from '@/lib/reasoning-types';

const ZONE_COLORS: Record<RiskZone, string> = {
  critical: '#FEE2E2', // red-100
  high:     '#FEF3C7', // amber-100
  medium:   '#FEF9C3', // yellow-100
  low:      '#DCFCE7', // green-100
};

const ZONE_STROKE: Record<RiskZone, string> = {
  critical: '#EF4444',
  high:     '#F59E0B',
  medium:   '#EAB308',
  low:      '#22C55E',
};

const ZONE_TEXT: Record<RiskZone, string> = {
  critical: '#B91C1C',
  high:     '#B45309',
  medium:   '#854D0E',
  low:      '#15803D',
};

function zoneFromScore(impact: number, likelihood: number): RiskZone {
  const score = impact * likelihood;
  if (score >= 15) return 'critical';
  if (score >= 10) return 'high';
  if (score >= 5)  return 'medium';
  return 'low';
}

interface Props {
  data: RiskHeatMapData;
}

export function RiskHeatMap({ data }: Props) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [tooltip, setTooltip] = useState<{
    control: HeatMapControlPoint;
    x: number;
    y: number;
  } | null>(null);

  useEffect(() => {
    if (!svgRef.current) return;

    const container = svgRef.current.parentElement;
    const containerWidth = container?.clientWidth ?? 480;
    const margin = { top: 40, right: 20, bottom: 50, left: 60 };
    const size = Math.min(containerWidth - margin.left - margin.right, 360);
    const cellSize = size / 5;
    const totalW = size + margin.left + margin.right;
    const totalH = size + margin.top + margin.bottom;

    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();
    svg.attr('width', totalW).attr('height', totalH);

    const g = svg.append('g').attr('transform', `translate(${margin.left},${margin.top})`);

    // Axis labels
    const impactLabels  = ['Minimal', 'Minor', 'Moderate', 'Major', 'Catastrophic'];
    const likelihoodLabels = ['Rare', 'Unlikely', 'Possible', 'Likely', 'Almost Certain'];

    // Draw heat map cells
    for (let impact = 1; impact <= 5; impact++) {
      for (let likelihood = 1; likelihood <= 5; likelihood++) {
        const zone = zoneFromScore(impact, likelihood);
        const x = (likelihood - 1) * cellSize;
        const y = (5 - impact) * cellSize;

        g.append('rect')
          .attr('x', x).attr('y', y)
          .attr('width', cellSize).attr('height', cellSize)
          .attr('fill', ZONE_COLORS[zone])
          .attr('stroke', '#CBD5E1')
          .attr('stroke-width', 0.5);

        // Score label in corner
        g.append('text')
          .attr('x', x + 4).attr('y', y + 12)
          .attr('font-size', '9px')
          .attr('fill', '#94A3B8')
          .text(impact * likelihood);
      }
    }

    // Plot control points
    const cellCounts: Map<string, number> = new Map();
    data.controls.forEach(ctrl => {
      const key = `${ctrl.impact}-${ctrl.likelihood}`;
      cellCounts.set(key, (cellCounts.get(key) ?? 0) + 1);
    });

    const cellOffsets: Map<string, number> = new Map();
    data.controls.forEach(ctrl => {
      const key = `${ctrl.impact}-${ctrl.likelihood}`;
      const idx = cellOffsets.get(key) ?? 0;
      cellOffsets.set(key, idx + 1);
      const count = cellCounts.get(key) ?? 1;

      const cx = (ctrl.likelihood - 1) * cellSize + cellSize / 2;
      const cy = (5 - ctrl.impact) * cellSize + cellSize / 2;

      const offsetX = count > 1 ? (idx - (count - 1) / 2) * 14 : 0;

      const zone = ctrl.zone;
      const circleG = g.append('g')
        .attr('transform', `translate(${cx + offsetX}, ${cy})`)
        .attr('cursor', 'pointer')
        .on('mouseenter', function(event: MouseEvent) {
          const rect = svgRef.current!.getBoundingClientRect();
          setTooltip({
            control: ctrl,
            x: event.clientX - rect.left,
            y: event.clientY - rect.top,
          });
        })
        .on('mouseleave', () => setTooltip(null));

      circleG.append('circle')
        .attr('r', 10)
        .attr('fill', ZONE_STROKE[zone])
        .attr('stroke', 'white')
        .attr('stroke-width', 2)
        .attr('opacity', 0.9);

      circleG.append('text')
        .attr('text-anchor', 'middle')
        .attr('dominant-baseline', 'middle')
        .attr('font-size', '7px')
        .attr('font-weight', '700')
        .attr('fill', 'white')
        .text(ctrl.article_ref.toUpperCase());
    });

    // X axis (likelihood)
    likelihoodLabels.forEach((label, i) => {
      g.append('text')
        .attr('x', i * cellSize + cellSize / 2)
        .attr('y', size + 16)
        .attr('text-anchor', 'middle')
        .attr('font-size', '9px')
        .attr('fill', '#64748B')
        .text(label);
    });

    g.append('text')
      .attr('x', size / 2)
      .attr('y', size + 38)
      .attr('text-anchor', 'middle')
      .attr('font-size', '11px')
      .attr('font-weight', '600')
      .attr('fill', '#475569')
      .text('Likelihood of Failure →');

    // Y axis (impact)
    impactLabels.forEach((label, i) => {
      g.append('text')
        .attr('x', -8)
        .attr('y', (4 - i) * cellSize + cellSize / 2)
        .attr('text-anchor', 'end')
        .attr('dominant-baseline', 'middle')
        .attr('font-size', '9px')
        .attr('fill', '#64748B')
        .text(label);
    });

    g.append('text')
      .attr('transform', `rotate(-90)`)
      .attr('x', -size / 2)
      .attr('y', -48)
      .attr('text-anchor', 'middle')
      .attr('font-size', '11px')
      .attr('font-weight', '600')
      .attr('fill', '#475569')
      .text('← Impact');

  }, [data]);

  return (
    <div className="relative" data-testid="risk-heat-map">
      <svg ref={svgRef} className="w-full" />

      {/* Legend */}
      <div className="flex flex-wrap gap-3 mt-2">
        {(['critical', 'high', 'medium', 'low'] as RiskZone[]).map(zone => (
          <div key={zone} className="flex items-center gap-1.5">
            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: ZONE_STROKE[zone] }} />
            <span className="text-xs capitalize" style={{ color: ZONE_TEXT[zone] }}>{zone}</span>
          </div>
        ))}
      </div>

      {/* Zone summary badges */}
      <div className="flex flex-wrap gap-2 mt-3">
        {(['critical', 'high', 'medium', 'low'] as RiskZone[]).map(zone => (
          <span
            key={zone}
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium"
            style={{ backgroundColor: ZONE_COLORS[zone], color: ZONE_TEXT[zone] }}
          >
            {data.zone_summary[zone]} {zone}
          </span>
        ))}
      </div>

      {/* Tooltip */}
      {tooltip && (
        <div
          className="absolute z-10 bg-white border border-slate-200 shadow-lg rounded-lg p-3 text-xs pointer-events-none w-52"
          style={{ left: tooltip.x + 12, top: tooltip.y - 40 }}
          data-testid="heat-map-tooltip"
        >
          <p className="font-semibold text-slate-900 mb-1">{tooltip.control.control_title}</p>
          <div className="space-y-0.5 text-slate-600">
            <p>Article: <span className="font-medium">{tooltip.control.article_ref}</span></p>
            <p>Impact: <span className="font-medium">{tooltip.control.impact} / 5</span></p>
            <p>Likelihood: <span className="font-medium">{tooltip.control.likelihood} / 5</span></p>
            <p>Score: <span className="font-medium">{tooltip.control.impact * tooltip.control.likelihood} / 25</span></p>
            <p>Zone:&nbsp;
              <span className="font-semibold capitalize" style={{ color: ZONE_TEXT[tooltip.control.zone] }}>
                {tooltip.control.zone}
              </span>
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
