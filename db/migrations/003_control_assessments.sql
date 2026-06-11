-- AIAUDITOR Control Assessments Table
-- Migration: 003_control_assessments.sql
-- Adds control_assessments table for tracking assessment lifecycle per control/engagement

BEGIN;

-- ============================================================
-- CONTROL ASSESSMENTS
-- Tracks the assessment status of each control within an engagement.
-- Status state machine: not_started -> in_progress -> assessed <-> remediation
-- ============================================================
CREATE TABLE IF NOT EXISTS control_assessments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    engagement_id       UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    control_id          UUID NOT NULL REFERENCES controls(id) ON DELETE CASCADE,
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'not_started'
                        CHECK (status IN ('not_started', 'in_progress', 'assessed', 'remediation')),
    assessed_by_id      UUID REFERENCES users(id),
    score               NUMERIC(5,4) CHECK (score BETWEEN 0 AND 1),
    notes               TEXT,
    evidence_ids        UUID[] NOT NULL DEFAULT '{}',
    transitioned_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (engagement_id, control_id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_control_assessments_engagement ON control_assessments(engagement_id);
CREATE INDEX IF NOT EXISTS idx_control_assessments_control ON control_assessments(control_id);
CREATE INDEX IF NOT EXISTS idx_control_assessments_org ON control_assessments(org_id);
CREATE INDEX IF NOT EXISTS idx_control_assessments_status ON control_assessments(status);

-- RLS
ALTER TABLE control_assessments ENABLE ROW LEVEL SECURITY;

CREATE POLICY org_isolation ON control_assessments
    USING (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE POLICY service_bypass ON control_assessments TO aiauditor_service
    USING (true) WITH CHECK (true);

-- Updated_at trigger
CREATE TRIGGER update_control_assessments_updated_at
    BEFORE UPDATE ON control_assessments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- FRAMEWORK MAPPINGS SEED DATA
-- NIS 2 <-> NIST CSF 2.0 <-> ISO 27001:2022 <-> CIS Controls v8
--
-- Note: Target controls reference system org (00000000-0000-0000-0000-000000000001)
-- and the NIS 2 framework (00000000-0000-0000-0000-000000000010).
-- Cross-framework mappings point to controls seeded in peer services or added here.
-- This migration seeds the mapping schema; actual cross-framework control records
-- are loaded by the framework-registry migration in a future phase.
-- ============================================================

-- Create NIST CSF 2.0, ISO 27001:2022, and CIS Controls v8 frameworks
-- (published: false until full control libraries are loaded)
INSERT INTO control_frameworks (id, org_id, name, version, authority, description, is_published, metadata)
VALUES
(
    '00000000-0000-0000-0000-000000000020',
    '00000000-0000-0000-0000-000000000001',
    'NIST Cybersecurity Framework',
    '2.0',
    'NIST',
    'NIST CSF 2.0 - Framework for Improving Critical Infrastructure Cybersecurity',
    false,
    '{"jurisdiction": "US", "url": "https://www.nist.gov/cyberframework"}'
),
(
    '00000000-0000-0000-0000-000000000030',
    '00000000-0000-0000-0000-000000000001',
    'ISO/IEC 27001',
    '2022',
    'ISO/IEC',
    'ISO/IEC 27001:2022 - Information security, cybersecurity and privacy protection',
    false,
    '{"jurisdiction": "International", "url": "https://www.iso.org/standard/27001"}'
),
(
    '00000000-0000-0000-0000-000000000040',
    '00000000-0000-0000-0000-000000000001',
    'CIS Controls',
    'v8',
    'Center for Internet Security',
    'CIS Critical Security Controls Version 8',
    false,
    '{"jurisdiction": "International", "url": "https://www.cisecurity.org/controls"}'
)
ON CONFLICT DO NOTHING;

-- Representative cross-framework control entries for mapping schema validation
-- NIST CSF 2.0 controls (subset for NIS 2 mapping)
INSERT INTO controls (id, framework_id, org_id, control_id, title, description, category, domain, article_ref, risk_weight, tags)
VALUES
(
    '20000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000020',
    '00000000-0000-0000-0000-000000000001',
    'NIST-GV.RM-01',
    'Risk management objectives',
    'Risk management objectives are established and agreed to by organizational stakeholders.',
    'Govern',
    'Risk Management',
    'GV.RM-01',
    1.50,
    ARRAY['nist-csf', 'risk-management', 'governance']
),
(
    '20000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000020',
    '00000000-0000-0000-0000-000000000001',
    'NIST-RS.MA-01',
    'Incident response plan execution',
    'The incident response plan is executed in coordination with relevant third parties.',
    'Respond',
    'Incident Management',
    'RS.MA-01',
    2.00,
    ARRAY['nist-csf', 'incident-response']
),
(
    '20000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000020',
    '00000000-0000-0000-0000-000000000001',
    'NIST-PR.AA-01',
    'Identities and credentials managed',
    'Identities and credentials for authorized users, services, and hardware are managed by the organization.',
    'Protect',
    'Identity Management',
    'PR.AA-01',
    2.00,
    ARRAY['nist-csf', 'identity', 'access-control']
)
ON CONFLICT (framework_id, control_id) DO NOTHING;

-- ISO 27001:2022 controls (subset)
INSERT INTO controls (id, framework_id, org_id, control_id, title, description, category, domain, article_ref, risk_weight, tags)
VALUES
(
    '30000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000030',
    '00000000-0000-0000-0000-000000000001',
    'ISO-8.2',
    'Information security risk assessment',
    'The organization shall perform information security risk assessments at planned intervals.',
    'Operations',
    'Risk Management',
    'Clause 8.2',
    2.00,
    ARRAY['iso27001', 'risk-assessment']
),
(
    '30000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000030',
    '00000000-0000-0000-0000-000000000001',
    'ISO-A.5.24',
    'Information security incident management planning and preparation',
    'The organization shall plan and prepare for managing information security incidents by defining processes.',
    'Annex A',
    'Incident Management',
    'A.5.24',
    2.00,
    ARRAY['iso27001', 'incident-response']
),
(
    '30000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000030',
    '00000000-0000-0000-0000-000000000001',
    'ISO-A.8.3',
    'Information access restriction',
    'Access to information and other associated assets shall be restricted in accordance with the access control policy.',
    'Annex A',
    'Access Control',
    'A.8.3',
    2.00,
    ARRAY['iso27001', 'access-control']
)
ON CONFLICT (framework_id, control_id) DO NOTHING;

-- CIS Controls v8 (subset)
INSERT INTO controls (id, framework_id, org_id, control_id, title, description, category, domain, article_ref, risk_weight, tags)
VALUES
(
    '40000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000040',
    '00000000-0000-0000-0000-000000000001',
    'CIS-18.1',
    'Establish a security awareness program',
    'Establish and maintain a security awareness program to influence behavior among the workforce.',
    'Implementation Group 1',
    'Security Awareness',
    'Control 18',
    1.50,
    ARRAY['cis-v8', 'awareness', 'training']
),
(
    '40000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000040',
    '00000000-0000-0000-0000-000000000001',
    'CIS-6.3',
    'Require MFA for externally-exposed applications',
    'Require all externally-exposed enterprise or third-party applications to enforce MFA.',
    'Implementation Group 1',
    'Access Control',
    'Control 6',
    2.50,
    ARRAY['cis-v8', 'mfa', 'authentication']
),
(
    '40000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000040',
    '00000000-0000-0000-0000-000000000001',
    'CIS-11.2',
    'Perform automated backups',
    'Perform automated backups of in-scope enterprise assets and store backup data in a secure location.',
    'Implementation Group 1',
    'Data Recovery',
    'Control 11',
    2.00,
    ARRAY['cis-v8', 'backup', 'business-continuity']
)
ON CONFLICT (framework_id, control_id) DO NOTHING;

-- NIS 2 -> NIST CSF 2.0 mappings
INSERT INTO framework_mappings
    (id, source_framework_id, target_framework_id, source_control_id, target_control_id, mapping_type, confidence, notes)
VALUES
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000020',
    '10000000-0000-0000-0000-000000000001',  -- NIS2-21a
    '20000000-0000-0000-0000-000000000001',  -- NIST-GV.RM-01
    'equivalent',
    0.90,
    'Both require systematic risk management objectives and governance'
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000020',
    '10000000-0000-0000-0000-000000000003',  -- NIS2-21b
    '20000000-0000-0000-0000-000000000002',  -- NIST-RS.MA-01
    'equivalent',
    0.85,
    'Both address incident response plan execution and coordination'
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000020',
    '10000000-0000-0000-0000-000000000013',  -- NIS2-21j
    '20000000-0000-0000-0000-000000000003',  -- NIST-PR.AA-01
    'partial',
    0.75,
    'NIS2-21j is broader (secure comms); NIST-PR.AA-01 focuses on identity management'
)
ON CONFLICT (source_control_id, target_control_id) DO NOTHING;

-- NIS 2 -> ISO 27001:2022 mappings
INSERT INTO framework_mappings
    (id, source_framework_id, target_framework_id, source_control_id, target_control_id, mapping_type, confidence, notes)
VALUES
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000030',
    '10000000-0000-0000-0000-000000000001',  -- NIS2-21a
    '30000000-0000-0000-0000-000000000001',  -- ISO-8.2
    'equivalent',
    0.92,
    'Both require formal risk assessment processes at planned intervals'
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000030',
    '10000000-0000-0000-0000-000000000003',  -- NIS2-21b
    '30000000-0000-0000-0000-000000000002',  -- ISO-A.5.24
    'equivalent',
    0.88,
    'Both require incident management planning and preparation'
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000030',
    '10000000-0000-0000-0000-000000000011',  -- NIS2-21i
    '30000000-0000-0000-0000-000000000003',  -- ISO-A.8.3
    'subset',
    0.80,
    'NIS2-21i covers HR security + access control; ISO-A.8.3 is focused on access restriction'
)
ON CONFLICT (source_control_id, target_control_id) DO NOTHING;

-- NIS 2 -> CIS Controls v8 mappings
INSERT INTO framework_mappings
    (id, source_framework_id, target_framework_id, source_control_id, target_control_id, mapping_type, confidence, notes)
VALUES
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000040',
    '10000000-0000-0000-0000-000000000009',  -- NIS2-21g
    '40000000-0000-0000-0000-000000000001',  -- CIS-18.1
    'equivalent',
    0.88,
    'Both require security awareness programs'
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000040',
    '10000000-0000-0000-0000-000000000013',  -- NIS2-21j
    '40000000-0000-0000-0000-000000000002',  -- CIS-6.3
    'equivalent',
    0.92,
    'Both require MFA for system access'
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000040',
    '10000000-0000-0000-0000-000000000005',  -- NIS2-21c
    '40000000-0000-0000-0000-000000000003',  -- CIS-11.2
    'partial',
    0.78,
    'NIS2-21c covers full BCP/DRP; CIS-11.2 focuses specifically on automated backup'
)
ON CONFLICT (source_control_id, target_control_id) DO NOTHING;

COMMIT;

