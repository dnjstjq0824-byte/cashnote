-- Migration 0003: Domain 3 (Connectors) - 7 Tables
CREATE TABLE provider_catalog (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_code VARCHAR(64) NOT NULL UNIQUE,
    category VARCHAR(32) NOT NULL CHECK (category IN ('NTS_PUBLIC', 'CARD_ASSOCIATION', 'CARD_ISSUER', 'OPEN_BANKING', 'HOMETAX_ASP', 'PG', 'VAN', 'POS', 'DELIVERY_PLATFORM', 'REVIEW_PLATFORM', 'MESSAGING')),
    display_name VARCHAR(120) NOT NULL,
    access_status VARCHAR(32) NOT NULL CHECK (access_status IN ('PUBLIC_API', 'APPROVAL_REQUIRED', 'CONTRACT_REQUIRED', 'LICENSE_REQUIRED', 'PARTNER_ONLY', 'COMMERCIAL_PROVIDER', 'NOT_PUBLIC', 'NOT_VERIFIED', 'UNSUPPORTED', 'BLOCKED')),
    supports_level_2_file BOOLEAN NOT NULL DEFAULT TRUE,
    supports_level_3_api BOOLEAN NOT NULL DEFAULT FALSE,
    is_live_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    contract_reference_no VARCHAR(128),
    official_doc_url TEXT,
    rate_limit_per_min INT NOT NULL DEFAULT 60,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE data_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES provider_catalog(id),
    connection_mode VARCHAR(32) NOT NULL CHECK (connection_mode IN ('LEVEL_1_MANUAL', 'LEVEL_2_FILE_ONLY', 'LEVEL_3_AUTHORIZED_API', 'LEVEL_3_WEBHOOK')),
    status VARCHAR(32) NOT NULL CHECK (status IN ('ACTIVE', 'PENDING_AUTH', 'TOKEN_EXPIRED', 'ERROR', 'BLOCKED_CONTRACT', 'REVOKED')),
    external_merchant_id VARCHAR(128),
    credentials_encrypted BYTEA,
    kms_key_id VARCHAR(128),
    last_synced_at TIMESTAMPTZ,
    last_error_code VARCHAR(64),
    last_error_message TEXT,
    created_by_user_id UUID NOT NULL REFERENCES users(id),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE connection_consents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    data_connection_id UUID REFERENCES data_connections(id) ON DELETE SET NULL,
    consented_by_user_id UUID NOT NULL REFERENCES users(id),
    consent_type VARCHAR(64) NOT NULL CHECK (consent_type IN ('TERMS_OF_SERVICE', 'PRIVACY_COLLECTION', 'THIRD_PARTY_DATA_PROVISION', 'FINANCIAL_DATA_DELEGATION', 'TAX_AGENT_SHARING', 'MARKETING_OPT_IN')),
    policy_version VARCHAR(32) NOT NULL,
    scope_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_agreed BOOLEAN NOT NULL,
    ip_address INET,
    user_agent TEXT,
    consented_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE TABLE raw_import_batches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    provider_id UUID REFERENCES provider_catalog(id),
    source_level VARCHAR(32) NOT NULL CHECK (source_level IN ('LEVEL_1_MANUAL', 'LEVEL_2_FILE', 'LEVEL_3_API')),
    file_name VARCHAR(255),
    file_sha256 CHAR(64),
    s3_object_key VARCHAR(512),
    total_rows INT NOT NULL DEFAULT 0,
    parsed_rows INT NOT NULL DEFAULT 0,
    duplicate_rows INT NOT NULL DEFAULT 0,
    error_rows INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL CHECK (status IN ('UPLOADED', 'PROCESSING', 'COMPLETED', 'PARTIAL_COMPLETED', 'FAILED')),
    uploaded_by_user_id UUID REFERENCES users(id),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, file_sha256)
);

CREATE TABLE sync_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    data_connection_id UUID NOT NULL REFERENCES data_connections(id) ON DELETE CASCADE,
    raw_import_batch_id UUID REFERENCES raw_import_batches(id),
    trigger_type VARCHAR(32) NOT NULL CHECK (trigger_type IN ('SCHEDULED', 'MANUAL_USER', 'WEBHOOK', 'ADMIN_RETRY')),
    target_date_from DATE NOT NULL,
    target_date_to DATE NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('QUEUED', 'RUNNING', 'SUCCEEDED', 'PARTIAL_FAILED', 'FAILED', 'BLOCKED')),
    attempt_count SMALLINT NOT NULL DEFAULT 1,
    records_fetched INT NOT NULL DEFAULT 0,
    records_upserted INT NOT NULL DEFAULT 0,
    error_code VARCHAR(64),
    error_detail TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sync_cursors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    data_connection_id UUID NOT NULL REFERENCES data_connections(id) ON DELETE CASCADE,
    stream_name VARCHAR(64) NOT NULL,
    cursor_value VARCHAR(255) NOT NULL,
    last_event_timestamp TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (data_connection_id, stream_name)
);

CREATE TABLE ingestion_errors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    raw_import_batch_id UUID REFERENCES raw_import_batches(id) ON DELETE CASCADE,
    sync_job_id UUID REFERENCES sync_jobs(id) ON DELETE CASCADE,
    row_number INT,
    error_code VARCHAR(64) NOT NULL,
    error_message TEXT NOT NULL,
    raw_payload_masked JSONB,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_by_user_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
