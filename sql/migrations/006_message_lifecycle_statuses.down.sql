UPDATE messaging.messages
SET status = CASE
    WHEN status IN ('created', 'validated', 'provider_submitted') THEN 'queued'
    WHEN status = 'opened' THEN 'read'
    WHEN status IN ('clicked', 'replied') THEN 'read'
    WHEN status IN ('cancelled', 'expired', 'suppressed') THEN 'failed'
    ELSE status
END;

ALTER TABLE messaging.messages
    DROP CONSTRAINT IF EXISTS messages_status_check;

ALTER TABLE messaging.messages
    ADD CONSTRAINT messages_status_check
    CHECK (status IN ('queued', 'sent', 'delivered', 'read', 'failed'));

ALTER TABLE messaging.messages
    DROP COLUMN IF EXISTS updated_at;
