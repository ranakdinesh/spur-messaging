DROP POLICY IF EXISTS tenant_isolation_whatsapp_onboarding_sessions ON messaging.whatsapp_onboarding_sessions;
DROP POLICY IF EXISTS tenant_isolation_whatsapp_phone_numbers ON messaging.whatsapp_phone_numbers;
DROP POLICY IF EXISTS tenant_isolation_whatsapp_business_accounts ON messaging.whatsapp_business_accounts;

DROP INDEX IF EXISTS messaging.idx_whatsapp_onboarding_sessions_status;
DROP INDEX IF EXISTS messaging.idx_whatsapp_onboarding_sessions_tenant;
DROP INDEX IF EXISTS messaging.idx_whatsapp_phone_numbers_status;
DROP INDEX IF EXISTS messaging.idx_whatsapp_phone_numbers_waba;
DROP INDEX IF EXISTS messaging.idx_whatsapp_phone_numbers_tenant;
DROP INDEX IF EXISTS messaging.idx_whatsapp_business_accounts_meta_business;
DROP INDEX IF EXISTS messaging.idx_whatsapp_business_accounts_provider_config;
DROP INDEX IF EXISTS messaging.idx_whatsapp_business_accounts_tenant;

DROP TABLE IF EXISTS messaging.whatsapp_onboarding_sessions;
DROP TABLE IF EXISTS messaging.whatsapp_phone_numbers;
DROP TABLE IF EXISTS messaging.whatsapp_business_accounts;

DROP INDEX IF EXISTS messaging.idx_provider_configs_tenant_id;
