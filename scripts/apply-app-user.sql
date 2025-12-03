-- Script to add app_user to an existing database
-- Run this if you already have data and don't want to reset the entire database

-- Create application user with proper RLS enforcement
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user WITH LOGIN PASSWORD 'mcp_password' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
        RAISE NOTICE 'Created app_user role';
    ELSE
        RAISE NOTICE 'app_user role already exists';
        -- Ensure it has the correct attributes
        ALTER ROLE app_user WITH LOGIN PASSWORD 'mcp_password' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
        RAISE NOTICE 'Updated app_user attributes';
    END IF;
END
$$;

-- Grant permissions to app_user
GRANT CONNECT ON DATABASE mcp_db TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO app_user;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO app_user;

-- Verify the setup
SELECT rolname, rolsuper, rolbypassrls
FROM pg_roles
WHERE rolname IN ('mcp_user', 'app_user')
ORDER BY rolname;
