-- AIAUDITOR PostgreSQL DDL
-- Migration: 001_initial_schema.sql
-- Creates all core tables with RLS policies

BEGIN;

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ORGANIZATIONS
-- ============================================================
CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    tier            TEXT NOT NULL DEFAULT 'standard' CHECK (tier IN ('standard', 'professional', 'enterprise')),
    industry        TEXT,
    country_code    CHAR(2),
    settings        JSONB NOT NULL DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- USERS
-- ============================================================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email           TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    password_hash   TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, email)
);

-- ============================================================
-- ROLES
-- ============================================================
CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    description     TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, slug)
);

-- ============================================================
-- USER ROLES
-- ============================================================
CREATE TABLE user_roles (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_by  UUID REFERENCES users(id),
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ,
    UNIQUE (user_id, role_id)
);

-- ============================================================
-- PERMISSIONS
-- ============================================================
CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource    TEXT NOT NULL,
    action      TEXT NOT NULL,
    conditions  JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (role_id, resource, action)
);

-- ============================================================
-- CONTROL FRAMEWORKS
-- ============================================================
CREATE TABLE control_frameworks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    version         TEXT NOT NULL,
    authority       TEXT NOT NULL,
    description     TEXT,
    is_published    BOOLEAN NOT NULL DEFAULT false,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, name, version)
);

-- ============================================================
-- CONTROLS
-- ============================================================
CREATE TABLE controls (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    framework_id        UUID NOT NULL REFERENCES control_frameworks(id) ON DELETE CASCADE,
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    parent_id           UUID REFERENCES controls(id),
    control_id          TEXT NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,
    objective           TEXT,
    category            TEXT,
    domain              TEXT,
    article_ref         TEXT,
    implementation_notes TEXT,
    test_procedures     JSONB NOT NULL DEFAULT '[]',
    evidence_requirements JSONB NOT NULL DEFAULT '[]',
    risk_weight         NUMERIC(3,2) NOT NULL DEFAULT 1.00 CHECK (risk_weight BETWEEN 0 AND 5),
    maturity_levels     JSONB NOT NULL DEFAULT '{}',
    tags                TEXT[] NOT NULL DEFAULT '{}',
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (framework_id, control_id)
);

-- ============================================================
-- FRAMEWORK MAPPINGS
-- ============================================================
CREATE TABLE framework_mappings (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_framework_id UUID NOT NULL REFERENCES control_frameworks(id),
    target_framework_id UUID NOT NULL REFERENCES control_frameworks(id),
    source_control_id   UUID NOT NULL REFERENCES controls(id),
    target_control_id   UUID NOT NULL REFERENCES controls(id),
    mapping_type        TEXT NOT NULL DEFAULT 'equivalent' CHECK (mapping_type IN ('equivalent', 'subset', 'superset', 'partial', 'none')),
    confidence          NUMERIC(3,2) NOT NULL DEFAULT 1.00 CHECK (confidence BETWEEN 0 AND 1),
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_control_id, target_control_id)
);

-- ============================================================
-- ENGAGEMENTS
-- ============================================================
CREATE TABLE engagements (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    framework_id        UUID NOT NULL REFERENCES control_frameworks(id),
    name                TEXT NOT NULL,
    description         TEXT,
    status              TEXT NOT NULL DEFAULT 'planning' CHECK (status IN (
                            'planning', 'fieldwork', 'review', 'reporting', 'completed', 'cancelled'
                        )),
    scope_json          JSONB NOT NULL DEFAULT '{}',
    lead_auditor_id     UUID REFERENCES users(id),
    target_start_date   DATE,
    target_end_date     DATE,
    actual_start_date   DATE,
    actual_end_date     DATE,
    overall_score       NUMERIC(5,2),
    risk_rating         TEXT CHECK (risk_rating IN ('critical', 'high', 'medium', 'low', 'informational')),
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- ENGAGEMENT PHASES
-- ============================================================
CREATE TABLE engagement_phases (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    engagement_id   UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES organizations(id),
    phase_name      TEXT NOT NULL,
    phase_order     INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'skipped')),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (engagement_id, phase_order)
);

-- ============================================================
-- EVIDENCE
-- ============================================================
CREATE TABLE evidence (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    engagement_id       UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    control_id          UUID NOT NULL REFERENCES controls(id),
    uploaded_by_id      UUID REFERENCES users(id),
    title               TEXT NOT NULL,
    description         TEXT,
    source_type         TEXT NOT NULL CHECK (source_type IN ('manual_upload', 'api_integration', 'automated_scan', 'screenshot', 'log_export', 'configuration_export')),
    source_ref          TEXT,
    file_path           TEXT,
    file_hash           TEXT,
    file_size_bytes     BIGINT,
    mime_type           TEXT,
    content_text        TEXT,
    collection_date     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_start        DATE,
    period_end          DATE,
    status              TEXT NOT NULL DEFAULT 'pending_review' CHECK (status IN ('pending_review', 'accepted', 'rejected', 'archived')),
    is_sufficient       BOOLEAN,
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- EVIDENCE CLASSIFICATIONS
-- ============================================================
CREATE TABLE evidence_classifications (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evidence_id     UUID NOT NULL REFERENCES evidence(id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES organizations(id),
    classifier      TEXT NOT NULL,
    classification  TEXT NOT NULL,
    confidence      NUMERIC(5,4) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    reasoning       TEXT,
    model_version   TEXT,
    classified_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- EVIDENCE QUALITY SCORES
-- ============================================================
CREATE TABLE evidence_quality_scores (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evidence_id         UUID NOT NULL UNIQUE REFERENCES evidence(id) ON DELETE CASCADE,
    org_id              UUID NOT NULL REFERENCES organizations(id),
    completeness_score  NUMERIC(5,4) NOT NULL CHECK (completeness_score BETWEEN 0 AND 1),
    accuracy_score      NUMERIC(5,4) NOT NULL CHECK (accuracy_score BETWEEN 0 AND 1),
    timeliness_score    NUMERIC(5,4) NOT NULL CHECK (timeliness_score BETWEEN 0 AND 1),
    relevance_score     NUMERIC(5,4) NOT NULL CHECK (relevance_score BETWEEN 0 AND 1),
    aggregate_score     NUMERIC(5,4) NOT NULL CHECK (aggregate_score BETWEEN 0 AND 1),
    deficiencies        JSONB NOT NULL DEFAULT '[]',
    scored_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- FINDINGS
-- ============================================================
CREATE TABLE findings (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    engagement_id       UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    control_id          UUID NOT NULL REFERENCES controls(id),
    assigned_to_id      UUID REFERENCES users(id),
    finding_ref         TEXT NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT NOT NULL,
    root_cause          TEXT,
    impact              TEXT,
    likelihood          TEXT CHECK (likelihood IN ('certain', 'likely', 'possible', 'unlikely', 'rare')),
    severity            TEXT NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'informational')),
    status              TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_remediation', 'remediated', 'accepted_risk', 'false_positive', 'closed')),
    due_date            DATE,
    closed_at           TIMESTAMPTZ,
    evidence_ids        UUID[] NOT NULL DEFAULT '{}',
    tags                TEXT[] NOT NULL DEFAULT '{}',
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (engagement_id, finding_ref)
);

-- ============================================================
-- RECOMMENDATIONS
-- ============================================================
CREATE TABLE recommendations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    finding_id      UUID NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES organizations(id),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    priority        TEXT NOT NULL CHECK (priority IN ('immediate', 'short_term', 'medium_term', 'long_term')),
    effort          TEXT CHECK (effort IN ('low', 'medium', 'high')),
    status          TEXT NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'accepted', 'rejected', 'in_progress', 'implemented')),
    implementation_notes TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- AUDIT EVENTS (append-only with hash chain)
-- ============================================================
CREATE TABLE audit_events (
    id              BIGSERIAL PRIMARY KEY,
    org_id          UUID NOT NULL REFERENCES organizations(id),
    event_id        UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    actor_id        UUID REFERENCES users(id),
    actor_email     TEXT,
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     UUID,
    changes         JSONB,
    context         JSONB NOT NULL DEFAULT '{}',
    ip_address      INET,
    user_agent      TEXT,
    previous_hash   TEXT NOT NULL DEFAULT '',
    event_hash      TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent updates and deletes on audit_events
CREATE RULE audit_events_no_update AS ON UPDATE TO audit_events DO INSTEAD NOTHING;
CREATE RULE audit_events_no_delete AS ON DELETE TO audit_events DO INSTEAD NOTHING;

-- ============================================================
-- INDEXES
-- ============================================================
CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_roles_org_id ON roles(org_id);
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_controls_framework_id ON controls(framework_id);
CREATE INDEX idx_controls_org_id ON controls(org_id);
CREATE INDEX idx_controls_domain ON controls(domain);
CREATE INDEX idx_controls_article_ref ON controls(article_ref);
CREATE INDEX idx_engagements_org_id ON engagements(org_id);
CREATE INDEX idx_engagements_status ON engagements(status);
CREATE INDEX idx_evidence_engagement_id ON evidence(engagement_id);
CREATE INDEX idx_evidence_control_id ON evidence(control_id);
CREATE INDEX idx_evidence_org_id ON evidence(org_id);
CREATE INDEX idx_findings_engagement_id ON findings(engagement_id);
CREATE INDEX idx_findings_control_id ON findings(control_id);
CREATE INDEX idx_findings_severity ON findings(severity);
CREATE INDEX idx_findings_status ON findings(status);
CREATE INDEX idx_audit_events_org_id ON audit_events(org_id);
CREATE INDEX idx_audit_events_actor_id ON audit_events(actor_id);
CREATE INDEX idx_audit_events_resource ON audit_events(resource_type, resource_id);
CREATE INDEX idx_audit_events_occurred_at ON audit_events(occurred_at DESC);

-- ============================================================
-- ROW LEVEL SECURITY
-- ============================================================
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE control_frameworks ENABLE ROW LEVEL SECURITY;
ALTER TABLE controls ENABLE ROW LEVEL SECURITY;
ALTER TABLE framework_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE engagements ENABLE ROW LEVEL SECURITY;
ALTER TABLE engagement_phases ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_classifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_quality_scores ENABLE ROW LEVEL SECURITY;
ALTER TABLE findings ENABLE ROW LEVEL SECURITY;
ALTER TABLE recommendations ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;

-- RLS policies: tenant isolation via current_setting
CREATE POLICY org_isolation ON organizations
    USING (id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON users
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON roles
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON user_roles
    USING (user_id IN (
        SELECT id FROM users
        WHERE org_id = current_setting('app.current_org_id', true)::UUID
    ));

CREATE POLICY org_isolation ON permissions
    USING (role_id IN (
        SELECT id FROM roles
        WHERE org_id = current_setting('app.current_org_id', true)::UUID
    ));

CREATE POLICY org_isolation ON control_frameworks
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON controls
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON framework_mappings
    USING (source_framework_id IN (
        SELECT id FROM control_frameworks
        WHERE org_id = current_setting('app.current_org_id', true)::UUID
    ));

CREATE POLICY org_isolation ON engagements
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON engagement_phases
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON evidence
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON evidence_classifications
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON evidence_quality_scores
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON findings
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON recommendations
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY org_isolation ON audit_events
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

-- Service role bypass (for migrations and backend services)
CREATE POLICY service_bypass ON organizations TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON users TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON roles TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON user_roles TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON permissions TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON control_frameworks TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON controls TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON framework_mappings TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON engagements TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON engagement_phases TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON evidence TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON evidence_classifications TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON evidence_quality_scores TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON findings TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON recommendations TO aiauditor_service USING (true) WITH CHECK (true);
CREATE POLICY service_bypass ON audit_events TO aiauditor_service USING (true) WITH CHECK (true);

-- ============================================================
-- UPDATED_AT TRIGGER
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_control_frameworks_updated_at BEFORE UPDATE ON control_frameworks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_controls_updated_at BEFORE UPDATE ON controls
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_engagements_updated_at BEFORE UPDATE ON engagements
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_evidence_updated_at BEFORE UPDATE ON evidence
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_findings_updated_at BEFORE UPDATE ON findings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_recommendations_updated_at BEFORE UPDATE ON recommendations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMIT;

