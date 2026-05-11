ALTER TABLE messaging.messages
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_tenant_idempotency
    ON messaging.messages (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
