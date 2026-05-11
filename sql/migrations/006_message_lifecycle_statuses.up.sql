ALTER TABLE messaging.messages
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE messaging.messages
    DROP CONSTRAINT IF EXISTS messages_status_check;

ALTER TABLE messaging.messages
    ADD CONSTRAINT messages_status_check
    CHECK (status IN (
        'created',
        'validated',
        'queued',
        'provider_submitted',
        'sent',
        'delivered',
        'read',
        'opened',
        'clicked',
        'replied',
        'failed',
        'cancelled',
        'expired',
        'suppressed'
    ));
