-- sql/queries/campaigns.sql

-- name: CreateCampaign :one
INSERT INTO messaging.campaigns (
    tenant_id, name, channel, template_id, template_params, segment_id, contact_ids, scheduled_at, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetCampaignByID :one
SELECT * FROM messaging.campaigns
WHERE tenant_id = $1 AND id = $2;

-- name: ListCampaigns :many
SELECT *, count(*) OVER() as total_count
FROM messaging.campaigns
WHERE tenant_id = $1
AND ($2::text IS NULL OR status = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateCampaign :one
UPDATE messaging.campaigns
SET name = $3,
    template_id = $4,
    template_params = $5,
    segment_id = $6,
    contact_ids = $7,
    scheduled_at = $8,
    status = $9,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: UpdateCampaignStatus :exec
UPDATE messaging.campaigns
SET status = $3,
    started_at = CASE WHEN $3 = 'running' AND started_at IS NULL THEN now() ELSE started_at END,
    completed_at = CASE WHEN $3 = 'completed' THEN now() ELSE completed_at END,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2;

-- name: UpdateCampaignStats :exec
UPDATE messaging.campaigns
SET stats = $3,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2;

-- name: DeleteCampaign :exec
DELETE FROM messaging.campaigns
WHERE tenant_id = $1 AND id = $2;

-- name: GetScheduledCampaigns :many
SELECT * FROM messaging.campaigns
WHERE status = 'scheduled' AND scheduled_at <= $1;
