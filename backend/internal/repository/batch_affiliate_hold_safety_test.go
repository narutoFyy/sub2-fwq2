//go:build unit

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
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
	"github.com/Wei-Shaw/sub2api/migrations"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// The suite truncates fixtures and temporarily rebuilds migration 255. Only its
// dedicated local database is accepted; it must never be pointed at an app DB.
func TestBatchAffiliateHoldSafetyPostgres(t *testing.T) {
	dsn := os.Getenv("BATCH_AFFILIATE_HOLD_TEST_DSN")
	if dsn == "" {
		t.Skip("set BATCH_AFFILIATE_HOLD_TEST_DSN to a disposable batch_affiliate_hold_test database")
	}
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "batch_affiliate_hold_test", strings.TrimPrefix(parsed.Path, "/"))
	require.Contains(t, []string{"localhost", "127.0.0.1"}, parsed.Hostname())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(12)
	require.NoError(t, ApplyMigrations(ctx, db))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	affiliate := &affiliateRepository{client: client}
	billing := NewUsageBillingRepository(client, db)
	exec := func(t *testing.T, query string, args ...any) {
		t.Helper()
		_, err := db.ExecContext(ctx, query, args...)
		require.NoError(t, err)
	}
	scalar := func(t *testing.T, query string, args ...any) float64 {
		t.Helper()
		var result float64
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&result))
		return result
	}
	textValue := func(t *testing.T, query string, args ...any) string {
		t.Helper()
		var result string
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&result))
		return result
	}
	type fixture struct{ userID, keyID, inviterID int64 }
	setup := func(t *testing.T) fixture {
		t.Helper()
		exec(t, "TRUNCATE users,payment_orders,batch_image_jobs,usage_billing_batch_image_holds,usage_billing_dedup,usage_billing_dedup_archive RESTART IDENTITY CASCADE")
		inviter, err := client.User.Create().SetEmail("batch-inviter@example.com").SetPasswordHash("isolated-test").Save(ctx)
		require.NoError(t, err)
		_, err = affiliate.EnsureUserAffiliate(ctx, inviter.ID)
		require.NoError(t, err)
		u, err := client.User.Create().SetEmail("batch-invitee@example.com").SetPasswordHash("isolated-test").SetBalance(10).Save(ctx)
		require.NoError(t, err)
		_, err = affiliate.EnsureUserAffiliate(ctx, u.ID)
		require.NoError(t, err)
		bound, err := affiliate.BindInviter(ctx, u.ID, inviter.ID)
		require.NoError(t, err)
		require.True(t, bound)
		key, err := client.APIKey.Create().SetUserID(u.ID).SetKey("batch-hold-isolated-test-key").SetName("test").Save(ctx)
		require.NoError(t, err)
		exec(t, `UPDATE user_affiliates SET trial_bonus_amount=1,trial_bonus_status='active',trial_granted_at=NOW(),trial_expires_at=NOW()+INTERVAL '1 day',total_paid_amount=10 WHERE user_id=$1`, u.ID)
		return fixture{u.ID, key.ID, inviter.ID}
	}
	command := func(f fixture, batch, action string, held, actual float64) *service.BatchImageBalanceHoldCommand {
		var requestID string
		switch action {
		case "reserve":
			requestID = service.BatchImageHoldRequestID(batch)
		case "capture":
			requestID = service.BatchImageCaptureRequestID(batch)
		case "release":
			requestID = service.BatchImageReleaseRequestID(batch)
		}
		return &service.BatchImageBalanceHoldCommand{RequestID: requestID, UserID: f.userID, APIKeyID: f.keyID, BatchID: batch, HoldAmount: held, ActualAmount: actual, ClientIP: "203.0.113.40"}
	}
	reserve := func(t *testing.T, f fixture, batch string, amount float64) {
		t.Helper()
		out, err := billing.ReserveBatchImageBalance(ctx, command(f, batch, "reserve", amount, 0))
		require.NoError(t, err)
		require.True(t, out.Applied)
	}
	assertWallet := func(t *testing.T, f fixture, balance, frozen, consumed, trialConsumed, expired float64) {
		t.Helper()
		require.InDelta(t, balance, scalar(t, "SELECT balance FROM users WHERE id=$1", f.userID), 1e-8)
		require.InDelta(t, frozen, scalar(t, "SELECT frozen_balance FROM users WHERE id=$1", f.userID), 1e-8)
		require.InDelta(t, consumed, scalar(t, "SELECT total_consumed_amount FROM user_affiliates WHERE user_id=$1", f.userID), 1e-8)
		require.InDelta(t, trialConsumed, scalar(t, "SELECT trial_consumed_amount FROM user_affiliates WHERE user_id=$1", f.userID), 1e-8)
		require.InDelta(t, expired, scalar(t, "SELECT trial_expired_amount FROM user_affiliates WHERE user_id=$1", f.userID), 1e-8)
	}
	makeDue := func(t *testing.T, f fixture) {
		t.Helper()
		exec(t, "UPDATE user_affiliates SET trial_expires_at=NOW()-INTERVAL '1 second' WHERE user_id=$1", f.userID)
	}

	t.Run("reservation_does_not_count_as_consumption", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "reserve-only", 2)
		assertWallet(t, f, 8, 2, 0, 0, 0)
		require.Equal(t, 1.0, scalar(t, "SELECT trial_reserved_amount FROM usage_billing_batch_image_holds WHERE batch_id='reserve-only'"))
		require.Zero(t, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants"))
	})
	t.Run("capture_counts_actual_cost_and_awards_once", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "capture-award", 4)
		cmd := command(f, "capture-award", "capture", 4, 3)
		out, err := billing.CaptureBatchImageBalance(ctx, cmd)
		require.NoError(t, err)
		require.True(t, out.Applied)
		assertWallet(t, f, 7, 0, 3, 1, 0)
		require.Equal(t, 3.0, scalar(t, "SELECT balance FROM users WHERE id=$1", f.inviterID))
		require.Equal(t, "203.0.113.40", textValue(t, "SELECT last_active_ip FROM user_affiliates WHERE user_id=$1", f.userID))
		out, err = billing.CaptureBatchImageBalance(ctx, cmd)
		require.NoError(t, err)
		require.False(t, out.Applied)
		assertWallet(t, f, 7, 0, 3, 1, 0)
		require.Equal(t, 1.0, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants"))
	})
	t.Run("release_before_expiry_restores_balance_without_consumption", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "release-active", 2)
		cmd := command(f, "release-active", "release", 2, 0)
		out, err := billing.ReleaseBatchImageBalance(ctx, cmd)
		require.NoError(t, err)
		require.True(t, out.Applied)
		out, err = billing.ReleaseBatchImageBalance(ctx, cmd)
		require.NoError(t, err)
		require.False(t, out.Applied)
		assertWallet(t, f, 10, 0, 0, 0, 0)
	})
	t.Run("release_before_reservation_does_not_poison_later_release", func(t *testing.T) {
		f := setup(t)
		cmd := command(f, "release-before-reserve", "release", 2, 0)
		out, err := billing.ReleaseBatchImageBalance(ctx, cmd)
		require.NoError(t, err)
		require.False(t, out.Applied)
		require.Zero(t, scalar(t, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id=$1", cmd.RequestID))
		reserve(t, f, "release-before-reserve", 2)
		out, err = billing.ReleaseBatchImageBalance(ctx, cmd)
		require.NoError(t, err)
		require.True(t, out.Applied)
		assertWallet(t, f, 10, 0, 0, 0, 0)
	})
	t.Run("durable_hold_releases_even_if_old_reserve_claim_was_removed", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "durable-release", 2)
		holdID := service.BatchImageHoldRequestID("durable-release")
		exec(t, "DELETE FROM usage_billing_dedup WHERE request_id=$1 AND api_key_id=$2", holdID, f.keyID)
		exec(t, "DELETE FROM usage_billing_dedup_archive WHERE request_id=$1 AND api_key_id=$2", holdID, f.keyID)
		out, err := billing.ReleaseBatchImageBalance(ctx, command(f, "durable-release", "release", 2, 0))
		require.NoError(t, err)
		require.True(t, out.Applied)
		assertWallet(t, f, 10, 0, 0, 0, 0)
		require.Equal(t, "released", textValue(t, "SELECT status FROM usage_billing_batch_image_holds WHERE batch_id='durable-release'"))
	})
	t.Run("expired_partial_capture_reclaims_only_unused_trial", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "capture-expired", 2)
		makeDue(t, f)
		_, err := affiliate.ExpireUserTrialBonus(ctx, f.userID)
		require.NoError(t, err)
		assertWallet(t, f, 8, 2, 0, 0, 0)
		_, err = billing.CaptureBatchImageBalance(ctx, command(f, "capture-expired", "capture", 2, 0.4))
		require.NoError(t, err)
		assertWallet(t, f, 9, 0, 0.4, 0.4, 0.6)
		_, err = affiliate.ExpireUserTrialBonus(ctx, f.userID)
		require.NoError(t, err)
		assertWallet(t, f, 9, 0, 0.4, 0.4, 0.6)
	})
	t.Run("full_cross_period_capture_uses_reserved_trial_without_expiry_charge", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "capture-full-expired", 1)
		makeDue(t, f)
		_, err := affiliate.ExpireUserTrialBonus(ctx, f.userID)
		require.NoError(t, err)
		_, err = billing.CaptureBatchImageBalance(ctx, command(f, "capture-full-expired", "capture", 1, 1))
		require.NoError(t, err)
		assertWallet(t, f, 9, 0, 1, 1, 0)
		require.Equal(t, "used", textValue(t, "SELECT trial_bonus_status FROM user_affiliates WHERE user_id=$1", f.userID))
	})
	t.Run("expired_release_cannot_resurrect_trial_credit", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "release-expired", 2)
		makeDue(t, f)
		_, err := billing.ReleaseBatchImageBalance(ctx, command(f, "release-expired", "release", 2, 0))
		require.NoError(t, err)
		assertWallet(t, f, 9, 0, 0, 0, 1)
		_, err = affiliate.ExpireUserTrialBonus(ctx, f.userID)
		require.NoError(t, err)
		assertWallet(t, f, 9, 0, 0, 0, 1)
	})
	t.Run("multiple_holds_and_regular_charge_do_not_share_reserved_trial", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "multi-a", 0.7)
		reserve(t, f, "multi-b", 0.7)
		_, err := billing.Apply(ctx, &service.UsageBillingCommand{RequestID: "regular-with-holds", APIKeyID: f.keyID, UserID: f.userID, BalanceCost: 0.5})
		require.NoError(t, err)
		assertWallet(t, f, 8.1, 1.4, 0.5, 0, 0)
		makeDue(t, f)
		_, err = affiliate.ExpireUserTrialBonus(ctx, f.userID)
		require.NoError(t, err)
		for _, batch := range []string{"multi-a", "multi-b"} {
			_, err = billing.ReleaseBatchImageBalance(ctx, command(f, batch, "release", 0.7, 0))
			require.NoError(t, err)
		}
		assertWallet(t, f, 8.5, 0, 0.5, 0, 1)
	})
	t.Run("released_live_trial_can_fund_regular_usage_before_deadline", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "live-a", 0.7)
		reserve(t, f, "live-b", 0.7)
		_, err := billing.CaptureBatchImageBalance(ctx, command(f, "live-a", "capture", 0.7, 0.2))
		require.NoError(t, err)
		_, err = billing.Apply(ctx, &service.UsageBillingCommand{RequestID: "regular-after-capture", APIKeyID: f.keyID, UserID: f.userID, BalanceCost: 0.5})
		require.NoError(t, err)
		_, err = billing.ReleaseBatchImageBalance(ctx, command(f, "live-b", "release", 0.7, 0))
		require.NoError(t, err)
		assertWallet(t, f, 9.3, 0, 0.7, 0.7, 0)
		makeDue(t, f)
		_, err = affiliate.ExpireUserTrialBonus(ctx, f.userID)
		require.NoError(t, err)
		assertWallet(t, f, 9, 0, 0.7, 0.7, 0.3)
	})
	t.Run("capture_and_release_have_one_terminal_winner", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			f := setup(t)
			reserve(t, f, "terminal-race", 2)
			reserve(t, f, "other-hold", 3)
			start := make(chan struct{})
			captureErr, releaseErr := make(chan error, 1), make(chan error, 1)
			go func() {
				<-start
				_, err := billing.CaptureBatchImageBalance(ctx, command(f, "terminal-race", "capture", 2, 1))
				captureErr <- err
			}()
			go func() {
				<-start
				_, err := billing.ReleaseBatchImageBalance(ctx, command(f, "terminal-race", "release", 2, 0))
				releaseErr <- err
			}()
			close(start)
			capErr, relErr := <-captureErr, <-releaseErr
			require.NoError(t, relErr)
			state := textValue(t, "SELECT status FROM usage_billing_batch_image_holds WHERE batch_id='terminal-race'")
			if state == "captured" {
				require.NoError(t, capErr)
				assertWallet(t, f, 6, 3, 1, 1, 0)
			} else {
				require.Equal(t, "released", state)
				require.Error(t, capErr, "a released hold cannot be reported as successfully captured")
				assertWallet(t, f, 7, 3, 0, 0, 0)
			}
			require.Equal(t, "reserved", textValue(t, "SELECT status FROM usage_billing_batch_image_holds WHERE batch_id='other-hold'"))
		}
	})
	t.Run("concurrent_holds_and_regular_usage_preserve_trial_budget", func(t *testing.T) {
		f := setup(t)
		start := make(chan struct{})
		errs := make(chan error, 3)
		var wg sync.WaitGroup
		for _, batch := range []string{"parallel-a", "parallel-b"} {
			wg.Add(1)
			go func(batch string) {
				defer wg.Done()
				<-start
				_, err := billing.ReserveBatchImageBalance(ctx, command(f, batch, "reserve", 0.6, 0))
				errs <- err
			}(batch)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := billing.Apply(ctx, &service.UsageBillingCommand{RequestID: "parallel-regular", APIKeyID: f.keyID, UserID: f.userID, BalanceCost: 0.3})
			errs <- err
		}()
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		trialUsed := scalar(t, "SELECT trial_consumed_amount FROM user_affiliates WHERE user_id=$1", f.userID)
		reserved := scalar(t, "SELECT SUM(trial_reserved_amount) FROM usage_billing_batch_image_holds WHERE user_id=$1 AND status='reserved'", f.userID)
		require.InDelta(t, 1, trialUsed+reserved, 1e-8)
		makeDue(t, f)
		for _, batch := range []string{"parallel-a", "parallel-b"} {
			_, err := billing.ReleaseBatchImageBalance(ctx, command(f, batch, "release", 0.6, 0))
			require.NoError(t, err)
		}
		assertWallet(t, f, 10-0.3-(1-trialUsed), 0, 0.3, trialUsed, 1-trialUsed)
	})
	t.Run("capture_failure_rolls_back_all_financial_and_reward_effects", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "capture-retry", 4)
		exec(t, `CREATE OR REPLACE FUNCTION batch_affiliate_reject_consumption() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.total_consumed_amount > OLD.total_consumed_amount THEN RAISE EXCEPTION 'injected batch consumption failure'; END IF; RETURN NEW; END $$`)
		exec(t, `CREATE TRIGGER batch_affiliate_reject_consumption BEFORE UPDATE ON user_affiliates FOR EACH ROW EXECUTE FUNCTION batch_affiliate_reject_consumption()`)
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS batch_affiliate_reject_consumption ON user_affiliates; DROP FUNCTION IF EXISTS batch_affiliate_reject_consumption()`)
		})
		cmd := command(f, "capture-retry", "capture", 4, 3)
		_, err := billing.CaptureBatchImageBalance(ctx, cmd)
		require.ErrorContains(t, err, "injected batch consumption failure")
		assertWallet(t, f, 6, 4, 0, 0, 0)
		require.Equal(t, "reserved", textValue(t, "SELECT status FROM usage_billing_batch_image_holds WHERE batch_id='capture-retry'"))
		require.Zero(t, scalar(t, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id=$1", cmd.RequestID))
		exec(t, "DROP TRIGGER batch_affiliate_reject_consumption ON user_affiliates")
		_, err = billing.CaptureBatchImageBalance(ctx, cmd)
		require.NoError(t, err)
		assertWallet(t, f, 7, 0, 3, 1, 0)
		require.Equal(t, 3.0, scalar(t, "SELECT balance FROM users WHERE id=$1", f.inviterID))
	})
	t.Run("hold_snapshot_rejects_changed_owner_key_or_amount", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "snapshot", 2)
		for _, field := range []string{"user", "key", "amount"} {
			cmd := command(f, "snapshot", "capture", 2, 0.5)
			switch field {
			case "user":
				cmd.UserID = f.inviterID
			case "key":
				cmd.APIKeyID++
			case "amount":
				cmd.HoldAmount = 1
			}
			_, err := billing.CaptureBatchImageBalance(ctx, cmd)
			require.Error(t, err, "changed %s must not consume a different hold", field)
			assertWallet(t, f, 8, 2, 0, 0, 0)
		}
	})
	t.Run("refund_clawback_then_release_does_not_recover_trial_twice", func(t *testing.T) {
		for _, expireFirst := range []bool{false, true} {
			t.Run(fmt.Sprint(expireFirst), func(t *testing.T) {
				f := setup(t)
				reserve(t, f, "clawback-release", 0.6)
				if expireFirst {
					makeDue(t, f)
					_, err := affiliate.ExpireUserTrialBonus(ctx, f.userID)
					require.NoError(t, err)
				}
				exec(t, "INSERT INTO payment_orders(id,user_id,amount,pay_amount,status,expires_at) VALUES(9900,$1,10,10,'COMPLETED',NOW())", f.userID)
				_, err := affiliate.ClawbackOrderAffiliateRewards(ctx, 9900, f.userID, 10)
				require.NoError(t, err)
				makeDue(t, f)
				_, err = billing.ReleaseBatchImageBalance(ctx, command(f, "clawback-release", "release", 0.6, 0))
				require.NoError(t, err)
				expired := 0.0
				if expireFirst {
					expired = 0.4
				}
				assertWallet(t, f, 9, 0, 0, 0, expired)
				require.Equal(t, "clawed_back", textValue(t, "SELECT trial_bonus_status FROM user_affiliates WHERE user_id=$1", f.userID))
			})
		}
	})
	t.Run("capture_after_refund_keeps_trial_clawed_back_and_counts_actual_cost", func(t *testing.T) {
		f := setup(t)
		reserve(t, f, "clawback-capture", 1)
		exec(t, "INSERT INTO payment_orders(id,user_id,amount,pay_amount,status,expires_at) VALUES(9901,$1,10,10,'COMPLETED',NOW())", f.userID)
		_, err := affiliate.ClawbackOrderAffiliateRewards(ctx, 9901, f.userID, 10)
		require.NoError(t, err)
		makeDue(t, f)
		_, err = billing.CaptureBatchImageBalance(ctx, command(f, "clawback-capture", "capture", 1, 0.4))
		require.NoError(t, err)
		assertWallet(t, f, 8.6, 0, 0.4, 0, 0)
		require.Equal(t, "clawed_back", textValue(t, "SELECT trial_bonus_status FROM user_affiliates WHERE user_id=$1", f.userID))
		require.Zero(t, scalar(t, "SELECT COUNT(*) FROM affiliate_reward_grants"))
	})
	t.Run("reserve_after_expiry_cannot_freeze_expired_credit", func(t *testing.T) {
		f := setup(t)
		makeDue(t, f)
		reserve(t, f, "late-reserve", 2)
		assertWallet(t, f, 7, 2, 0, 0, 1)
		require.Zero(t, scalar(t, "SELECT trial_reserved_amount FROM usage_billing_batch_image_holds WHERE batch_id='late-reserve'"))
	})
	t.Run("legacy_fully_frozen_trial_releases_without_resurrection_or_double_recovery", func(t *testing.T) {
		matches, err := fs.Glob(migrations.FS, "255_*.sql")
		require.NoError(t, err)
		require.Len(t, matches, 1)
		migration, err := migrations.FS.ReadFile(matches[0])
		require.NoError(t, err)
		for _, priorStatus := range []string{"active", "expired", "clawed_back"} {
			t.Run(priorStatus, func(t *testing.T) {
				f := setup(t)
				expired := 0.0
				if priorStatus == "expired" {
					expired = 1
				}
				exec(t, "UPDATE users SET balance=0,frozen_balance=1 WHERE id=$1", f.userID)
				exec(t, `UPDATE user_affiliates SET trial_bonus_status=$1,trial_expired_amount=$2,trial_granted_at=NOW()-INTERVAL '2 days',trial_expires_at=NOW()-INTERVAL '1 second' WHERE user_id=$3`, priorStatus, expired, f.userID)
				exec(t, `INSERT INTO batch_image_jobs(batch_id,user_id,api_key_id,provider,model,status,item_count,hold_amount,created_at)
VALUES('legacy-frozen',$1,$2,'test','test','running',1,1,NOW()-INTERVAL '1 day')`, f.userID, f.keyID)
				exec(t, "INSERT INTO usage_billing_dedup_archive(request_id,api_key_id,request_fingerprint,created_at) VALUES($1,$2,'legacy',NOW()-INTERVAL '1 day')", service.BatchImageHoldRequestID("legacy-frozen"), f.keyID)
				exec(t, string(migration))
				reserved := 0.0
				if priorStatus == "active" {
					reserved = 1
				}
				require.Equal(t, reserved, scalar(t, "SELECT trial_reserved_amount FROM usage_billing_batch_image_holds WHERE batch_id='legacy-frozen'"))
				cmd := command(f, "legacy-frozen", "release", 1, 0)
				_, err := billing.ReleaseBatchImageBalance(ctx, cmd)
				require.NoError(t, err)
				if priorStatus == "active" {
					expired = 1
				}
				assertWallet(t, f, 1-reserved, 0, 0, 0, expired)
				exec(t, string(migration))
				out, err := billing.ReleaseBatchImageBalance(ctx, cmd)
				require.NoError(t, err)
				require.False(t, out.Applied)
				assertWallet(t, f, 1-reserved, 0, 0, 0, expired)
			})
		}
	})
	t.Run("legacy_migration_reconstructs_hot_and_archived_hold_terminals", func(t *testing.T) {
		f := setup(t)
		matches, err := fs.Glob(migrations.FS, "255_*.sql")
		require.NoError(t, err)
		require.Len(t, matches, 1)
		migration, err := migrations.FS.ReadFile(matches[0])
		require.NoError(t, err)
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()
		mustExec := func(query string, args ...any) {
			_, err := tx.ExecContext(ctx, query, args...)
			require.NoError(t, err)
		}
		mustScalar := func(query string, args ...any) float64 {
			var v float64
			require.NoError(t, tx.QueryRowContext(ctx, query, args...).Scan(&v))
			return v
		}
		mustExec("DROP TABLE usage_billing_batch_image_holds")
		mustExec("ALTER TABLE user_affiliates DROP COLUMN trial_consumed_amount")
		for i, batch := range []string{"legacy-hot", "legacy-archive", "legacy-captured", "legacy-released", "legacy-never-held"} {
			mustExec(`INSERT INTO batch_image_jobs(batch_id,user_id,api_key_id,provider,model,status,item_count,estimated_cost,hold_amount,actual_cost,created_at)
VALUES($1,$2,$3,'test','test','running',1,0.6,0.6,0.2,NOW()+$4*INTERVAL '1 second')`, batch, f.userID, f.keyID, i)
			if batch == "legacy-never-held" {
				continue
			}
			if batch == "legacy-archive" {
				mustExec("INSERT INTO usage_billing_dedup_archive(request_id,api_key_id,request_fingerprint,created_at) VALUES($1,$2,'legacy',NOW())", service.BatchImageHoldRequestID(batch), f.keyID)
			} else {
				mustExec("INSERT INTO usage_billing_dedup(request_id,api_key_id,request_fingerprint) VALUES($1,$2,'legacy')", service.BatchImageHoldRequestID(batch), f.keyID)
			}
		}
		mustExec("INSERT INTO usage_billing_dedup_archive(request_id,api_key_id,request_fingerprint,created_at) VALUES($1,$2,'legacy',NOW())", service.BatchImageCaptureRequestID("legacy-captured"), f.keyID)
		mustExec("INSERT INTO usage_billing_dedup(request_id,api_key_id,request_fingerprint) VALUES($1,$2,'legacy')", service.BatchImageReleaseRequestID("legacy-released"), f.keyID)
		mustExec(string(migration))
		require.Equal(t, 4.0, mustScalar("SELECT COUNT(*) FROM usage_billing_batch_image_holds"))
		require.InDelta(t, 0.6, mustScalar("SELECT trial_reserved_amount FROM usage_billing_batch_image_holds WHERE batch_id='legacy-hot'"), 1e-8)
		require.InDelta(t, 0.4, mustScalar("SELECT trial_reserved_amount FROM usage_billing_batch_image_holds WHERE batch_id='legacy-archive'"), 1e-8)
		require.Equal(t, 1.0, mustScalar("SELECT COUNT(*) FROM usage_billing_batch_image_holds WHERE batch_id='legacy-captured' AND status='captured'"))
		require.Equal(t, 1.0, mustScalar("SELECT COUNT(*) FROM usage_billing_batch_image_holds WHERE batch_id='legacy-released' AND status='released'"))
		require.Zero(t, mustScalar("SELECT SUM(trial_reserved_amount) FROM usage_billing_batch_image_holds WHERE status<>'reserved'"))
		// Simulate real accounting after the first migration. Replay must preserve
		// both the consumed principal and the remaining per-hold assignments.
		mustExec("UPDATE user_affiliates SET trial_consumed_amount=0.25 WHERE user_id=$1", f.userID)
		mustExec("UPDATE usage_billing_batch_image_holds SET trial_reserved_amount=0.35 WHERE batch_id='legacy-hot'")
		mustExec(string(migration))
		require.Equal(t, 4.0, mustScalar("SELECT COUNT(*) FROM usage_billing_batch_image_holds"))
		require.Equal(t, 0.25, mustScalar("SELECT trial_consumed_amount FROM user_affiliates WHERE user_id=$1", f.userID))
		require.Equal(t, 0.35, mustScalar("SELECT trial_reserved_amount FROM usage_billing_batch_image_holds WHERE batch_id='legacy-hot'"))
		require.Equal(t, 0.4, mustScalar("SELECT trial_reserved_amount FROM usage_billing_batch_image_holds WHERE batch_id='legacy-archive'"))
		// All schema and fixture changes roll back together, preserving the migrated test DB.
	})
}
