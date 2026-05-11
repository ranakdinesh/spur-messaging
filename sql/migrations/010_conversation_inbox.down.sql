DROP INDEX IF EXISTS messaging.idx_conversations_tags;
DROP INDEX IF EXISTS messaging.idx_conversations_assignee;
DROP INDEX IF EXISTS messaging.idx_conversations_inbox;

ALTER TABLE messaging.conversations
    DROP CONSTRAINT IF EXISTS conversations_priority_check;

ALTER TABLE messaging.conversations
    DROP CONSTRAINT IF EXISTS conversations_status_check;

ALTER TABLE messaging.conversations
    ADD CONSTRAINT conversations_status_check
    CHECK (status IN ('open', 'closed'));

ALTER TABLE messaging.conversations
    DROP COLUMN IF EXISTS closed_at,
    DROP COLUMN IF EXISTS resolution_due_at,
    DROP COLUMN IF EXISTS first_response_due_at,
    DROP COLUMN IF EXISTS internal_notes,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS priority,
    DROP COLUMN IF EXISTS assigned_team,
    DROP COLUMN IF EXISTS assigned_agent_id;
