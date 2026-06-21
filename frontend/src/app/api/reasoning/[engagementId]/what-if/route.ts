import { NextRequest, NextResponse } from 'next/server';
import { simulateWhatIf } from '@/lib/mock-data';
import type { WhatIfQuery, WhatIfResult } from '@/lib/reasoning-types';

/**
 * POST /api/reasoning/:engagementId/what-if
 *
 * Body: WhatIfQuery
 * Returns: WhatIfResult (deterministic simulation — no LLM)
 *
 * Simulates the effect of adding or removing evidence items on the
 * overall confidence score and quality gate status for a given control.
 * All simulation logic is deterministic.
 */
export async function POST(
  req: NextRequest,
  context: { params: Promise<{ engagementId: string }> },
): Promise<NextResponse<WhatIfResult | { error: string }>> {
  const { engagementId } = await context.params;

  let body: Partial<WhatIfQuery>;
  try {
    body = (await req.json()) as Partial<WhatIfQuery>;
  } catch {
    return NextResponse.json({ error: 'Invalid JSON body' }, { status: 400 });
  }

  // Validate required fields
  if (
    !body.control_id ||
    !body.action ||
    body.evidence_quality_score === undefined ||
    body.count === undefined
  ) {
    return NextResponse.json(
      { error: 'Missing required fields: control_id, action, evidence_quality_score, count' },
      { status: 400 },
    );
  }

  if (!['add_evidence', 'remove_evidence'].includes(body.action)) {
    return NextResponse.json(
      { error: 'action must be "add_evidence" or "remove_evidence"' },
      { status: 400 },
    );
  }

  if (body.evidence_quality_score < 0 || body.evidence_quality_score > 100) {
    return NextResponse.json(
      { error: 'evidence_quality_score must be between 0 and 100' },
      { status: 400 },
    );
  }

  if (body.count < 1) {
    return NextResponse.json({ error: 'count must be >= 1' }, { status: 400 });
  }

  const query: WhatIfQuery = {
    engagement_id: engagementId,
    control_id: body.control_id,
    action: body.action as WhatIfQuery['action'],
    evidence_quality_score: body.evidence_quality_score,
    count: body.count,
  };

  const result = simulateWhatIf(query);
  return NextResponse.json(result, { status: 200 });
}
