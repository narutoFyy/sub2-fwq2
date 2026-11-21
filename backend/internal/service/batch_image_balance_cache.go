package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type batchImageBalanceCacheInvalidator interface {
	InvalidateUserBalance(context.Context, int64) error
}

func optionalBatchImageBalanceCache(cache *BillingCacheService) batchImageBalanceCacheInvalidator {
	if cache == nil {
		return nil
	}
	return cache
}

// Holds, capture refunds, expiry and referral awards all change the wallet.
// Clear both caches only after the repository transaction has committed.
func invalidateBatchImageBalanceCaches(ctx context.Context, auth APIKeyAuthCacheInvalidator, balance batchImageBalanceCacheInvalidator, userID int64) {
	if userID <= 0 {
		return
	}
	// The wallet transaction is already committed. Client cancellation must not
	// prevent invalidating a previously cached, higher balance.
	detachedCtx := context.WithoutCancel(ctx)
	if auth != nil {
		cacheCtx, cancel := context.WithTimeout(detachedCtx, cacheWriteTimeout)
		auth.InvalidateAuthCacheByUserID(cacheCtx, userID)
		cancel()
	}
	if balance != nil {
		cacheCtx, cancel := context.WithTimeout(detachedCtx, cacheWriteTimeout)
		defer cancel()
		if err := balance.InvalidateUserBalance(cacheCtx, userID); err != nil {
			logger.L().Warn("batch_image.balance_cache_invalidation_failed", zap.Int64("user_id", userID), zap.Error(err))
		}
	}
}
