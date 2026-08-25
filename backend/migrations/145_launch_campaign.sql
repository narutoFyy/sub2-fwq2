CREATE TABLE IF NOT EXISTS affiliate_campaigns (
    campaign_key VARCHAR(64) PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ NOT NULL,
    bonus_rate_percent DECIMAL(8,4) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO affiliate_campaigns (
    campaign_key,
    name,
    starts_at,
    ends_at,
    bonus_rate_percent,
    status
)
VALUES (
    'launch-rebate-2026-08',
    '拉新返利活动',
    NOW(),
    TIMESTAMPTZ '2026-08-15 23:59:59+08',
    50,
    'active'
)
ON CONFLICT (campaign_key) DO NOTHING;

CREATE TABLE IF NOT EXISTS affiliate_campaign_credits (
    id BIGSERIAL PRIMARY KEY,
    campaign_key VARCHAR(64) NOT NULL REFERENCES affiliate_campaigns(campaign_key),
    inviter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invitee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redeem_code_id BIGINT NOT NULL REFERENCES redeem_codes(id) ON DELETE RESTRICT,
    payment_order_id BIGINT NULL REFERENCES payment_orders(id) ON DELETE SET NULL,
    qualifying_amount DECIMAL(20,8) NOT NULL,
    bonus_amount DECIMAL(20,8) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    qualified_at TIMESTAMPTZ NOT NULL,
    reversed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT affiliate_campaign_credit_amount_positive CHECK (qualifying_amount > 0),
    CONSTRAINT affiliate_campaign_bonus_amount_positive CHECK (bonus_amount > 0),
    CONSTRAINT affiliate_campaign_credit_status_valid CHECK (status IN ('active', 'reversed')),
    UNIQUE (campaign_key, invitee_id),
    UNIQUE (campaign_key, redeem_code_id)
);

CREATE INDEX IF NOT EXISTS idx_affiliate_campaign_credits_rank
    ON affiliate_campaign_credits(campaign_key, status, inviter_id, qualifying_amount DESC);

CREATE INDEX IF NOT EXISTS idx_affiliate_campaign_credits_payment
    ON affiliate_campaign_credits(payment_order_id)
    WHERE payment_order_id IS NOT NULL;

COMMENT ON TABLE affiliate_campaigns IS '限时邀请活动定义';
COMMENT ON TABLE affiliate_campaign_credits IS '限时邀请活动首次余额入账及额外返利记录';
