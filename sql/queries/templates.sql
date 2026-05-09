-- sql/queries/templates.sql

-- name: CreateTemplate :one
INSERT INTO messaging.templates (
    tenant_id, channel, name, language, category, components, status, provider_template_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetTemplateByID :one
SELECT * FROM messaging.templates
WHERE tenant_id = $1 AND id = $2;

-- name: GetTemplateByName :one
SELECT * FROM messaging.templates
WHERE tenant_id = $1 AND name = $2 AND language = $3;

-- name: ListTemplates :many
SELECT *, count(*) OVER() as total_count
FROM messaging.templates
WHERE tenant_id = $1
AND ($2::text IS NULL OR channel = $2)
AND ($3::text IS NULL OR status = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: UpdateTemplate :one
UPDATE messaging.templates
SET category = $3,
    components = $4,
    status = $5,
    provider_template_id = COALESCE($6, provider_template_id),
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: UpdateTemplateStatus :exec
UPDATE messaging.templates
SET status = $3,
    provider_template_id = COALESCE($4, provider_template_id),
    rejection_reason = $5,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2;

-- name: DeleteTemplate :exec
DELETE FROM messaging.templates
WHERE tenant_id = $1 AND id = $2;
