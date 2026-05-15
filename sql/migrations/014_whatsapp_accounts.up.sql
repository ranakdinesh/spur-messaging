-- WhatsApp Tech Provider onboarding state.
--
-- Design note:
-- provider_configs remains the generic credential/provider boundary. WhatsApp
-- account records reference provider_configs through provider_config_id when an
-- onboarded WABA is backed by stored encrypted Meta credentials. This keeps
-- provider_configs channel-agnostic and avoids adding WABA-specific foreign keys
-- to the generic provider table. Phone numbers reference the WABA aggregate by
-- (tenant_id, waba_id).

CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_configs_tenant_id
    ON messaging.provider_configs (tenant_id, id);

CREATE TABLE IF NOT EXISTS messaging.whatsapp_business_accounts (
    id                           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                    UUID NOT NULL,
    meta_business_id              TEXT NOT NULL,
    waba_id                      TEXT NOT NULL,
    name                         TEXT NOT NULL DEFAULT '',
    currency                     TEXT NOT NULL DEFAULT '',
    timezone_id                  TEXT NOT NULL DEFAULT '',
    business_verification_status TEXT NOT NULL DEFAULT 'unknown'
        CHECK (business_verification_status IN ('not_verified', 'pending', 'verified', 'rejected', 'unknown')),
    onboarding_status            TEXT NOT NULL DEFAULT 'pending'
        CHECK (onboarding_status IN ('pending', 'in_progress', 'completed', 'failed', 'expired', 'cancelled')),
    provider_config_id           UUID,
    last_synced_at               TIMESTAMPTZ,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT whatsapp_business_accounts_tenant_waba_unique UNIQUE (tenant_id, waba_id),
    CONSTRAINT whatsapp_business_accounts_provider_config_fk
        FOREIGN KEY (tenant_id, provider_config_id)
        REFERENCES messaging.provider_configs (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_business_accounts_tenant
    ON messaging.whatsapp_business_accounts (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_whatsapp_business_accounts_provider_config
    ON messaging.whatsapp_business_accounts (tenant_id, provider_config_id)
    WHERE provider_config_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_whatsapp_business_accounts_meta_business
    ON messaging.whatsapp_business_accounts (tenant_id, meta_business_id);

CREATE TABLE IF NOT EXISTS messaging.whatsapp_phone_numbers (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    waba_id                  TEXT NOT NULL,
    phone_number_id          TEXT NOT NULL,
    display_phone_number     TEXT NOT NULL DEFAULT '',
    verified_name            TEXT NOT NULL DEFAULT '',
    quality_rating           TEXT NOT NULL DEFAULT 'unknown'
        CHECK (quality_rating IN ('green', 'yellow', 'red', 'unknown')),
    messaging_limit_tier     TEXT NOT NULL DEFAULT '',
    status                   TEXT NOT NULL DEFAULT 'unknown'
        CHECK (status IN ('pending_verification', 'connected', 'disconnected', 'flagged', 'restricted', 'banned', 'unknown')),
    code_verification_status TEXT NOT NULL DEFAULT 'unknown'
        CHECK (code_verification_status IN ('not_verified', 'pending', 'verified', 'failed', 'expired', 'unknown')),
    last_synced_at           TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT whatsapp_phone_numbers_tenant_phone_unique UNIQUE (tenant_id, phone_number_id),
    CONSTRAINT whatsapp_phone_numbers_waba_fk
        FOREIGN KEY (tenant_id, waba_id)
        REFERENCES messaging.whatsapp_business_accounts (tenant_id, waba_id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_phone_numbers_tenant
    ON messaging.whatsapp_phone_numbers (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_whatsapp_phone_numbers_waba
    ON messaging.whatsapp_phone_numbers (tenant_id, waba_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_phone_numbers_status
    ON messaging.whatsapp_phone_numbers (tenant_id, status, quality_rating);

CREATE TABLE IF NOT EXISTS messaging.whatsapp_onboarding_sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    state         TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'expired', 'cancelled')),
    error_message TEXT,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT whatsapp_onboarding_sessions_state_unique UNIQUE (tenant_id, state)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_onboarding_sessions_tenant
    ON messaging.whatsapp_onboarding_sessions (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_whatsapp_onboarding_sessions_status
    ON messaging.whatsapp_onboarding_sessions (tenant_id, status, created_at DESC);

ALTER TABLE messaging.whatsapp_business_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.whatsapp_phone_numbers ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.whatsapp_onboarding_sessions ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_whatsapp_business_accounts ON messaging.whatsapp_business_accounts;
CREATE POLICY tenant_isolation_whatsapp_business_accounts ON messaging.whatsapp_business_accounts
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

DROP POLICY IF EXISTS tenant_isolation_whatsapp_phone_numbers ON messaging.whatsapp_phone_numbers;
CREATE POLICY tenant_isolation_whatsapp_phone_numbers ON messaging.whatsapp_phone_numbers
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

DROP POLICY IF EXISTS tenant_isolation_whatsapp_onboarding_sessions ON messaging.whatsapp_onboarding_sessions;
CREATE POLICY tenant_isolation_whatsapp_onboarding_sessions ON messaging.whatsapp_onboarding_sessions
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
