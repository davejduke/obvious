-- AIAUDITOR PostgreSQL DDL
-- Migration: 004_search_notifications.sql
-- Adds NOTIFY triggers on findings, evidence, and controls for CDC pipeline.
-- The search service listens on channels: aiauditor_findings, aiauditor_evidence,
-- aiauditor_controls and re-indexes changed rows in OpenSearch.

BEGIN;

-- ============================================================
-- NOTIFY trigger function (generic, reused by all tables)
-- Emits JSON payload: {"op":"INSERT|UPDATE|DELETE","table":"...","id":"...","org_id":"..."}
-- ============================================================
CREATE OR REPLACE FUNCTION aiauditor_notify_change()
RETURNS TRIGGER AS $$
DECLARE
  payload JSON;
  channel TEXT;
BEGIN
  channel := 'aiauditor_' || TG_TABLE_NAME;

  IF TG_OP = 'DELETE' THEN
    payload := json_build_object(
      'op',     'DELETE',
      'table',  TG_TABLE_NAME,
      'id',     OLD.id,
      'org_id', OLD.org_id
    );
  ELSE
    payload := json_build_object(
      'op',     TG_OP,
      'table',  TG_TABLE_NAME,
      'id',     NEW.id,
      'org_id', NEW.org_id
    );
  END IF;

  -- NOTIFY payload is limited to 8000 bytes; we send only IDs, not full rows.
  PERFORM pg_notify(channel, payload::TEXT);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- FINDINGS change trigger
-- ============================================================
DROP TRIGGER IF EXISTS trg_findings_search_notify ON findings;
CREATE TRIGGER trg_findings_search_notify
  AFTER INSERT OR UPDATE OR DELETE ON findings
  FOR EACH ROW EXECUTE FUNCTION aiauditor_notify_change();

-- ============================================================
-- EVIDENCE change trigger
-- ============================================================
DROP TRIGGER IF EXISTS trg_evidence_search_notify ON evidence;
CREATE TRIGGER trg_evidence_search_notify
  AFTER INSERT OR UPDATE OR DELETE ON evidence
  FOR EACH ROW EXECUTE FUNCTION aiauditor_notify_change();

-- ============================================================
-- CONTROLS change trigger
-- ============================================================
DROP TRIGGER IF EXISTS trg_controls_search_notify ON controls;
CREATE TRIGGER trg_controls_search_notify
  AFTER INSERT OR UPDATE OR DELETE ON controls
  FOR EACH ROW EXECUTE FUNCTION aiauditor_notify_change();

COMMIT;

