-- Model radar stores one OpenAI probe configuration per group and the two
-- probe results used by the user-facing radar page.
CREATE TABLE IF NOT EXISTS model_radar_configs (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL UNIQUE REFERENCES groups(id) ON DELETE CASCADE,
    model_id VARCHAR(128) NOT NULL DEFAULT '',
    reasoning_effort VARCHAR(20) NOT NULL DEFAULT 'medium',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS model_radar_results (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    model_id VARCHAR(128) NOT NULL,
    test_type VARCHAR(20) NOT NULL CHECK (test_type IN ('logic', 'drawing')),
    status VARCHAR(30) NOT NULL CHECK (status IN ('passed', 'failed', 'request_failed', 'pending_review')),
    response_text TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    latency_ms BIGINT NOT NULL DEFAULT 0,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    review_status VARCHAR(20) NOT NULL DEFAULT '',
    reviewed_at TIMESTAMPTZ,
    reviewed_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_model_radar_results_group_type_time
    ON model_radar_results (group_id, test_type, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_model_radar_results_review
    ON model_radar_results (review_status, detected_at DESC);
