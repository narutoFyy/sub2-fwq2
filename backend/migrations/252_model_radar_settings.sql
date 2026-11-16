-- 降智雷达的全局设置（只有 id = 1 这一行），以及分组级的间隔和是否对用户展示。
CREATE TABLE IF NOT EXISTS model_radar_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    interval_minutes INT NOT NULL DEFAULT 30,
    ciallo_attempts INT NOT NULL DEFAULT 3,
    nerf_confirm_rounds INT NOT NULL DEFAULT 1,
    drawing_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    retention_days INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO model_radar_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- interval_minutes = 0 表示跟随全局间隔；visible = FALSE 表示照常检测但不给用户看。
ALTER TABLE model_radar_configs ADD COLUMN IF NOT EXISTS interval_minutes INT NOT NULL DEFAULT 0;
ALTER TABLE model_radar_configs ADD COLUMN IF NOT EXISTS visible BOOLEAN NOT NULL DEFAULT TRUE;
