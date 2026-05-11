DELETE FROM messaging.suppressions
WHERE channel <> 'email';

UPDATE messaging.suppressions
SET email = recipient
WHERE email IS NULL;

DROP INDEX IF EXISTS messaging.idx_suppressions_tenant_channel_recipient;
DROP INDEX IF EXISTS messaging.idx_suppressions_tenant_recipient;
DROP INDEX IF EXISTS messaging.idx_suppressions_tenant_email;

ALTER TABLE messaging.suppressions
    DROP CONSTRAINT IF EXISTS suppressions_channel_check;

ALTER TABLE messaging.suppressions
    ALTER COLUMN email SET NOT NULL;

ALTER TABLE messaging.suppressions
    ADD CONSTRAINT suppressions_tenant_id_email_key UNIQUE (tenant_id, email);

CREATE INDEX IF NOT EXISTS idx_suppressions_tenant_email
    ON messaging.suppressions (tenant_id, email);

ALTER TABLE messaging.suppressions
    DROP COLUMN IF EXISTS recipient;

ALTER TABLE messaging.suppressions
    DROP COLUMN IF EXISTS channel;
