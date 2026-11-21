-- yf-mail 拉新返利活动方案（正式版 v2）数据库迁移
-- 1. 扩展 user_affiliates：新人 1 元体验金、注册/调用 IP 与同 IP 加权预警、单人两步奖（3元+5元）、30天有效邀请归因与邀请人连带风控。

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS register_ip VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS register_ip_subnet VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS last_active_ip VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS same_ip_warning BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS risk_tags VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_bonus_amount DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_bonus_status VARCHAR(24) NOT NULL DEFAULT 'none';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_skip_reason VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_granted_at TIMESTAMPTZ NULL;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_expires_at TIMESTAMPTZ NULL;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_expired_amount DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_rpm_limited BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_grant_paused BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS trial_grant_pause_reason VARCHAR(128) NOT NULL DEFAULT '';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS total_paid_amount DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS total_consumed_amount DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS step1_reward_status VARCHAR(24) NOT NULL DEFAULT 'none';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS step1_reward_amount DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS step2_reward_status VARCHAR(24) NOT NULL DEFAULT 'none';

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS step2_reward_amount DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS qualified_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_trial_granted_at
    ON user_affiliates (trial_granted_at)
    WHERE trial_granted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_register_ip_subnet
    ON user_affiliates (register_ip_subnet, trial_granted_at)
    WHERE trial_granted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_trial_expires
    ON user_affiliates (trial_expires_at)
    WHERE trial_bonus_status = 'active';

CREATE INDEX IF NOT EXISTS idx_user_affiliates_qualified
    ON user_affiliates (inviter_id, qualified_at)
    WHERE qualified_at IS NOT NULL;

-- 2. 单人两步奖与阶梯奖（补差额模式）发放记录与人工审核队列表（含 operation_id 唯一幂等约束）
CREATE TABLE IF NOT EXISTS affiliate_reward_grants (
    id BIGSERIAL PRIMARY KEY,
    operation_id VARCHAR(128) NOT NULL UNIQUE,
    inviter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invitee_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    reward_type VARCHAR(32) NOT NULL,
    tier_target INTEGER NOT NULL DEFAULT 0,
    amount DECIMAL(20,8) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'granted',
    review_reason VARCHAR(128) NOT NULL DEFAULT '',
    review_note TEXT NOT NULL DEFAULT '',
    reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ NULL,
    granted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_affiliate_reward_grants_inviter
    ON affiliate_reward_grants (inviter_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_affiliate_reward_grants_status
    ON affiliate_reward_grants (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_affiliate_reward_grants_granted_at
    ON affiliate_reward_grants (inviter_id, granted_at)
    WHERE status = 'granted';

-- 3. 预设拉新返利 v2 默认配置（仅在未配置时写入，不覆盖管理员已有修改）
INSERT INTO settings (key, value, updated_at)
VALUES
    ('affiliate_rebate_rate', '5', NOW()),
    ('affiliate_rebate_freeze_hours', '72', NOW()),
    ('affiliate_feishu_webhook_url', '', NOW())
ON CONFLICT (key) DO NOTHING;
