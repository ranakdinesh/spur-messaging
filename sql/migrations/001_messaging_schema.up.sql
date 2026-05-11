-- sql/migrations/001_messaging_schema.up.sql

CREATE SCHEMA IF NOT EXISTS messaging;

-- Provider configurations (tenant's own WhatsApp/SMS/Email credentials)
CREATE TABLE messaging.provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    channel         TEXT NOT NULL CHECK (channel IN ('whatsapp', 'sms', 'email')),
    provider        TEXT NOT NULL CHECK (provider IN ('meta_cloud', 'msg91', 'twilio', 'sendgrid', 'mailgun', 'postmark')),
    credentials     BYTEA NOT NULL,         -- AES-256-GCM encrypted
    webhook_secret  TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    phone_number_id TEXT,                    -- WhatsApp: Meta phone number ID
    waba_id         TEXT,                    -- WhatsApp: Business Account ID
    business_id     TEXT,                    -- Meta Business ID
    display_phone   TEXT,                    -- Display phone number
    from_email      TEXT,                    -- Email: verified sender address
    from_name       TEXT,                    -- Email: sender display name
    reply_to_email  TEXT,                    -- Email: reply-to address
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, channel, provider)
);

-- Message templates
CREATE TABLE messaging.templates (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    channel              TEXT NOT NULL DEFAULT 'whatsapp',
    name                 TEXT NOT NULL,
    language             TEXT NOT NULL DEFAULT 'en',
    category             TEXT NOT NULL CHECK (category IN ('marketing', 'utility', 'authentication')),
    components           JSONB NOT NULL DEFAULT '[]',
    status               TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'approved', 'rejected')),
    provider_template_id TEXT,
    rejection_reason     TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name, language)
);

-- Contacts
CREATE TABLE messaging.contacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    phone           TEXT,                    -- E.164 format
    email           TEXT,
    name            TEXT,
    attributes      JSONB NOT NULL DEFAULT '{}',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    opt_in_whatsapp TEXT NOT NULL DEFAULT 'pending' CHECK (opt_in_whatsapp IN ('pending', 'opted_in', 'opted_out')),
    opt_in_sms      TEXT NOT NULL DEFAULT 'pending' CHECK (opt_in_sms IN ('pending', 'opted_in', 'opted_out')),
    opt_in_email    TEXT NOT NULL DEFAULT 'pending' CHECK (opt_in_email IN ('pending', 'opted_in', 'opted_out')),
    opted_in_at     TIMESTAMPTZ,
    opted_out_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_contacts_tenant_phone ON messaging.contacts (tenant_id, phone) WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX idx_contacts_tenant_email ON messaging.contacts (tenant_id, email) WHERE email IS NOT NULL;

-- Messages (outbound and inbound)
CREATE TABLE messaging.messages (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    campaign_id         UUID,
    conversation_id     UUID,
    channel             TEXT NOT NULL,
    direction           TEXT NOT NULL DEFAULT 'outbound' CHECK (direction IN ('outbound', 'inbound')),
    recipient           TEXT NOT NULL,
    sender              TEXT,
    message_type        TEXT NOT NULL CHECK (message_type IN ('template', 'text', 'media', 'interactive', 'location')),
    template_id         UUID REFERENCES messaging.templates(id),
    template_name       TEXT,
    template_params     JSONB,
    text_body           TEXT,
    media_url           TEXT,
    media_type          TEXT,
    provider_message_id TEXT,
    idempotency_key     TEXT,
    status              TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('created', 'validated', 'queued', 'provider_submitted', 'sent', 'delivered', 'read', 'opened', 'clicked', 'replied', 'failed', 'cancelled', 'expired', 'suppressed')),
    error_code          TEXT,
    error_message       TEXT,
    cost                DECIMAL(10, 6),
    sent_at             TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    read_at             TIMESTAMPTZ,
    failed_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata            JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_messages_tenant_status ON messaging.messages (tenant_id, status);
CREATE INDEX idx_messages_tenant_recipient ON messaging.messages (tenant_id, recipient);
CREATE INDEX idx_messages_tenant_campaign ON messaging.messages (tenant_id, campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX idx_messages_provider_id ON messaging.messages (provider_message_id) WHERE provider_message_id IS NOT NULL;
CREATE INDEX idx_messages_tenant_created ON messaging.messages (tenant_id, created_at DESC);
CREATE UNIQUE INDEX idx_messages_tenant_idempotency ON messaging.messages (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Segments
CREATE TABLE messaging.segments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    is_dynamic  BOOLEAN NOT NULL DEFAULT false,
    rules       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

-- Static segment membership (for non-dynamic segments)
CREATE TABLE messaging.segment_contacts (
    segment_id UUID NOT NULL REFERENCES messaging.segments(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES messaging.contacts(id) ON DELETE CASCADE,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (segment_id, contact_id)
);

-- Campaigns
CREATE TABLE messaging.campaigns (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,
    channel         TEXT NOT NULL,
    template_id     UUID NOT NULL REFERENCES messaging.templates(id),
    template_params JSONB NOT NULL DEFAULT '{}',
    segment_id      UUID REFERENCES messaging.segments(id),
    contact_ids     UUID[],                  -- explicit contact list (alternative to segment)
    scheduled_at    TIMESTAMPTZ,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'scheduled', 'running', 'paused', 'completed', 'failed')),
    stats           JSONB NOT NULL DEFAULT '{"total":0,"queued":0,"sent":0,"delivered":0,"read":0,"failed":0}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RLS policies
ALTER TABLE messaging.provider_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.segment_contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.campaigns ENABLE ROW LEVEL SECURITY;

-- RLS policy: tenant can only see their own data
-- The GUC variable app.tenant_id is set by the identity module's middleware
CREATE POLICY tenant_isolation_provider_configs ON messaging.provider_configs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_templates ON messaging.templates
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_contacts ON messaging.contacts
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_messages ON messaging.messages
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_segments ON messaging.segments
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_campaigns ON messaging.campaigns
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
