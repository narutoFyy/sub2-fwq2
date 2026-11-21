-- Cumulative refunded paid value already applied to affiliate balances.
-- Keep this in the same transaction as clawbacks so repeated callbacks are harmless.
ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS affiliate_refunded_paid_amount DECIMAL(20,8) NOT NULL DEFAULT 0;
