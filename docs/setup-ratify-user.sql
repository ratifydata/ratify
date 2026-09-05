\set ON_ERROR_STOP on
\set database_name 'my_app'
-- The remaining privileges are scoped to the target database.
\connect :"database_name"

-- Replace these values before running the script.
\set ratify_username 'ratify_readonly'
\set ratify_password 'password'
\set schema_name 'public'
\set object_owner 'my_app_owner'


SELECT EXISTS(
    SELECT FROM pg_catalog.pg_roles WHERE pg_roles.rolname = :'ratify_username'
) AS role_exist;
\gset

\if :role_exist
\echo 'Role Already Exist. Updating privileges only'
\else
CREATE ROLE :"ratify_username"
    WITH LOGIN
    PASSWORD :'ratify_password'
    NOSUPERUSER
    NOCREATEDB
    NOCREATEROLE
    NOINHERIT;

\echo 'Role created successfully'

\endif

-- Permit the Ratify user to connect to the target database.
GRANT CONNECT ON DATABASE :"database_name" TO :"ratify_username";


\echo 'Database Name: '+:database_name

-- Permit access to objects in the target schema and read existing tables.
GRANT USAGE ON SCHEMA :"schema_name" TO :"ratify_username";
GRANT SELECT ON ALL TABLES IN SCHEMA :"schema_name" TO :"ratify_username";

-- Automatically grant SELECT on tables subsequently created by object_owner.
-- Repeat this statement for every role that creates tables in the schema.
ALTER DEFAULT PRIVILEGES FOR ROLE :"object_owner"
    IN SCHEMA :"schema_name"
    GRANT SELECT ON TABLES TO :"ratify_username";
