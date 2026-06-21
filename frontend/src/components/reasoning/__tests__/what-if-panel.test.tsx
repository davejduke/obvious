/**
 * What If Panel — simulation interaction tests.
 *
 * Tests cover:
 *  - Panel renders with controls list
 *  - Action toggle (add / remove)
 *  - Simulate button calls onSimulate with correct query
 *  - Result is displayed after simulation
 *  - Narrative text appears
 *  - Gate change badge appears
 */
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { WhatIfPanel } from '../what-if-panel';
import type { WhatIfQuery, WhatIfResult } from '@/lib/reasoning-types';
import { mockReasoningState } from '@/lib/mock-data';

const mockResult: WhatIfResult = {
  original_score: 72,
  simulated_score: 79,
  delta: 7,
  affected_factors: [
    { factor: 'Quality',             original: 48, simulated: 63 },
    { factor: 'Evidence Sufficiency', original: 33, simulated: 57 },
  ],
  gate_change: 'now_passes',
  narrative: 'Adding 2 evidence items at 80% quality to "Incident Response Plan" increases overall confidence by 7 points (quality gate now PASSES).',
};

describe('WhatIfPanel', () => {
  const onSimulate = jest.fn().mockResolvedValue(mockResult);

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders with control options', () => {
    render(
      <WhatIfPanel
        engagementId="eng-001"
        controls={mockReasoningState.quality_gates}
        onSimulate={onSimulate}
      />,
    );
    expect(screen.getByTestId('what-if-panel')).toBeInTheDocument();
    expect(screen.getByTestId('what-if-control-select')).toBeInTheDocument();
  });

  it('defaults to add_evidence action', () => {
    render(
      <WhatIfPanel
        engagementId="eng-001"
        controls={mockReasoningState.quality_gates}
        onSimulate={onSimulate}
      />,
    );
    // Add evidence button should be highlighted (active)
    const addBtn = screen.getByTestId('action-add_evidence');
    expect(addBtn).toBeInTheDocument();
  });

  it('toggles to remove_evidence', () => {
    render(
      <WhatIfPanel
        engagementId="eng-001"
        controls={mockReasoningState.quality_gates}
        onSimulate={onSimulate}
      />,
    );
    fireEvent.click(screen.getByTestId('action-remove_evidence'));
    // Button should now be active (has blue bg class)
    expect(screen.getByTestId('action-remove_evidence').className).toContain('bg-blue-600');
  });

  it('calls onSimulate when simulate button clicked', async () => {
    render(
      <WhatIfPanel
        engagementId="eng-001"
        controls={mockReasoningState.quality_gates}
        onSimulate={onSimulate}
      />,
    );
    fireEvent.click(screen.getByTestId('simulate-button'));
    await waitFor(() => expect(onSimulate).toHaveBeenCalledTimes(1));
    const call = onSimulate.mock.calls[0][0] as WhatIfQuery;
    expect(call.engagement_id).toBe('eng-001');
    expect(call.action).toBe('add_evidence');
    expect(call.count).toBe(1);
  });

  it('shows result after simulation', async () => {
    render(
      <WhatIfPanel
        engagementId="eng-001"
        controls={mockReasoningState.quality_gates}
        onSimulate={onSimulate}
      />,
    );
    fireEvent.click(screen.getByTestId('simulate-button'));
    await waitFor(() => screen.getByTestId('what-if-result'));
    expect(screen.getByTestId('what-if-result')).toBeInTheDocument();
  });

  it('displays narrative text in result', async () => {
    render(
      <WhatIfPanel
        engagementId="eng-001"
        controls={mockReasoningState.quality_gates}
        onSimulate={onSimulate}
      />,
    );
    fireEvent.click(screen.getByTestId('simulate-button'));
    await waitFor(() => screen.getByText(/increases overall confidence/i));
    expect(screen.getByText(/increases overall confidence/i)).toBeInTheDocument();
  });

  it('shows gate change badge', async () => {
    render(
      <WhatIfPanel
        engagementId="eng-001"
        controls={mockReasoningState.quality_gates}
        onSimulate={onSimulate}
      />,
    );
    fireEvent.click(screen.getByTestId('simulate-button'));
    await waitFor(() => screen.getByTestId('gate-change-badge'));
    const badge = screen.getByTestId('gate-change-badge');
    expect(badge).toBeInTheDocument();
    // gate_change value is set as a data attribute for reliable assertion
    expect(badge).toHaveAttribute('data-gate-change', 'now_passes');
  });
});
