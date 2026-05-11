CREATE TABLE IF NOT EXISTS messaging.conversations (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    channel              TEXT NOT NULL CHECK (channel IN ('whatsapp', 'sms', 'email')),
    recipient            TEXT NOT NULL,
    status               TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'pending', 'resolved', 'closed')),
    handoff_status       TEXT NOT NULL DEFAULT 'bot' CHECK (handoff_status IN ('bot', 'agent', 'closed', 'waiting_customer')),
    assigned_agent_id    UUID,
    assigned_team        TEXT,
    priority             TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    tags                 TEXT[] NOT NULL DEFAULT '{}',
    internal_notes       JSONB NOT NULL DEFAULT '[]',
    last_inbound_at      TIMESTAMPTZ,
    last_outbound_at     TIMESTAMPTZ,
    service_window_until TIMESTAMPTZ,
    first_response_due_at TIMESTAMPTZ,
    resolution_due_at    TIMESTAMPTZ,
    closed_at            TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, channel, recipient)
);

CREATE INDEX IF NOT EXISTS idx_conversations_tenant_recipient
    ON messaging.conversations (tenant_id, channel, recipient);

CREATE INDEX IF NOT EXISTS idx_conversations_service_window
    ON messaging.conversations (tenant_id, channel, service_window_until)
    WHERE service_window_until IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_inbox
    ON messaging.conversations (tenant_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_assignee
    ON messaging.conversations (tenant_id, assigned_agent_id, status);
CREATE INDEX IF NOT EXISTS idx_conversations_tags
    ON messaging.conversations USING GIN (tags);

ALTER TABLE messaging.conversations ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_conversations ON messaging.conversations;
CREATE POLICY tenant_isolation_conversations ON messaging.conversations
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
