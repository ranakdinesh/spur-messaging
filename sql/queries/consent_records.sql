-- sql/queries/consent_records.sql

-- name: CreateConsentRecord :one
INSERT INTO messaging.consent_records (
    tenant_id, contact_id, channel, status, source, purpose, proof, ip_address, user_agent, brand
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: ListConsentRecords :many
SELECT * FROM messaging.consent_records
WHERE tenant_id = $1 AND contact_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
