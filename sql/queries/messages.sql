-- sql/queries/messages.sql

-- name: CreateMessage :one
INSERT INTO messaging.messages (
    tenant_id, campaign_id, conversation_id, channel, direction, recipient, sender,
    message_type, template_id, template_name, template_params, text_body,
    media_url, media_type, provider_message_id, status, error_code, error_message,
    cost, sent_at, delivered_at, read_at, failed_at, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
) RETURNING *;

-- name: GetMessageByID :one
SELECT * FROM messaging.messages
WHERE tenant_id = $1 AND id = $2;

-- name: ListMessages :many
SELECT *, count(*) OVER() as total_count
FROM messaging.messages
WHERE tenant_id = $1
AND ($2::text IS NULL OR channel = $2)
AND ($3::text IS NULL OR status = $3)
AND ($4::text IS NULL OR recipient = $4)
AND ($5::uuid IS NULL OR campaign_id = $5)
AND ($6::timestamptz IS NULL OR created_at >= $6)
AND ($7::timestamptz IS NULL OR created_at <= $7)
ORDER BY created_at DESC
LIMIT $8 OFFSET $9;

-- name: UpdateMessageStatus :exec
UPDATE messaging.messages
SET status = $3,
    provider_message_id = COALESCE($4, provider_message_id),
    updated_at = now()
WHERE tenant_id = $1 AND id = $2;

-- name: UpdateMessageStatusByProviderID :exec
UPDATE messaging.messages
SET status = $2,
    delivered_at = CASE WHEN $2 = 'delivered' THEN $3 ELSE delivered_at END,
    read_at = CASE WHEN $2 = 'read' THEN $3 ELSE read_at END,
    failed_at = CASE WHEN $2 = 'failed' THEN $3 ELSE failed_at END,
    updated_at = now()
WHERE provider_message_id = $1;

-- name: GetMessagesByCampaignID :many
SELECT *, count(*) OVER() as total_count
FROM messaging.messages
WHERE tenant_id = $1 AND campaign_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
