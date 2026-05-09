-- sql/queries/provider_configs.sql

-- name: CreateProviderConfig :one
INSERT INTO messaging.provider_configs (
    tenant_id, channel, provider, credentials, webhook_secret, is_active,
    phone_number_id, waba_id, business_id, display_phone, from_email, from_name, reply_to_email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetProviderConfigByID :one
SELECT * FROM messaging.provider_configs
WHERE tenant_id = $1 AND id = $2;

-- name: GetProviderConfigByChannel :one
SELECT * FROM messaging.provider_configs
WHERE tenant_id = $1 AND channel = $2 AND is_active = true;

-- name: GetProviderConfigByWABAID :one
SELECT * FROM messaging.provider_configs
WHERE waba_id = $1;

-- name: ListProviderConfigs :many
SELECT * FROM messaging.provider_configs
WHERE tenant_id = $1;

-- name: UpdateProviderConfig :one
UPDATE messaging.provider_configs
SET provider = $3,
    credentials = $4,
    webhook_secret = $5,
    is_active = $6,
    phone_number_id = $7,
    waba_id = $8,
    business_id = $9,
    display_phone = $10,
    from_email = $11,
    from_name = $12,
    reply_to_email = $13,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteProviderConfig :exec
DELETE FROM messaging.provider_configs
WHERE tenant_id = $1 AND id = $2;

-- name: UpdateProviderConfigIsActive :exec
UPDATE messaging.provider_configs
SET is_active = $3,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2;
