DROP INDEX IF EXISTS messaging.idx_messages_tenant_idempotency;

ALTER TABLE messaging.messages
    DROP COLUMN IF EXISTS idempotency_key;
