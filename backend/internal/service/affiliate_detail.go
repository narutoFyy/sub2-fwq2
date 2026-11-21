package service

import (
	"context"
	"time"
)

// AffiliateInviteeTotals is computed over all invitees, independently of the
// page displayed in the invitation history.
type AffiliateInviteeTotals struct {
	InviteeCount       int
	QualifiedCount     int
	QualifiedPaidTotal float64
	PendingStepTotal   float64
	GrantedStepTotal   float64
	RemindCount        int
}

type AffiliateDetailRepository interface {
	GetInviteeTotals(context.Context, int64, time.Time) (*AffiliateInviteeTotals, error)
	ListInviteesPage(context.Context, int64, int, int) ([]AffiliateInvitee, error)
}

// LockRewardScope is acquired before order/user row locks in a transaction
// that also changes affiliate rewards, matching the billing lock order.
func (s *AffiliateService) LockRewardScope(ctx context.Context, userID int64) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if repo, ok := s.repo.(interface {
		LockAffiliateRewardScope(context.Context, int64) error
	}); ok {
		return repo.LockAffiliateRewardScope(ctx, userID)
	}
	return nil
}
