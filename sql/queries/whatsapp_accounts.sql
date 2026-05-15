-- sql/queries/whatsapp_accounts.sql

-- name: CreateWhatsAppBusinessAccount :one
INSERT INTO messaging.whatsapp_business_accounts (
    tenant_id,
    meta_business_id,
    waba_id,
    name,
    currency,
    timezone_id,
    business_verification_status,
    onboarding_status,
    provider_config_id,
    last_synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetWhatsAppBusinessAccount :one
SELECT * FROM messaging.whatsapp_business_accounts
WHERE tenant_id = $1 AND id = $2;

-- name: GetWhatsAppBusinessAccountByWABAID :one
SELECT * FROM messaging.whatsapp_business_accounts
WHERE tenant_id = $1 AND waba_id = $2;

-- name: ListWhatsAppBusinessAccounts :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.whatsapp_business_accounts
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWhatsAppBusinessAccountStatus :one
UPDATE messaging.whatsapp_business_accounts
SET business_verification_status = $3,
    onboarding_status = $4,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: UpdateWhatsAppBusinessAccountSync :one
UPDATE messaging.whatsapp_business_accounts
SET meta_business_id = $3,
    name = $4,
    currency = $5,
    timezone_id = $6,
    business_verification_status = $7,
    onboarding_status = $8,
    provider_config_id = $9,
    last_synced_at = $10,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteWhatsAppBusinessAccount :exec
DELETE FROM messaging.whatsapp_business_accounts
WHERE tenant_id = $1 AND id = $2;

-- name: CreateWhatsAppPhoneNumber :one
INSERT INTO messaging.whatsapp_phone_numbers (
    tenant_id,
    waba_id,
    phone_number_id,
    display_phone_number,
    verified_name,
    quality_rating,
    messaging_limit_tier,
    status,
    code_verification_status,
    last_synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetWhatsAppPhoneNumber :one
SELECT * FROM messaging.whatsapp_phone_numbers
WHERE tenant_id = $1 AND id = $2;

-- name: GetWhatsAppPhoneNumberByPhoneNumberID :one
SELECT * FROM messaging.whatsapp_phone_numbers
WHERE tenant_id = $1 AND phone_number_id = $2;

-- name: ListWhatsAppPhoneNumbersByWABA :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.whatsapp_phone_numbers
WHERE tenant_id = $1 AND waba_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListWhatsAppPhoneNumbers :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.whatsapp_phone_numbers
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWhatsAppPhoneNumberStatus :one
UPDATE messaging.whatsapp_phone_numbers
SET quality_rating = $3,
    messaging_limit_tier = $4,
    status = $5,
    code_verification_status = $6,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: UpdateWhatsAppPhoneNumberSync :one
UPDATE messaging.whatsapp_phone_numbers
SET display_phone_number = $3,
    verified_name = $4,
    quality_rating = $5,
    messaging_limit_tier = $6,
    status = $7,
    code_verification_status = $8,
    last_synced_at = $9,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteWhatsAppPhoneNumber :exec
DELETE FROM messaging.whatsapp_phone_numbers
WHERE tenant_id = $1 AND id = $2;

-- name: CreateWhatsAppOnboardingSession :one
INSERT INTO messaging.whatsapp_onboarding_sessions (
    tenant_id,
    state,
    status
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetWhatsAppOnboardingSession :one
SELECT * FROM messaging.whatsapp_onboarding_sessions
WHERE tenant_id = $1 AND id = $2;

-- name: GetWhatsAppOnboardingSessionByState :one
SELECT * FROM messaging.whatsapp_onboarding_sessions
WHERE tenant_id = $1 AND state = $2;

-- name: UpdateWhatsAppOnboardingSessionStatus :one
UPDATE messaging.whatsapp_onboarding_sessions
SET status = $3,
    error_message = NULL,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: CompleteWhatsAppOnboardingSession :one
UPDATE messaging.whatsapp_onboarding_sessions
SET status = 'completed',
    error_message = NULL,
    completed_at = now(),
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: FailWhatsAppOnboardingSession :one
UPDATE messaging.whatsapp_onboarding_sessions
SET status = 'failed',
    error_message = $3,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;
