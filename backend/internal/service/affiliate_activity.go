package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type AffiliateUserAdmission struct {
	Balance       float64
	ExpiredAmount float64
	TrialExpired  bool // Retain invalidation after an earlier failed cache refresh.
}

type affiliateUserAdmissionRepository interface {
	PrepareAffiliateUserForRequest(context.Context, int64, string) (*AffiliateUserAdmission, error)
}

// PrepareAffiliateUserForRequest runs before any balance-based authorization.
// It does not mutate the shared auth-cache object held by another request.
func (s *APIKeyService) PrepareAffiliateUserForRequest(ctx context.Context, apiKey *APIKey, clientIP string) (*APIKey, error) {
	if s == nil || apiKey == nil || apiKey.User == nil {
		return apiKey, nil
	}
	repo, ok := s.userRepo.(affiliateUserAdmissionRepository)
	if !ok {
		return apiKey, nil
	}
	state, err := repo.PrepareAffiliateUserForRequest(ctx, apiKey.User.ID, clientIP)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return apiKey, nil
	}
	keyCopy, userCopy := *apiKey, *apiKey.User
	userCopy.Balance = state.Balance
	keyCopy.User = &userCopy
	if state.ExpiredAmount > 0 || state.TrialExpired {
		s.InvalidateAuthCacheByUserID(ctx, userCopy.ID)
		if invalidator, ok := s.rateLimitCacheInvalid.(interface {
			InvalidateUserBalance(context.Context, int64) error
		}); ok {
			if err := invalidator.InvalidateUserBalance(ctx, userCopy.ID); err != nil {
				return nil, err
			}
		}
	}
	return &keyCopy, nil
}

// RecordRegistrationIP is called only on new-account bootstrap, never login.
func (s *AffiliateService) RecordRegistrationIP(ctx context.Context, userID int64) {
	if s == nil || s.repo == nil {
		return
	}
	repo, ok := s.repo.(interface {
		RecordRegistrationIP(context.Context, int64, string) error
	})
	if !ok {
		return
	}
	if err := repo.RecordRegistrationIP(ctx, userID, AffiliateClientIPFromContext(ctx)); err != nil {
		logger.LegacyPrintf("service.affiliate", "record registration IP failed user=%d: %v", userID, err)
	}
}

// refreshExpiredAffiliateTrial covers long-lived connections, where a later
// model turn may begin after the original authentication's expiry check.
func (s *BillingCacheService) refreshExpiredAffiliateTrial(ctx context.Context, userID int64) error {
	repo, ok := s.userRepo.(interface {
		ExpiredAffiliateTrialAdmission(context.Context, int64) (*AffiliateUserAdmission, error)
	})
	if !ok {
		return nil
	}
	state, err := repo.ExpiredAffiliateTrialAdmission(ctx, userID)
	if err != nil {
		return ErrBillingServiceUnavailable
	}
	if state == nil {
		return nil
	}
	if err := s.InvalidateUserBalance(ctx, userID); err != nil {
		return ErrBillingServiceUnavailable
	}
	if s.balanceBelowEligibilityThreshold(state.Balance) {
		return ErrInsufficientBalance
	}
	return nil
}
