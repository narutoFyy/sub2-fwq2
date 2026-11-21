//go:build unit

package repository

import (
	"context"
	"database/sql"
	"net/url"
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

func TestAffiliateDetailAllInviteesAndPagination(t *testing.T) {
	dsn := os.Getenv("AFFILIATE_SUMMARY_TEST_DSN")
	if dsn == "" {
		t.Skip("set AFFILIATE_SUMMARY_TEST_DSN to an isolated localhost affiliate_summary_test database")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"127.0.0.1", "localhost"}, u.Hostname())
	require.Equal(t, "/affiliate_summary_test", u.Path)
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, ApplyMigrations(ctx, db))
	_, err = db.ExecContext(ctx, "TRUNCATE TABLE users, payment_orders, usage_billing_dedup, usage_billing_dedup_archive RESTART IDENTITY CASCADE")
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := &affiliateRepository{client: client}
	inviter, err := client.User.Create().SetEmail("summary-inviter@example.test").SetPasswordHash("test-only").Save(ctx)
	require.NoError(t, err)
	_, err = repo.EnsureUserAffiliate(ctx, inviter.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
WITH seeded AS (
  INSERT INTO users(email,password_hash)
  SELECT 'summary-' || n || '@example.test','test-only' FROM generate_series(1,101) n RETURNING id
)
INSERT INTO user_affiliates(user_id,aff_code,inviter_id,created_at,total_paid_amount,total_consumed_amount,qualified_at,step1_reward_status,step1_reward_amount)
SELECT id,'SUMMARY'||id,$1,NOW()-INTERVAL '3 days',10,3,NOW()-INTERVAL '1 day','granted',3 FROM seeded`, inviter.ID)
	require.NoError(t, err)
	for i, amount := range []float64{10, 50, 50} {
		user, err := client.User.Create().SetEmail([]string{"pending-one@example.test", "pending-two@example.test", "expired@example.test"}[i]).SetPasswordHash("test-only").Save(ctx)
		require.NoError(t, err)
		_, err = repo.EnsureUserAffiliate(ctx, user.ID)
		require.NoError(t, err)
		created := time.Now()
		if i == 2 {
			created = created.Add(-31 * 24 * time.Hour)
		}
		_, err = db.ExecContext(ctx, `UPDATE user_affiliates SET inviter_id=$1, created_at=$2, total_paid_amount=$3,
step1_reward_status='pending_consume',step1_reward_amount=3,
step2_reward_status=CASE WHEN $3::numeric >= 50 THEN 'pending_consume' ELSE 'none' END WHERE user_id=$4`, inviter.ID, created, amount, user.ID)
		require.NoError(t, err)
	}
	svc := service.NewAffiliateService(repo, nil, nil, nil)
	first, err := svc.GetAffiliateDetailPage(ctx, inviter.ID, 1, 2)
	require.NoError(t, err)
	require.Len(t, first.Invitees, 2)
	require.Equal(t, 104, first.InviteesTotal)
	require.Equal(t, 104, first.AffCount)
	require.Equal(t, 101, first.QualifiedInviteeCount)
	require.InDelta(t, 1010, first.QualifiedInviteesPaidTotal, 1e-8)
	require.InDelta(t, 11, first.PendingStepBonusTotal, 1e-8)
	require.InDelta(t, 303, first.GrantedStepBonusTotal, 1e-8)
	require.Equal(t, 2, first.RemindConsumeCount)
	for _, invitee := range first.Invitees {
		require.False(t, invitee.Qualified)
		require.Contains(t, invitee.Email, "***")
	}
	second, err := svc.GetAffiliateDetailPage(ctx, inviter.ID, 2, 2)
	require.NoError(t, err)
	require.Equal(t, first.QualifiedInviteeCount, second.QualifiedInviteeCount)
	require.Equal(t, first.PendingStepBonusTotal, second.PendingStepBonusTotal)
	require.Equal(t, first.GrantedStepBonusTotal, second.GrantedStepBonusTotal)
	require.NotEqual(t, first.Invitees[0].UserID, second.Invitees[0].UserID)
	last, err := svc.GetAffiliateDetailPage(ctx, inviter.ID, 999999999, 2)
	require.NoError(t, err)
	require.Equal(t, 52, last.InviteesPage)
	require.Len(t, last.Invitees, 2)
	require.Equal(t, "expired_window", last.Invitees[1].DisplayStatus)
	require.Zero(t, last.Invitees[1].PendingBonusAmount)
	_, err = db.ExecContext(ctx, `UPDATE user_affiliates SET qualified_at=created_at+INTERVAL '31 days' WHERE user_id=$1`, second.Invitees[0].UserID)
	require.NoError(t, err)
	totals, err := repo.GetInviteeTotals(ctx, inviter.ID, time.Now())
	require.NoError(t, err)
	require.Equal(t, 100, totals.QualifiedCount, "legacy late qualifications must not contribute to a ladder")
}
