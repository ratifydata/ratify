-- =============================================================================
-- Migration 000002: Reversal
-- =============================================================================
-- Drops all tables created in 000002_core_schema.up.sql.
-- Tables are dropped in reverse dependency order — most dependent first.
-- =============================================================================

DROP TABLE IF EXISTS notification_logs;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS breaches;
DROP TABLE IF EXISTS consumer_responses;
DROP TABLE IF EXISTS proposal_changes;
DROP TABLE IF EXISTS proposals;
DROP TABLE IF EXISTS consumer_registrations;
DROP TABLE IF EXISTS contract_columns;
DROP TABLE IF EXISTS contract_versions;
DROP TABLE IF EXISTS contracts;
DROP TABLE IF EXISTS database_connections;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS organizations;