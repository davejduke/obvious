'use client';
import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import type { ConfidenceFactor, OverallConfidence } from '@/lib/reasoning-types';

const factorColors: Record<string, { bar: string; bg: string; text: string }> = {
  scope:   { bar: 'bg-blue-500',   bg: 'bg-blue-50',   text: 'text-blue-700' },
  risk:    { bar: 'bg-orange-500', bg: 'bg-orange-50', text: 'text-orange-700' },
  quality: { bar: 'bg-purple-500', bg: 'bg-purple-50', text: 'text-purple-700' },
  economy: { bar: 'bg-green-500',  bg: 'bg-green-50',  text: 'text-green-700' },
};

function ScoreBar({ value, colorClass }: { value: number; colorClass: string }) {
  return (
    <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
      <div
        className={`h-2 rounded-full transition-all duration-500 ${colorClass}`}
        style={{ width: `${Math.max(0, Math.min(100, value))}%` }}
        role="progressbar"
        aria-valuenow={value}
        aria-valuemin={0}
        aria-valuemax={100}
      />
    </div>
  );
}

function FactorCard({ factor }: { factor: ConfidenceFactor }) {
  const [expanded, setExpanded] = useState(false);
  const colors = factorColors[factor.name] ?? factorColors.scope;

  return (
    <div className={`rounded-lg border border-slate-200 overflow-hidden`}>
      <button
        className="w-full flex items-center gap-3 px-4 py-3 hover:bg-slate-50 transition-colors text-left"
        onClick={() => setExpanded(e => !e)}
        aria-expanded={expanded}
        data-testid={`factor-${factor.name}`}
      >
        <div className={`w-8 h-8 rounded-md flex items-center justify-center text-xs font-bold ${colors.bg} ${colors.text}`}>
          {factor.name.charAt(0).toUpperCase()}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm font-semibold text-slate-800">{factor.label}</span>
            <span className={`text-sm font-bold ml-2 ${colors.text}`}>{factor.score}</span>
          </div>
          <ScoreBar value={factor.score} colorClass={colors.bar} />
        </div>
        <div className="text-slate-400 ml-2">
          <span className="text-xs text-slate-400 mr-2">×{factor.weight} = {factor.contribution.toFixed(1)}</span>
          {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </div>
      </button>

      {expanded && (
        <div className={`border-t border-slate-100 px-4 py-3 space-y-2 ${colors.bg}`}>
          {factor.breakdown.map((sub, i) => (
            <div key={i} className="space-y-1">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-slate-600">{sub.label}</span>
                <span className="text-xs font-semibold text-slate-700">{sub.value}</span>
              </div>
              <ScoreBar value={sub.value} colorClass={colors.bar} />
              <p className="text-xs text-slate-500">{sub.detail}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

interface Props {
  confidence: OverallConfidence;
}

export function ConfidenceFactorBreakdown({ confidence }: Props) {
  const scoreColor =
    confidence.score >= 75 ? 'text-green-600' :
    confidence.score >= 55 ? 'text-yellow-600' : 'text-red-600';

  const scoreRing =
    confidence.score >= 75 ? 'stroke-green-500' :
    confidence.score >= 55 ? 'stroke-yellow-500' : 'stroke-red-500';

  const circumference = 2 * Math.PI * 36;
  const offset = circumference - (confidence.score / 100) * circumference;

  return (
    <div data-testid="confidence-factor-breakdown">
      {/* Overall score gauge */}
      <div className="flex items-center gap-6 mb-6">
        <div className="relative w-24 h-24 shrink-0">
          <svg viewBox="0 0 80 80" className="w-24 h-24 -rotate-90">
            <circle cx="40" cy="40" r="36" fill="none" stroke="#E2E8F0" strokeWidth="6" />
            <circle
              cx="40" cy="40" r="36" fill="none" strokeWidth="6"
              strokeDasharray={circumference}
              strokeDashoffset={offset}
              strokeLinecap="round"
              className={`transition-all duration-700 ${scoreRing}`}
            />
          </svg>
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <span className={`text-2xl font-bold ${scoreColor}`}>{confidence.score}</span>
            <span className="text-xs text-slate-400">/ 100</span>
          </div>
        </div>
        <div>
          <p className="text-sm font-semibold text-slate-700">Overall Confidence</p>
          <p className="text-xs text-slate-500 mt-1">
            Weighted composite of scope, risk, quality, and economy factors.
            All values are deterministic — no LLM inference.
          </p>
          <p className="text-xs text-slate-400 mt-1">
            Computed {new Date(confidence.computed_at).toLocaleString()}
          </p>
        </div>
      </div>

      {/* Factor cards */}
      <div className="space-y-2">
        {confidence.factors.map(factor => (
          <FactorCard key={factor.name} factor={factor} />
        ))}
      </div>

      {/* Weight legend */}
      <div className="mt-4 p-3 bg-slate-50 rounded-lg text-xs text-slate-500">
        <span className="font-medium text-slate-600">Weights: </span>
        {confidence.factors.map((f, i) => (
          <span key={f.name}>
            {f.label} {Math.round(f.weight * 100)}%
            {i < confidence.factors.length - 1 ? ' · ' : ''}
          </span>
        ))}
      </div>
    </div>
  );
}
