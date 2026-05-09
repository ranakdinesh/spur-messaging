-- sql/queries/suppressions.sql

-- name: CreateSuppression :one
INSERT INTO messaging.suppressions (
    tenant_id, email, reason, source
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: IsSuppressed :one
SELECT EXISTS(
    SELECT 1 FROM messaging.suppressions
    WHERE tenant_id = $1 AND email = $2
);

-- name: ListSuppressions :many
SELECT *, count(*) OVER() as total_count
FROM messaging.suppressions
WHERE tenant_id = $1
AND ($2::text IS NULL OR reason = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: DeleteSuppression :exec
DELETE FROM messaging.suppressions
WHERE tenant_id = $1 AND id = $2;

-- name: BulkCheckSuppression :many
SELECT email FROM messaging.suppressions
WHERE tenant_id = $1 AND email = ANY($2::text[]);
