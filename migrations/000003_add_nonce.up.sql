-- =============================================================================
-- Migration 000003: Update database_connection table
-- =============================================================================
-- Adds nonce to the database_connection table
-- =============================================================================

ALTER TABLE IF EXISTS database_connections ADD column nonce bytea NOT NULL;
ALTER TABLE IF EXISTS database_connections ALTER COLUMN password_encrypted TYPE bytea USING decode(password_encrypted, 'base64');;