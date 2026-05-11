-- sql/migrations/004_email_system.up.sql

-- HTML email templates (separate from WhatsApp templates)
CREATE TABLE messaging.email_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    name          TEXT NOT NULL,
    subject       TEXT NOT NULL,
    preview_text  TEXT NOT NULL DEFAULT '',
    html_body     TEXT NOT NULL,
    text_body     TEXT NOT NULL DEFAULT '',    -- auto-generated from HTML if empty
    category      TEXT NOT NULL DEFAULT 'transactional' CHECK (category IN ('transactional', 'marketing', 'notification')),
    variables     TEXT[] NOT NULL DEFAULT '{}', -- variable names used in template
    is_active     BOOLEAN NOT NULL DEFAULT true,
    version       INT NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

-- Email lifecycle events (open, click, bounce, complaint, etc.)
-- High-volume table — partitioned by month if needed at scale
CREATE TABLE messaging.email_events (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    message_id        UUID NOT NULL REFERENCES messaging.messages(id),
    campaign_id       UUID,
    event_type        TEXT NOT NULL CHECK (event_type IN (
        'delivered', 'bounce', 'soft_bounce', 'open', 'click',
        'unsubscribe', 'complaint', 'dropped', 'deferred'
    )),
    recipient         TEXT NOT NULL,
    timestamp         TIMESTAMPTZ NOT NULL,
    provider_event_id TEXT,                   -- for dedup
    user_agent        TEXT,
    ip_address        TEXT,
    url               TEXT,                   -- clicked URL
    bounce_type       TEXT,                   -- 'hard' or 'soft'
    bounce_reason     TEXT,                   -- SMTP error
    complaint_feedback TEXT,
    raw_payload       JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_email_events_provider_id ON messaging.email_events (provider_event_id) WHERE provider_event_id IS NOT NULL;
CREATE INDEX idx_email_events_message ON messaging.email_events (tenant_id, message_id);
CREATE INDEX idx_email_events_campaign ON messaging.email_events (tenant_id, campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX idx_email_events_type_time ON messaging.email_events (tenant_id, event_type, timestamp DESC);

-- Unsubscribe list (multi-level: global, category, campaign)
CREATE TABLE messaging.unsubscribes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    email       TEXT NOT NULL,
    scope       TEXT NOT NULL CHECK (scope IN ('global', 'category', 'campaign')),
    campaign_id UUID,                  -- only if scope='campaign'
    reason      TEXT NOT NULL DEFAULT 'manual', -- 'manual', 'link_click', 'complaint', 'bounce'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_unsubscribes_tenant_email_scope ON messaging.unsubscribes (tenant_id, email, scope, COALESCE(campaign_id, '00000000-0000-0000-0000-000000000000'::uuid));
CREATE INDEX idx_unsubscribes_tenant_email ON messaging.unsubscribes (tenant_id, email);

-- Suppression list (hard bounces, complaints — NEVER send to these)
CREATE TABLE messaging.suppressions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    channel     TEXT NOT NULL DEFAULT 'email' CHECK (channel IN ('whatsapp', 'sms', 'email')),
    recipient   TEXT NOT NULL,
    email       TEXT,
    reason      TEXT NOT NULL CHECK (reason IN ('hard_bounce', 'complaint', 'manual', 'invalid')),
    source      TEXT NOT NULL DEFAULT 'manual', -- 'bounce_webhook', 'complaint_webhook', 'manual', 'import'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, channel, recipient)
);

CREATE INDEX idx_suppressions_tenant_recipient ON messaging.suppressions (tenant_id, channel, recipient);
CREATE INDEX idx_suppressions_tenant_email ON messaging.suppressions (tenant_id, email) WHERE email IS NOT NULL;

-- RLS
ALTER TABLE messaging.email_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.email_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.unsubscribes ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.suppressions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_email_templates ON messaging.email_templates
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_email_events ON messaging.email_events
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_unsubscribes ON messaging.unsubscribes
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_suppressions ON messaging.suppressions
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
