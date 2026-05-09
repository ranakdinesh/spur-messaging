-- sql/queries/unsubscribes.sql

-- name: CreateUnsubscribe :one
INSERT INTO messaging.unsubscribes (
    tenant_id, email, scope, campaign_id, reason
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: IsUnsubscribed :one
SELECT EXISTS(
    SELECT 1 FROM messaging.unsubscribes
    WHERE tenant_id = $1 AND email = $2
    AND (
        scope = 'global' OR
        (scope = 'category' AND $3::text = 'marketing') OR
        (scope = 'campaign' AND campaign_id = $4)
    )
);

-- name: ListUnsubscribes :many
SELECT *, count(*) OVER() as total_count
FROM messaging.unsubscribes
WHERE tenant_id = $1
AND ($2::text IS NULL OR scope = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: DeleteUnsubscribe :exec
DELETE FROM messaging.unsubscribes
WHERE tenant_id = $1 AND id = $2;

-- name: GetUnsubscribesByEmail :many
SELECT * FROM messaging.unsubscribes
WHERE tenant_id = $1 AND email = $2;
