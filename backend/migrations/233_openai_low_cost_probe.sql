-- Shared OpenAI low-cost account probe state and rolling 24-hour budget ledger.
CREATE TABLE IF NOT EXISTS openai_low_cost_probe_states (
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    last_probe_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    latency_ms BIGINT,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    consecutive_successes INTEGER NOT NULL DEFAULT 0,
    disabled_until TIMESTAMPTZ,
    estimated_spend_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_openai_low_cost_probe_states_account
    ON openai_low_cost_probe_states (account_id);

CREATE INDEX IF NOT EXISTS idx_openai_low_cost_probe_states_status
    ON openai_low_cost_probe_states (status, disabled_until);

CREATE TABLE IF NOT EXISTS openai_low_cost_probe_budget_ledger (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    estimated_spend_usd DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_openai_low_cost_probe_budget_created
    ON openai_low_cost_probe_budget_ledger (created_at DESC);
