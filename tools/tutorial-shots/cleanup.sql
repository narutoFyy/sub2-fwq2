-- 还原 seed.sql 的改动
BEGIN;
DELETE FROM usage_logs WHERE user_id IN (SELECT id FROM users WHERE email = 'shots@local.test');
DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE email = 'shots@local.test');
DELETE FROM users WHERE email = 'shots@local.test';
DELETE FROM redeem_codes WHERE code LIKE 'DEMO-%';
DELETE FROM groups WHERE name IN ('Claude 经济', 'Claude 满血', 'GPT 经济', 'GPT 满血');
UPDATE groups g SET status = b.status FROM _shots_group_backup b WHERE g.id = b.id;
DROP TABLE IF EXISTS _shots_group_backup;
UPDATE settings SET value = '[]' WHERE key = 'custom_menu_items';
COMMIT;
