-- 新手引导本地测试数据：一个余额为 0 的新用户 + 两个未使用的兑换码 + 一个站内「购买套餐」外链菜单（模拟线上）。
-- 密码见执行脚本时传入的 :hash；清理见 guide_cleanup.sql。
BEGIN;
INSERT INTO users (email, password_hash, role, balance, status, username)
VALUES ('guide-test@local.test', :'hash', 'user', 0, 'active', 'guide-test')
ON CONFLICT DO NOTHING;
INSERT INTO redeem_codes (code, type, value, status) VALUES ('GUIDE-TEST-0001', 'balance', 10, 'unused'), ('GUIDE-TEST-0002', 'balance', 10, 'unused')
ON CONFLICT DO NOTHING;
UPDATE settings SET value = '[{"id":"buy-plan","label":"购买套餐","icon_svg":"","url":"https://example.com/shop","visibility":"user","sort_order":0,"open_in_new_tab":true}]' WHERE key = 'custom_menu_items';
COMMIT;
