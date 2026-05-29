CREATE TABLE IF NOT EXISTS ai_generation_actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    generation_id UUID NOT NULL REFERENCES ai_generations(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    draft_id TEXT,
    note TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ai_generation_actions_action_check') THEN
        ALTER TABLE ai_generation_actions
            ADD CONSTRAINT ai_generation_actions_action_check
            CHECK (action IN ('copy', 'save_record', 'save_healing', 'review_confirmed', 'review_skipped'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_ai_generation_actions_user_generation_created
    ON ai_generation_actions(user_id, generation_id, created_at DESC);
