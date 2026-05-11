ALTER TABLE messaging.suppressions
    ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'email';

ALTER TABLE messaging.suppressions
    ADD COLUMN IF NOT EXISTS recipient TEXT;

UPDATE messaging.suppressions
SET recipient = email
WHERE recipient IS NULL;

ALTER TABLE messaging.suppressions
    ALTER COLUMN recipient SET NOT NULL;

ALTER TABLE messaging.suppressions
    ALTER COLUMN email DROP NOT NULL;

ALTER TABLE messaging.suppressions
    DROP CONSTRAINT IF EXISTS suppressions_channel_check;

ALTER TABLE messaging.suppressions
    ADD CONSTRAINT suppressions_channel_check
    CHECK (channel IN ('whatsapp', 'sms', 'email'));

DROP INDEX IF EXISTS messaging.idx_suppressions_tenant_email;
CREATE INDEX IF NOT EXISTS idx_suppressions_tenant_email
    ON messaging.suppressions (tenant_id, email)
    WHERE email IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_suppressions_tenant_recipient
    ON messaging.suppressions (tenant_id, channel, recipient);

ALTER TABLE messaging.suppressions
    DROP CONSTRAINT IF EXISTS suppressions_tenant_id_email_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_suppressions_tenant_channel_recipient
    ON messaging.suppressions (tenant_id, channel, recipient);
