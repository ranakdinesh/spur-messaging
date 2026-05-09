-- sql/queries/analytics.sql

-- name: GetMessageStats :one
SELECT
    count(*) as total,
    count(*) FILTER (WHERE status = 'sent') as sent,
    count(*) FILTER (WHERE status = 'delivered') as delivered,
    count(*) FILTER (WHERE status = 'read') as read,
    count(*) FILTER (WHERE status = 'failed') as failed
FROM messaging.messages
WHERE tenant_id = $1
AND ($2::timestamptz IS NULL OR created_at >= $2)
AND ($3::timestamptz IS NULL OR created_at <= $3)
AND ($4::text IS NULL OR channel = $4);

-- name: GetCampaignStats :one
SELECT
    stats
FROM messaging.campaigns
WHERE tenant_id = $1 AND id = $2;
