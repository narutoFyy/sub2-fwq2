-- 降智雷达本地演示数据：给几个 [测试] OpenAI 分组打开自动检测，并灌入 ciallo 结果。
-- 回复文字取自 2026-11-15 对 gpt-6-astra / gpt-6.1-sol 的真实调用，时间线是编的。
-- 仅用于本地看页面；清理：DELETE FROM model_radar_results; DELETE FROM model_radar_configs;
DELETE FROM model_radar_results;
DELETE FROM model_radar_configs;

INSERT INTO model_radar_configs (group_id, model_id, reasoning_effort, enabled, next_run_at, last_run_at)
SELECT id, model, 'high', TRUE, NOW() + INTERVAL '1 day', NOW() - INTERVAL '4 minutes'
FROM (VALUES (5, 'gpt-6-astra'), (6, 'gpt-6.1-sol'), (7, 'gpt-6-astra'), (12, 'gpt-6-astra')) AS v(id, model)
WHERE EXISTS (SELECT 1 FROM groups g WHERE g.id = v.id);

-- 分组 6：一直正常
INSERT INTO model_radar_results (group_id, model_id, test_type, status, response_text, latency_ms, detected_at)
SELECT 6, 'gpt-6.1-sol', 'ciallo', 'passed', 'Ciallo～(∠・ω< )⌒★ 今天想一起做点什么？', 6400 + n * 37, NOW() - INTERVAL '4 minutes' - n * INTERVAL '30 minutes'
FROM generate_series(0, 23) AS n;

-- 分组 5：最近 3 轮没带颜文字，之前正常
INSERT INTO model_radar_results (group_id, model_id, test_type, status, response_text, latency_ms, detected_at)
SELECT 5, 'gpt-6-astra', 'ciallo',
       CASE WHEN n < 3 THEN 'failed' ELSE 'passed' END,
       CASE WHEN n < 3 THEN 'Ciallo～ 👋 What can I help you with?' ELSE 'Ciallo～(∠・ω< )⌒★ 有什么我可以帮你的吗？' END,
       5300 + n * 41, NOW() - INTERVAL '4 minutes' - n * INTERVAL '30 minutes'
FROM generate_series(0, 17) AS n;

-- 分组 7：正常，中间夹过一次请求失败
INSERT INTO model_radar_results (group_id, model_id, test_type, status, response_text, error_message, latency_ms, detected_at)
SELECT 7, 'gpt-6-astra', 'ciallo',
       CASE WHEN n = 5 THEN 'request_failed' ELSE 'passed' END,
       CASE WHEN n = 5 THEN '' ELSE 'Ciallo～(∠・ω< )⌒☆' END,
       CASE WHEN n = 5 THEN 'upstream 503' ELSE '' END,
       7200 + n * 29, NOW() - INTERVAL '4 minutes' - n * INTERVAL '30 minutes'
FROM generate_series(0, 23) AS n;

INSERT INTO model_radar_results (group_id, model_id, test_type, status, response_text, latency_ms, detected_at)
VALUES (6, 'gpt-6.1-sol', 'drawing', 'passed',
        '<svg viewBox="0 0 320 180" xmlns="http://www.w3.org/2000/svg"><rect width="320" height="180" fill="#e0f2fe"/><circle cx="110" cy="130" r="26" fill="none" stroke="#111" stroke-width="4"/><circle cx="210" cy="130" r="26" fill="none" stroke="#111" stroke-width="4"/><path d="M110 130 L150 90 L210 130 M150 90 L180 90" stroke="#111" stroke-width="4" fill="none"/><ellipse cx="155" cy="70" rx="26" ry="18" fill="#fff" stroke="#111" stroke-width="3"/><circle cx="182" cy="50" r="11" fill="#fff" stroke="#111" stroke-width="3"/><path d="M190 50 L226 56 L190 60 Z" fill="#f59e0b" stroke="#111" stroke-width="2"/><text x="160" y="172" text-anchor="middle" font-size="10" fill="#555">本地演示样张（手写，非模型输出）</text></svg>',
        48000, NOW() - INTERVAL '3 minutes');
