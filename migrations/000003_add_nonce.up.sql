-- =============================================================================
-- Migration 000003: Update database_connection table
-- =============================================================================
-- Adds nonce to the database_connection table
-- =============================================================================

ALTER TABLE IF EXISTS database_connections ADD column nonce TEXT NOT NULL;