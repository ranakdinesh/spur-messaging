ALTER TABLE messaging.provider_configs
    DROP CONSTRAINT IF EXISTS provider_configs_provider_check;

ALTER TABLE messaging.provider_configs
    ADD CONSTRAINT provider_configs_provider_check
    CHECK (provider IN ('meta_cloud', 'msg91', 'twilio', 'sendgrid', 'mailgun', 'postmark', 'smtp', 'dev_email'));
