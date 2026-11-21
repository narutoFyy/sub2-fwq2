//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type batchAffiliateBalanceCache struct {
	users         []int64
	contextErrors []error
}

func (c *batchAffiliateBalanceCache) InvalidateUserBalance(ctx context.Context, userID int64) error {
	c.users = append(c.users, userID)
	c.contextErrors = append(c.contextErrors, ctx.Err())
	return nil
}

type batchAffiliateContextAuthCache struct {
	fakeBatchImageAuthCacheInvalidator
	contextError error
	hasDeadline  bool
}

func (c *batchAffiliateContextAuthCache) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	c.contextError = ctx.Err()
	_, c.hasDeadline = ctx.Deadline()
	c.fakeBatchImageAuthCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
}

func TestBatchImageCommittedWalletInvalidatesCacheAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	auth, balance := &batchAffiliateContextAuthCache{}, &batchAffiliateBalanceCache{}
	invalidateBatchImageBalanceCaches(ctx, auth, balance, 123)
	require.Equal(t, []int64{123}, auth.userIDs)
	require.Equal(t, []int64{123}, balance.users)
	require.NoError(t, auth.contextError)
	require.True(t, auth.hasDeadline)
	require.NoError(t, balance.contextErrors[0])
}

type batchAffiliateCaptureRepo struct {
	fakeBatchImageBillingRepo
	result BatchImageBalanceHoldResult
}

func (r *batchAffiliateCaptureRepo) CaptureBatchImageBalance(ctx context.Context, cmd *BatchImageBalanceHoldCommand) (*BatchImageBalanceHoldResult, error) {
	if _, err := r.fakeBatchImageBillingRepo.CaptureBatchImageBalance(ctx, cmd); err != nil {
		return nil, err
	}
	return &r.result, nil
}

func TestBatchImageSettlementInvalidatesBothWalletCachesIncludingRetry(t *testing.T) {
	for _, applied := range []bool{true, false} {
		name := "new_capture"
		if !applied {
			name = "retry_after_committed_capture"
		}
		t.Run(name, func(t *testing.T) {
			repo := newFakeBatchImageRepository()
			job := testSettlingBatchImageJob("affiliate_cache_" + name)
			repo.jobs[job.BatchID] = job
			billing := &batchAffiliateCaptureRepo{result: BatchImageBalanceHoldResult{
				Applied: applied, AffiliateCreditedInviterID: 987, TrialBonusExpiredAmount: 0.2,
			}}
			auth, balance := &fakeBatchImageAuthCacheInvalidator{}, &batchAffiliateBalanceCache{}
			svc := &BatchImageSettlementService{Repo: repo, BillingRepo: billing,
				Pricing: &fakeBatchImagePricingResolver{unitPrice: 0.25}, AuthCache: auth, BalanceCache: balance}
			_, err := svc.Settle(context.Background(), job.BatchID)
			require.NoError(t, err)
			require.Equal(t, []int64{job.UserID, 987}, auth.userIDs)
			require.Equal(t, auth.userIDs, balance.users)
		})
	}
}

func TestBatchImageFailedCaptureDoesNotInvalidateUncommittedRewards(t *testing.T) {
	repo := newFakeBatchImageRepository()
	job := testSettlingBatchImageJob("affiliate_failed_capture")
	repo.jobs[job.BatchID] = job
	billing := &batchAffiliateCaptureRepo{fakeBatchImageBillingRepo: fakeBatchImageBillingRepo{captureErr: errors.New("reward update failed")},
		result: BatchImageBalanceHoldResult{AffiliateCreditedInviterID: 987}}
	auth, balance := &fakeBatchImageAuthCacheInvalidator{}, &batchAffiliateBalanceCache{}
	svc := &BatchImageSettlementService{Repo: repo, BillingRepo: billing,
		Pricing: &fakeBatchImagePricingResolver{unitPrice: 0.25}, AuthCache: auth, BalanceCache: balance}
	_, err := svc.Settle(context.Background(), job.BatchID)
	require.Error(t, err)
	require.Empty(t, auth.userIDs)
	require.Empty(t, balance.users)
	require.NotEqual(t, BatchImageJobStatusCompleted, repo.jobs[job.BatchID].Status)
}

func TestBatchImageReserveCarriesIPWithoutChangingIdempotency(t *testing.T) {
	job := testSettlingBatchImageJob("affiliate_submit_ip")
	repo := &fakeBatchImageBillingRepo{}
	for _, clientIP := range []string{"203.0.113.10", "203.0.113.11"} {
		ctx := WithAffiliateClientIP(context.Background(), clientIP)
		require.NoError(t, reserveBatchImageBalanceHold(ctx, repo, job, "same-request"))
	}
	require.Len(t, repo.reserves, 2)
	require.Equal(t, "203.0.113.10", repo.reserves[0].ClientIP)
	require.Equal(t, "203.0.113.11", repo.reserves[1].ClientIP)
	require.Equal(t, repo.reserves[0].RequestFingerprint, repo.reserves[1].RequestFingerprint)
}

func TestBatchImageSubmitAndTerminalReleaseInvalidateBalanceCache(t *testing.T) {
	svc, _, _, _, _ := newTestBatchImagePublicService(true)
	balance := &batchAffiliateBalanceCache{}
	svc.BalanceCache = balance
	owner := testBatchImageOwner()
	_, err := svc.Submit(context.Background(), owner, validBatchImageSubmitRequest(), "affiliate-reserve-cache")
	require.NoError(t, err)
	require.Contains(t, balance.users, owner.UserID)

	auth := &fakeBatchImageAuthCacheInvalidator{}
	job := testSettlingBatchImageJob("affiliate_release_cache")
	job.Status = BatchImageJobStatusCancelled
	processor := &BatchImageProviderProcessor{BillingRepo: &fakeBatchImageBillingRepo{}, AuthCache: auth, BalanceCache: balance}
	require.NoError(t, processor.releaseTerminalHold(context.Background(), job))
	require.Equal(t, []int64{job.UserID}, auth.userIDs)
	require.Equal(t, job.UserID, balance.users[len(balance.users)-1])
}
