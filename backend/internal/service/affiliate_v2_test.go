//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsMainstreamTrialEmail(t *testing.T) {
	t.Parallel()

	allowed := []string{
		"user@qq.com",
		"user@foxmail.com",
		"user@163.com",
		"user@126.com",
		"user@gmail.com",
		"user+alias@googlemail.com",
		"user@outlook.com",
		"user@hotmail.com",
		"user@live.com",
		"user@icloud.com",
		"user@me.com",
		"  USER@GMAIL.COM  ",
	}
	for _, email := range allowed {
		require.True(t, IsMainstreamTrialEmail(email), "expected mainstream email allowed: %s", email)
	}

	rejected := []string{
		"",
		"invalid",
		"user@tempmail.org",
		"user@mailinator.com",
		"user@mycustomdomain.xyz",
		"user@fake-gmail.com",
	}
	for _, email := range rejected {
		require.False(t, IsMainstreamTrialEmail(email), "expected non-mainstream email rejected: %s", email)
	}
}

func TestComputeIPSubnet(t *testing.T) {
	t.Parallel()

	ip, subnet := ComputeIPSubnet("203.0.113.45:54321")
	require.Equal(t, "203.0.113.45", ip)
	require.Equal(t, "203.0.113.0/24", subnet)

	ip2, subnet2 := ComputeIPSubnet("203.0.113.199")
	require.Equal(t, "203.0.113.199", ip2)
	require.Equal(t, "203.0.113.0/24", subnet2)

	ip6, subnet6 := ComputeIPSubnet("2001:db8:abcd:0012:1234:5678:9abc:def0")
	require.Equal(t, "2001:db8:abcd:12:1234:5678:9abc:def0", ip6)
	require.Equal(t, "2001:db8:abcd:12::/64", subnet6)

	emptyIP, emptySubnet := ComputeIPSubnet("   ")
	require.Equal(t, "", emptyIP)
	require.Equal(t, "", emptySubnet)
}

func TestFormatAffiliateInviteeV2_TwoStepDisplayStates(t *testing.T) {
	t.Parallel()
	now := time.Now()
	joined := now.Add(-2 * 24 * time.Hour)

	// 1) 实付满 10 元但未消耗满 3 元 → 待到账 3 元 + 提醒好友配置 API Key 试用
	inv1 := &AffiliateInvitee{
		UserID:              101,
		CreatedAt:           &joined,
		TotalPaidAmount:     10,
		TotalConsumedAmount: 1.2,
		Step1RewardStatus:   "pending_consume",
		Step2RewardStatus:   "none",
	}
	FormatAffiliateInviteeV2(inv1, now)
	require.Equal(t, "pending_consume_3", inv1.DisplayStatus)
	require.InDelta(t, 3.0, inv1.PendingBonusAmount, 1e-9)
	require.InDelta(t, 0.0, inv1.GrantedBonusAmount, 1e-9)
	require.True(t, inv1.NeedsConsumeReminder)

	// 2) 一次性首充 50 元但未消耗满 3 元 → 先显示「待到账 8 元」
	inv2 := &AffiliateInvitee{
		UserID:              102,
		CreatedAt:           &joined,
		TotalPaidAmount:     50,
		TotalConsumedAmount: 0.5,
		Step1RewardStatus:   "pending_consume",
		Step2RewardStatus:   "pending_consume",
	}
	FormatAffiliateInviteeV2(inv2, now)
	require.Equal(t, "pending_consume_8", inv2.DisplayStatus)
	require.InDelta(t, 8.0, inv2.PendingBonusAmount, 1e-9)
	require.InDelta(t, 0.0, inv2.GrantedBonusAmount, 1e-9)
	require.True(t, inv2.NeedsConsumeReminder)

	// 3) 首充 20 元且已消耗满 3 元 → 已到账 3 元，待冲满 50 元再得 5 元
	qualTime := now.Add(-1 * time.Hour)
	inv3 := &AffiliateInvitee{
		UserID:              103,
		CreatedAt:           &joined,
		TotalPaidAmount:     20,
		TotalConsumedAmount: 4.5,
		Step1RewardStatus:   "granted",
		Step2RewardStatus:   "none",
		QualifiedAt:         &qualTime,
	}
	FormatAffiliateInviteeV2(inv3, now)
	require.Equal(t, "granted_3_pending_5", inv3.DisplayStatus)
	require.InDelta(t, 0.0, inv3.PendingBonusAmount, 1e-9)
	require.InDelta(t, 3.0, inv3.GrantedBonusAmount, 1e-9)
	require.False(t, inv3.NeedsConsumeReminder)
	require.True(t, inv3.Qualified)

	// 4) 累计实付满 50 元且已消耗满 3 元 → 全部已到账 8 元
	inv4 := &AffiliateInvitee{
		UserID:              104,
		CreatedAt:           &joined,
		TotalPaidAmount:     70,
		TotalConsumedAmount: 6.0,
		Step1RewardStatus:   "granted",
		Step2RewardStatus:   "granted",
		QualifiedAt:         &qualTime,
	}
	FormatAffiliateInviteeV2(inv4, now)
	require.Equal(t, "granted_8", inv4.DisplayStatus)
	require.InDelta(t, 0.0, inv4.PendingBonusAmount, 1e-9)
	require.InDelta(t, 8.0, inv4.GrantedBonusAmount, 1e-9)
	require.False(t, inv4.NeedsConsumeReminder)
	require.True(t, inv4.Qualified)
}

type affiliateV2RepoStub struct {
	AffiliateRepository
	summary           *AffiliateSummary
	invitees          []AffiliateInvitee
	grants            []AffiliateRewardGrantRecord
	postBindCalled    bool
	postBindIP        string
	postBindSubnet    string
	paidRechargeCalls int
	consumptionCalls  int
}

func (s *affiliateV2RepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	if s.summary != nil {
		return s.summary, nil
	}
	return &AffiliateSummary{UserID: userID, AffCode: "TESTCODE2026"}, nil
}

func (s *affiliateV2RepoStub) ThawFrozenQuota(_ context.Context, _ int64) (float64, error) {
	return 0, nil
}

func (s *affiliateV2RepoStub) ListInvitees(_ context.Context, _ int64, _ int) ([]AffiliateInvitee, error) {
	return s.invitees, nil
}

func (s *affiliateV2RepoStub) ProcessInviteePostBind(_ context.Context, _, _ int64, clientIP, ipSubnet string, _ int) (*AffiliateTrialGrantOutcome, error) {
	s.postBindCalled = true
	s.postBindIP = clientIP
	s.postBindSubnet = ipSubnet
	return &AffiliateTrialGrantOutcome{Granted: true, Amount: 1.0, TodayGrantedCount: 1}, nil
}

func (s *affiliateV2RepoStub) RecordInviteePaidRecharge(_ context.Context, inviteeID int64, _ float64, _ *int64, _ int) (*AffiliateRewardEvalOutcome, error) {
	s.paidRechargeCalls++
	return &AffiliateRewardEvalOutcome{InviterID: 1, InviteeID: inviteeID, Step1Granted: 3.0}, nil
}

func (s *affiliateV2RepoStub) RecordInviteeConsumption(_ context.Context, inviteeID int64, _ float64, _ string) (*AffiliateRewardEvalOutcome, error) {
	s.consumptionCalls++
	return &AffiliateRewardEvalOutcome{InviterID: 1, InviteeID: inviteeID, Step1Granted: 3.0}, nil
}

func (s *affiliateV2RepoStub) ExpireUserTrialBonus(_ context.Context, _ int64) (float64, error) {
	return 0, nil
}

func (s *affiliateV2RepoStub) SweepExpiredTrialBonuses(_ context.Context, _ int) ([]int64, error) {
	return nil, nil
}

func (s *affiliateV2RepoStub) ListUserRewardGrants(_ context.Context, _ int64) ([]AffiliateRewardGrantRecord, error) {
	return s.grants, nil
}

func (s *affiliateV2RepoStub) ListAdminRewardGrants(_ context.Context, _ AffiliateRecordFilter) ([]AffiliateRewardGrantRecord, int64, error) {
	return s.grants, int64(len(s.grants)), nil
}

func (s *affiliateV2RepoStub) ApproveRewardGrant(_ context.Context, grantID, reviewerID int64, note string) (*AffiliateRewardGrantRecord, error) {
	return &AffiliateRewardGrantRecord{ID: grantID, InviterID: 1, Status: "granted", ReviewedBy: &reviewerID, ReviewNote: note}, nil
}

func (s *affiliateV2RepoStub) RejectRewardGrant(_ context.Context, grantID, reviewerID int64, note string) (*AffiliateRewardGrantRecord, error) {
	return &AffiliateRewardGrantRecord{ID: grantID, InviterID: 1, Status: "rejected", ReviewedBy: &reviewerID, ReviewNote: note}, nil
}

func (s *affiliateV2RepoStub) SetInviterTrialGrantPaused(_ context.Context, _ int64, _ bool, _ string) error {
	return nil
}

func (s *affiliateV2RepoStub) ClawbackOrderAffiliateRewards(_ context.Context, _, _ int64, _ float64) ([]int64, error) {
	return []int64{1, 2}, nil
}

func TestGetAffiliateDetail_EnrichesV2LadderAndTotals(t *testing.T) {
	t.Parallel()
	now := time.Now()
	joined := now.Add(-24 * time.Hour)
	rate := 5.0

	repo := &affiliateV2RepoStub{
		summary: &AffiliateSummary{
			UserID:               1,
			AffCode:              "YFMAIL2026",
			AffCount:             3,
			AffQuota:             320.5,
			AffFrozenQuota:       4.0,
			AffHistoryQuota:      324.5,
			AffRebateRatePercent: &rate,
		},
		invitees: []AffiliateInvitee{
			{
				UserID:              11,
				Email:               "friendA@qq.com",
				CreatedAt:           &joined,
				TotalPaidAmount:     10,
				TotalConsumedAmount: 3.5,
				TotalRebate:         0.5,
				Step1RewardStatus:   "granted",
				Step2RewardStatus:   "none",
				QualifiedAt:         &now,
			},
			{
				UserID:              12,
				Email:               "friendB@gmail.com",
				CreatedAt:           &joined,
				TotalPaidAmount:     70,
				TotalConsumedAmount: 10,
				TotalRebate:         3.5,
				Step1RewardStatus:   "granted",
				Step2RewardStatus:   "granted",
				QualifiedAt:         &now,
			},
			{
				UserID:              13,
				Email:               "friendC@163.com",
				CreatedAt:           &joined,
				TotalPaidAmount:     10,
				TotalConsumedAmount: 0.8,
				TotalRebate:         0.5,
				Step1RewardStatus:   "pending_consume",
				Step2RewardStatus:   "none",
			},
		},
		grants: []AffiliateRewardGrantRecord{
			{ID: 1, InviterID: 1, RewardType: "tier_5", TierTarget: 5, Amount: 10, Status: "granted"},
			{ID: 2, InviterID: 1, RewardType: "tier_50", TierTarget: 50, Amount: 60, Status: "pending_review"},
		},
	}

	svc := NewAffiliateService(repo, nil, nil, nil)
	detail, err := svc.GetAffiliateDetail(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, detail)

	require.Equal(t, 2, detail.QualifiedInviteeCount)
	require.InDelta(t, 80.0, detail.QualifiedInviteesPaidTotal, 1e-9)
	require.InDelta(t, 3.0, detail.PendingStepBonusTotal, 1e-9)
	require.InDelta(t, 11.0, detail.GrantedStepBonusTotal, 1e-9)
	require.InDelta(t, 10.0, detail.GrantedLadderBonusTotal, 1e-9)
	require.InDelta(t, 60.0, detail.PendingLadderBonusTotal, 1e-9)
	require.Equal(t, 1, detail.RemindConsumeCount)
	require.True(t, detail.CanApplyWithdraw)
	require.InDelta(t, 300.0, detail.WithdrawThreshold, 1e-9)
	require.Len(t, detail.LadderTiers, 4)
	require.Equal(t, "granted", detail.LadderTiers[0].Status)
	require.Equal(t, "locked", detail.LadderTiers[1].Status)
	require.Equal(t, "pending_review", detail.LadderTiers[2].Status)
}
