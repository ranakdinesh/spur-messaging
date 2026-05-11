-- sql/queries/webhooks.sql

-- name: CreateWebhookEndpoint :one
INSERT INTO messaging.webhook_endpoints (
    id, tenant_id, url, secret, events, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetWebhookEndpoint :one
SELECT * FROM messaging.webhook_endpoints
WHERE tenant_id = $1 AND id = $2;

-- name: ListWebhookEndpoints :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.webhook_endpoints
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWebhookEndpoint :one
UPDATE messaging.webhook_endpoints
SET
    url = $3,
    secret = $4,
    events = $5,
    is_active = $6,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteWebhookEndpoint :exec
DELETE FROM messaging.webhook_endpoints
WHERE tenant_id = $1 AND id = $2;

-- name: CreateWebhookDelivery :one
INSERT INTO messaging.webhook_deliveries (
    id, tenant_id, webhook_id, event_id, event_type, payload, status, attempt_count
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetWebhookDelivery :one
SELECT * FROM messaging.webhook_deliveries
WHERE tenant_id = $1 AND id = $2;

-- name: ListWebhookDeliveries :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.webhook_deliveries
WHERE tenant_id = sqlc.arg('tenant_id')
AND (sqlc.narg('webhook_id')::uuid IS NULL OR webhook_id = sqlc.narg('webhook_id')::uuid)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListDueWebhookDeliveries :many
SELECT * FROM messaging.webhook_deliveries
WHERE status = 'retrying'
AND next_attempt_at IS NOT NULL
AND next_attempt_at <= $1
ORDER BY next_attempt_at ASC
LIMIT $2;

-- name: UpdateWebhookDelivery :one
UPDATE messaging.webhook_deliveries
SET
    status = $3,
    attempt_count = $4,
    next_attempt_at = $5,
    last_attempt_at = $6,
    response_status = $7,
    response_body = $8,
    error_message = $9,
    signature = $10,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;
