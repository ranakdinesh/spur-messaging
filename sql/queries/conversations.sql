-- sql/queries/conversations.sql

-- name: GetConversationByID :one
SELECT * FROM messaging.conversations
WHERE tenant_id = $1
  AND id = $2;

-- name: ListConversations :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.conversations
WHERE tenant_id = sqlc.arg('tenant_id')
  AND (sqlc.narg('channel')::text IS NULL OR channel = sqlc.narg('channel')::text)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('handoff_status')::text IS NULL OR handoff_status = sqlc.narg('handoff_status')::text)
  AND (sqlc.narg('assigned_agent_id')::uuid IS NULL OR assigned_agent_id = sqlc.narg('assigned_agent_id')::uuid)
  AND (sqlc.narg('recipient')::text IS NULL OR recipient ILIKE '%' || sqlc.narg('recipient')::text || '%')
  AND (sqlc.narg('tag')::text IS NULL OR sqlc.narg('tag')::text = ANY(tags))
ORDER BY updated_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetActiveConversationByRecipient :one
SELECT * FROM messaging.conversations
WHERE tenant_id = $1
  AND channel = $2
  AND recipient = $3
  AND status = 'open'
  AND service_window_until IS NOT NULL
  AND service_window_until > $4;

-- name: UpsertConversationInbound :one
INSERT INTO messaging.conversations (
    tenant_id, channel, recipient, status, handoff_status, last_inbound_at, service_window_until
) VALUES (
    $1, $2, $3, 'open', 'bot', $4, $4::timestamptz + interval '24 hours'
)
ON CONFLICT (tenant_id, channel, recipient) DO UPDATE
SET status = 'open',
    handoff_status = CASE
        WHEN messaging.conversations.handoff_status = 'closed' THEN 'bot'
        ELSE messaging.conversations.handoff_status
    END,
    last_inbound_at = GREATEST(COALESCE(messaging.conversations.last_inbound_at, EXCLUDED.last_inbound_at), EXCLUDED.last_inbound_at),
    service_window_until = GREATEST(COALESCE(messaging.conversations.service_window_until, EXCLUDED.service_window_until), EXCLUDED.service_window_until),
    updated_at = now()
RETURNING *;

-- name: UpsertConversationOutbound :one
INSERT INTO messaging.conversations (
    tenant_id, channel, recipient, status, handoff_status, last_outbound_at
) VALUES (
    $1, $2, $3, 'open', 'bot', $4
)
ON CONFLICT (tenant_id, channel, recipient) DO UPDATE
SET last_outbound_at = GREATEST(COALESCE(messaging.conversations.last_outbound_at, EXCLUDED.last_outbound_at), EXCLUDED.last_outbound_at),
    updated_at = now()
RETURNING *;

-- name: UpdateConversationInbox :one
UPDATE messaging.conversations
SET status = COALESCE(sqlc.narg('status')::text, status),
    handoff_status = COALESCE(sqlc.narg('handoff_status')::text, handoff_status),
    assigned_agent_id = COALESCE(sqlc.narg('assigned_agent_id')::uuid, assigned_agent_id),
    assigned_team = COALESCE(sqlc.narg('assigned_team')::text, assigned_team),
    priority = COALESCE(sqlc.narg('priority')::text, priority),
    tags = COALESCE(sqlc.narg('tags')::text[], tags),
    first_response_due_at = COALESCE(sqlc.narg('first_response_due_at')::timestamptz, first_response_due_at),
    resolution_due_at = COALESCE(sqlc.narg('resolution_due_at')::timestamptz, resolution_due_at),
    closed_at = CASE
        WHEN sqlc.narg('status')::text = 'closed' THEN COALESCE(closed_at, now())
        WHEN sqlc.narg('status')::text IN ('open', 'pending') THEN NULL
        ELSE closed_at
    END,
    updated_at = now()
WHERE tenant_id = sqlc.arg('tenant_id')
  AND id = sqlc.arg('id')
RETURNING *;

-- name: AddConversationNote :one
UPDATE messaging.conversations
SET internal_notes = internal_notes || jsonb_build_array(jsonb_build_object(
        'id', $3::text,
        'author_id', $4::text,
        'body', $5::text,
        'created_at', $6::timestamptz
    )),
    updated_at = now()
WHERE tenant_id = $1
  AND id = $2
RETURNING *;
