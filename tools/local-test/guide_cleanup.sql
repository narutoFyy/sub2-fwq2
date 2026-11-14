BEGIN;
DELETE FROM usage_logs WHERE user_id IN (SELECT id FROM users WHERE email = 'guide-test@local.test');
DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE email = 'guide-test@local.test');
DELETE FROM redeem_codes WHERE code LIKE 'GUIDE-TEST-%';
DELETE FROM users WHERE email = 'guide-test@local.test';
UPDATE settings SET value = '[]' WHERE key = 'custom_menu_items';
COMMIT;
