-- AIAUDITOR NIS 2 Article 21 Seed Data
-- Migration: 002_seed_nis2.sql
-- Seeds NIS 2 Directive Article 21(a-j) control framework

BEGIN;

-- Create a system organization for seed data
INSERT INTO organizations (id, name, slug, tier, industry, country_code, settings)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'AIAUDITOR System',
    'aiauditor-system',
    'enterprise',
    'Technology',
    'EU',
    '{"is_system": true}'
) ON CONFLICT (slug) DO NOTHING;

-- Create NIS 2 framework
INSERT INTO control_frameworks (
    id,
    org_id,
    name,
    version,
    authority,
    description,
    is_published,
    metadata
) VALUES (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'NIS 2 Directive',
    '2022/2555',
    'European Parliament and Council',
    'Directive on measures for a high common level of cybersecurity across the Union, Article 21 Security Requirements',
    true,
    '{"jurisdiction": "EU", "effective_date": "2024-10-18", "url": "https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX%3A32022L2555"}'
) ON CONFLICT DO NOTHING;

-- ============================================================
-- NIS 2 Article 21(a) — Risk analysis and information system security policies
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21a',
    'Risk analysis and information system security policies',
    'Entities shall implement policies on risk analysis and information system security.',
    'Establish a systematic approach to identifying, assessing, and managing cybersecurity risks.',
    'Risk Management',
    'Governance',
    'Art. 21(2)(a)',
    '[{"id": "T1", "description": "Review the risk management policy document", "expected_evidence": "Risk management policy signed by management"}, {"id": "T2", "description": "Interview risk owners to verify understanding", "expected_evidence": "Interview notes or attestation"}]',
    '[{"type": "policy_document", "description": "Information security risk policy", "required": true}, {"type": "risk_register", "description": "Current risk register with assessments", "required": true}]',
    2.00,
    ARRAY['risk-management', 'governance', 'nis2', 'article-21a']
),
(
    '10000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    'NIS2-21a.1',
    'Risk assessment methodology',
    'Define and document a repeatable methodology for identifying and assessing cybersecurity risks.',
    'Ensure consistent and documented risk assessments.',
    'Risk Management',
    'Governance',
    'Art. 21(2)(a)',
    '[{"id": "T1", "description": "Review risk assessment methodology documentation", "expected_evidence": "Methodology document with scoring criteria"}]',
    '[{"type": "methodology_document", "description": "Risk assessment methodology", "required": true}]',
    1.50,
    ARRAY['risk-assessment', 'methodology', 'nis2']
);

-- ============================================================
-- NIS 2 Article 21(b) — Incident handling
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21b',
    'Incident handling',
    'Entities shall implement policies and procedures for incident handling, including detection, analysis, containment, and recovery.',
    'Ensure effective and timely response to cybersecurity incidents.',
    'Incident Response',
    'Operations',
    'Art. 21(2)(b)',
    '[{"id": "T1", "description": "Review incident response plan", "expected_evidence": "Documented IRP with roles and procedures"}, {"id": "T2", "description": "Review recent incident reports or tabletop exercise results", "expected_evidence": "Incident log or exercise report"}]',
    '[{"type": "incident_response_plan", "description": "Incident response policy and procedures", "required": true}, {"type": "incident_log", "description": "Log of incidents in past 12 months", "required": false}]',
    2.50,
    ARRAY['incident-response', 'operations', 'nis2', 'article-21b']
),
(
    '10000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000003',
    'NIS2-21b.1',
    'Incident classification and severity scoring',
    'Define classification criteria for cybersecurity incidents including severity levels.',
    'Enable consistent and accurate incident triage.',
    'Incident Response',
    'Operations',
    'Art. 21(2)(b)',
    '[{"id": "T1", "description": "Review incident classification matrix", "expected_evidence": "Documented classification matrix"}]',
    '[{"type": "classification_matrix", "description": "Incident classification matrix", "required": true}]',
    1.50,
    ARRAY['incident-classification', 'nis2']
);

-- ============================================================
-- NIS 2 Article 21(c) — Business continuity and crisis management
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21c',
    'Business continuity and crisis management',
    'Entities shall implement business continuity measures including backup management, disaster recovery, and crisis management.',
    'Ensure resilience and recovery capability for critical systems.',
    'Business Continuity',
    'Resilience',
    'Art. 21(2)(c)',
    '[{"id": "T1", "description": "Review BCP/DRP documentation", "expected_evidence": "BCP and DRP documents"}, {"id": "T2", "description": "Verify backup testing results", "expected_evidence": "Backup test logs from past 12 months"}]',
    '[{"type": "bcp_document", "description": "Business continuity plan", "required": true}, {"type": "drp_document", "description": "Disaster recovery plan", "required": true}, {"type": "backup_test_results", "description": "Recent backup test results", "required": true}]',
    2.00,
    ARRAY['business-continuity', 'disaster-recovery', 'backups', 'nis2', 'article-21c']
);

-- ============================================================
-- NIS 2 Article 21(d) — Supply chain security
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21d',
    'Supply chain security',
    'Entities shall address security aspects of relationships between entities and their direct suppliers and service providers.',
    'Manage cybersecurity risks arising from third-party relationships and supply chains.',
    'Third-Party Risk',
    'Supply Chain',
    'Art. 21(2)(d)',
    '[{"id": "T1", "description": "Review supplier security assessment process", "expected_evidence": "Supplier risk assessment procedure"}, {"id": "T2", "description": "Sample supplier contracts for security clauses", "expected_evidence": "Contracts with security requirements"}]',
    '[{"type": "supplier_policy", "description": "Supply chain security policy", "required": true}, {"type": "supplier_assessments", "description": "Third-party risk assessments", "required": true}]',
    2.00,
    ARRAY['supply-chain', 'third-party', 'vendor-management', 'nis2', 'article-21d']
);

-- ============================================================
-- NIS 2 Article 21(e) — Security in network and information systems acquisition, development and maintenance
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000007',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21e',
    'Security in network and information systems acquisition, development and maintenance',
    'Entities shall include security in the acquisition, development and maintenance of network and information systems, including vulnerability handling and disclosure.',
    'Embed security throughout the system development lifecycle.',
    'Development Security',
    'Technology',
    'Art. 21(2)(e)',
    '[{"id": "T1", "description": "Review SDLC security policy", "expected_evidence": "Secure SDLC documentation"}, {"id": "T2", "description": "Review vulnerability management process", "expected_evidence": "Vulnerability management procedure and recent scan results"}]',
    '[{"type": "sdlc_policy", "description": "Secure development lifecycle policy", "required": true}, {"type": "vulnerability_scans", "description": "Recent vulnerability scan results", "required": true}]',
    2.00,
    ARRAY['secure-development', 'vulnerability-management', 'sdlc', 'nis2', 'article-21e']
);

-- ============================================================
-- NIS 2 Article 21(f) — Policies and procedures to assess the effectiveness of cybersecurity risk management measures
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000008',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21f',
    'Policies and procedures to assess the effectiveness of cybersecurity risk management measures',
    'Entities shall implement policies and procedures to assess effectiveness of cybersecurity risk management measures.',
    'Continuously monitor and improve the effectiveness of security controls.',
    'Performance Management',
    'Governance',
    'Art. 21(2)(f)',
    '[{"id": "T1", "description": "Review KPIs and metrics programme", "expected_evidence": "Security metrics dashboard or report"}, {"id": "T2", "description": "Review audit and review schedule", "expected_evidence": "Internal audit plan"}]',
    '[{"type": "metrics_report", "description": "Cybersecurity metrics and KPIs", "required": true}, {"type": "audit_plan", "description": "Internal audit and review plan", "required": true}]',
    1.50,
    ARRAY['metrics', 'performance', 'audit', 'nis2', 'article-21f']
);

-- ============================================================
-- NIS 2 Article 21(g) — Basic cyber hygiene practices and cybersecurity training
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000009',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21g',
    'Basic cyber hygiene practices and cybersecurity training',
    'Entities shall implement basic cyber hygiene practices and provide cybersecurity training.',
    'Build cybersecurity awareness and competency across the organization.',
    'Security Awareness',
    'People',
    'Art. 21(2)(g)',
    '[{"id": "T1", "description": "Review security awareness training programme", "expected_evidence": "Training completion records"}, {"id": "T2", "description": "Review cyber hygiene policy", "expected_evidence": "Acceptable use and cyber hygiene policies"}]',
    '[{"type": "training_records", "description": "Security awareness training completion records", "required": true}, {"type": "hygiene_policy", "description": "Cyber hygiene policy", "required": true}]',
    1.50,
    ARRAY['training', 'awareness', 'cyber-hygiene', 'nis2', 'article-21g']
);

-- ============================================================
-- NIS 2 Article 21(h) — Policies and procedures regarding the use of cryptography
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21h',
    'Policies and procedures regarding the use of cryptography and, where appropriate, encryption',
    'Entities shall implement policies and procedures regarding cryptographic controls and encryption of data.',
    'Protect data confidentiality and integrity through appropriate cryptographic measures.',
    'Cryptography',
    'Technology',
    'Art. 21(2)(h)',
    '[{"id": "T1", "description": "Review cryptography policy", "expected_evidence": "Cryptography policy with approved algorithms"}, {"id": "T2", "description": "Sample data-at-rest and in-transit encryption configuration", "expected_evidence": "Configuration screenshots or export"}]',
    '[{"type": "crypto_policy", "description": "Cryptography and encryption policy", "required": true}, {"type": "encryption_config", "description": "Evidence of encryption implementation", "required": true}]',
    2.00,
    ARRAY['cryptography', 'encryption', 'data-protection', 'nis2', 'article-21h']
);

-- ============================================================
-- NIS 2 Article 21(i) — Human resources security, access control policies and asset management
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21i',
    'Human resources security, access control policies and asset management',
    'Entities shall implement human resources security measures, access control policies, and asset management practices.',
    'Ensure appropriate access controls, asset inventory, and HR security practices.',
    'Access Control',
    'Identity',
    'Art. 21(2)(i)',
    '[{"id": "T1", "description": "Review access control policy and procedures", "expected_evidence": "Access control policy with least-privilege principle"}, {"id": "T2", "description": "Sample user access reviews", "expected_evidence": "Recent access review records"}, {"id": "T3", "description": "Review asset inventory", "expected_evidence": "IT asset register"}]',
    '[{"type": "access_control_policy", "description": "Access control policy", "required": true}, {"type": "access_review_records", "description": "Periodic access review records", "required": true}, {"type": "asset_inventory", "description": "IT asset inventory", "required": true}]',
    2.00,
    ARRAY['access-control', 'identity', 'asset-management', 'hr-security', 'nis2', 'article-21i']
),
(
    '10000000-0000-0000-0000-000000000012',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000011',
    'NIS2-21i.1',
    'Multi-factor authentication and privileged access management',
    'Implement MFA and controls for privileged accounts.',
    'Reduce risk of account compromise and unauthorized privileged access.',
    'Access Control',
    'Identity',
    'Art. 21(2)(i)',
    '[{"id": "T1", "description": "Verify MFA enforcement for remote and privileged access", "expected_evidence": "MFA configuration screenshots"}]',
    '[{"type": "mfa_config", "description": "MFA configuration evidence", "required": true}]',
    2.50,
    ARRAY['mfa', 'privileged-access', 'pam', 'nis2']
);

-- ============================================================
-- NIS 2 Article 21(j) — Use of multi-factor authentication or continuous authentication
-- ============================================================
INSERT INTO controls (id, framework_id, org_id, parent_id, control_id, title, description, objective, category, domain, article_ref, test_procedures, evidence_requirements, risk_weight, tags)
VALUES
(
    '10000000-0000-0000-0000-000000000013',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    NULL,
    'NIS2-21j',
    'Use of multi-factor authentication or continuous authentication solutions, secured voice, video and text communications and secured emergency communication systems',
    'Entities shall use multi-factor authentication or continuous authentication solutions and secure communications.',
    'Strengthen authentication and protect communications from unauthorized interception or tampering.',
    'Authentication',
    'Technology',
    'Art. 21(2)(j)',
    '[{"id": "T1", "description": "Review MFA deployment scope and coverage", "expected_evidence": "MFA enrollment report"}, {"id": "T2", "description": "Review secure communications policy", "expected_evidence": "Secure communications policy and tool inventory"}, {"id": "T3", "description": "Verify emergency communication procedures", "expected_evidence": "Emergency communication plan"}]',
    '[{"type": "mfa_report", "description": "MFA coverage and enrollment report", "required": true}, {"type": "secure_comms_policy", "description": "Secure communications policy", "required": true}, {"type": "emergency_comms_plan", "description": "Emergency communication plan", "required": false}]',
    2.50,
    ARRAY['mfa', 'authentication', 'secure-communications', 'nis2', 'article-21j']
);

COMMIT;

