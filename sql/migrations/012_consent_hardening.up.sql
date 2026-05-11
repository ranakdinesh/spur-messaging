-- Consent expiry, double opt-in pending state, and inbound keyword evidence.

ALTER TABLE messaging.contacts
    DROP CONSTRAINT IF EXISTS contacts_opt_in_whatsapp_check;
ALTER TABLE messaging.contacts
    DROP CONSTRAINT IF EXISTS contacts_opt_in_sms_check;
ALTER TABLE messaging.contacts
    DROP CONSTRAINT IF EXISTS contacts_opt_in_email_check;

ALTER TABLE messaging.contacts
    ADD CONSTRAINT contacts_opt_in_whatsapp_check
        CHECK (opt_in_whatsapp IN ('pending', 'double_opt_in_pending', 'opted_in', 'opted_out'));
ALTER TABLE messaging.contacts
    ADD CONSTRAINT contacts_opt_in_sms_check
        CHECK (opt_in_sms IN ('pending', 'double_opt_in_pending', 'opted_in', 'opted_out'));
ALTER TABLE messaging.contacts
    ADD CONSTRAINT contacts_opt_in_email_check
        CHECK (opt_in_email IN ('pending', 'double_opt_in_pending', 'opted_in', 'opted_out'));

ALTER TABLE messaging.consent_records
    DROP CONSTRAINT IF EXISTS consent_records_status_check;

ALTER TABLE messaging.consent_records
    ADD CONSTRAINT consent_records_status_check
        CHECK (status IN ('double_opt_in_pending', 'opted_in', 'opted_out'));

ALTER TABLE messaging.consent_records
    ADD COLUMN IF NOT EXISTS keyword TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_consent_records_expiry
    ON messaging.consent_records (tenant_id, channel, expires_at)
    WHERE expires_at IS NOT NULL;
