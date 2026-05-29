-- =============================================================================
-- Migration 000001: Initial schema setup
-- =============================================================================
-- This migration establishes the database configuration that all subsequent
-- migrations depend on. It enables the pgcrypto extension for UUID generation
-- using gen_random_uuid(), which every table in the schema uses as its
-- primary key type.
-- =============================================================================

-- Enable the pgcrypto extension.
-- Required for gen_random_uuid() which generates UUID primary keys.
-- All tables in the Ratify schema use UUID primary keys.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";