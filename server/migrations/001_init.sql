CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    school TEXT,
    stage TEXT,
    subject TEXT,
    is_head_teacher BOOLEAN NOT NULL DEFAULT FALSE,
    pro_status TEXT NOT NULL DEFAULT 'free',
    reminder_policy TEXT NOT NULL DEFAULT 'low_interrupt',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS courses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    class_name TEXT NOT NULL,
    location TEXT,
    weekday SMALLINT NOT NULL CHECK (weekday >= 0 AND weekday <= 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    repeat_rule TEXT NOT NULL DEFAULT 'weekly',
    color TEXT,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS parent_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    student_name TEXT NOT NULL,
    class_name TEXT NOT NULL,
    parent_name TEXT NOT NULL,
    relationship TEXT NOT NULL,
    contact TEXT,
    communication_style TEXT,
    risk_level TEXT NOT NULL DEFAULT 'low',
    important_notes TEXT,
    next_action TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES parent_profiles(id) ON DELETE SET NULL,
    course_id UUID REFERENCES courses(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'personal',
    remind_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    note TEXT,
    done_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS communication_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES parent_profiles(id) ON DELETE SET NULL,
    student TEXT NOT NULL,
    channel TEXT NOT NULL,
    reason TEXT NOT NULL,
    summary TEXT NOT NULL,
    result TEXT,
    risk_level TEXT NOT NULL DEFAULT 'low',
    follow_up_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS healing_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    mood TEXT,
    content TEXT,
    ai_reply TEXT,
    visibility TEXT NOT NULL DEFAULT 'private',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ai_generations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    scenario TEXT NOT NULL,
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    safety_label TEXT NOT NULL DEFAULT 'teacher_review_required',
    token_usage INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS favorites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    source_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_courses_user_weekday ON courses(user_id, weekday);
CREATE INDEX IF NOT EXISTS idx_reminders_user_time_status ON reminders(user_id, remind_at, status);
CREATE INDEX IF NOT EXISTS idx_parent_profiles_user_risk ON parent_profiles(user_id, risk_level);
CREATE INDEX IF NOT EXISTS idx_records_user_parent ON communication_records(user_id, parent_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_healing_user_created ON healing_entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_generations_user_created ON ai_generations(user_id, created_at DESC);
