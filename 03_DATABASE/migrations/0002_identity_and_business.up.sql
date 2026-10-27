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
