-- =============================================================================
-- Migration 000005: Drops column deleted_at to table team_members
-- =============================================================================

ALTER TABLE IF EXISTS team_members DROP COLUMN deleted_at;
