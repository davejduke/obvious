-- AIAUDITOR Approval Workflows
-- Migration: 006_approval_workflows.sql
-- Three workflow types: audit_plan | finding_signoff | report_release
-- State machine: draft -> pending_approval -> approved -> locked
--                         (reject returns to draft)

BEGIN;

-- ============================================================
-- APPROVAL WORKFLOWS
-- One row per approval workflow instance.
-- ============================================================
CREATE TABLE IF NOT EXISTS approval_workflows (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    workflow_type   TEXT NOT NULL
                    CHECK (workflow_type IN ('audit_plan', 'finding_signoff', 'report_release')),
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'pending_approval', 'approved', 'locked', 'rejected')),

    -- Polymorphic reference: which resource this workflow governs.
    resource_type   TEXT NOT NULL,   -- 'engagement' | 'finding' | 'report'
    resource_id     UUID NOT NULL,
    engagement_id   UUID REFERENCES engagements(id) ON DELETE CASCADE,

    -- Who submitted the workflow for review.
    submitted_by_id     UUID REFERENCES users(id),
    submitted_by_email  TEXT,
    submitted_at        TIMESTAMPTZ,

    -- Final approver details (set on approval/rejection).
    decided_by_id       UUID REFERENCES users(id),
    decided_by_email    TEXT,
    decided_at          TIMESTAMPTZ,

    -- Human-readable comment from the most recent action.
    latest_comment  TEXT,

    -- Rejection reason (populated when status = 'rejected').
    rejection_reason TEXT,

    -- Metadata for extra context.
    metadata        JSONB NOT NULL DEFAULT '{}',

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_approval_workflows_org      ON approval_workflows(org_id);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_resource ON approval_workflows(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_eng      ON approval_workflows(engagement_id);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_status   ON approval_workflows(status);

ALTER TABLE approval_workflows ENABLE ROW LEVEL SECURITY;

CREATE POLICY org_isolation ON approval_workflows
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY service_bypass ON approval_workflows TO aiauditor_service
    USING (true) WITH CHECK (true);

CREATE TRIGGER update_approval_workflows_updated_at
    BEFORE UPDATE ON approval_workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- APPROVAL HISTORY
-- Immutable event log: one row per action on a workflow.
-- ============================================================
CREATE TABLE IF NOT EXISTS approval_history (
    id              BIGSERIAL PRIMARY KEY,
    workflow_id     UUID NOT NULL REFERENCES approval_workflows(id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Actor
    actor_id        UUID REFERENCES users(id),
    actor_email     TEXT NOT NULL,
    actor_role      TEXT NOT NULL,  -- 'engagement_lead' | 'chief_audit_executive' | 'reviewer'

    -- What happened
    action          TEXT NOT NULL
                    CHECK (action IN ('submitted', 'approved', 'rejected', 'locked', 'returned_to_draft')),
    from_status     TEXT NOT NULL,
    to_status       TEXT NOT NULL,
    comment         TEXT,

    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_approval_history_workflow ON approval_history(workflow_id);
CREATE INDEX IF NOT EXISTS idx_approval_history_org      ON approval_history(org_id);
CREATE INDEX IF NOT EXISTS idx_approval_history_actor    ON approval_history(actor_id);

ALTER TABLE approval_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY org_isolation ON approval_history
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY service_bypass ON approval_history TO aiauditor_service
    USING (true) WITH CHECK (true);

COMMIT;
