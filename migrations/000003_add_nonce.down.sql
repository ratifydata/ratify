-- =============================================================================
-- Migration 000003: Reversal
-- =============================================================================
-- Drops database_connection in 000003_add_nonce_schema.up.sql.
-- =============================================================================

ALTER TABLE IF EXISTS database_connections DROP COLUMN nonce;
ALTER TABLE IF EXISTS database_connections DROP COLUMN password_encrypted;