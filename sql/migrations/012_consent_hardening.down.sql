DROP INDEX IF EXISTS messaging.idx_consent_records_expiry;

ALTER TABLE messaging.consent_records
    DROP COLUMN IF EXISTS confirmed_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS locale,
    DROP COLUMN IF EXISTS keyword;

ALTER TABLE messaging.consent_records
    DROP CONSTRAINT IF EXISTS consent_records_status_check;
ALTER TABLE messaging.consent_records
    ADD CONSTRAINT consent_records_status_check
        CHECK (status IN ('opted_in', 'opted_out'));

ALTER TABLE messaging.contacts
    DROP CONSTRAINT IF EXISTS contacts_opt_in_whatsapp_check;
ALTER TABLE messaging.contacts
    DROP CONSTRAINT IF EXISTS contacts_opt_in_sms_check;
ALTER TABLE messaging.contacts
    DROP CONSTRAINT IF EXISTS contacts_opt_in_email_check;

UPDATE messaging.contacts
SET opt_in_whatsapp = 'pending'
WHERE opt_in_whatsapp = 'double_opt_in_pending';

UPDATE messaging.contacts
SET opt_in_sms = 'pending'
WHERE opt_in_sms = 'double_opt_in_pending';

UPDATE messaging.contacts
SET opt_in_email = 'pending'
WHERE opt_in_email = 'double_opt_in_pending';

ALTER TABLE messaging.contacts
    ADD CONSTRAINT contacts_opt_in_whatsapp_check
        CHECK (opt_in_whatsapp IN ('pending', 'opted_in', 'opted_out'));
ALTER TABLE messaging.contacts
    ADD CONSTRAINT contacts_opt_in_sms_check
        CHECK (opt_in_sms IN ('pending', 'opted_in', 'opted_out'));
ALTER TABLE messaging.contacts
    ADD CONSTRAINT contacts_opt_in_email_check
        CHECK (opt_in_email IN ('pending', 'opted_in', 'opted_out'));
