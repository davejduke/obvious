// Reasoning Engine — deterministic confidence factor types
// No LLM explanation; all values are computed, not generated.

export type RiskZone = 'critical' | 'high' | 'medium' | 'low';

// ----- Confidence factors -----
export interface ConfidenceFactor {
  name: 'scope' | 'risk' | 'quality' | 'economy';
  label: string;
  score: number;          // 0–100
  weight: number;         // relative weight 0–1
  contribution: number;   // score × weight
  breakdown: ConfidenceSubFactor[];
}

export interface ConfidenceSubFactor {
  label: string;
  value: number;          // 0–100
  detail: string;
}

export interface OverallConfidence {
  score: number;           // 0–100 weighted composite
  factors: ConfidenceFactor[];
  computed_at: string;
  engagement_id: string;
}

// ----- Quality gate -----
export type GateStatus = 'passed' | 'blocked' | 'pending';

export interface QualityGateControl {
  control_id: string;
  control_title: string;
  article_ref: string;
  status: GateStatus;
  score: number;           // 0–100
  threshold: number;       // required minimum
  evidence_count: number;
  required_evidence: number;
  block_reasons: string[];
}

// ----- Evidence sufficiency -----
export interface EvidenceSufficiencyItem {
  control_id: string;
  control_title: string;
  article_ref: string;
  collected: number;
  required: number;
  sufficiency_pct: number; // 0–100
  status: 'sufficient' | 'partial' | 'insufficient';
}

// ----- Scope DAG with evidence counts -----
export interface ScopeDAGNode {
  id: string;
  label: string;
  type: 'start' | 'end' | 'process' | 'decision';
  status: 'completed' | 'in_progress' | 'pending';
  x: number;
  y: number;
  evidence_count?: number;
  children_ids?: string[];
  expanded?: boolean;
}

// ----- Risk heat map -----
export interface HeatMapControlPoint {
  control_id: string;
  control_title: string;
  impact: number;     // 1–5
  likelihood: number; // 1–5
  zone: RiskZone;
  article_ref: string;
}

export interface RiskHeatMapData {
  controls: HeatMapControlPoint[];
  zone_summary: Record<RiskZone, number>;
}

// ----- Full overlay state -----
export interface ReasoningEngineState {
  engagement_id: string;
  engagement_name: string;
  overall_confidence: OverallConfidence;
  quality_gates: QualityGateControl[];
  evidence_sufficiency: EvidenceSufficiencyItem[];
  heat_map: RiskHeatMapData;
  dag_nodes: ScopeDAGNode[];
  dag_edges: { source: string; target: string }[];
}

// ----- What If panel -----
export interface WhatIfQuery {
  engagement_id: string;
  control_id: string;
  action: 'add_evidence' | 'remove_evidence';
  evidence_quality_score: number; // 0–100
  count: number;
}

export interface WhatIfResult {
  original_score: number;
  simulated_score: number;
  delta: number;
  affected_factors: { factor: string; original: number; simulated: number }[];
  gate_change: 'no_change' | 'now_passes' | 'now_blocks';
  narrative: string; // deterministic text, no LLM
}

