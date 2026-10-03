-- =============================================================================
-- Migration 000005: Adds active to table team_members
-- =============================================================================

ALTER TABLE IF EXISTS team_members ADD COLUMN deleted_at TIMESTAMPTZ;
