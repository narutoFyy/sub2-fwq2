-- 图文教程截图用的干净本地数据：演示用户（余额 0、没有 Key）、演示兑换码、几个命名清爽的分组，
-- 并把其它分组临时停用（备份在 _shots_group_backup，cleanup.sql 会还原）。
-- 用法见 capture.py 顶部说明。
BEGIN;

CREATE TABLE IF NOT EXISTS _shots_group_backup (id bigint PRIMARY KEY, status text NOT NULL);

INSERT INTO users (email, password_hash, role, balance, status, username)
VALUES ('shots@local.test', :'hash', 'user', 0, 'active', 'demo')
ON CONFLICT DO NOTHING;
UPDATE users SET balance = 0, password_hash = :'hash' WHERE email = 'shots@local.test';
DELETE FROM usage_logs WHERE user_id IN (SELECT id FROM users WHERE email = 'shots@local.test');
DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE email = 'shots@local.test');

DELETE FROM redeem_codes WHERE code LIKE 'DEMO-%';
INSERT INTO redeem_codes (code, type, value, status) VALUES
  ('DEMO-7K2Q-9XPA-4M8D', 'balance', 10, 'unused'),
  ('DEMO-3B6F-1ZRC-8T5W', 'balance', 10, 'unused');

-- 只在第一次备份：重复执行不会把「已停用」当成原状态
INSERT INTO _shots_group_backup (id, status)
SELECT id, status FROM groups WHERE deleted_at IS NULL AND name NOT IN ('Claude 经济', 'Claude 满血', 'GPT 经济', 'GPT 满血')
ON CONFLICT DO NOTHING;
UPDATE groups SET status = 'inactive' WHERE id IN (SELECT id FROM _shots_group_backup);

INSERT INTO groups (name, description, platform, rate_multiplier, status)
SELECT v.name, v.description, v.platform, v.rate, 'active'
FROM (VALUES
  ('Claude 经济', '性价比之选，适合日常编码', 'anthropic', 0.4),
  ('Claude 满血', '官方满血，稳定优先', 'anthropic', 1.0),
  ('GPT 经济', '性价比之选，适合日常编码', 'openai', 0.3),
  ('GPT 满血', '官方满血，稳定优先', 'openai', 1.0)
) AS v(name, description, platform, rate)
WHERE NOT EXISTS (SELECT 1 FROM groups g WHERE g.name = v.name AND g.deleted_at IS NULL);
UPDATE groups SET status = 'active' WHERE name IN ('Claude 经济', 'Claude 满血', 'GPT 经济', 'GPT 满血') AND deleted_at IS NULL;

-- 和线上一致：侧栏里有一个指向小铺的「购买套餐」自定义菜单
UPDATE settings SET value = '[{"id":"buy-plan","label":"购买套餐","icon_svg":"","url":"https://wzyp.cn/shop/PL2VA4CE","visibility":"user","sort_order":0,"open_in_new_tab":true}]'
WHERE key = 'custom_menu_items';
COMMIT;
