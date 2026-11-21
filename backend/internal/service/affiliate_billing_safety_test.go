//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBillingCommandCarriesIPWithoutChangingIdempotency(t *testing.T) {
	ip := "203.0.113.9"
	params := &postUsageBillingParams{Cost: &CostBreakdown{ActualCost: 3}, User: &User{ID: 1}, APIKey: &APIKey{ID: 2}, Account: &Account{ID: 3}}
	cmd := buildUsageBillingCommand("same-request", &UsageLog{IPAddress: &ip}, params)
	require.Equal(t, ip, cmd.ClientIP)
	first := cmd.RequestFingerprint
	ip = "203.0.113.99"
	retry := buildUsageBillingCommand("same-request", &UsageLog{IPAddress: &ip}, params)
	require.Equal(t, first, retry.RequestFingerprint)
}

func TestBillingExpiryInvalidatesCacheEvenWithPositiveRemainingBalance(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 10}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)
	remaining := 8.6
	syncBalanceCacheAfterDeduction(context.Background(), &postUsageBillingParams{Cost: &CostBreakdown{ActualCost: 0.4}, User: &User{ID: 1}}, &billingDeps{billingCacheService: svc}, &UsageBillingApplyResult{NewBalance: &remaining, TrialBonusExpiredAmount: 1})
	require.Equal(t, int64(1), cache.invalidateCalls.Load())
	require.Zero(t, cache.deductCalls.Load(), "subtracting only the request cost would preserve expired credit")
}

type affiliateAdmissionTestUsers struct {
	UserRepository
	state AffiliateUserAdmission
}

func (r *affiliateAdmissionTestUsers) PrepareAffiliateUserForRequest(context.Context, int64, string) (*AffiliateUserAdmission, error) {
	return &r.state, nil
}

type affiliateAdmissionTestKeys struct{ APIKeyRepository }

func (*affiliateAdmissionTestKeys) ListKeysByUserID(context.Context, int64) ([]string, error) {
	return nil, nil
}

type affiliateAdmissionTestCache struct{ calls int }

func (*affiliateAdmissionTestCache) InvalidateAPIKeyRateLimit(context.Context, int64) error {
	return nil
}
func (c *affiliateAdmissionTestCache) InvalidateUserBalance(context.Context, int64) error {
	c.calls++
	return nil
}

func TestAffiliateAdmissionRetriesCacheRefreshAfterEarlierExpiry(t *testing.T) {
	// A previous request committed expiry but failed its Redis invalidation.
	// The next request must retry invalidation despite ExpiredAmount being zero.
	users := &affiliateAdmissionTestUsers{state: AffiliateUserAdmission{Balance: 8, TrialExpired: true}}
	cache := &affiliateAdmissionTestCache{}
	svc := NewAPIKeyService(&affiliateAdmissionTestKeys{}, users, nil, nil, nil, nil, &config.Config{})
	svc.SetRateLimitCacheInvalidator(cache)
	original := &APIKey{ID: 2, User: &User{ID: 1, Balance: 9}}
	result, err := svc.PrepareAffiliateUserForRequest(context.Background(), original, "203.0.113.9")
	require.NoError(t, err)
	require.Equal(t, 8.0, result.User.Balance)
	require.Equal(t, 9.0, original.User.Balance)
	require.Equal(t, 1, cache.calls)
}

type expiredTrialBillingUsers struct{ UserRepository }

func (*expiredTrialBillingUsers) ExpiredAffiliateTrialAdmission(context.Context, int64) (*AffiliateUserAdmission, error) {
	return &AffiliateUserAdmission{Balance: 0, TrialExpired: true}, nil
}

func TestLongLivedConnectionCannotSpendExpiredTrialFromWarmCache(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 1}
	svc := NewBillingCacheService(cache, &expiredTrialBillingUsers{}, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)
	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1, Balance: 1}, nil, nil, nil, "")
	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.Equal(t, int64(1), cache.invalidateCalls.Load())
}
