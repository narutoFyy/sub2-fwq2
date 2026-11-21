package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	// 新人注册即领 1 元体验金与准入风控常量
	AffiliateTrialBonusAmount          = 1.0
	AffiliateTrialExpireDays           = 7
	AffiliateTrialUserRPMLimit         = 3
	AffiliateTrialSameIP24hLimit       = 2
	AffiliateTrialDailyGlobalLimit     = 150
	AffiliateInviterLowConvMinInvitees = 10
	AffiliateInviterLowConvMinRate     = 0.10

	// 单人两步奖（3元 + 5元 = 8元）与有效邀请口径常量
	AffiliateQualifyWindowDays = 30
	AffiliateStep1MinPaid      = 10.0
	AffiliateStep2MinPaid      = 50.0
	AffiliateMinConsumed       = 3.0
	AffiliateStep1Bonus        = 3.0
	AffiliateStep2Bonus        = 5.0

	// 付费后风控与提现门槛常量
	AffiliateDailyAutoRewardLimit     = 10
	AffiliateWithdrawThresholdDefault = 300.0
	AffiliateSameIPBurst24hThreshold  = 3

	// 飞书告警 Webhook 配置键
	SettingKeyAffiliateFeishuWebhookURL = "affiliate_feishu_webhook_url"
)

var (
	ErrAffiliateRewardGrantNotFound   = infraerrors.NotFound("AFFILIATE_REWARD_GRANT_NOT_FOUND", "affiliate reward grant not found")
	ErrAffiliateRewardGrantNotPending = infraerrors.Conflict("AFFILIATE_REWARD_GRANT_NOT_PENDING", "affiliate reward grant is not in pending_review status")
)

type ctxKeyAffiliateClientIP struct{}

// WithAffiliateClientIP 在请求上下文中携带客户端真实 IP，用于注册与模型调用风控。
func WithAffiliateClientIP(ctx context.Context, clientIP string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyAffiliateClientIP{}, strings.TrimSpace(clientIP))
}

// AffiliateClientIPFromContext 从上下文读取客户端 IP。
func AffiliateClientIPFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyAffiliateClientIP{}).(string)
	return strings.TrimSpace(v)
}

// mainstreamTrialEmailDomains 定义方案 A 主流邮箱白名单：
// 仅 QQ、163、126、Foxmail、Gmail、Outlook/Hotmail、iCloud 等主流邮箱注册自动发放 1 元体验金。
var mainstreamTrialEmailDomains = map[string]struct{}{
	"qq.com":         {},
	"foxmail.com":    {},
	"vip.qq.com":     {},
	"163.com":        {},
	"126.com":        {},
	"yeah.net":       {},
	"vip.163.com":    {},
	"gmail.com":      {},
	"googlemail.com": {},
	"outlook.com":    {},
	"hotmail.com":    {},
	"live.com":       {},
	"msn.com":        {},
	"outlook.cn":     {},
	"live.cn":        {},
	"icloud.com":     {},
	"me.com":         {},
	"mac.com":        {},
}

// IsMainstreamTrialEmail 判断邮箱是否属于方案 A 主流邮箱白名单。
func IsMainstreamTrialEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at <= 0 || at >= len(email)-1 {
		return false
	}
	domain := strings.TrimSpace(email[at+1:])
	_, ok := mainstreamTrialEmailDomains[domain]
	return ok
}

// ComputeIPSubnet 规范化客户端 IP 并计算网段标识（IPv4 为 /24，IPv6 为 /64）。
func ComputeIPSubnet(rawIP string) (string, string) {
	rawIP = strings.TrimSpace(rawIP)
	if rawIP == "" {
		return "", ""
	}
	if host, _, err := net.SplitHostPort(rawIP); err == nil && host != "" {
		rawIP = host
	}
	parsed := net.ParseIP(rawIP)
	if parsed == nil {
		return rawIP, rawIP
	}
	if ip4 := parsed.To4(); ip4 != nil {
		clean := ip4.String()
		subnet := fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
		return clean, subnet
	}
	ip16 := parsed.To16()
	if ip16 == nil {
		return rawIP, rawIP
	}
	clean := ip16.String()
	masked := ip16.Mask(net.CIDRMask(64, 128))
	return clean, fmt.Sprintf("%s/64", masked.String())
}

// DefaultAffiliateLadderTiers 返回拉新返利 v2 的 4 档阶梯奖定义（按档补差额，封顶 200 元余额）。
func DefaultAffiliateLadderTiers() []AffiliateLadderTier {
	return []AffiliateLadderTier{
		{
			TierTarget:           5,
			CumulativeBonus:      10,
			DeltaBonus:           10,
			RequiresMinTotalPaid: 0,
			AutoGrant:            true,
			Status:               "locked",
		},
		{
			TierTarget:           20,
			CumulativeBonus:      40,
			DeltaBonus:           30,
			RequiresMinTotalPaid: 0,
			AutoGrant:            true,
			Status:               "locked",
		},
		{
			TierTarget:           50,
			CumulativeBonus:      100,
			DeltaBonus:           60,
			RequiresMinTotalPaid: 0,
			AutoGrant:            false,
			Status:               "locked",
		},
		{
			TierTarget:           100,
			CumulativeBonus:      200,
			DeltaBonus:           100,
			RequiresMinTotalPaid: 3000,
			AutoGrant:            false,
			Status:               "locked",
		},
	}
}

type AffiliateTrialGrantOutcome struct {
	Granted                 bool    `json:"granted"`
	Amount                  float64 `json:"amount"`
	SkipReason              string  `json:"skip_reason,omitempty"`
	TodayGrantedCount       int     `json:"today_granted_count"`
	CircuitBreakerTriggered bool    `json:"circuit_breaker_triggered"`
	InviterPausedNow        bool    `json:"inviter_paused_now"`
}

type AffiliateRewardEvalOutcome struct {
	InviterID         int64   `json:"inviter_id"`
	InviteeID         int64   `json:"invitee_id"`
	TrialRPMLifted    bool    `json:"trial_rpm_lifted"`
	QualifiedNow      bool    `json:"qualified_now"`
	Step1Granted      float64 `json:"step1_granted"`
	Step1PendingRev   bool    `json:"step1_pending_review"`
	Step2Granted      float64 `json:"step2_granted"`
	Step2PendingRev   bool    `json:"step2_pending_review"`
	TierBonusGranted  float64 `json:"tier_bonus_granted"`
	TierPendingReview bool    `json:"tier_pending_review"`
}

type AffiliateRewardGrantRecord struct {
	ID              int64      `json:"id"`
	OperationID     string     `json:"operation_id"`
	InviterID       int64      `json:"inviter_id"`
	InviterEmail    string     `json:"inviter_email"`
	InviterUsername string     `json:"inviter_username"`
	InviteeID       *int64     `json:"invitee_id,omitempty"`
	InviteeEmail    string     `json:"invitee_email,omitempty"`
	InviteeUsername string     `json:"invitee_username,omitempty"`
	RewardType      string     `json:"reward_type"`
	TierTarget      int        `json:"tier_target"`
	Amount          float64    `json:"amount"`
	Status          string     `json:"status"`
	ReviewReason    string     `json:"review_reason"`
	ReviewNote      string     `json:"review_note"`
	ReviewedBy      *int64     `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	GrantedAt       *time.Time `json:"granted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// AffiliateV2Repository 是拉新返利 v2 的扩展仓储接口。
type AffiliateV2Repository interface {
	ProcessInviteePostBind(ctx context.Context, inviteeID, inviterID int64, clientIP, ipSubnet string, defaultRPMLimit int) (*AffiliateTrialGrantOutcome, error)
	RecordInviteePaidRecharge(ctx context.Context, inviteeID int64, paidAmount float64, sourceOrderID *int64, defaultRPMLimit int) (*AffiliateRewardEvalOutcome, error)
	RecordInviteeConsumption(ctx context.Context, inviteeID int64, consumedAmount float64, clientIP string) (*AffiliateRewardEvalOutcome, error)
	ExpireUserTrialBonus(ctx context.Context, userID int64) (float64, error)
	SweepExpiredTrialBonuses(ctx context.Context, limit int) ([]int64, error)
	ListUserRewardGrants(ctx context.Context, inviterID int64) ([]AffiliateRewardGrantRecord, error)
	ListAdminRewardGrants(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateRewardGrantRecord, int64, error)
	ApproveRewardGrant(ctx context.Context, grantID, reviewerID int64, note string) (*AffiliateRewardGrantRecord, error)
	RejectRewardGrant(ctx context.Context, grantID, reviewerID int64, note string) (*AffiliateRewardGrantRecord, error)
	SetInviterTrialGrantPaused(ctx context.Context, inviterID int64, paused bool, reason string) error
	ClawbackOrderAffiliateRewards(ctx context.Context, orderID, inviteeID int64, refundedPaidAmount float64) ([]int64, error)
}

var (
	feishuCircuitAlertMu      sync.Mutex
	feishuCircuitAlertLastDay string
)

func (s *AffiliateService) v2Repo() (AffiliateV2Repository, bool) {
	if s == nil || s.repo == nil {
		return nil, false
	}
	r, ok := s.repo.(AffiliateV2Repository)
	return r, ok
}

func (s *AffiliateService) defaultUserRPMLimit(ctx context.Context) int {
	if s == nil || s.settingService == nil {
		return 0
	}
	return s.settingService.GetDefaultUserRPMLimit(ctx)
}

// postBindInviterV2 在受邀新用户成功绑定邀请人后执行：
// 记录注册 IP/网段、校验主流邮箱白名单、同 IP 24h 限 2 笔、每日 150 笔全局熔断、邀请人低转化率连带风控，
// 达标则立即发放 1 元体验金（7 天有效）并设置未充值体验号 3 RPM 限速。
func (s *AffiliateService) postBindInviterV2(ctx context.Context, inviteeID, inviterID int64) {
	repo, ok := s.v2Repo()
	if !ok {
		return
	}
	rawIP := AffiliateClientIPFromContext(ctx)
	cleanIP, subnet := ComputeIPSubnet(rawIP)
	defaultRPM := s.defaultUserRPMLimit(ctx)

	outcome, err := repo.ProcessInviteePostBind(ctx, inviteeID, inviterID, cleanIP, subnet, defaultRPM)
	if err != nil {
		logger.LegacyPrintf("service.affiliate", "[AffiliateV2] ProcessInviteePostBind failed invitee=%d inviter=%d: %v", inviteeID, inviterID, err)
		return
	}
	if outcome == nil {
		return
	}
	if outcome.Granted {
		s.invalidateAffiliateCaches(ctx, inviteeID)
	}
	if outcome.CircuitBreakerTriggered {
		go s.notifyTrialCircuitBreakerFeishu(outcome.TodayGrantedCount)
	}
}

// postPaidRechargeV2 在被邀请人完成实付充值时执行：
// 累加实付金额、解除未充值体验号 3 RPM 限速，并评估单人两步奖（10元->3元、50元->5元）与阶梯奖（5/20/50/100人）。
func (s *AffiliateService) postPaidRechargeV2(ctx context.Context, inviteeID int64, paidAmount float64, sourceOrderID *int64) error {
	repo, ok := s.v2Repo()
	if !ok || inviteeID <= 0 || paidAmount <= 0 {
		return nil
	}
	defaultRPM := s.defaultUserRPMLimit(ctx)
	outcome, err := repo.RecordInviteePaidRecharge(ctx, inviteeID, roundTo(paidAmount, 8), sourceOrderID, defaultRPM)
	if err != nil {
		return fmt.Errorf("record affiliate paid recharge: %w", err)
	}
	if outcome == nil {
		return nil
	}
	if outcome.TrialRPMLifted {
		s.invalidateAffiliateCaches(ctx, inviteeID)
	}
	if outcome.InviterID > 0 && (outcome.Step1Granted > 0 || outcome.Step2Granted > 0 || outcome.TierBonusGranted > 0) {
		s.invalidateAffiliateCaches(ctx, outcome.InviterID)
	}
	return nil
}

// RecordUserConsumption 在用户调用模型产生实际扣费后调用：
// 累加累计消耗金额，若累计消耗满 3 元则触发解锁邀请人的单人两步奖与阶梯奖。
func (s *AffiliateService) RecordUserConsumption(ctx context.Context, userID int64, consumedAmount float64, clientIP string) {
	if s == nil || userID <= 0 || consumedAmount <= 0 || math.IsNaN(consumedAmount) || math.IsInf(consumedAmount, 0) {
		return
	}
	repo, ok := s.v2Repo()
	if !ok {
		return
	}
	cleanIP, _ := ComputeIPSubnet(clientIP)
	outcome, err := repo.RecordInviteeConsumption(ctx, userID, roundTo(consumedAmount, 8), cleanIP)
	if err != nil {
		logger.LegacyPrintf("service.affiliate", "[AffiliateV2] RecordInviteeConsumption failed user=%d cost=%.6f: %v", userID, consumedAmount, err)
		return
	}
	if outcome != nil && outcome.InviterID > 0 && (outcome.Step1Granted > 0 || outcome.Step2Granted > 0 || outcome.TierBonusGranted > 0) {
		s.invalidateAffiliateCaches(ctx, outcome.InviterID)
	}
}

// tryExpireTrialBonus 懒检查并清理当前用户已满 7 天未使用的体验金。
func (s *AffiliateService) tryExpireTrialBonus(ctx context.Context, userID int64) {
	repo, ok := s.v2Repo()
	if !ok || userID <= 0 {
		return
	}
	deducted, err := repo.ExpireUserTrialBonus(ctx, userID)
	if err == nil && deducted > 0 {
		s.invalidateAffiliateCaches(ctx, userID)
	}
}

// SweepExpiredTrialBonuses 批量扫描并清零到期未使用的 1 元体验金。
func (s *AffiliateService) SweepExpiredTrialBonuses(ctx context.Context, limit int) (int, error) {
	repo, ok := s.v2Repo()
	if !ok {
		return 0, nil
	}
	if limit <= 0 {
		limit = 200
	}
	affectedUsers, err := repo.SweepExpiredTrialBonuses(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, uid := range affectedUsers {
		s.invalidateAffiliateCaches(ctx, uid)
	}
	return len(affectedUsers), nil
}

// ClawbackOrderAffiliateRewards 在订单发生异常退款或争议时，同步扣回邀请奖励与返利额度（余额不足扣为负值）。
func (s *AffiliateService) ClawbackOrderAffiliateRewards(ctx context.Context, orderID, inviteeID int64, refundedPaidAmount float64) error {
	repo, ok := s.v2Repo()
	if !ok || orderID <= 0 || inviteeID <= 0 || refundedPaidAmount <= 0 {
		return nil
	}
	affectedUsers, err := repo.ClawbackOrderAffiliateRewards(ctx, orderID, inviteeID, roundTo(refundedPaidAmount, 8))
	if err != nil {
		return err
	}
	for _, uid := range affectedUsers {
		s.invalidateAffiliateCaches(ctx, uid)
	}
	return nil
}

// enrichAffiliateDetailV2 计算邀请页改版所需的阶梯进度条、待到账/已到账明细与提醒好友激活状态。
func (s *AffiliateService) enrichAffiliateDetailV2(ctx context.Context, summary *AffiliateSummary, detail *AffiliateDetail) error {
	if detail == nil {
		return nil
	}
	now := time.Now()
	var qualifiedCount int
	var qualifiedPaidTotal float64
	var pendingStepTotal float64
	var grantedStepTotal float64
	var remindCount int

	for i := range detail.Invitees {
		inv := &detail.Invitees[i]
		FormatAffiliateInviteeV2(inv, now)
		if inv.Qualified {
			qualifiedCount++
			qualifiedPaidTotal = roundTo(qualifiedPaidTotal+inv.TotalPaidAmount, 8)
		}
		pendingStepTotal = roundTo(pendingStepTotal+inv.PendingBonusAmount, 8)
		grantedStepTotal = roundTo(grantedStepTotal+inv.GrantedBonusAmount, 8)
		if inv.NeedsConsumeReminder {
			remindCount++
		}
	}

	detail.QualifiedInviteeCount = qualifiedCount
	detail.QualifiedInviteesPaidTotal = qualifiedPaidTotal
	detail.PendingStepBonusTotal = pendingStepTotal
	detail.GrantedStepBonusTotal = grantedStepTotal
	detail.RemindConsumeCount = remindCount

	tiers := DefaultAffiliateLadderTiers()
	if repo, ok := s.v2Repo(); ok && summary != nil {
		grants, err := repo.ListUserRewardGrants(ctx, summary.UserID)
		if err != nil {
			return err
		}
		if err == nil {
			tierStatusMap := make(map[int]string, len(grants))
			var pendingLadder, grantedLadder float64
			for _, g := range grants {
				if g.TierTarget > 0 {
					tierStatusMap[g.TierTarget] = g.Status
					switch g.Status {
					case "granted":
						grantedLadder = roundTo(grantedLadder+g.Amount, 8)
					case "pending_review":
						pendingLadder = roundTo(pendingLadder+g.Amount, 8)
					}
				}
			}
			for i := range tiers {
				if st, exists := tierStatusMap[tiers[i].TierTarget]; exists {
					tiers[i].Status = st
				}
			}
			detail.PendingLadderBonusTotal = pendingLadder
			detail.GrantedLadderBonusTotal = grantedLadder
		}
	}
	detail.LadderTiers = tiers
	return nil
}

// FormatAffiliateInviteeV2 根据被邀请人的累计实付、累计消耗和两步奖状态，计算前端展示字段：
// - 实付满 10 元未消耗满 3 元 → 显示「待到账 3 元」（needs_consume_reminder = true）
// - 一次性首充 50 元未消耗满 3 元 → 显示「待到账 8 元」（needs_consume_reminder = true）
// - 消耗满 3 元后 → 转为「已到账 3 元」或「已到账 8 元」
func FormatAffiliateInviteeV2(inv *AffiliateInvitee, now time.Time) {
	if inv == nil {
		return
	}
	if inv.Step1RewardStatus == "" {
		inv.Step1RewardStatus = "none"
	}
	if inv.Step2RewardStatus == "" {
		inv.Step2RewardStatus = "none"
	}
	inv.Qualified = inv.QualifiedAt != nil && inv.TotalPaidAmount >= AffiliateStep1MinPaid && inv.TotalConsumedAmount >= AffiliateMinConsumed &&
		(inv.CreatedAt == nil || !inv.QualifiedAt.After(inv.CreatedAt.Add(AffiliateQualifyWindowDays*24*time.Hour)))
	withinWindow := inv.CreatedAt == nil || !now.After(inv.CreatedAt.Add(AffiliateQualifyWindowDays*24*time.Hour))
	if !withinWindow {
		if inv.Step1RewardStatus == "pending_consume" {
			inv.Step1RewardStatus = "expired"
		}
		if inv.Step2RewardStatus == "pending_consume" {
			inv.Step2RewardStatus = "expired"
		}
	}

	var pending float64
	var granted float64

	switch inv.Step1RewardStatus {
	case "granted":
		granted += AffiliateStep1Bonus
	case "pending_consume", "pending_review":
		pending += AffiliateStep1Bonus
	}
	switch inv.Step2RewardStatus {
	case "granted":
		granted += AffiliateStep2Bonus
	case "pending_consume", "pending_review":
		pending += AffiliateStep2Bonus
	}

	// 兼容：若状态列尚未更新但实付已达标且在 30 天有效期内
	if withinWindow && inv.TotalPaidAmount >= AffiliateStep1MinPaid && inv.TotalConsumedAmount < AffiliateMinConsumed {
		if inv.Step1RewardStatus == "none" {
			pending += AffiliateStep1Bonus
			inv.Step1RewardStatus = "pending_consume"
		}
		if inv.TotalPaidAmount >= AffiliateStep2MinPaid && inv.Step2RewardStatus == "none" {
			pending += AffiliateStep2Bonus
			inv.Step2RewardStatus = "pending_consume"
		}
	}

	inv.PendingBonusAmount = roundTo(pending, 2)
	inv.GrantedBonusAmount = roundTo(granted, 2)
	inv.NeedsConsumeReminder = withinWindow && inv.TotalPaidAmount >= AffiliateStep1MinPaid && inv.TotalConsumedAmount < AffiliateMinConsumed

	switch {
	case inv.Step1RewardStatus == "granted" && inv.Step2RewardStatus == "granted":
		inv.DisplayStatus = "granted_8"
	case inv.Step1RewardStatus == "pending_review" || inv.Step2RewardStatus == "pending_review":
		inv.DisplayStatus = "pending_review"
	case inv.NeedsConsumeReminder && inv.TotalPaidAmount >= AffiliateStep2MinPaid:
		inv.DisplayStatus = "pending_consume_8"
	case inv.NeedsConsumeReminder:
		inv.DisplayStatus = "pending_consume_3"
	case inv.Step1RewardStatus == "granted":
		inv.DisplayStatus = "granted_3_pending_5"
	case !withinWindow:
		inv.DisplayStatus = "expired_window"
	default:
		inv.DisplayStatus = "registered"
	}
}

// =========================
// Admin: 阶梯奖与风控奖励审核
// =========================

func (s *AffiliateService) AdminListRewardGrants(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateRewardGrantRecord, int64, error) {
	repo, ok := s.v2Repo()
	if !ok {
		return []AffiliateRewardGrantRecord{}, 0, nil
	}
	return repo.ListAdminRewardGrants(ctx, normalizeAffiliateRecordFilter(filter))
}

func (s *AffiliateService) AdminApproveRewardGrant(ctx context.Context, grantID, reviewerID int64, note string) (*AffiliateRewardGrantRecord, error) {
	if grantID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_GRANT_ID", "invalid reward grant id")
	}
	repo, ok := s.v2Repo()
	if !ok {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate v2 repository unavailable")
	}
	record, err := repo.ApproveRewardGrant(ctx, grantID, reviewerID, strings.TrimSpace(note))
	if err != nil {
		return nil, err
	}
	if record != nil && record.InviterID > 0 {
		s.invalidateAffiliateCaches(ctx, record.InviterID)
	}
	return record, nil
}

func (s *AffiliateService) AdminRejectRewardGrant(ctx context.Context, grantID, reviewerID int64, note string) (*AffiliateRewardGrantRecord, error) {
	if grantID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_GRANT_ID", "invalid reward grant id")
	}
	repo, ok := s.v2Repo()
	if !ok {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate v2 repository unavailable")
	}
	return repo.RejectRewardGrant(ctx, grantID, reviewerID, strings.TrimSpace(note))
}

func (s *AffiliateService) AdminSetInviterTrialGrantPaused(ctx context.Context, inviterID int64, paused bool, reason string) error {
	if inviterID <= 0 {
		return infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	repo, ok := s.v2Repo()
	if !ok {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate v2 repository unavailable")
	}
	return repo.SetInviterTrialGrantPaused(ctx, inviterID, paused, strings.TrimSpace(reason))
}

func (s *AffiliateService) notifyTrialCircuitBreakerFeishu(todayCount int) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	todayStr := time.Now().In(loc).Format("2006-01-02")

	feishuCircuitAlertMu.Lock()
	if feishuCircuitAlertLastDay == todayStr {
		feishuCircuitAlertMu.Unlock()
		return
	}
	feishuCircuitAlertLastDay = todayStr
	feishuCircuitAlertMu.Unlock()

	logger.LegacyPrintf("service.affiliate", "ALERT: [AffiliateV2] Daily trial bonus circuit breaker triggered! date=%s granted=%d limit=%d",
		todayStr, todayCount, AffiliateTrialDailyGlobalLimit)

	webhookURL := strings.TrimSpace(os.Getenv("AFFILIATE_FEISHU_WEBHOOK_URL"))
	if webhookURL == "" {
		webhookURL = strings.TrimSpace(os.Getenv("FEISHU_WEBHOOK_URL"))
	}
	if webhookURL == "" && s != nil && s.settingService != nil && s.settingService.settingRepo != nil {
		if val, err := s.settingService.settingRepo.GetValue(context.Background(), SettingKeyAffiliateFeishuWebhookURL); err == nil {
			webhookURL = strings.TrimSpace(val)
		}
	}
	if webhookURL == "" {
		return
	}

	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]string{
			"text": fmt.Sprintf("【yf-mail 拉新风控告警】\n全站今日（%s）自动发放新人 1 元体验金已达上限 %d 笔（当前 %d 笔），已自动触发每日全局熔断，暂停今日后续新人体验金自动发放，请及时核查注册流量。",
				todayStr, AffiliateTrialDailyGlobalLimit, todayCount),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.LegacyPrintf("service.affiliate", "[AffiliateV2] Failed to send Feishu circuit breaker alert: %v", err)
		return
	}
	_ = resp.Body.Close()
}
