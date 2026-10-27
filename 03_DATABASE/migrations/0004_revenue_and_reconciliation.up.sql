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
