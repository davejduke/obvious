-- AIAUDITOR Audit Planning Module
-- Migration: 007_audit_planning.sql
-- Strategic (3-year) plans, annual plans, assurance maps, resource calendars

BEGIN;

-- ============================================================
-- STRATEGIC PLANS — 3-year rolling risk-based plan
-- ============================================================
CREATE TABLE IF NOT EXISTS strategic_plans (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    start_year      INT  NOT NULL,
    end_year        INT  NOT NULL,
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'approved', 'active', 'archived')),
    version         INT  NOT NULL DEFAULT 1,
    entities        JSONB NOT NULL DEFAULT '[]',
    approval_id     UUID REFERENCES approval_workflows(id),
    approved_at     TIMESTAMPTZ,
    approved_by     TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategic_plans_org ON strategic_plans(org_id);
CREATE INDEX IF NOT EXISTS idx_strategic_plans_status ON strategic_plans(org_id, status);

-- ============================================================
-- ANNUAL PLANS — schedule of engagements for a given year
-- ============================================================
CREATE TABLE IF NOT EXISTS annual_plans (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    strategic_plan_id   UUID REFERENCES strategic_plans(id),
    year                INT  NOT NULL,
    name                TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'approved', 'active', 'archived')),
    version             INT  NOT NULL DEFAULT 1,
    engagements         JSONB NOT NULL DEFAULT '[]',
    approval_id         UUID REFERENCES approval_workflows(id),
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_annual_plans_org ON annual_plans(org_id);
CREATE INDEX IF NOT EXISTS idx_annual_plans_year ON annual_plans(org_id, year);

-- ============================================================
-- ASSURANCE MAPS — business unit x control domain coverage matrix
-- ============================================================
CREATE TABLE IF NOT EXISTS assurance_maps (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id            UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    annual_plan_id    UUID REFERENCES annual_plans(id),
    year              INT  NOT NULL,
    name              TEXT NOT NULL,
    business_units    JSONB NOT NULL DEFAULT '[]',
    control_domains   JSONB NOT NULL DEFAULT '[]',
    matrix            JSONB NOT NULL DEFAULT '[]',
    metadata          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assurance_maps_org ON assurance_maps(org_id);

-- ============================================================
-- RESOURCE CALENDARS — auditor availability and workload
-- ============================================================
CREATE TABLE IF NOT EXISTS resource_calendars (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id            UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    annual_plan_id    UUID REFERENCES annual_plans(id),
    year              INT  NOT NULL,
    name              TEXT NOT NULL,
    auditors          JSONB NOT NULL DEFAULT '[]',
    metadata          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resource_calendars_org ON resource_calendars(org_id);

COMMIT;
