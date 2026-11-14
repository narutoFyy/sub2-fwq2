-- 清理 plaza_seed.sql 写入的测试数据，并恢复「广场必须登录」。
BEGIN;
DELETE FROM account_groups WHERE account_id IN (SELECT id FROM accounts WHERE name LIKE '[测试]%')
                              OR group_id IN (SELECT id FROM groups WHERE name LIKE '[测试]%');
DELETE FROM accounts WHERE name LIKE '[测试]%';
DELETE FROM groups WHERE name LIKE '[测试]%';
UPDATE settings SET value = 'true' WHERE key = 'model_plaza_require_auth';
COMMIT;
