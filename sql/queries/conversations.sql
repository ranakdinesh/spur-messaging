-- sql/queries/conversations.sql

-- name: GetConversationByID :one
SELECT * FROM messaging.conversations
WHERE tenant_id = $1
  AND id = $2;

-- name: ListConversations :many
SELECT *, count(*) OVER() AS total_count
FROM messaging.conversations
WHERE tenant_id = $1
  AND ($2 = '' OR channel = $2)
  AND ($3 = '' OR status = $3)
  AND ($4 = '' OR handoff_status = $4)
  AND ($5::uuid IS NULL OR assigned_agent_id = $5)
  AND ($6 = '' OR recipient ILIKE '%' || $6 || '%')
  AND ($7 = '' OR $7 = ANY(tags))
ORDER BY updated_at DESC
LIMIT $8 OFFSET $9;

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
SET status = COALESCE($3, status),
    handoff_status = COALESCE($4, handoff_status),
    assigned_agent_id = COALESCE($5, assigned_agent_id),
    assigned_team = COALESCE($6, assigned_team),
    priority = COALESCE($7, priority),
    tags = COALESCE($8::text[], tags),
    first_response_due_at = COALESCE($9, first_response_due_at),
    resolution_due_at = COALESCE($10, resolution_due_at),
    closed_at = CASE
        WHEN $3 = 'closed' THEN COALESCE(closed_at, now())
        WHEN $3 IN ('open', 'pending') THEN NULL
        ELSE closed_at
    END,
    updated_at = now()
WHERE tenant_id = $1
  AND id = $2
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
