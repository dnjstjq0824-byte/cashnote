-- Migration 0001: Extensions, Custom Types, and Helper Functions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Helper function for current tenant store_id in RLS policies
CREATE OR REPLACE FUNCTION current_tenant_store_id() RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_store_id', true), '')::UUID;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION current_tenant_business_id() RETURNS UUID AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_business_id', true), '')::UUID;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;
