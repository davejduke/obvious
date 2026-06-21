import { NextRequest, NextResponse } from 'next/server';
import { mockReasoningState } from '@/lib/mock-data';
import type { ReasoningEngineState } from '@/lib/reasoning-types';

/**
 * GET /api/reasoning/:engagementId/state
 *
 * Returns the full deterministic reasoning engine state for an engagement.
 * All confidence factors are computed; no LLM is queried.
 */
export async function GET(
  _req: NextRequest,
  context: { params: Promise<{ engagementId: string }> },
): Promise<NextResponse<ReasoningEngineState | { error: string }>> {
  const { engagementId } = await context.params;

  // In the real implementation this would query the reasoning engine service.
  // For the current phase, return mock state keyed by engagement ID.
  if (engagementId !== mockReasoningState.engagement_id) {
    return NextResponse.json({ error: 'Engagement not found' }, { status: 404 });
  }

  return NextResponse.json(mockReasoningState, { status: 200 });
}
