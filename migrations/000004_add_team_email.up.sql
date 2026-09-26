-- =============================================================================
-- Migration 000003: Reversal
-- =============================================================================
-- Drops database_connection in 000003_add_nonce_schema.up.sql.
-- =============================================================================

ALTER TABLE IF EXISTS teams ADD COLUMN email_address VARCHAR NOT NULL;
