-- sql/migrations/004_email_system.down.sql

DROP POLICY IF EXISTS tenant_isolation_suppressions ON messaging.suppressions;
DROP POLICY IF EXISTS tenant_isolation_unsubscribes ON messaging.unsubscribes;
DROP POLICY IF EXISTS tenant_isolation_email_events ON messaging.email_events;
DROP POLICY IF EXISTS tenant_isolation_email_templates ON messaging.email_templates;

DROP TABLE IF EXISTS messaging.suppressions;
DROP TABLE IF EXISTS messaging.unsubscribes;
DROP TABLE IF EXISTS messaging.email_events;
DROP TABLE IF EXISTS messaging.email_templates;
