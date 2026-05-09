-- sql/queries/contacts.sql

-- name: CreateContact :one
INSERT INTO messaging.contacts (
    tenant_id, phone, email, name, attributes, tags, opt_in_whatsapp, opt_in_sms, opt_in_email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetContactByID :one
SELECT * FROM messaging.contacts
WHERE tenant_id = $1 AND id = $2;

-- name: GetContactByPhone :one
SELECT * FROM messaging.contacts
WHERE tenant_id = $1 AND phone = $2;

-- name: GetContactByEmail :one
SELECT * FROM messaging.contacts
WHERE tenant_id = $1 AND email = $2;

-- name: ListContacts :many
SELECT *, count(*) OVER() as total_count
FROM messaging.contacts
WHERE tenant_id = $1
AND ($2::text IS NULL OR phone = $2)
AND ($3::text IS NULL OR email = $3)
AND ($4::text IS NULL OR tags @> ARRAY[$4::text])
AND ($5::text IS NULL OR (
    ($5 = 'whatsapp' AND opt_in_whatsapp = 'opted_in') OR
    ($5 = 'sms' AND opt_in_sms = 'opted_in') OR
    ($5 = 'email' AND opt_in_email = 'opted_in')
))
ORDER BY created_at DESC
LIMIT $6 OFFSET $7;

-- name: UpdateContact :one
UPDATE messaging.contacts
SET phone = $3,
    email = $4,
    name = $5,
    attributes = $6,
    tags = $7,
    opt_in_whatsapp = $8,
    opt_in_sms = $9,
    opt_in_email = $10,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteContact :exec
DELETE FROM messaging.contacts
WHERE tenant_id = $1 AND id = $2;

-- name: UpdateOptIn :exec
UPDATE messaging.contacts
SET opt_in_whatsapp = CASE WHEN $3::text = 'whatsapp' THEN $4::text ELSE opt_in_whatsapp END,
    opt_in_sms = CASE WHEN $3::text = 'sms' THEN $4::text ELSE opt_in_sms END,
    opt_in_email = CASE WHEN $3::text = 'email' THEN $4::text ELSE opt_in_email END,
    opted_in_at = CASE WHEN $4::text = 'opted_in' THEN now() ELSE opted_in_at END,
    opted_out_at = CASE WHEN $4::text = 'opted_out' THEN now() ELSE opted_out_at END,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2;

-- name: BulkCreateContacts :copyfrom
INSERT INTO messaging.contacts (
    tenant_id, phone, email, name, attributes, tags, opt_in_whatsapp, opt_in_sms, opt_in_email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: GetContactsBySegment :many
SELECT c.*, count(*) OVER() as total_count
FROM messaging.contacts c
JOIN messaging.segment_contacts sc ON c.id = sc.contact_id
WHERE c.tenant_id = $1 AND sc.segment_id = $2
ORDER BY c.created_at DESC
LIMIT $3 OFFSET $4;
