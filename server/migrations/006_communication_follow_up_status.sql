ALTER TABLE communication_records
    ADD COLUMN IF NOT EXISTS follow_up_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS followed_up_at TIMESTAMPTZ;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'communication_records_follow_up_status_check') THEN
        ALTER TABLE communication_records
            ADD CONSTRAINT communication_records_follow_up_status_check
            CHECK (follow_up_status IN ('pending', 'done'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_records_user_follow_up
    ON communication_records(user_id, follow_up_status, follow_up_at)
    WHERE follow_up_at IS NOT NULL;
