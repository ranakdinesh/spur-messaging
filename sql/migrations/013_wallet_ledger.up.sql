-- Wallet ledger and configurable pricing foundation.

CREATE TABLE IF NOT EXISTS messaging.wallet_ledger (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    entry_type     TEXT NOT NULL CHECK (entry_type IN ('credit', 'debit', 'hold', 'release', 'refund', 'adjustment')),
    amount         NUMERIC(12, 6) NOT NULL,
    currency       TEXT NOT NULL DEFAULT 'INR',
    channel        TEXT CHECK (channel IN ('whatsapp', 'sms', 'email')),
    category       TEXT NOT NULL DEFAULT '',
    reference_type TEXT NOT NULL DEFAULT '',
    reference_id   UUID,
    description    TEXT NOT NULL DEFAULT '',
    metadata       JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT wallet_ledger_amount_not_zero CHECK (amount <> 0)
);

CREATE INDEX IF NOT EXISTS idx_wallet_ledger_tenant_currency
    ON messaging.wallet_ledger (tenant_id, currency, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_reference
    ON messaging.wallet_ledger (tenant_id, reference_type, reference_id, entry_type)
    WHERE reference_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS messaging.rate_cards (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID,
    channel        TEXT NOT NULL CHECK (channel IN ('whatsapp', 'sms', 'email')),
    category       TEXT NOT NULL DEFAULT 'service',
    country        TEXT NOT NULL DEFAULT 'IN',
    currency       TEXT NOT NULL DEFAULT 'INR',
    unit_price     NUMERIC(12, 6) NOT NULL CHECK (unit_price >= 0),
    effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rate_cards_lookup
    ON messaging.rate_cards (tenant_id, channel, category, country, currency, effective_from DESC);

ALTER TABLE messaging.wallet_ledger ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.rate_cards ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_wallet_ledger ON messaging.wallet_ledger
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY tenant_isolation_rate_cards ON messaging.rate_cards
    USING (tenant_id IS NULL OR tenant_id = current_setting('app.tenant_id')::uuid);
