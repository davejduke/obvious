/**
 * Reasoning Engine Overlay — rendering and interaction tests.
 *
 * Tests cover:
 *  - Overlay does not render when open=false
 *  - Overlay renders with correct title when open=true
 *  - Overall confidence score is displayed
 *  - All tabs are accessible
 *  - Tab switching shows correct content
 *  - Escape key closes the overlay
 *  - Close button closes the overlay
 */
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ReasoningEngineOverlay, ReasoningEngineTrigger } from '../reasoning-engine-overlay';
import { mockReasoningState } from '@/lib/mock-data';

// D3 makes DOM calls not supported in jsdom — stub it out
jest.mock('d3', () => ({
  select: () => ({
    selectAll: () => ({ remove: jest.fn() }),
    attr: jest.fn().mockReturnThis(),
    append: jest.fn().mockReturnThis(),
    call: jest.fn().mockReturnThis(),
    on: jest.fn().mockReturnThis(),
  }),
  zoom: () => ({
    scaleExtent: jest.fn().mockReturnThis(),
    on: jest.fn().mockReturnThis(),
  }),
}));

describe('ReasoningEngineOverlay', () => {
  const defaultProps = {
    state: mockReasoningState,
    open: true,
    onClose: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('does not render when open is false', () => {
    render(<ReasoningEngineOverlay {...defaultProps} open={false} />);
    expect(screen.queryByTestId('reasoning-engine-overlay')).not.toBeInTheDocument();
  });

  it('renders overlay when open is true', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    expect(screen.getByTestId('reasoning-engine-overlay')).toBeInTheDocument();
  });

  it('shows engagement name in header', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    expect(screen.getByText('NIS 2 Article 21 Audit 2024')).toBeInTheDocument();
  });

  it('displays the overall confidence score', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    // Score appears in header badge
    expect(screen.getAllByText('72').length).toBeGreaterThanOrEqual(1);
  });

  it('renders all six tabs', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    expect(screen.getByTestId('tab-confidence')).toBeInTheDocument();
    expect(screen.getByTestId('tab-heatmap')).toBeInTheDocument();
    expect(screen.getByTestId('tab-quality')).toBeInTheDocument();
    expect(screen.getByTestId('tab-sufficiency')).toBeInTheDocument();
    expect(screen.getByTestId('tab-dag')).toBeInTheDocument();
    expect(screen.getByTestId('tab-whatif')).toBeInTheDocument();
  });

  it('shows confidence factor breakdown on default tab', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    expect(screen.getByTestId('confidence-factor-breakdown')).toBeInTheDocument();
  });

  it('switches to quality gate tab', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    fireEvent.click(screen.getByTestId('tab-quality'));
    expect(screen.getByTestId('quality-gate-status')).toBeInTheDocument();
  });

  it('switches to evidence sufficiency tab', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    fireEvent.click(screen.getByTestId('tab-sufficiency'));
    expect(screen.getByTestId('evidence-sufficiency')).toBeInTheDocument();
  });

  it('switches to What If tab', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    fireEvent.click(screen.getByTestId('tab-whatif'));
    expect(screen.getByTestId('what-if-panel')).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    fireEvent.click(screen.getByTestId('overlay-close'));
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose on Escape key', () => {
    render(<ReasoningEngineOverlay {...defaultProps} />);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });
});

describe('ReasoningEngineTrigger', () => {
  it('renders trigger button with confidence score', () => {
    render(<ReasoningEngineTrigger state={mockReasoningState} />);
    expect(screen.getByTestId('reasoning-engine-trigger')).toBeInTheDocument();
    expect(screen.getByText('72')).toBeInTheDocument();
  });

  it('opens overlay on click', () => {
    render(<ReasoningEngineTrigger state={mockReasoningState} />);
    fireEvent.click(screen.getByTestId('reasoning-engine-trigger'));
    expect(screen.getByTestId('reasoning-engine-overlay')).toBeInTheDocument();
  });
});
