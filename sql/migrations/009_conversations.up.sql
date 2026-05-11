CREATE TABLE IF NOT EXISTS messaging.conversations (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    channel              TEXT NOT NULL CHECK (channel IN ('whatsapp', 'sms', 'email')),
    recipient            TEXT NOT NULL,
    status               TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    handoff_status       TEXT NOT NULL DEFAULT 'bot' CHECK (handoff_status IN ('bot', 'agent', 'closed', 'waiting_customer')),
    last_inbound_at      TIMESTAMPTZ,
    last_outbound_at     TIMESTAMPTZ,
    service_window_until TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, channel, recipient)
);

CREATE INDEX IF NOT EXISTS idx_conversations_tenant_recipient
    ON messaging.conversations (tenant_id, channel, recipient);

CREATE INDEX IF NOT EXISTS idx_conversations_service_window
    ON messaging.conversations (tenant_id, channel, service_window_until)
    WHERE service_window_until IS NOT NULL;

ALTER TABLE messaging.conversations ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_conversations ON messaging.conversations;
CREATE POLICY tenant_isolation_conversations ON messaging.conversations
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
