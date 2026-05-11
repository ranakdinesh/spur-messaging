ALTER TABLE messaging.conversations
    ADD COLUMN IF NOT EXISTS assigned_agent_id UUID,
    ADD COLUMN IF NOT EXISTS assigned_team TEXT,
    ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'medium',
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS internal_notes JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS first_response_due_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS resolution_due_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'conversations_status_check'
          AND conrelid = 'messaging.conversations'::regclass
    ) THEN
        ALTER TABLE messaging.conversations DROP CONSTRAINT conversations_status_check;
    END IF;
END $$;

ALTER TABLE messaging.conversations
    ADD CONSTRAINT conversations_status_check
    CHECK (status IN ('open', 'pending', 'resolved', 'closed'));

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'conversations_priority_check'
          AND conrelid = 'messaging.conversations'::regclass
    ) THEN
        ALTER TABLE messaging.conversations
            ADD CONSTRAINT conversations_priority_check
            CHECK (priority IN ('low', 'medium', 'high', 'urgent'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_conversations_inbox
    ON messaging.conversations (tenant_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversations_assignee
    ON messaging.conversations (tenant_id, assigned_agent_id, status);

CREATE INDEX IF NOT EXISTS idx_conversations_tags
    ON messaging.conversations USING GIN (tags);
