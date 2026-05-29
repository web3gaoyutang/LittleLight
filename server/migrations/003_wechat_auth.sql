ALTER TABLE users
    ADD COLUMN IF NOT EXISTS wechat_open_id TEXT,
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

UPDATE users
SET wechat_open_id = 'mock-openid-littlelight-teacher'
WHERE id = '00000000-0000-0000-0000-000000000001'
  AND wechat_open_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_wechat_open_id
    ON users(wechat_open_id)
    WHERE wechat_open_id IS NOT NULL;
