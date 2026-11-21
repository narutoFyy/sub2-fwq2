//go:build unit

package repository

import (
	"context"
	"database/sql"
	"os"
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

// AFFILIATE_BILLING_TEST_DSN must name a disposable, isolated PostgreSQL database.
func TestAffiliateBillingSafetyPostgres(t *testing.T) {
	dsn := os.Getenv("AFFILIATE_BILLING_TEST_DSN")
	if dsn == "" {
		t.Skip("set AFFILIATE_BILLING_TEST_DSN to an isolated PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, ApplyMigrations(ctx, db))
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	affiliate := &affiliateRepository{client: client}
	users := &userRepository{client: client}
	billing := NewUsageBillingRepository(client, db)
	exec := func(q string, args ...any) { _, err := db.ExecContext(ctx, q, args...); require.NoError(t, err) }
	scalar := func(q string, args ...any) float64 {
		var v float64
		require.NoError(t, db.QueryRowContext(ctx, q, args...).Scan(&v))
		return v
	}
	fixture := func(t *testing.T) (int64, int64) {
		exec("TRUNCATE users, payment_orders, usage_billing_dedup, usage_billing_dedup_archive RESTART IDENTITY CASCADE")
		u, err := client.User.Create().SetEmail("billing@example.com").SetPasswordHash("test").SetBalance(10).Save(ctx)
		require.NoError(t, err)
		_, err = affiliate.EnsureUserAffiliate(ctx, u.ID)
		require.NoError(t, err)
		key, err := client.APIKey.Create().SetUserID(u.ID).SetKey("billing-safety-key").SetName("test").Save(ctx)
		require.NoError(t, err)
		return u.ID, key.ID
	}
	t.Run("expired_trial_is_removed_at_admission_without_invite_page_visit", func(t *testing.T) {
		uid, _ := fixture(t)
		exec("UPDATE users SET balance=1 WHERE id=$1", uid)
		exec("UPDATE user_affiliates SET trial_bonus_status='active', trial_bonus_amount=1, trial_expires_at=NOW()-INTERVAL '1 second' WHERE user_id=$1", uid)
		state, err := users.PrepareAffiliateUserForRequest(ctx, uid, "203.0.113.9")
		require.NoError(t, err)
		require.Zero(t, state.Balance)
		require.Equal(t, 1.0, state.ExpiredAmount)
		turn, err := users.ExpiredAffiliateTrialAdmission(ctx, uid)
		require.NoError(t, err)
		require.NotNil(t, turn)
		require.Zero(t, turn.Balance)
		state, err = users.PrepareAffiliateUserForRequest(ctx, uid, "203.0.113.9")
		require.NoError(t, err)
		require.Zero(t, state.ExpiredAmount)
		require.Equal(t, 1.0, scalar("SELECT COUNT(*) FROM user_affiliate_ledger WHERE user_id=$1 AND action='trial_expire'", uid))
		var active, registration string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT last_active_ip, register_ip FROM user_affiliates WHERE user_id=$1", uid).Scan(&active, &registration))
		require.Equal(t, "203.0.113.9", active)
		require.Empty(t, registration, "an old user's call must not invent a registration IP")
	})
	t.Run("late_expiry_returns_committed_balance", func(t *testing.T) {
		uid, key := fixture(t)
		exec("UPDATE user_affiliates SET trial_bonus_status='active', trial_bonus_amount=1, trial_expires_at=NOW()-INTERVAL '1 second' WHERE user_id=$1", uid)
		out, err := billing.Apply(ctx, &service.UsageBillingCommand{RequestID: "expiry", APIKeyID: key, UserID: uid, BalanceCost: 0.4, ClientIP: "203.0.113.5"})
		require.NoError(t, err)
		require.NotNil(t, out.NewBalance)
		require.InDelta(t, 8.6, *out.NewBalance, 1e-8)
		require.InDelta(t, scalar("SELECT balance FROM users WHERE id=$1", uid), *out.NewBalance, 1e-8)
		require.Equal(t, 1.0, out.TrialBonusExpiredAmount)
	})
	t.Run("failed_consumption_rolls_back_charge_and_claim_then_retry_is_exactly_once", func(t *testing.T) {
		uid, key := fixture(t)
		exec(`CREATE OR REPLACE FUNCTION affiliate_test_reject_consumption() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.total_consumed_amount > OLD.total_consumed_amount THEN RAISE EXCEPTION 'injected consumption failure'; END IF; RETURN NEW; END $$`)
		exec(`CREATE TRIGGER affiliate_test_reject_consumption BEFORE UPDATE ON user_affiliates FOR EACH ROW EXECUTE FUNCTION affiliate_test_reject_consumption()`)
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS affiliate_test_reject_consumption ON user_affiliates; DROP FUNCTION IF EXISTS affiliate_test_reject_consumption()`)
		})
		cmd := &service.UsageBillingCommand{RequestID: "retry", APIKeyID: key, UserID: uid, BalanceCost: 3, ClientIP: "203.0.113.7"}
		out, err := billing.Apply(ctx, cmd)
		require.Error(t, err)
		require.Nil(t, out)
		require.Equal(t, 10.0, scalar("SELECT balance FROM users WHERE id=$1", uid))
		require.Zero(t, scalar("SELECT COUNT(*) FROM usage_billing_dedup"))
		exec("DROP TRIGGER affiliate_test_reject_consumption ON user_affiliates")
		out, err = billing.Apply(ctx, cmd)
		require.NoError(t, err)
		require.True(t, out.Applied)
		out, err = billing.Apply(ctx, cmd)
		require.NoError(t, err)
		require.False(t, out.Applied)
		require.Equal(t, 7.0, scalar("SELECT balance FROM users WHERE id=$1", uid))
		require.Equal(t, 3.0, scalar("SELECT total_consumed_amount FROM user_affiliates WHERE user_id=$1", uid))
		var active string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT last_active_ip FROM user_affiliates WHERE user_id=$1", uid).Scan(&active))
		require.Equal(t, "203.0.113.7", active)
	})
	t.Run("registration_ip_is_recorded_for_uninvited_users_and_remains_immutable", func(t *testing.T) {
		uid, _ := fixture(t)
		require.NoError(t, affiliate.RecordRegistrationIP(ctx, uid, "203.0.113.8"))
		require.NoError(t, affiliate.RecordRegistrationIP(ctx, uid, "203.0.113.99"))
		var registration, subnet string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT register_ip, register_ip_subnet FROM user_affiliates WHERE user_id=$1", uid).Scan(&registration, &subnet))
		require.Equal(t, "203.0.113.8", registration)
		require.Equal(t, "203.0.113.0/24", subnet)
	})
}
