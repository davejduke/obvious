-- AIAUDITOR PostgreSQL DDL
-- Migration: 005_webhooks.sql
-- Webhook subscription management and delivery tracking
-- Supersedes placeholder webhook_endpoints schema from release branch.

BEGIN;

-- Drop the placeholder webhook_endpoints table if it was applied from an
-- earlier draft migration; this service uses webhook_subscriptions.
DROP TABLE IF EXISTS webhook_deliveries CASCADE;
DROP TABLE IF EXISTS webhook_endpoints CASCADE;

-- ============================================================
-- WEBHOOK SUBSCRIPTIONS
-- ============================================================
CREATE TABLE webhook_subscriptions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    secret_hash     TEXT NOT NULL,  -- SHA-256 of secret for breach-safe storage
    secret_value    TEXT NOT NULL,  -- actual secret used for HMAC signing
    event_types     TEXT[] NOT NULL DEFAULT '{}',
    description     TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- WEBHOOK DELIVERIES (delivery log + dead-letter)
-- ============================================================
CREATE TABLE webhook_deliveries (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscription_id     UUID NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_type          TEXT NOT NULL,
    payload             JSONB NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'delivered', 'failed')),
    attempt_count       INT NOT NULL DEFAULT 0,
    last_attempt_at     TIMESTAMPTZ,
    next_retry_at       TIMESTAMPTZ,
    response_status     INT,        -- HTTP status code from last attempt
    error_message       TEXT,       -- error detail for dead-letter inspection
    delivered_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- INDEXES
-- ============================================================
CREATE INDEX idx_webhook_subscriptions_org_id ON webhook_subscriptions(org_id);
CREATE INDEX idx_webhook_subscriptions_active ON webhook_subscriptions(org_id, is_active);
CREATE INDEX idx_webhook_deliveries_subscription_id ON webhook_deliveries(subscription_id);
CREATE INDEX idx_webhook_deliveries_org_id ON webhook_deliveries(org_id);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(status, next_retry_at)
    WHERE status = 'pending';
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);

-- ============================================================
-- ROW LEVEL SECURITY
-- ============================================================
ALTER TABLE webhook_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;

CREATE POLICY org_isolation ON webhook_subscriptions
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON webhook_deliveries
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY service_bypass ON webhook_subscriptions TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON webhook_deliveries TO aiauditor_service USING (true) WITH CHECK (true);

-- ============================================================
-- UPDATED_AT TRIGGER
-- ============================================================
CREATE TRIGGER update_webhook_subscriptions_updated_at
    BEFORE UPDATE ON webhook_subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMIT;
