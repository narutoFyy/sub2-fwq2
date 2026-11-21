package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// These destructive fixture tests only accept the explicitly named disposable database.
func TestAffiliateCoreIntegrity(t *testing.T) {
	dsn := os.Getenv("AFFILIATE_CORE_TEST_DSN")
	if dsn == "" {
		t.Skip("set AFFILIATE_CORE_TEST_DSN to a disposable affiliate_core_test database")
	}
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "affiliate_core_test", strings.TrimPrefix(parsed.Path, "/"))
	require.Contains(t, []string{"localhost", "127.0.0.1"}, parsed.Hostname())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(12)
	require.NoError(t, db.PingContext(ctx))
	require.NoError(t, ApplyMigrations(ctx, db))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := &affiliateRepository{client: client}
	exec := func(t *testing.T, q string, args ...any) {
		t.Helper()
		_, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
	}
	scalar := func(t *testing.T, q string, args ...any) float64 {
		t.Helper()
		var v float64
		require.NoError(t, db.QueryRowContext(ctx, q, args...).Scan(&v))
		return v
	}
	reset := func(t *testing.T) {
		exec(t, "TRUNCATE TABLE users,payment_orders,usage_billing_dedup,usage_billing_dedup_archive RESTART IDENTITY CASCADE")
	}
	mkuser := func(t *testing.T, label string) int64 {
		t.Helper()
		u, err := client.User.Create().SetEmail(label + "@gmail.com").SetPasswordHash("isolated-test").Save(ctx)
		require.NoError(t, err)
		_, err = repo.EnsureUserAffiliate(ctx, u.ID)
		require.NoError(t, err)
		return u.ID
	}
	bind := func(t *testing.T, inviter int64, label string) int64 {
		t.Helper()
		id := mkuser(t, label)
		ok, err := repo.BindInviter(ctx, id, inviter)
		require.NoError(t, err)
		require.True(t, ok)
		return id
	}
	order := func(t *testing.T, id, oid int64, paid float64) {
		t.Helper()
		exec(t, `INSERT INTO payment_orders(id,user_id,amount,pay_amount,status,expires_at) VALUES($1,$2,$3,$3,'COMPLETED',NOW())`, oid, id, paid)
	}
	concurrent := func(t *testing.T, count int, fn func(int) error) {
		t.Helper()
		start := make(chan struct{})
		errs := make(chan error, count)
		var wg sync.WaitGroup
		for i := 0; i < count; i++ {
			wg.Add(1)
			go func(i int) { defer wg.Done(); <-start; errs <- fn(i) }(i)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
	}

	t.Run("payment_consumption_grants_and_pending_status_use_valid_SQL", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "sql-inv")
		id := bind(t, inv, "sql-child")
		result, err := repo.RecordInviteePaidRecharge(ctx, id, 50, nil, 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 2.0, scalar(t, `SELECT (step1_reward_status = 'pending_consume')::int + (step2_reward_status = 'pending_consume')::int FROM user_affiliates WHERE user_id=$1`, id))
		result, err = repo.RecordInviteeConsumption(ctx, id, 3, "")
		require.NoError(t, err)
		require.Equal(t, 3.0, result.Step1Granted)
		require.Equal(t, 5.0, result.Step2Granted)
		require.Equal(t, 8.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
		_, err = repo.RecordInviteeConsumption(ctx, id, 1, "")
		require.NoError(t, err)
		require.Equal(t, 8.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
	})
	t.Run("same_IP_and_subnet_limit_survives_concurrent_registration", func(t *testing.T) {
		for _, sameIP := range []bool{true, false} {
			t.Run(fmt.Sprint(sameIP), func(t *testing.T) {
				reset(t)
				inv := mkuser(t, "trial-inv")
				ids := []int64{bind(t, inv, "trial-a"), bind(t, inv, "trial-b"), bind(t, inv, "trial-c")}
				concurrent(t, 3, func(i int) error {
					ip := "203.0.113.7"
					if !sameIP {
						ip = fmt.Sprintf("203.0.113.%d", i+7)
					}
					_, err := repo.ProcessInviteePostBind(ctx, ids[i], inv, ip, "203.0.113.0/24", 0)
					return err
				})
				require.Equal(t, 2.0, scalar(t, "SELECT COUNT(*) FROM user_affiliates WHERE trial_granted_at IS NOT NULL"))
			})
		}
	})
	t.Run("global_daily_limit_survives_concurrent_registration", func(t *testing.T) {
		reset(t)
		exec(t, `WITH seed AS (INSERT INTO users(email,password_hash) SELECT 'seed-'||n||'@gmail.com','isolated-test' FROM generate_series(1,149)n RETURNING id) INSERT INTO user_affiliates(user_id,aff_code,trial_bonus_status,trial_granted_at) SELECT id,'SEED'||id,'active',NOW() FROM seed`)
		inv := mkuser(t, "global-inv")
		ids := []int64{bind(t, inv, "global-a"), bind(t, inv, "global-b"), bind(t, inv, "global-c")}
		concurrent(t, 3, func(i int) error {
			_, err := repo.ProcessInviteePostBind(ctx, ids[i], inv, fmt.Sprintf("198.51.%d.7", i), fmt.Sprintf("198.51.%d.0/24", i), 0)
			return err
		})
		require.Equal(t, 150.0, scalar(t, "SELECT COUNT(*) FROM user_affiliates WHERE trial_granted_at IS NOT NULL"))
	})
	t.Run("inviter_daily_limit_and_ladder_claim_are_serialized", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "auto-inv")
		ids := []int64{bind(t, inv, "auto-a"), bind(t, inv, "auto-b"), bind(t, inv, "auto-c")}
		exec(t, "UPDATE user_affiliates SET total_paid_amount=10 WHERE inviter_id=$1", inv)
		exec(t, `INSERT INTO affiliate_reward_grants(operation_id,inviter_id,reward_type,amount,status,granted_at) SELECT 'seedauto-'||n,$1,'test_seed',0,'granted',NOW() FROM generate_series(1,9)n`, inv)
		concurrent(t, 3, func(i int) error { _, err := repo.RecordInviteeConsumption(ctx, ids[i], 3, ""); return err })
		require.Equal(t, 10.0, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants WHERE inviter_id=$1 AND status='granted'", inv))
		require.Equal(t, 2.0, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants WHERE inviter_id=$1 AND status='pending_review'", inv))
		require.Equal(t, 3.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
	})
	t.Run("refund_revokes_step_ladder_and_trial_without_replaying", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "refund-inv")
		ids := make([]int64, 5)
		for i := range ids {
			id := bind(t, inv, fmt.Sprintf("refund-%d", i))
			ids[i] = id
			order(t, id, int64(1000+i), 10)
			_, err := repo.ProcessInviteePostBind(ctx, id, inv, fmt.Sprintf("198.18.%d.1", i), fmt.Sprintf("198.18.%d.0/24", i), 0)
			require.NoError(t, err)
			_, err = repo.RecordInviteePaidRecharge(ctx, id, 10, nil, 0)
			require.NoError(t, err)
			_, err = repo.RecordInviteeConsumption(ctx, id, 3, "")
			require.NoError(t, err)
		}
		require.Equal(t, 25.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
		for i, id := range ids {
			_, err := repo.ClawbackOrderAffiliateRewards(ctx, int64(1000+i), id, 10)
			require.NoError(t, err)
			_, err = repo.ClawbackOrderAffiliateRewards(ctx, int64(1000+i), id, 10)
			require.NoError(t, err)
		}
		require.Equal(t, 0.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
		require.Equal(t, 6.0, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants WHERE status='clawed_back'"))
		require.Equal(t, 0.0, scalar(t, "SELECT COUNT(*) FROM user_affiliates WHERE qualified_at IS NOT NULL"))
		require.Equal(t, 5.0, scalar(t, "SELECT COUNT(*) FROM user_affiliate_ledger WHERE action='clawback_trial'"))
	})
	t.Run("refund_cancels_pending_review_and_pending_consumption", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "pending-inv")
		id := bind(t, inv, "pending-child")
		order(t, id, 2000, 50)
		exec(t, "UPDATE user_affiliates SET total_paid_amount=50,total_consumed_amount=3,qualified_at=NOW(),step1_reward_status='pending_review',step1_reward_amount=3,step2_reward_status='pending_consume',step2_reward_amount=5 WHERE user_id=$1", id)
		var gid int64
		require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO affiliate_reward_grants(operation_id,inviter_id,invitee_id,reward_type,amount,status) VALUES($1,$2,$3,'step1',3,'pending_review') RETURNING id`, fmt.Sprintf("step1:%d", id), inv, id).Scan(&gid))
		_, err := repo.ClawbackOrderAffiliateRewards(ctx, 2000, id, 50)
		require.NoError(t, err)
		_, err = repo.ApproveRewardGrant(ctx, gid, inv, "after refund")
		require.ErrorIs(t, err, service.ErrAffiliateRewardGrantNotPending)
		require.Equal(t, 0.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
		require.Equal(t, 2.0, scalar(t, `SELECT (step1_reward_status='clawed_back')::int+(step2_reward_status='clawed_back')::int FROM user_affiliates WHERE user_id=$1`, id))
	})
	t.Run("approval_rechecks_live_qualification", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "approval-inv")
		id := bind(t, inv, "approval-child")
		exec(t, "UPDATE user_affiliates SET total_paid_amount=0,total_consumed_amount=3,qualified_at=NOW(),step1_reward_status='pending_review' WHERE user_id=$1", id)
		var gid int64
		require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO affiliate_reward_grants(operation_id,inviter_id,invitee_id,reward_type,amount,status) VALUES($1,$2,$3,'step1',3,'pending_review') RETURNING id`, fmt.Sprintf("step1:%d", id), inv, id).Scan(&gid))
		_, err := repo.ApproveRewardGrant(ctx, gid, inv, "")
		require.Error(t, err)
		exec(t, "UPDATE user_affiliates SET total_paid_amount=10 WHERE user_id=$1", id)
		_, err = repo.ApproveRewardGrant(ctx, gid, inv, "")
		require.NoError(t, err)
		require.Equal(t, 3.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
	})
	t.Run("partial_refund_is_cumulative_proportional_and_thaws_net_rebate", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "partial-inv")
		id := bind(t, inv, "partial-child")
		oid := int64(3000)
		order(t, id, oid, 100)
		exec(t, "UPDATE user_affiliates SET total_paid_amount=100 WHERE user_id=$1", id)
		_, err := repo.AccrueQuota(ctx, inv, id, 5, 72, &oid)
		require.NoError(t, err)
		for _, cumulative := range []float64{1, 1, 2, 1} {
			_, err = repo.ClawbackOrderAffiliateRewards(ctx, oid, id, cumulative)
			require.NoError(t, err)
		}
		require.InDelta(t, 4.9, scalar(t, "SELECT aff_frozen_quota FROM user_affiliates WHERE user_id=$1", inv), 1e-8)
		require.Equal(t, 98.0, scalar(t, "SELECT total_paid_amount FROM user_affiliates WHERE user_id=$1", id))
		exec(t, "UPDATE user_affiliate_ledger SET frozen_until=NOW()-INTERVAL '1 second' WHERE frozen_until IS NOT NULL")
		thawed, err := repo.ThawFrozenQuota(ctx, inv)
		require.NoError(t, err)
		require.InDelta(t, 4.9, thawed, 1e-8)
		_, err = repo.ClawbackOrderAffiliateRewards(ctx, oid, id, 3)
		require.NoError(t, err)
		require.InDelta(t, 4.85, scalar(t, "SELECT aff_quota FROM user_affiliates WHERE user_id=$1", inv), 1e-8)
		_, _, err = repo.TransferQuotaToBalance(ctx, inv)
		require.NoError(t, err)
		_, err = repo.ClawbackOrderAffiliateRewards(ctx, oid, id, 100)
		require.NoError(t, err)
		require.InDelta(t, 0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv), 1e-8)
		require.InDelta(t, 0, scalar(t, "SELECT aff_history_quota FROM user_affiliates WHERE user_id=$1", inv), 1e-8)
		require.InDelta(t, 0, scalar(t, "SELECT SUM(amount) FROM user_affiliate_ledger WHERE action IN ('accrue','clawback_rebate')"), 1e-8)
	})
	t.Run("pending_consume_or_previous_qualification_does_not_extend_30_days", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "deadline-inv")
		id := bind(t, inv, "deadline-child")
		exec(t, "UPDATE user_affiliates SET created_at=NOW()-INTERVAL '31 days',total_paid_amount=10,step1_reward_status='pending_consume',step1_reward_amount=3 WHERE user_id=$1", id)
		_, err := repo.RecordInviteeConsumption(ctx, id, 3, "")
		require.NoError(t, err)
		require.Equal(t, 0.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
		exec(t, "UPDATE user_affiliates SET qualified_at=created_at+INTERVAL '1 day',step1_reward_status='granted' WHERE user_id=$1", id)
		_, err = repo.RecordInviteePaidRecharge(ctx, id, 40, nil, 0)
		require.NoError(t, err)
		require.Equal(t, 0.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
	})
	t.Run("refund_rechecks_100_person_paid_threshold_and_pending_ladders", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "tier100-inv")
		exec(t, `WITH seed AS (INSERT INTO users(email,password_hash) SELECT 'tier100-'||n||'@gmail.com','isolated-test' FROM generate_series(1,100)n RETURNING id)
INSERT INTO user_affiliates(user_id,aff_code,inviter_id,total_paid_amount,total_consumed_amount,qualified_at) SELECT id,'TIER'||id,$1,30,3,NOW() FROM seed`, inv)
		id := int64(scalar(t, "SELECT MIN(user_id) FROM user_affiliates WHERE inviter_id=$1", inv))
		order(t, id, 5000, 30)
		exec(t, `INSERT INTO affiliate_reward_grants(operation_id,inviter_id,reward_type,tier_target,amount,status,granted_at)
VALUES($1,$2,'tier_100',100,100,'granted',NOW()),($3,$2,'tier_50',50,60,'pending_review',NULL)`, fmt.Sprintf("tier:%d:100", inv), inv, fmt.Sprintf("tier:%d:50", inv))
		exec(t, "UPDATE users SET balance=100 WHERE id=$1", inv)
		_, err := repo.ClawbackOrderAffiliateRewards(ctx, 5000, id, 1)
		require.NoError(t, err)
		require.Equal(t, 100.0, scalar(t, "SELECT COUNT(*) FROM user_affiliates WHERE qualified_at IS NOT NULL"))
		require.Equal(t, 0.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
		require.Equal(t, 1.0, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants WHERE reward_type='tier_100' AND status='clawed_back'"))
		gid := int64(scalar(t, "SELECT id FROM affiliate_reward_grants WHERE reward_type='tier_50'"))
		_, err = repo.ApproveRewardGrant(ctx, gid, inv, "")
		require.NoError(t, err)
		exec(t, "UPDATE user_affiliates SET total_paid_amount=0,qualified_at=NULL WHERE inviter_id=$1 AND user_id<>$2", inv, id)
		_, err = repo.ClawbackOrderAffiliateRewards(ctx, 5000, id, 2)
		require.NoError(t, err)
		require.Equal(t, 0.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
	})
	t.Run("concurrent_refund_and_approval_cannot_leave_reward_credited", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "race-inv")
		id := bind(t, inv, "race-child")
		order(t, id, 6000, 10)
		exec(t, "UPDATE user_affiliates SET total_paid_amount=10,total_consumed_amount=3,qualified_at=NOW(),step1_reward_status='pending_review' WHERE user_id=$1", id)
		var gid int64
		require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO affiliate_reward_grants(operation_id,inviter_id,invitee_id,reward_type,amount,status) VALUES($1,$2,$3,'step1',3,'pending_review') RETURNING id`, fmt.Sprintf("step1:%d", id), inv, id).Scan(&gid))
		concurrent(t, 2, func(i int) error {
			if i == 0 {
				_, err := repo.ClawbackOrderAffiliateRewards(ctx, 6000, id, 10)
				return err
			}
			_, err := repo.ApproveRewardGrant(ctx, gid, inv, "")
			if errors.Is(err, service.ErrAffiliateRewardGrantNotPending) {
				return nil
			}
			return err
		})
		require.Equal(t, 0.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
		require.Equal(t, 1.0, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants WHERE status='clawed_back'"))
	})
	t.Run("transfer_waits_before_wallet_lock_during_legacy_refund", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "transfer-refund-inv")
		id := bind(t, inv, "transfer-refund-child")
		order(t, id, 6500, 10)
		exec(t, "UPDATE users SET balance=20 WHERE id=$1", id)
		exec(t, "UPDATE user_affiliates SET total_paid_amount=10,aff_quota=5,aff_history_quota=5 WHERE user_id=$1", id)

		refundTx, err := client.Tx(ctx)
		require.NoError(t, err)
		defer func() { _ = refundTx.Rollback() }()
		refundCtx := dbent.NewTxContext(ctx, refundTx)
		require.NoError(t, repo.LockAffiliateRewardScope(refundCtx, id))
		// Old pending refunds deduct balance at finalization before recovering rewards.
		require.NoError(t, refundTx.Client().User.UpdateOneID(id).AddBalance(-10).Exec(refundCtx))
		rows, err := refundTx.Client().QueryContext(refundCtx, "SELECT pg_backend_pid()")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var refundPID int
		require.NoError(t, rows.Scan(&refundPID))
		require.NoError(t, rows.Close())

		transferCtx, cancelTransfer := context.WithTimeout(ctx, 5*time.Second)
		type transferResult struct {
			amount  float64
			balance float64
			err     error
		}
		result := make(chan transferResult, 1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			amount, balance, err := repo.TransferQuotaToBalance(transferCtx, id)
			result <- transferResult{amount, balance, err}
		}()
		defer func() {
			cancelTransfer()
			_ = refundTx.Rollback()
			<-done
		}()
		require.Eventually(t, func() bool {
			var waiting int
			err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pg_stat_activity
WHERE datname = current_database() AND wait_event_type = 'Lock' AND wait_event = 'advisory'
AND $1 = ANY(pg_blocking_pids(pid))`, refundPID).Scan(&waiting)
			return err == nil && waiting == 1
		}, 2*time.Second, 10*time.Millisecond, "transfer must wait on the scope before acquiring the wallet row")
		// This would deadlock if transfer held the wallet while waiting on our user row.
		_, err = repo.ClawbackOrderAffiliateRewards(refundCtx, 6500, id, 10)
		require.NoError(t, err)
		require.NoError(t, refundTx.Commit())
		transferred := <-result
		require.NoError(t, transferred.err)
		require.Equal(t, 5.0, transferred.amount)
		require.Equal(t, 15.0, transferred.balance)
		require.Equal(t, 0.0, scalar(t, "SELECT aff_quota FROM user_affiliates WHERE user_id=$1", id))
		require.Equal(t, 10.0, scalar(t, "SELECT affiliate_refunded_paid_amount FROM payment_orders WHERE id=6500"))
	})
	t.Run("refund_failure_rolls_back_watermark_and_all_wallet_changes", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "rollback-inv")
		id := bind(t, inv, "rollback-child")
		oid := int64(7000)
		order(t, id, oid, 100)
		exec(t, "UPDATE user_affiliates SET total_paid_amount=100 WHERE user_id=$1", id)
		_, err := repo.AccrueQuota(ctx, inv, id, 5, 72, &oid)
		require.NoError(t, err)
		exec(t, `CREATE OR REPLACE FUNCTION core_reject_clawback() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='clawback_rebate' THEN RAISE EXCEPTION 'injected refund ledger failure'; END IF; RETURN NEW; END $$`)
		exec(t, `CREATE TRIGGER core_reject_clawback BEFORE INSERT ON user_affiliate_ledger FOR EACH ROW EXECUTE FUNCTION core_reject_clawback()`)
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS core_reject_clawback ON user_affiliate_ledger; DROP FUNCTION IF EXISTS core_reject_clawback()`)
		})
		_, err = repo.ClawbackOrderAffiliateRewards(ctx, oid, id, 1)
		require.Error(t, err)
		require.Equal(t, 0.0, scalar(t, "SELECT affiliate_refunded_paid_amount FROM payment_orders WHERE id=$1", oid))
		require.Equal(t, 100.0, scalar(t, "SELECT total_paid_amount FROM user_affiliates WHERE user_id=$1", id))
		require.Equal(t, 5.0, scalar(t, "SELECT aff_frozen_quota FROM user_affiliates WHERE user_id=$1", inv))
		exec(t, `DROP TRIGGER core_reject_clawback ON user_affiliate_ledger`)
		_, err = repo.ClawbackOrderAffiliateRewards(ctx, oid, id, 1)
		require.NoError(t, err)
		require.InDelta(t, 4.95, scalar(t, "SELECT aff_frozen_quota FROM user_affiliates WHERE user_id=$1", inv), 1e-8)
	})
	t.Run("approval_after_day30_accepts_only_qualification_earned_within_window", func(t *testing.T) {
		reset(t)
		inv := mkuser(t, "late-review-inv")
		id := bind(t, inv, "late-review-child")
		exec(t, "UPDATE user_affiliates SET created_at=NOW()-INTERVAL '31 days',qualified_at=NOW()-INTERVAL '2 days',total_paid_amount=10,total_consumed_amount=3,step1_reward_status='pending_review' WHERE user_id=$1", id)
		var gid int64
		require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO affiliate_reward_grants(operation_id,inviter_id,invitee_id,reward_type,amount,status,created_at) VALUES($1,$2,$3,'step1',3,'pending_review',NOW()-INTERVAL '2 days') RETURNING id`, fmt.Sprintf("step1:%d", id), inv, id).Scan(&gid))
		_, err := repo.ApproveRewardGrant(ctx, gid, inv, "")
		require.NoError(t, err)
		require.Equal(t, 3.0, scalar(t, "SELECT balance FROM users WHERE id=$1", inv))
	})

}
