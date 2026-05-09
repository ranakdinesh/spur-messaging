-- sql/migrations/001_messaging_schema.down.sql

DROP POLICY IF EXISTS tenant_isolation_campaigns ON messaging.campaigns;
DROP POLICY IF EXISTS tenant_isolation_segments ON messaging.segments;
DROP POLICY IF EXISTS tenant_isolation_messages ON messaging.messages;
DROP POLICY IF EXISTS tenant_isolation_contacts ON messaging.contacts;
DROP POLICY IF EXISTS tenant_isolation_templates ON messaging.templates;
DROP POLICY IF EXISTS tenant_isolation_provider_configs ON messaging.provider_configs;

DROP TABLE IF EXISTS messaging.campaigns;
DROP TABLE IF EXISTS messaging.segment_contacts;
DROP TABLE IF EXISTS messaging.segments;
DROP TABLE IF EXISTS messaging.messages;
DROP TABLE IF EXISTS messaging.contacts;
DROP TABLE IF EXISTS messaging.templates;
DROP TABLE IF EXISTS messaging.provider_configs;

DROP SCHEMA IF EXISTS messaging;
