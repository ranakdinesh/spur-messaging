CREATE TABLE IF NOT EXISTS messaging.consent_records (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,
    contact_id UUID NOT NULL REFERENCES messaging.contacts(id) ON DELETE CASCADE,
    channel    TEXT NOT NULL CHECK (channel IN ('whatsapp', 'sms', 'email')),
    status     TEXT NOT NULL CHECK (status IN ('opted_in', 'opted_out')),
    source     TEXT NOT NULL DEFAULT 'manual',
    purpose    TEXT NOT NULL DEFAULT '',
    proof      TEXT NOT NULL DEFAULT '',
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    brand      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_consent_records_contact
    ON messaging.consent_records (tenant_id, contact_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_consent_records_channel
    ON messaging.consent_records (tenant_id, channel, status, created_at DESC);

ALTER TABLE messaging.consent_records ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_consent_records ON messaging.consent_records;
CREATE POLICY tenant_isolation_consent_records ON messaging.consent_records
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
