-- sql/queries/segments.sql

-- name: CreateSegment :one
INSERT INTO messaging.segments (
    tenant_id, name, is_dynamic, rules
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetSegmentByID :one
SELECT * FROM messaging.segments
WHERE tenant_id = $1 AND id = $2;

-- name: ListSegments :many
SELECT * FROM messaging.segments
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: UpdateSegment :one
UPDATE messaging.segments
SET name = $3,
    is_dynamic = $4,
    rules = $5,
    updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: DeleteSegment :exec
DELETE FROM messaging.segments
WHERE tenant_id = $1 AND id = $2;

-- name: ResolveContacts :many
SELECT c.*, count(*) OVER() as total_count
FROM messaging.contacts c
JOIN messaging.segment_contacts sc ON c.id = sc.contact_id
WHERE c.tenant_id = $1 AND sc.segment_id = $2
ORDER BY c.created_at DESC
LIMIT $3 OFFSET $4;

-- name: AddContactToSegment :exec
INSERT INTO messaging.segment_contacts (segment_id, contact_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveContactFromSegment :exec
DELETE FROM messaging.segment_contacts
WHERE segment_id = $1 AND contact_id = $2;

-- name: ClearSegmentContacts :exec
DELETE FROM messaging.segment_contacts
WHERE segment_id = $1;
