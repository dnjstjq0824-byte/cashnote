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
