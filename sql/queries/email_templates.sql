-- sql/queries/email_templates.sql

-- name: CreateEmailTemplate :one
INSERT INTO messaging.email_templates (
    tenant_id, name, subject, preview_text, html_body, text_body, category, variables, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetEmailTemplateByID :one
SELECT * FROM messaging.email_templates
WHERE tenant_id = $1 AND id = $2;

-- name: GetEmailTemplateByName :one
SELECT * FROM messaging.email_templates
WHERE tenant_id = $1 AND name = $2;

-- name: ListEmailTemplates :many
SELECT *, count(*) OVER() as total_count
FROM messaging.email_templates
WHERE tenant_id = $1
AND ($2::text IS NULL OR category = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateEmailTemplate :one
UPDATE messaging.email_templates
SET subject = COALESCE($3, subject),
    preview_text = COALESCE($4, preview_text),
    html_body = COALESCE($5, html_body),
    text_body = COALESCE($6, text_body),
    category = COALESCE($7, category),
    variables = COALESCE($8, variables),
    is_active = COALESCE($9, is_active),
    version = version + 1,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteEmailTemplate :exec
DELETE FROM messaging.email_templates
WHERE tenant_id = $1 AND id = $2;
