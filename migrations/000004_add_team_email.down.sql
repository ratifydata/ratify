-- =============================================================================
-- Migration 000004: Drops email_address in table teams
-- =============================================================================
-- Drops column email_address  in 000004_add_team_email_schema.up.sql.
-- =============================================================================

ALTER TABLE IF EXISTS teams DROP COLUMN email_address;
