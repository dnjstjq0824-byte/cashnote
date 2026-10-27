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
