ALTER TABLE courses ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE parent_profiles ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE reminders ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE communication_records ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE healing_entries ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE ai_generations ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE favorites ALTER COLUMN user_id SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_pro_status_check') THEN
        ALTER TABLE users
            ADD CONSTRAINT users_pro_status_check
            CHECK (pro_status IN ('free', 'trial', 'pro', 'expired'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_reminder_policy_check') THEN
        ALTER TABLE users
            ADD CONSTRAINT users_reminder_policy_check
            CHECK (reminder_policy IN ('normal', 'low_interrupt'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'courses_time_order_check') THEN
        ALTER TABLE courses
            ADD CONSTRAINT courses_time_order_check
            CHECK (end_time > start_time);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parent_profiles_risk_level_check') THEN
        ALTER TABLE parent_profiles
            ADD CONSTRAINT parent_profiles_risk_level_check
            CHECK (risk_level IN ('low', 'medium', 'high'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'reminders_status_check') THEN
        ALTER TABLE reminders
            ADD CONSTRAINT reminders_status_check
            CHECK (status IN ('pending', 'done', 'snoozed', 'deleted'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'communication_records_risk_level_check') THEN
        ALTER TABLE communication_records
            ADD CONSTRAINT communication_records_risk_level_check
            CHECK (risk_level IN ('low', 'medium', 'high'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'healing_entries_type_check') THEN
        ALTER TABLE healing_entries
            ADD CONSTRAINT healing_entries_type_check
            CHECK (type IN ('breath', 'praise', 'treehole', 'sound'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'healing_entries_visibility_check') THEN
        ALTER TABLE healing_entries
            ADD CONSTRAINT healing_entries_visibility_check
            CHECK (visibility IN ('private'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ai_generations_scenario_check') THEN
        ALTER TABLE ai_generations
            ADD CONSTRAINT ai_generations_scenario_check
            CHECK (scenario IN ('parent_drafts', 'praise'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ai_generations_safety_label_check') THEN
        ALTER TABLE ai_generations
            ADD CONSTRAINT ai_generations_safety_label_check
            CHECK (safety_label IN (
                'teacher_review_required',
                'self_care',
                'crisis_support_required',
                'student_safety_review_required',
                'medical_review_required'
            ));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'favorites_type_check') THEN
        ALTER TABLE favorites
            ADD CONSTRAINT favorites_type_check
            CHECK (type IN ('communication_template', 'ai_praise', 'class_feedback'));
    END IF;
END $$;
