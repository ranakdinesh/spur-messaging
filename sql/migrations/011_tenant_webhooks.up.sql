-- Tenant-facing webhook endpoints and delivery logs.

CREATE TABLE IF NOT EXISTS messaging.webhook_endpoints (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    url           TEXT NOT NULL,
    secret        TEXT NOT NULL,
    events        TEXT[] NOT NULL DEFAULT '{}',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    failure_count INTEGER NOT NULL DEFAULT 0,
    disabled_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT webhook_url_https CHECK (url LIKE 'https://%'),
    CONSTRAINT webhook_secret_min_length CHECK (length(secret) >= 32)
);

CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_tenant
    ON messaging.webhook_endpoints (tenant_id, is_active, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_events
    ON messaging.webhook_endpoints USING GIN (events);

CREATE TABLE IF NOT EXISTS messaging.webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    webhook_id      UUID NOT NULL REFERENCES messaging.webhook_endpoints(id) ON DELETE CASCADE,
    event_id        UUID NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'succeeded', 'retrying', 'failed')),
    attempt_count   INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    last_attempt_at TIMESTAMPTZ,
    response_status INTEGER,
    response_body   TEXT,
    error_message   TEXT,
    signature       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_tenant
    ON messaging.webhook_deliveries (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint
    ON messaging.webhook_deliveries (tenant_id, webhook_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_retry
    ON messaging.webhook_deliveries (status, next_attempt_at)
    WHERE status = 'retrying';
CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_deliveries_event_endpoint
    ON messaging.webhook_deliveries (webhook_id, event_id);

ALTER TABLE messaging.webhook_endpoints ENABLE ROW LEVEL SECURITY;
ALTER TABLE messaging.webhook_deliveries ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_webhook_endpoints ON messaging.webhook_endpoints;
CREATE POLICY tenant_isolation_webhook_endpoints ON messaging.webhook_endpoints
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

DROP POLICY IF EXISTS tenant_isolation_webhook_deliveries ON messaging.webhook_deliveries;
CREATE POLICY tenant_isolation_webhook_deliveries ON messaging.webhook_deliveries
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
