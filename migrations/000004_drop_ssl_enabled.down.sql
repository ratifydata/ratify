-- =============================================================================
-- Migration 000004: Reversal
-- =============================================================================
-- Adds column ssl_enabled
-- =============================================================================

ALTER TABLE IF EXISTS database_connections ADD COLUMN ssl_enabled BOOLEAN NOT NULL DEFAULT TRUE;
