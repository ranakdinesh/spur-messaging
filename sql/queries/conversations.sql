-- sql/queries/conversations.sql

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
