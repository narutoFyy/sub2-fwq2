-- 降智雷达的文字题从数列题换成「ciallo」题：新结果用 test_type = 'ciallo'，
-- 旧的 'logic' 历史结果保留但不再展示。
ALTER TABLE model_radar_results DROP CONSTRAINT IF EXISTS model_radar_results_test_type_check;
ALTER TABLE model_radar_results
    ADD CONSTRAINT model_radar_results_test_type_check CHECK (test_type IN ('logic', 'drawing', 'ciallo'));

-- 新题按 high 推理强度校准（medium 及以下正常模型也只回 emoji），默认值跟着改。
ALTER TABLE model_radar_configs ALTER COLUMN reasoning_effort SET DEFAULT 'high';
UPDATE model_radar_configs SET reasoning_effort = 'high', updated_at = NOW()
WHERE reasoning_effort IN ('minimal', 'low', 'medium');
