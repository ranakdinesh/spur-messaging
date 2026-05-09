-- sql/queries/email_events.sql

-- name: CreateEmailEvent :one
INSERT INTO messaging.email_events (
    tenant_id, message_id, campaign_id, event_type, recipient, timestamp,
    provider_event_id, user_agent, ip_address, url, bounce_type, bounce_reason,
    complaint_feedback, raw_payload
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: CreateEmailEventBatch :copyfrom
INSERT INTO messaging.email_events (
    tenant_id, message_id, campaign_id, event_type, recipient, timestamp,
    provider_event_id, user_agent, ip_address, url, bounce_type, bounce_reason,
    complaint_feedback, raw_payload
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
);

-- name: GetEmailEventsByMessageID :many
SELECT * FROM messaging.email_events
WHERE tenant_id = $1 AND message_id = $2
ORDER BY timestamp ASC;

-- name: GetEmailEventsByCampaignID :many
SELECT *, count(*) OVER() as total_count
FROM messaging.email_events
WHERE tenant_id = $1 AND campaign_id = $2
AND ($3::text IS NULL OR event_type = $3)
ORDER BY timestamp DESC
LIMIT $4 OFFSET $5;

-- name: ExistsByProviderEventID :one
SELECT EXISTS(
    SELECT 1 FROM messaging.email_events
    WHERE provider_event_id = $1
);

-- name: GetEmailStats :one
SELECT
    COUNT(*) FILTER (WHERE event_type = 'delivered') as delivered,
    COUNT(*) FILTER (WHERE event_type = 'open') as opens,
    COUNT(DISTINCT recipient) FILTER (WHERE event_type = 'open') as unique_opens,
    COUNT(*) FILTER (WHERE event_type = 'click') as clicks,
    COUNT(DISTINCT recipient) FILTER (WHERE event_type = 'click') as unique_clicks,
    COUNT(*) FILTER (WHERE event_type = 'bounce') as bounces,
    COUNT(*) FILTER (WHERE event_type = 'bounce' AND bounce_type = 'hard') as hard_bounces,
    COUNT(*) FILTER (WHERE event_type = 'soft_bounce' OR (event_type = 'bounce' AND bounce_type = 'soft')) as soft_bounces,
    COUNT(*) FILTER (WHERE event_type = 'complaint') as complaints,
    COUNT(*) FILTER (WHERE event_type = 'unsubscribe') as unsubscribes
FROM messaging.email_events
WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3;

-- name: GetEmailCampaignStats :one
SELECT
    COUNT(*) FILTER (WHERE event_type = 'delivered') as delivered,
    COUNT(*) FILTER (WHERE event_type = 'open') as opens,
    COUNT(DISTINCT recipient) FILTER (WHERE event_type = 'open') as unique_opens,
    COUNT(*) FILTER (WHERE event_type = 'click') as clicks,
    COUNT(DISTINCT recipient) FILTER (WHERE event_type = 'click') as unique_clicks,
    COUNT(*) FILTER (WHERE event_type = 'bounce') as bounces,
    COUNT(*) FILTER (WHERE event_type = 'bounce' AND bounce_type = 'hard') as hard_bounces,
    COUNT(*) FILTER (WHERE event_type = 'soft_bounce' OR (event_type = 'bounce' AND bounce_type = 'soft')) as soft_bounces,
    COUNT(*) FILTER (WHERE event_type = 'complaint') as complaints,
    COUNT(*) FILTER (WHERE event_type = 'unsubscribe') as unsubscribes
FROM messaging.email_events
WHERE tenant_id = $1 AND campaign_id = $2;
