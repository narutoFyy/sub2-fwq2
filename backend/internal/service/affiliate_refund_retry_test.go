//go:build unit

package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type affiliateRefundFailingRepo struct {
	affiliateV2RepoStub
	fail  bool
	calls int
}

func (r *affiliateRefundFailingRepo) ClawbackOrderAffiliateRewards(ctx context.Context, _ int64, userID int64, _ float64) ([]int64, error) {
	tx := dbent.TxFromContext(ctx)
	if tx == nil {
		return nil, errors.New("reward recovery must use refund transaction")
	}
	r.calls++
	if err := tx.Client().User.UpdateOneID(userID).AddBalance(-3).Exec(ctx); err != nil {
		return nil, err
	}
	if r.fail {
		return nil, errors.New("injected reward recovery failure")
	}
	return []int64{userID}, nil
}

func TestRefundAffiliateFailureLeavesConfirmedRetryWithoutGateway(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createPendingRefundOrderForTest(t, ctx, client, "affiliate-retry")
	require.NoError(t, client.User.UpdateOneID(order.UserID).SetBalance(30).Exec(ctx))
	require.NoError(t, client.PaymentOrder.UpdateOneID(order.ID).SetStatus(OrderStatusRefunding).Exec(ctx))
	// Mirror the production uniqueness constraint to cover a previously pending attempt.
	_, err := client.ExecContext(ctx, "CREATE UNIQUE INDEX refund_test_unique_audit ON payment_audit_logs(order_id, action)")
	require.NoError(t, err)
	repo := &affiliateRefundFailingRepo{fail: true}
	svc := &PaymentService{entClient: client, affiliateService: NewAffiliateService(repo, nil, nil, nil)}
	plan := svc.refundFinalizePlan(order)
	plan.BalanceToDeduct = 100 // Already held by ExecuteRefund; finalization must not debit it again.
	result, err := svc.markRefundOk(ctx, plan)
	require.Nil(t, result)
	require.ErrorContains(t, err, "injected reward recovery failure")
	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusRefundPending, reloaded.Status)
	user, err := client.User.Get(ctx, order.UserID)
	require.NoError(t, err)
	require.Equal(t, 30.0, user.Balance, "failed recovery is rolled back")
	detail, err := svc.latestRefundPendingDetail(ctx, order.ID)
	require.NoError(t, err)
	require.True(t, detail.GatewayConfirmed)
	require.True(t, detail.DeductionHeld)
	require.Equal(t, 100.0, detail.BalanceDeducted)
	count, err := client.PaymentAuditLog.Query().Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)), paymentauditlog.ActionEQ("REFUND_SUCCESS")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	repo.fail = false
	// No payment provider or user balance deductor is configured: retry must only finalize locally.
	result, err = svc.QueryAndFinalizeRefund(ctx, order.ID)
	require.NoError(t, err)
	require.True(t, result.Success)
	user, err = client.User.Get(ctx, order.UserID)
	require.NoError(t, err)
	require.Equal(t, 27.0, user.Balance)
	require.Equal(t, 2, repo.calls)
	_, err = svc.QueryAndFinalizeRefund(ctx, order.ID)
	require.Error(t, err)
	require.Equal(t, 2, repo.calls)
}

func TestRefundQuerySuccessPersistsConfirmationBeforeAffiliateFailure(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createPendingRefundOrderWithDetailForTest(t, ctx, client, "query-affiliate-failure", `{"refundID":"rf_test","deductionHeld":true,"deductionType":"balance","balanceDeducted":100}`)
	repo := &affiliateRefundFailingRepo{fail: true}
	svc := &PaymentService{entClient: client, affiliateService: NewAffiliateService(repo, nil, nil, nil), loadBalancer: &captureLoadBalancer{}}
	provider := &refundQueryProviderTestDouble{refundResponse: &payment.RefundResponse{Status: payment.ProviderStatusSuccess}}
	restore := replacePaymentProviderFactoryForTest(t, provider)
	defer restore()
	_, err := svc.QueryAndFinalizeRefund(ctx, order.ID)
	require.ErrorContains(t, err, "injected reward recovery failure")
	detail, err := svc.latestRefundPendingDetail(ctx, order.ID)
	require.NoError(t, err)
	require.True(t, detail.GatewayConfirmed)
	require.Equal(t, 100.0, detail.BalanceDeducted)
	repo.fail = false
	createPaymentProviderFromInstance = func(string, string, map[string]string) (payment.Provider, error) {
		return nil, errors.New("retry must not query gateway again")
	}
	result, err := svc.QueryAndFinalizeRefund(ctx, order.ID)
	require.NoError(t, err)
	require.True(t, result.Success)
}

func TestRefundRepeatedPendingFailureReturnsHeldBalanceWithUniqueAudit(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createPendingRefundOrderWithDetailForTest(t, ctx, client, "repeated-pending-failure", `{"deductionHeld":true,"deductionType":"balance","balanceDeducted":100}`)
	_, err := client.ExecContext(ctx, "CREATE UNIQUE INDEX refund_test_unique_audit ON payment_audit_logs(order_id, action)")
	require.NoError(t, err)
	svc := &PaymentService{entClient: client, userRepo: &mockUserRepo{
		updateBalanceFn: func(ctx context.Context, userID int64, amount float64) error {
			tx := dbent.TxFromContext(ctx)
			require.NotNil(t, tx)
			return tx.Client().User.UpdateOneID(userID).AddBalance(amount).Exec(ctx)
		},
	}}
	plan := svc.refundFinalizePlan(order)
	plan.deductionHeld = true
	plan.BalanceToDeduct = 100
	result, err := svc.finalizeRefundFailed(ctx, plan, errors.New("first gateway failure"))
	require.NoError(t, err)
	require.False(t, result.Success)
	firstAudit, err := client.PaymentAuditLog.Query().Where(
		paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)), paymentauditlog.ActionEQ("REFUND_FAILED")).Only(ctx)
	require.NoError(t, err)
	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = client.ExecContext(ctx, "UPDATE payment_audit_logs SET created_at = $1 WHERE id = $2", oldTime, firstAudit.ID)
	require.NoError(t, err)

	// A new attempt holds the balance again and reuses the pending audit.
	require.NoError(t, client.User.UpdateOneID(order.UserID).SetBalance(0).Exec(ctx))
	require.NoError(t, client.PaymentOrder.UpdateOneID(order.ID).SetStatus(OrderStatusRefunding).Exec(ctx))
	_, err = svc.markRefundPending(ctx, plan, &payment.RefundResponse{Status: payment.ProviderStatusPending})
	require.NoError(t, err)
	result, err = svc.finalizeRefundFailed(ctx, plan, errors.New("second gateway failure"))
	require.NoError(t, err)
	require.False(t, result.Success)
	user, err := client.User.Get(ctx, order.UserID)
	require.NoError(t, err)
	require.Equal(t, 100.0, user.Balance)
	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusRefundFailed, reloaded.Status)
	secondAudit, err := client.PaymentAuditLog.Query().Where(
		paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)), paymentauditlog.ActionEQ("REFUND_FAILED")).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, firstAudit.ID, secondAudit.ID)
	require.Contains(t, secondAudit.Detail, "second gateway failure")
	require.True(t, secondAudit.CreatedAt.After(oldTime), "retry must refresh the failure audit timestamp")
}

type affiliatePaidFailureRepo struct{ affiliateV2RepoStub }

func (r *affiliatePaidFailureRepo) RecordInviteePaidRecharge(context.Context, int64, float64, *int64, int) (*AffiliateRewardEvalOutcome, error) {
	return nil, errors.New("injected paid recharge failure")
}

func TestAffiliatePaidRechargeErrorPropagatesToPayment(t *testing.T) {
	settings := NewSettingService(&paymentFulfillmentSettingRepoStub{values: map[string]string{SettingKeyAffiliateEnabled: "true"}}, nil)
	svc := NewAffiliateService(&affiliatePaidFailureRepo{}, settings, nil, nil)
	orderID := int64(10)
	_, err := svc.AccrueInviteRebateForOrder(context.Background(), 1, 10, &orderID)
	require.ErrorContains(t, err, "injected paid recharge failure")
}
