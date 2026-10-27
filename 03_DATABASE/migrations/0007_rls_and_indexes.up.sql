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
