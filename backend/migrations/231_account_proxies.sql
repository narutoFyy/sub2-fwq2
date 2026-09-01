CREATE TABLE IF NOT EXISTS account_proxies (
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    proxy_id BIGINT NOT NULL REFERENCES proxies(id) ON DELETE RESTRICT,
    concurrency INT NOT NULL DEFAULT 1 CHECK (concurrency >= 1),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, proxy_id)
);

CREATE INDEX IF NOT EXISTS account_proxies_proxy_id_idx ON account_proxies(proxy_id);
CREATE INDEX IF NOT EXISTS account_proxies_enabled_order_idx
    ON account_proxies(account_id, enabled, sort_order, proxy_id);
