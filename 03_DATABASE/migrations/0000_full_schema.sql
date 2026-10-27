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


-- Migration 0002: Domain 1 (Identity) & Domain 2 (Business) - 11 Tables
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    phone_encrypted BYTEA,
    phone_hmac_hash VARCHAR(64) UNIQUE,
    password_hash VARCHAR(255),
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED', 'DELETED')),
    mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100),
    locale VARCHAR(16) NOT NULL DEFAULT 'ko-KR',
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Seoul',
    large_font_mode BOOLEAN NOT NULL DEFAULT FALSE,
    ci_hash VARCHAR(128),
    identity_verified_at TIMESTAMPTZ,
    marketing_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    marketing_opt_in_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL CHECK (provider IN ('LOCAL', 'KAKAO', 'GOOGLE', 'APPLE', 'PASS_CI')),
    provider_subject_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_subject_id)
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(128) NOT NULL UNIQUE,
    token_family_id UUID NOT NULL,
    device_name VARCHAR(120),
    ip_address INET,
    user_agent TEXT,
    mfa_verified BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(64) NOT NULL UNIQUE CHECK (code IN ('OWNER', 'CO_OWNER', 'MANAGER', 'ACCOUNTANT', 'TAX_AGENT', 'STAFF', 'VIEWER', 'SUPPLIER', 'PARTNER_ADMIN', 'CS_OPERATOR', 'AUDITOR', 'SUPER_ADMIN')),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system_role BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(100) NOT NULL UNIQUE,
    domain VARCHAR(64) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE businesses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_code CHAR(2) NOT NULL DEFAULT 'KR',
    business_type VARCHAR(32) NOT NULL CHECK (business_type IN ('INDIVIDUAL_GENERAL', 'INDIVIDUAL_SIMPLIFIED', 'INDIVIDUAL_EXEMPT', 'CORPORATION', 'NON_PROFIT')),
    registration_number VARCHAR(32) NOT NULL,
    corporate_registration_number VARCHAR(32),
    legal_name VARCHAR(200) NOT NULL,
    representative_name VARCHAR(100) NOT NULL,
    opened_date DATE,
    nts_tax_type_code VARCHAR(16),
    nts_business_status VARCHAR(32) NOT NULL DEFAULT 'UNKNOWN' CHECK (nts_business_status IN ('UNKNOWN', 'ACTIVE', 'SUSPENDED', 'CLOSED')),
    nts_closed_date DATE,
    nts_last_checked_at TIMESTAMPTZ,
    ownership_verification_status VARCHAR(32) NOT NULL DEFAULT 'UNVERIFIED' CHECK (ownership_verification_status IN ('UNVERIFIED', 'NTS_VALIDATED', 'PENDING_REVIEW', 'VERIFIED_OWNER', 'DISPUTED', 'REJECTED', 'REVOKED')),
    primary_owner_user_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (country_code, registration_number)
);

CREATE TABLE business_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    requested_by_user_id UUID NOT NULL REFERENCES users(id),
    verification_method VARCHAR(32) NOT NULL CHECK (verification_method IN ('NTS_API_CHECK', 'PENNY_BANK_AUTH', 'OCR_CERTIFICATE', 'POWER_OF_ATTORNEY_MANUAL')),
    status VARCHAR(32) NOT NULL CHECK (status IN ('PENDING', 'VERIFIED', 'FAILED', 'EXPIRED', 'REJECTED')),
    evidence_s3_key VARCHAR(512),
    nts_response_payload JSONB,
    reviewer_admin_id UUID REFERENCES users(id),
    rejection_reason TEXT,
    verified_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE stores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
    store_code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    vertical_type VARCHAR(32) NOT NULL DEFAULT 'FNB_RESTAURANT' CHECK (vertical_type IN ('FNB_RESTAURANT', 'FNB_CAFE_BAKERY', 'RETAIL_COMMERCE', 'SERVICE_BEAUTY_ACADEMY', 'OTHER')),
    ksic_industry_code VARCHAR(16),
    currency_code CHAR(3) NOT NULL DEFAULT 'KRW',
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Seoul',
    business_day_cutoff_hour SMALLINT NOT NULL DEFAULT 0 CHECK (business_day_cutoff_hour BETWEEN 0 AND 12),
    region_code VARCHAR(20),
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    postal_code VARCHAR(20),
    phone VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ONBOARDING', 'ACTIVE', 'INACTIVE', 'ARCHIVED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE store_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id),
    invitation_status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE' CHECK (invitation_status IN ('INVITED', 'ACTIVE', 'SUSPENDED', 'REVOKED', 'LEFT')),
    invited_by_user_id UUID REFERENCES users(id),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, user_id)
);

CREATE TABLE store_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_member_id UUID NOT NULL REFERENCES store_members(id) ON DELETE CASCADE,
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    effect VARCHAR(16) NOT NULL DEFAULT 'ALLOW' CHECK (effect IN ('ALLOW', 'DENY')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_member_id, permission_id)
);


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


-- Migration 0004: Domain 4 (Revenue, Card Sales, Settlements, Bank Deposits & Reconciliation) - 10 Tables
CREATE TABLE sales_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
    source_level VARCHAR(32) NOT NULL CHECK (source_level IN ('LEVEL_1_MANUAL', 'LEVEL_2_FILE', 'LEVEL_3_API', 'LEVEL_4_RECONCILED')),
    source_provider_id UUID REFERENCES provider_catalog(id),
    raw_import_batch_id UUID REFERENCES raw_import_batches(id),
    external_transaction_id VARCHAR(128),
    dedup_fingerprint CHAR(64) NOT NULL,
    channel_type VARCHAR(32) NOT NULL CHECK (channel_type IN ('CARD_OFFLINE', 'CARD_ONLINE_PG', 'CASH', 'CASH_RECEIPT', 'DELIVERY_APP', 'BANK_TRANSFER', 'TAX_INVOICE', 'OTHER')),
    transaction_status VARCHAR(32) NOT NULL CHECK (transaction_status IN ('AUTHORIZED', 'CAPTURED', 'PARTIALLY_CANCELED', 'CANCELED', 'SETTLEMENT_SCHEDULED', 'SETTLED', 'DISPUTED')),
    transacted_at TIMESTAMPTZ NOT NULL,
    business_date DATE NOT NULL,
    currency_code CHAR(3) NOT NULL DEFAULT 'KRW',
    fx_rate_to_base NUMERIC(19,6) NOT NULL DEFAULT 1.000000,
    gross_amount NUMERIC(19,4) NOT NULL,
    discount_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    canceled_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    net_sales_amount NUMERIC(19,4) NOT NULL,
    supply_amount NUMERIC(19,4) NOT NULL,
    vat_amount NUMERIC(19,4) NOT NULL,
    service_tip_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    tax_exemption_type VARCHAR(16) NOT NULL DEFAULT 'TAXABLE' CHECK (tax_exemption_type IN ('TAXABLE', 'TAX_FREE', 'ZERO_RATED', 'MIXED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, dedup_fingerprint)
);

CREATE TABLE payment_authorizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    sales_transaction_id UUID NOT NULL REFERENCES sales_transactions(id) ON DELETE RESTRICT,
    card_issuer_code VARCHAR(32) NOT NULL,
    card_acquirer_code VARCHAR(32) NOT NULL,
    card_brand VARCHAR(32),
    card_type VARCHAR(16) CHECK (card_type IN ('CREDIT', 'CHECK_DEBIT', 'PREPAID', 'GIFT', 'CORPORATE', 'UNKNOWN')),
    masked_pan VARCHAR(32) NOT NULL,
    approval_number VARCHAR(32) NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL,
    installment_months SMALLINT NOT NULL DEFAULT 0,
    is_interest_free BOOLEAN NOT NULL DEFAULT FALSE,
    authorized_amount NUMERIC(19,4) NOT NULL CHECK (authorized_amount >= 0),
    terminal_id VARCHAR(64),
    van_code VARCHAR(32),
    auth_status VARCHAR(32) NOT NULL CHECK (auth_status IN ('APPROVED', 'VOIDED_SAME_DAY', 'PARTIALLY_CANCELED', 'CANCELED_AFTER_CAPTURE')),
    raw_import_batch_id UUID REFERENCES raw_import_batches(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, card_acquirer_code, approval_number, approved_at, authorized_amount)
);

CREATE TABLE payment_captures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    payment_authorization_id UUID NOT NULL REFERENCES payment_authorizations(id) ON DELETE RESTRICT,
    capture_date DATE NOT NULL,
    captured_amount NUMERIC(19,4) NOT NULL,
    base_fee_rate NUMERIC(8,5) NOT NULL DEFAULT 0.00000,
    preferential_fee_discount_rate NUMERIC(8,5) NOT NULL DEFAULT 0.00000,
    applied_fee_rate NUMERIC(8,5) NOT NULL,
    fee_amount NUMERIC(19,4) NOT NULL,
    vat_on_fee_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    net_expected_payout_amount NUMERIC(19,4) NOT NULL,
    expected_payout_date DATE NOT NULL,
    capture_status VARCHAR(32) NOT NULL CHECK (capture_status IN ('CAPTURED', 'HOLD', 'REJECTED', 'SETTLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payment_cancellations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    payment_authorization_id UUID NOT NULL REFERENCES payment_authorizations(id) ON DELETE RESTRICT,
    cancellation_type VARCHAR(32) NOT NULL CHECK (cancellation_type IN ('PRE_CAPTURE_VOID', 'POST_CAPTURE_FULL_CANCEL', 'POST_CAPTURE_PARTIAL_CANCEL', 'CHARGEBACK')),
    canceled_at TIMESTAMPTZ NOT NULL,
    canceled_amount NUMERIC(19,4) NOT NULL CHECK (canceled_amount > 0),
    fee_refunded_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    offset_settlement_date DATE,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE acquisition_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    card_acquirer_code VARCHAR(32) NOT NULL,
    acquisition_date DATE NOT NULL,
    expected_payout_date DATE NOT NULL,
    transaction_count INT NOT NULL CHECK (transaction_count >= 0),
    total_captured_amount NUMERIC(19,4) NOT NULL,
    total_canceled_offset_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    total_fee_amount NUMERIC(19,4) NOT NULL,
    net_scheduled_amount NUMERIC(19,4) NOT NULL,
    source_level VARCHAR(32) NOT NULL,
    raw_import_batch_id UUID REFERENCES raw_import_batches(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, card_acquirer_code, acquisition_date, expected_payout_date)
);

CREATE TABLE payout_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    settlement_source_type VARCHAR(32) NOT NULL CHECK (settlement_source_type IN ('CARD_ACQUIRER', 'PG_PROVIDER', 'DELIVERY_PLATFORM', 'MARKETPLACE')),
    source_partner_code VARCHAR(64) NOT NULL,
    sales_period_start DATE NOT NULL,
    sales_period_end DATE NOT NULL,
    scheduled_payout_date DATE NOT NULL,
    original_holiday_shifted_from DATE,
    gross_sales_amount NUMERIC(19,4) NOT NULL,
    total_deduction_fee_amount NUMERIC(19,4) NOT NULL,
    hold_or_offset_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    expected_net_deposit_amount NUMERIC(19,4) NOT NULL,
    payout_status VARCHAR(32) NOT NULL DEFAULT 'SCHEDULED' CHECK (payout_status IN ('SCHEDULED', 'DEPOSITED_MATCHED', 'PARTIALLY_DEPOSITED', 'DELAYED_CANDIDATE', 'CONFIRMED_ON_HOLD', 'CONFIRMED_UNPAID', 'SYNC_DELAY_SUSPECTED')),
    target_bank_code VARCHAR(16),
    target_account_masked VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE settlement_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    payout_schedule_id UUID REFERENCES payout_schedules(id),
    channel_type VARCHAR(32) NOT NULL,
    partner_code VARCHAR(64) NOT NULL,
    settlement_date DATE NOT NULL,
    gross_amount NUMERIC(19,4) NOT NULL,
    commission_fee_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    delivery_fee_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    payment_gateway_fee_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    ad_spend_deducted_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    merchant_coupon_burden_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    withholding_or_vat_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    net_settled_amount NUMERIC(19,4) NOT NULL,
    source_level VARCHAR(32) NOT NULL,
    raw_import_batch_id UUID REFERENCES raw_import_batches(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bank_deposits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    bank_code VARCHAR(16) NOT NULL,
    account_number_encrypted BYTEA NOT NULL,
    account_number_masked VARCHAR(32) NOT NULL,
    bank_tran_id VARCHAR(128) NOT NULL,
    transacted_at TIMESTAMPTZ NOT NULL,
    transaction_date DATE NOT NULL,
    direction VARCHAR(16) NOT NULL CHECK (direction IN ('INBOUND_DEPOSIT', 'OUTBOUND_WITHDRAWAL')),
    amount NUMERIC(19,4) NOT NULL CHECK (amount > 0),
    balance_after NUMERIC(19,4),
    counterparty_name VARCHAR(120),
    memo_text VARCHAR(255),
    reconciliation_status VARCHAR(32) NOT NULL DEFAULT 'UNMATCHED' CHECK (reconciliation_status IN ('UNMATCHED', 'AUTO_MATCHED', 'MANUAL_MATCHED', 'PARTIAL_MATCHED', 'IGNORED_NON_REVENUE')),
    source_level VARCHAR(32) NOT NULL,
    raw_import_batch_id UUID REFERENCES raw_import_batches(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, bank_code, bank_tran_id)
);

CREATE TABLE reconciliation_matches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    payout_schedule_id UUID NOT NULL REFERENCES payout_schedules(id) ON DELETE RESTRICT,
    bank_deposit_id UUID REFERENCES bank_deposits(id) ON DELETE RESTRICT,
    match_type VARCHAR(32) NOT NULL CHECK (match_type IN ('EXACT_1_TO_1', 'BATCH_N_TO_1', 'SPLIT_1_TO_N', 'FEE_TOLERANCE_MATCH', 'MANUAL_OVERRIDE', 'UNMATCHED_DIAGNOSTIC')),
    match_status VARCHAR(32) NOT NULL CHECK (match_status IN ('MATCHED', 'VARIANCE_DETECTED', 'DELAYED_CANDIDATE', 'CONFIRMED_UNPAID', 'SYNC_DELAY_PENDING', 'RESOLVED')),
    expected_amount NUMERIC(19,4) NOT NULL,
    actual_matched_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    variance_amount NUMERIC(19,4) NOT NULL,
    variance_reason_code VARCHAR(64) CHECK (variance_reason_code IN ('NONE', 'FEE_RATE_DIFFERENCE', 'POST_CAPTURE_CANCEL_OFFSET', 'HOLIDAY_SHIFT', 'PARTIAL_PAYOUT_HOLD', 'DATA_SYNC_LAGGING', 'UNKNOWN_SHORTFALL')),
    confidence_score NUMERIC(5,2) NOT NULL CHECK (confidence_score BETWEEN 0 AND 100),
    matched_by_user_id UUID REFERENCES users(id),
    matched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT
);

CREATE TABLE reconciliation_disputes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    reconciliation_match_id UUID NOT NULL REFERENCES reconciliation_matches(id) ON DELETE RESTRICT,
    raised_by_user_id UUID NOT NULL REFERENCES users(id),
    dispute_type VARCHAR(32) NOT NULL CHECK (dispute_type IN ('FALSE_POSITIVE_UNPAID', 'WRONG_BANK_MATCH', 'FEE_CALCULATION_ERROR', 'CARD_ISSUER_INQUIRY_NEEDED')),
    status VARCHAR(32) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'UNDER_REVIEW', 'RESOLVED_CORRECTED', 'RESOLVED_CONFIRMED_SHORTFALL', 'CLOSED')),
    merchant_comment TEXT NOT NULL,
    resolution_summary TEXT,
    resolved_by_user_id UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- Migration 0005: Domain 5 (Expenses & Immutable Double-Entry Ledger) & Domain 6 (Tax) - 12 Tables
CREATE TABLE expense_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(120) NOT NULL,
    cost_behavior VARCHAR(16) NOT NULL CHECK (cost_behavior IN ('FIXED', 'VARIABLE', 'MIXED')),
    pnl_group VARCHAR(32) NOT NULL CHECK (pnl_group IN ('COGS_FOOD_MATERIAL', 'LABOR_COST', 'RENT_OCCUPANCY', 'PLATFORM_PAYMENT_FEE', 'MARKETING_AD', 'UTILITIES', 'SUPPLIES', 'TAX_DUES', 'OTHER_OPEX')),
    default_vat_deductible BOOLEAN NOT NULL DEFAULT TRUE,
    is_system_default BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, code)
);

CREATE TABLE vendors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    vendor_name VARCHAR(200) NOT NULL,
    registration_number VARCHAR(32),
    representative_name VARCHAR(100),
    default_expense_category_id UUID REFERENCES expense_categories(id),
    bank_code VARCHAR(16),
    account_number_encrypted BYTEA,
    account_holder_name VARCHAR(100),
    phone VARCHAR(32),
    email VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
    expense_category_id UUID NOT NULL REFERENCES expense_categories(id),
    vendor_id UUID REFERENCES vendors(id),
    source_level VARCHAR(32) NOT NULL CHECK (source_level IN ('LEVEL_1_MANUAL', 'LEVEL_2_FILE', 'LEVEL_3_API', 'LEVEL_4_RECONCILED')),
    payment_method VARCHAR(32) NOT NULL CHECK (payment_method IN ('CORPORATE_CARD', 'BUSINESS_CARD', 'BANK_TRANSFER', 'CASH', 'AUTO_DEBIT', 'PLATFORM_DEDUCTION')),
    incurred_at TIMESTAMPTZ NOT NULL,
    business_date DATE NOT NULL,
    currency_code CHAR(3) NOT NULL DEFAULT 'KRW',
    total_amount NUMERIC(19,4) NOT NULL CHECK (total_amount >= 0),
    supply_amount NUMERIC(19,4) NOT NULL,
    vat_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    qualifying_evidence_type VARCHAR(32) NOT NULL DEFAULT 'NONE' CHECK (qualifying_evidence_type IN ('TAX_INVOICE', 'EXEMPT_INVOICE', 'CARD_SLIP', 'CASH_RECEIPT', 'PAPER_SIMPLIFIED_RECEIPT', 'PAYROLL_REPORT', 'NONE')),
    vat_deduction_status VARCHAR(32) NOT NULL DEFAULT 'ELIGIBLE_CANDIDATE' CHECK (vat_deduction_status IN ('ELIGIBLE_CANDIDATE', 'NON_DEDUCTIBLE_VEHICLE', 'NON_DEDUCTIBLE_ENTERTAINMENT', 'NON_DEDUCTIBLE_EXEMPT', 'NO_QUALIFYING_EVIDENCE', 'CONFIRMED_BY_TAX_AGENT')),
    receipt_s3_key VARCHAR(512),
    ocr_confidence NUMERIC(5,2),
    memo TEXT,
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    created_by_user_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE accounting_periods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    period_Year SMALLINT NOT NULL,
    period_month SMALLINT NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSING_REVIEW', 'LOCKED')),
    locked_by_user_id UUID REFERENCES users(id),
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, period_Year, period_month)
);

CREATE TABLE ledger_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    account_code VARCHAR(32) NOT NULL,
    account_name VARCHAR(120) NOT NULL,
    account_type VARCHAR(16) NOT NULL CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    normal_balance VARCHAR(8) NOT NULL CHECK (normal_balance IN ('DEBIT', 'CREDIT')),
    is_system_account BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, account_code)
);

CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    accounting_period_id UUID REFERENCES accounting_periods(id),
    entry_number VARCHAR(64) NOT NULL,
    entry_date DATE NOT NULL,
    posting_status VARCHAR(16) NOT NULL DEFAULT 'POSTED' CHECK (posting_status IN ('POSTED', 'REVERSED')),
    source_entity_type VARCHAR(32) NOT NULL CHECK (source_entity_type IN ('SALES_TRANSACTION', 'PAYMENT_CAPTURE', 'SETTLEMENT_RECORD', 'BANK_DEPOSIT', 'EXPENSE', 'TAX_INVOICE', 'MANUAL_ADJUSTMENT', 'CORRECTION_REVERSAL')),
    source_entity_id UUID NOT NULL,
    description TEXT NOT NULL,
    total_debit_amount NUMERIC(19,4) NOT NULL CHECK (total_debit_amount >= 0),
    total_credit_amount NUMERIC(19,4) NOT NULL CHECK (total_credit_amount >= 0),
    created_by_user_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_journal_balanced CHECK (total_debit_amount = total_credit_amount),
    UNIQUE (store_id, entry_number)
);

CREATE TABLE journal_lines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE RESTRICT,
    line_no SMALLINT NOT NULL,
    ledger_account_id UUID NOT NULL REFERENCES ledger_accounts(id) ON DELETE RESTRICT,
    debit_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000 CHECK (debit_amount >= 0),
    credit_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000 CHECK (credit_amount >= 0),
    line_memo VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_line_single_side CHECK ((debit_amount > 0 AND credit_amount = 0) OR (credit_amount > 0 AND debit_amount = 0)),
    UNIQUE (journal_entry_id, line_no)
);

CREATE TABLE corrections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    original_journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE RESTRICT,
    reversal_journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE RESTRICT,
    correcting_journal_entry_id UUID REFERENCES journal_entries(id) ON DELETE RESTRICT,
    reason_code VARCHAR(64) NOT NULL,
    reason_detail TEXT NOT NULL,
    before_snapshot JSONB NOT NULL,
    after_snapshot JSONB NOT NULL,
    corrected_by_user_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tax_invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
    direction VARCHAR(16) NOT NULL CHECK (direction IN ('SALES_ISSUED', 'PURCHASE_RECEIVED')),
    invoice_type VARCHAR(32) NOT NULL CHECK (invoice_type IN ('TAX_INVOICE', 'EXEMPT_INVOICE', 'MODIFIED_TAX_INVOICE')),
    nts_confirm_number VARCHAR(64) UNIQUE,
    write_date DATE NOT NULL,
    issue_date DATE NOT NULL,
    nts_transmitted_date DATE,
    supplier_reg_no VARCHAR(32) NOT NULL,
    supplier_name VARCHAR(200) NOT NULL,
    buyer_reg_no VARCHAR(32) NOT NULL,
    buyer_name VARCHAR(200) NOT NULL,
    supply_amount NUMERIC(19,4) NOT NULL,
    tax_amount NUMERIC(19,4) NOT NULL,
    total_amount NUMERIC(19,4) NOT NULL,
    purpose_type VARCHAR(16) NOT NULL CHECK (purpose_type IN ('BILLING_CHARGE', 'RECEIPT_PAID')),
    source_level VARCHAR(32) NOT NULL,
    linked_expense_id UUID REFERENCES expenses(id),
    linked_sales_id UUID REFERENCES sales_transactions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE cash_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
    direction VARCHAR(16) NOT NULL CHECK (direction IN ('SALES_ISSUED', 'PURCHASE_EXPENSE')),
    usage_type VARCHAR(32) NOT NULL CHECK (usage_type IN ('INCOME_DEDUCTION_PERSONAL', 'EXPENSE_PROOF_BUSINESS')),
    approval_number VARCHAR(32) NOT NULL,
    transacted_at TIMESTAMPTZ NOT NULL,
    supply_amount NUMERIC(19,4) NOT NULL,
    vat_amount NUMERIC(19,4) NOT NULL,
    service_tip_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    total_amount NUMERIC(19,4) NOT NULL,
    counterparty_reg_no VARCHAR(32),
    is_canceled BOOLEAN NOT NULL DEFAULT FALSE,
    source_level VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, approval_number, transacted_at, total_amount)
);

CREATE TABLE tax_estimates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
    tax_type VARCHAR(32) NOT NULL CHECK (tax_type IN ('VAT_PRELIMINARY', 'VAT_FINAL', 'INCOME_TAX_ESTIMATE', 'WITHHOLDING_TAX')),
    tax_year SMALLINT NOT NULL,
    tax_period_code VARCHAR(16) NOT NULL,
    taxable_sales_supply NUMERIC(19,4) NOT NULL,
    output_vat_amount NUMERIC(19,4) NOT NULL,
    deductible_purchase_supply NUMERIC(19,4) NOT NULL,
    input_vat_amount NUMERIC(19,4) NOT NULL,
    deemed_input_vat_credit NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    card_issuance_tax_credit NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    estimated_payable_tax_amount NUMERIC(19,4) NOT NULL,
    calculation_formula_snapshot JSONB NOT NULL,
    disclaimer_acknowledged BOOLEAN NOT NULL DEFAULT TRUE,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tax_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    document_type VARCHAR(64) NOT NULL CHECK (document_type IN ('PAPER_RECEIPT_SCAN', 'PAPER_TAX_INVOICE', 'WITHHOLDING_LEDGER', 'VAT_EXPORT_PACKAGE', 'BUSINESS_CERTIFICATE')),
    file_name VARCHAR(255) NOT NULL,
    s3_object_key VARCHAR(512) NOT NULL,
    file_sha256 CHAR(64) NOT NULL,
    linked_expense_id UUID REFERENCES expenses(id),
    shared_with_tax_agent_id UUID REFERENCES users(id),
    share_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- Migration 0006: Domain 7 (Store Operations), Domain 8 (CRM & Analytics), Domain 9 (Marketplace & Subscription), Domain 10 (Operations) - 25 Tables
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    sku_code VARCHAR(64) NOT NULL,
    barcode VARCHAR(64),
    name VARCHAR(200) NOT NULL,
    category_name VARCHAR(100),
    selling_price NUMERIC(19,4) NOT NULL,
    standard_unit_cost NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    tax_exemption_type VARCHAR(16) NOT NULL DEFAULT 'TAXABLE',
    pos_aliases JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, sku_code)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    sales_transaction_id UUID REFERENCES sales_transactions(id),
    channel_source VARCHAR(32) NOT NULL CHECK (channel_source IN ('POS_DINE_IN', 'POS_TAKEOUT', 'KIOSK', 'TABLE_ORDER', 'BAEMIN', 'COUPANG_EATS', 'YOGIYO', 'ONLINE_MALL')),
    external_order_no VARCHAR(128) NOT NULL,
    ordered_at TIMESTAMPTZ NOT NULL,
    order_status VARCHAR(32) NOT NULL CHECK (order_status IN ('COMPLETED', 'PARTIALLY_REFUNDED', 'CANCELED')),
    item_count INT NOT NULL DEFAULT 1,
    gross_order_amount NUMERIC(19,4) NOT NULL,
    discount_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    delivery_tip_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    line_items_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, channel_source, external_order_no)
);

CREATE TABLE inventory_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id),
    item_code VARCHAR(64) NOT NULL,
    item_name VARCHAR(200) NOT NULL,
    unit_of_measure VARCHAR(32) NOT NULL DEFAULT 'EA',
    current_quantity NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    reorder_point_quantity NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    average_unit_cost NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, item_code)
);

CREATE TABLE inventory_movements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    inventory_item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE RESTRICT,
    movement_type VARCHAR(32) NOT NULL CHECK (movement_type IN ('PURCHASE_IN', 'SALE_DEDUCTION', 'WASTE_LOSS', 'STOCKTAKE_ADJUSTMENT', 'STORE_TRANSFER')),
    quantity_delta NUMERIC(19,4) NOT NULL,
    unit_cost NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    reference_order_id UUID,
    recorded_by_user_id UUID REFERENCES users(id),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    employee_name VARCHAR(100) NOT NULL,
    employment_type VARCHAR(32) NOT NULL CHECK (employment_type IN ('FULL_TIME_REGULAR', 'PART_TIME_HOURLY', 'FREELANCER_3_3', 'DAILY_WORKER')),
    pay_type VARCHAR(16) NOT NULL CHECK (pay_type IN ('HOURLY', 'MONTHLY')),
    pay_rate_amount NUMERIC(19,4) NOT NULL,
    weekly_scheduled_hours NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    joined_date DATE NOT NULL,
    resigned_date DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE store_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    metric_month DATE NOT NULL,
    target_gross_sales NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    target_operating_profit NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    target_food_cost_ratio_pct NUMERIC(5,2) NOT NULL DEFAULT 30.00,
    target_labor_cost_ratio_pct NUMERIC(5,2) NOT NULL DEFAULT 25.00,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, metric_month)
);

CREATE TABLE customer_segments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    segment_code VARCHAR(64) NOT NULL,
    segment_name VARCHAR(120) NOT NULL,
    is_identifiable_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    criteria_json JSONB NOT NULL,
    estimated_customer_count INT NOT NULL DEFAULT 0,
    avg_order_value NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    repeat_visit_rate_pct NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, segment_code)
);

CREATE TABLE review_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    platform_code VARCHAR(32) NOT NULL CHECK (platform_code IN ('GOOGLE_BUSINESS', 'NAVER_PLACE', 'KAKAO_MAP', 'BAEMIN', 'COUPANG_EATS', 'YOGIYO', 'DIRECT_RECEIPT')),
    external_place_id VARCHAR(128),
    supports_direct_api_reply BOOLEAN NOT NULL DEFAULT FALSE,
    connection_status VARCHAR(32) NOT NULL DEFAULT 'FILE_OR_MANUAL_ONLY',
    last_synced_at TIMESTAMPTZ,
    UNIQUE (store_id, platform_code)
);

CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    review_source_id UUID NOT NULL REFERENCES review_sources(id) ON DELETE CASCADE,
    external_review_id VARCHAR(128) NOT NULL,
    author_display_name VARCHAR(100),
    rating NUMERIC(3,1) NOT NULL CHECK (rating BETWEEN 1.0 AND 5.0),
    content TEXT NOT NULL,
    ordered_menu_summary VARCHAR(255),
    sentiment_label VARCHAR(16) NOT NULL CHECK (sentiment_label IN ('POSITIVE', 'NEUTRAL', 'NEGATIVE')),
    extracted_keywords JSONB NOT NULL DEFAULT '[]'::jsonb,
    reply_status VARCHAR(32) NOT NULL DEFAULT 'UNREPLIED' CHECK (reply_status IN ('UNREPLIED', 'DRAFT_READY', 'REPLIED_VIA_API', 'COPIED_FOR_MANUAL_REPLY')),
    posted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (review_source_id, external_review_id)
);

CREATE TABLE review_responses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    response_text TEXT NOT NULL,
    is_ai_generated_draft BOOLEAN NOT NULL DEFAULT FALSE,
    published_via VARCHAR(32) NOT NULL CHECK (published_via IN ('OFFICIAL_API', 'CLIPBOARD_MANUAL_COPY')),
    responded_by_user_id UUID REFERENCES users(id),
    responded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE kpi_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    granularity VARCHAR(16) NOT NULL CHECK (granularity IN ('DAILY', 'WEEKLY', 'MONTHLY')),
    period_start_date DATE NOT NULL,
    period_end_date DATE NOT NULL,
    gross_sales_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    net_sales_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    card_sales_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    delivery_sales_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    cash_sales_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    total_expense_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    fixed_expense_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    variable_expense_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    estimated_operating_profit NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    net_cash_flow_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    transaction_count INT NOT NULL DEFAULT 0,
    avg_order_value NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    data_completeness_pct NUMERIC(5,2) NOT NULL DEFAULT 100.00,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (store_id, granularity, period_start_date)
);

CREATE TABLE forecasts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    forecast_type VARCHAR(32) NOT NULL CHECK (forecast_type IN ('NEXT_7_DAYS_SALES', 'MONTH_END_SALES', 'CASH_FLOW_30D', 'INVENTORY_DEPLETION')),
    target_date DATE NOT NULL,
    predicted_value NUMERIC(19,4) NOT NULL,
    lower_bound_p10 NUMERIC(19,4) NOT NULL,
    upper_bound_p90 NUMERIC(19,4) NOT NULL,
    confidence_level VARCHAR(16) NOT NULL CHECK (confidence_level IN ('HIGH', 'MEDIUM', 'LOW', 'INSUFFICIENT_DATA')),
    model_version VARCHAR(64) NOT NULL,
    feature_importance_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE insight_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    report_type VARCHAR(32) NOT NULL CHECK (report_type IN ('MORNING_BRIEFING', 'WEEKLY_EXECUTIVE', 'MONTHLY_PNL_DIAGNOSIS', 'EXPENSE_ANOMALY_ALERT')),
    report_date DATE NOT NULL,
    headline VARCHAR(255) NOT NULL,
    summary_markdown TEXT NOT NULL,
    metrics_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE suppliers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    business_id UUID REFERENCES businesses(id),
    supplier_name VARCHAR(200) NOT NULL,
    registration_number VARCHAR(32) NOT NULL UNIQUE,
    commerce_model VARCHAR(32) NOT NULL CHECK (commerce_model IN ('FULFILLMENT_DIRECT', 'CONSIGNMENT_ESCROW', 'AFFILIATE_OUTLINK')),
    commission_rate_pct NUMERIC(5,2) NOT NULL DEFAULT 5.00,
    verification_status VARCHAR(32) NOT NULL DEFAULT 'VERIFIED' CHECK (verification_status IN ('PENDING', 'VERIFIED', 'SUSPENDED')),
    settlement_bank_code VARCHAR(16),
    settlement_account_encrypted BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE marketplace_products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    sku VARCHAR(64) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    category VARCHAR(64) NOT NULL,
    commerce_model VARCHAR(32) NOT NULL CHECK (commerce_model IN ('FULFILLMENT_DIRECT', 'CONSIGNMENT_ESCROW', 'AFFILIATE_OUTLINK')),
    retail_reference_price NUMERIC(19,4),
    b2b_unit_price NUMERIC(19,4) NOT NULL,
    vat_included BOOLEAN NOT NULL DEFAULT TRUE,
    min_order_qty INT NOT NULL DEFAULT 1,
    stock_quantity INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE marketplace_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    buyer_user_id UUID NOT NULL REFERENCES users(id),
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    order_number VARCHAR(64) NOT NULL UNIQUE,
    order_status VARCHAR(32) NOT NULL CHECK (order_status IN ('PAID', 'PREPARING', 'SHIPPED', 'DELIVERED', 'PURCHASE_CONFIRMED', 'DISPUTED', 'REFUNDED')),
    total_product_amount NUMERIC(19,4) NOT NULL,
    shipping_fee_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    coupon_discount_amount NUMERIC(19,4) NOT NULL DEFAULT 0.0000,
    paid_amount NUMERIC(19,4) NOT NULL,
    auto_linked_expense_id UUID REFERENCES expenses(id),
    tracking_company VARCHAR(64),
    tracking_number VARCHAR(64),
    ordered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_code VARCHAR(32) NOT NULL UNIQUE CHECK (plan_code IN ('FREE', 'PLUS_MONTHLY', 'PLUS_ANNUAL', 'PRO_MULTI_STORE', 'ENTERPRISE')),
    plan_name VARCHAR(100) NOT NULL,
    billing_interval VARCHAR(16) NOT NULL CHECK (billing_interval IN ('NONE', 'MONTHLY', 'ANNUAL')),
    price_amount NUMERIC(19,4) NOT NULL,
    currency_code CHAR(3) NOT NULL DEFAULT 'KRW',
    included_stores_count SMALLINT NOT NULL DEFAULT 1,
    features_json JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
    membership_id UUID NOT NULL REFERENCES memberships(id),
    status VARCHAR(32) NOT NULL CHECK (status IN ('TRIALING', 'ACTIVE', 'PAST_DUE', 'CANCELED_AT_PERIOD_END', 'CANCELED', 'EXPIRED')),
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    pg_billing_key_encrypted BYTEA,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    subscription_id UUID REFERENCES subscriptions(id),
    marketplace_order_id UUID REFERENCES marketplace_orders(id),
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    pg_provider VARCHAR(32) NOT NULL,
    pg_payment_key VARCHAR(128) UNIQUE,
    amount NUMERIC(19,4) NOT NULL CHECK (amount >= 0),
    currency_code CHAR(3) NOT NULL DEFAULT 'KRW',
    payment_status VARCHAR(32) NOT NULL CHECK (payment_status IN ('READY', 'SUCCEEDED', 'FAILED', 'PARTIALLY_REFUNDED', 'REFUNDED')),
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
    refund_amount NUMERIC(19,4) NOT NULL CHECK (refund_amount > 0),
    reason VARCHAR(255) NOT NULL,
    refund_status VARCHAR(32) NOT NULL CHECK (refund_status IN ('REQUESTED', 'COMPLETED', 'FAILED')),
    processed_by_user_id UUID REFERENCES users(id),
    refunded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID REFERENCES stores(id) ON DELETE RESTRICT,
    business_id UUID REFERENCES businesses(id) ON DELETE RESTRICT,
    actor_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    actor_role VARCHAR(64) NOT NULL,
    action_code VARCHAR(100) NOT NULL,
    target_entity_type VARCHAR(64) NOT NULL,
    target_entity_id VARCHAR(128) NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    change_summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    prev_log_hash CHAR(64) NOT NULL,
    current_log_hash CHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(32) NOT NULL CHECK (channel IN ('KAKAO_ALIMTALK', 'PUSH_FCM_APNS', 'EMAIL', 'SMS_FALLBACK', 'IN_APP')),
    template_code VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body_masked TEXT NOT NULL,
    deep_link_url VARCHAR(512),
    delivery_status VARCHAR(32) NOT NULL CHECK (delivery_status IN ('QUEUED', 'SENT', 'DELIVERED', 'READ', 'FAILED_FALLBACK_TRIGGERED', 'BLOCKED_QUIET_HOURS')),
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE background_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    queue_name VARCHAR(64) NOT NULL,
    job_type VARCHAR(100) NOT NULL,
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    payload_json JSONB NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'RETRY_SCHEDULED', 'DEAD_LETTER')),
    attempts SMALLINT NOT NULL DEFAULT 0,
    max_attempts SMALLINT NOT NULL DEFAULT 5,
    last_error TEXT,
    run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flag_key VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    regulatory_gate_status VARCHAR(32) NOT NULL DEFAULT 'APPROVED' CHECK (regulatory_gate_status IN ('APPROVED', 'REGULATORY_BLOCKED', 'PARTNER_CONTRACT_PENDING', 'SANDBOX_ONLY')),
    target_store_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_by_admin_id UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE admin_actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_user_id UUID NOT NULL REFERENCES users(id),
    second_approver_user_id UUID REFERENCES users(id),
    action_type VARCHAR(100) NOT NULL,
    target_resource VARCHAR(128) NOT NULL,
    support_ticket_id VARCHAR(64),
    justification TEXT NOT NULL,
    before_state JSONB,
    after_state JSONB,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    published_to_bus BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- Migration 0007: Row-Level Security (RLS) Policies & Performance Indexes
-- Enable RLS on all tenant-scoped tables to enforce strict multi-tenant isolation
ALTER TABLE sales_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_authorizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_captures ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_cancellations ENABLE ROW LEVEL SECURITY;
ALTER TABLE acquisition_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE payout_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE settlement_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE bank_deposits ENABLE ROW LEVEL SECURITY;
ALTER TABLE reconciliation_matches ENABLE ROW LEVEL SECURITY;
ALTER TABLE reconciliation_disputes ENABLE ROW LEVEL SECURITY;
ALTER TABLE expenses ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounting_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE journal_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE journal_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE corrections ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_estimates ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE store_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE review_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE reviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE review_responses ENABLE ROW LEVEL SECURITY;
ALTER TABLE kpi_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE forecasts ENABLE ROW LEVEL SECURITY;
ALTER TABLE insight_reports ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_sales ON sales_transactions
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_auth ON payment_authorizations
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_captures ON payment_captures
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_payouts ON payout_schedules
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_bank_deposits ON bank_deposits
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_recon_matches ON reconciliation_matches
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_expenses ON expenses
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_journal_entries ON journal_entries
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_journal_lines ON journal_lines
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_tax_invoices ON tax_invoices
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_reviews ON reviews
    USING (store_id = current_tenant_store_id());

CREATE POLICY tenant_isolation_kpi_snapshots ON kpi_snapshots
    USING (store_id = current_tenant_store_id());

-- High-Selectivity Composite Indexes for Sub-100ms Queries
CREATE INDEX idx_sales_tx_store_date ON sales_transactions (store_id, business_date DESC, channel_type);
CREATE INDEX idx_pay_auth_store_approved ON payment_authorizations (store_id, approved_at DESC, card_acquirer_code);
CREATE INDEX idx_payout_sched_store_date_status ON payout_schedules (store_id, scheduled_payout_date DESC, payout_status);
CREATE INDEX idx_bank_dep_store_date_status ON bank_deposits (store_id, transaction_date DESC, reconciliation_status);
CREATE INDEX idx_recon_match_store_status ON reconciliation_matches (store_id, match_status, matched_at DESC);
CREATE INDEX idx_expenses_store_date_cat ON expenses (store_id, business_date DESC, expense_category_id);
CREATE INDEX idx_journal_entries_store_date ON journal_entries (store_id, entry_date DESC, source_entity_type);
CREATE INDEX idx_tax_invoices_store_date_dir ON tax_invoices (store_id, write_date DESC, direction);
CREATE INDEX idx_reviews_store_posted_rating ON reviews (store_id, posted_at DESC, rating, reply_status);
CREATE INDEX idx_audit_logs_store_created ON audit_logs (store_id, created_at DESC);
CREATE INDEX idx_outbox_unpublished ON outbox_events (published_to_bus, created_at ASC) WHERE published_to_bus = FALSE;
