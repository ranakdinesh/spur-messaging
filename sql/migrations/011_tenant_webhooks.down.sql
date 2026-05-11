DROP POLICY IF EXISTS tenant_isolation_webhook_deliveries ON messaging.webhook_deliveries;
DROP POLICY IF EXISTS tenant_isolation_webhook_endpoints ON messaging.webhook_endpoints;

DROP INDEX IF EXISTS messaging.idx_webhook_deliveries_event_endpoint;
DROP INDEX IF EXISTS messaging.idx_webhook_deliveries_retry;
DROP INDEX IF EXISTS messaging.idx_webhook_deliveries_endpoint;
DROP INDEX IF EXISTS messaging.idx_webhook_deliveries_tenant;
DROP INDEX IF EXISTS messaging.idx_webhook_endpoints_events;
DROP INDEX IF EXISTS messaging.idx_webhook_endpoints_tenant;

DROP TABLE IF EXISTS messaging.webhook_deliveries;
DROP TABLE IF EXISTS messaging.webhook_endpoints;
