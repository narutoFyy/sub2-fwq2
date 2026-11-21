package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func appendRiskTag(existing, tag string) string {
	existing = strings.TrimSpace(existing)
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return existing
	}
	if existing == "" {
		return tag
	}
	for _, part := range strings.Split(existing, ",") {
		if strings.TrimSpace(part) == tag {
			return existing
		}
	}
	return existing + "," + tag
}

// lockAffiliateUsersTx serializes reward evaluation for shared inviters. Always acquire
// all user scopes in numeric order before taking affiliate, grant or balance row locks.
func lockAffiliateUsersTx(ctx context.Context, q affiliateQueryExecer, ids ...int64) error {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var previous int64
	for _, id := range ids {
		if id <= 0 || id == previous {
			continue
		}
		if _, err := q.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, fmt.Sprintf("affiliate_rewards:%d", id)); err != nil {
			return fmt.Errorf("lock affiliate reward scope: %w", err)
		}
		previous = id
	}
	return nil
}

func lockAffiliateRewardScopeTx(ctx context.Context, q affiliateQueryExecer, userID int64) error {
	rows, err := q.QueryContext(ctx, `SELECT inviter_id FROM user_affiliates WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	var inviterID sql.NullInt64
	if rows.Next() {
		if err := rows.Scan(&inviterID); err != nil {
			_ = rows.Close()
			return err
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return lockAffiliateUsersTx(ctx, q, userID, inviterID.Int64)
}

// LockAffiliateRewardScope lets payment transactions acquire the same scopes before
// locking payment orders or user balances. A standalone call would release too early.
func (r *affiliateRepository) LockAffiliateRewardScope(ctx context.Context, userID int64) error {
	tx := dbent.TxFromContext(ctx)
	if tx == nil {
		return fmt.Errorf("affiliate reward scope requires an existing transaction")
	}
	return lockAffiliateRewardScopeTx(ctx, tx.Client(), userID)
}

// ProcessInviteePostBind 处理受邀新用户绑定邀请人后的准入校验与 1 元体验金发放。
func (r *affiliateRepository) ProcessInviteePostBind(
	ctx context.Context,
	inviteeID, inviterID int64,
	clientIP, ipSubnet string,
	defaultRPMLimit int,
) (*service.AffiliateTrialGrantOutcome, error) {
	if inviteeID <= 0 || inviterID <= 0 {
		return nil, nil
	}

	outcome := &service.AffiliateTrialGrantOutcome{}
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		// A shared transaction lock protects both rolling IP/subnet and global-day quotas.
		if _, err := txClient.ExecContext(txCtx, `SELECT pg_advisory_xact_lock(hashtextextended('affiliate_trial_grants', 0))`); err != nil {
			return err
		}
		if err := lockAffiliateUsersTx(txCtx, txClient, inviteeID, inviterID); err != nil {
			return err
		}

		rows, err := txClient.QueryContext(txCtx, `
SELECT COALESCE(u.email, ''),
       COALESCE(u.rpm_limit, 0),
       COALESCE(ua.trial_bonus_status, 'none'),
       COALESCE(ua.risk_tags, ''),
       COALESCE(ua.same_ip_warning, false)
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id
WHERE ua.user_id = $1
FOR UPDATE OF ua`, inviteeID)
		if err != nil {
			return fmt.Errorf("lock invitee affiliate for trial grant: %w", err)
		}
		if !rows.Next() {
			_ = rows.Close()
			return nil
		}
		var inviteeEmail string
		var currentRPM int
		var trialStatus string
		var riskTags string
		var sameIPWarning bool
		if err := rows.Scan(&inviteeEmail, &currentRPM, &trialStatus, &riskTags, &sameIPWarning); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}

		// 幂等保护：已处理过体验金状态的不再重复处理
		if trialStatus != "" && trialStatus != "none" {
			return nil
		}

		// 读取邀请人 IP 与连带风控状态
		inviterRows, err := txClient.QueryContext(txCtx, `
SELECT COALESCE(register_ip, ''),
       COALESCE(last_active_ip, ''),
       COALESCE(trial_grant_paused, false)
FROM user_affiliates
WHERE user_id = $1`, inviterID)
		if err != nil {
			return fmt.Errorf("query inviter affiliate for trial check: %w", err)
		}
		var inviterRegIP, inviterActiveIP string
		var inviterPaused bool
		if inviterRows.Next() {
			if err := inviterRows.Scan(&inviterRegIP, &inviterActiveIP, &inviterPaused); err != nil {
				_ = inviterRows.Close()
				return err
			}
		}
		if err := inviterRows.Close(); err != nil {
			return err
		}

		if clientIP != "" && (clientIP == inviterRegIP || clientIP == inviterActiveIP) {
			sameIPWarning = true
			riskTags = appendRiskTag(riskTags, "same_ip")
		}

		saveSkipped := func(reason, extraTag string) error {
			if extraTag != "" {
				riskTags = appendRiskTag(riskTags, extraTag)
			}
			outcome.Granted = false
			outcome.SkipReason = reason
			_, updateErr := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET register_ip = CASE WHEN $1 <> '' THEN $1 ELSE register_ip END,
    register_ip_subnet = CASE WHEN $2 <> '' THEN $2 ELSE register_ip_subnet END,
    last_active_ip = CASE WHEN $1 <> '' THEN $1 ELSE last_active_ip END,
    same_ip_warning = $3,
    risk_tags = $4,
    trial_bonus_status = 'skipped',
    trial_skip_reason = $5,
    updated_at = NOW()
WHERE user_id = $6`, clientIP, ipSubnet, sameIPWarning, riskTags, reason, inviteeID)
			return updateErr
		}

		// 1) 主流邮箱白名单校验（方案 A）
		if !service.IsMainstreamTrialEmail(inviteeEmail) {
			return saveSkipped("email_not_whitelisted", "non_mainstream_email")
		}

		// 2) 邀请人连带风控：若某邀请人名下新注册 >= 10 人且付费率 < 10%，自动暂停其邀请链接的新人体验金发放
		if inviterPaused {
			return saveSkipped("inviter_paused", "inviter_paused")
		}
		convRows, err := txClient.QueryContext(txCtx, `
SELECT COUNT(*)::integer AS total_invitees,
       COUNT(*) FILTER (WHERE total_paid_amount > 0)::integer AS paid_invitees
FROM user_affiliates
WHERE inviter_id = $1`, inviterID)
		if err != nil {
			return fmt.Errorf("query inviter conversion rate: %w", err)
		}
		var totalInvitees, paidInvitees int
		if convRows.Next() {
			if err := convRows.Scan(&totalInvitees, &paidInvitees); err != nil {
				_ = convRows.Close()
				return err
			}
		}
		if err := convRows.Close(); err != nil {
			return err
		}
		if totalInvitees >= service.AffiliateInviterLowConvMinInvitees &&
			float64(paidInvitees)/float64(totalInvitees) < service.AffiliateInviterLowConvMinRate {
			if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET trial_grant_paused = true,
    trial_grant_pause_reason = '名下新注册≥10人且付费率<10%，自动触发连带风控转人工复核',
    updated_at = NOW()
WHERE user_id = $1`, inviterID); err != nil {
				return fmt.Errorf("pause inviter trial grant: %w", err)
			}
			outcome.InviterPausedNow = true
			return saveSkipped("inviter_paused", "inviter_paused")
		}

		// 3) 同 IP（或同 /24 网段）24 小时内最多发放 2 笔新人体验金
		if clientIP != "" {
			ipRows, err := txClient.QueryContext(txCtx, `
SELECT COUNT(*)::integer
FROM user_affiliates
WHERE trial_granted_at IS NOT NULL
  AND trial_granted_at >= NOW() - INTERVAL '24 hours'
  AND (register_ip = $1 OR ($2 <> '' AND register_ip_subnet = $2))`, clientIP, ipSubnet)
			if err != nil {
				return fmt.Errorf("count same ip 24h trial grants: %w", err)
			}
			var ipGrantCount int
			if ipRows.Next() {
				if err := ipRows.Scan(&ipGrantCount); err != nil {
					_ = ipRows.Close()
					return err
				}
			}
			if err := ipRows.Close(); err != nil {
				return err
			}
			if ipGrantCount >= service.AffiliateTrialSameIP24hLimit {
				return saveSkipped("ip_rate_limited", "ip_rate_limited")
			}
		}

		// 4) 全站每日全局熔断（150 笔/日，按东八区自然日统计）
		dailyRows, err := txClient.QueryContext(txCtx, `
SELECT COUNT(*)::integer
FROM user_affiliates
WHERE trial_granted_at IS NOT NULL
  AND trial_granted_at >= (date_trunc('day', NOW() AT TIME ZONE 'Asia/Shanghai') AT TIME ZONE 'Asia/Shanghai')`)
		if err != nil {
			return fmt.Errorf("count daily global trial grants: %w", err)
		}
		var todayCount int
		if dailyRows.Next() {
			if err := dailyRows.Scan(&todayCount); err != nil {
				_ = dailyRows.Close()
				return err
			}
		}
		if err := dailyRows.Close(); err != nil {
			return err
		}
		outcome.TodayGrantedCount = todayCount
		if todayCount >= service.AffiliateTrialDailyGlobalLimit {
			outcome.CircuitBreakerTriggered = true
			return saveSkipped("daily_circuit_breaker", "daily_circuit_breaker")
		}

		// 5) 准入通过：发放 1 元体验金（仅计入 balance，不计入 total_recharged），设置 7 天有效期与未充值体验号 3 RPM 限速
		trialAmount := service.AffiliateTrialBonusAmount
		applyTrialRPM := currentRPM == 0 || currentRPM > service.AffiliateTrialUserRPMLimit

		userUpdate := txClient.User.Update().
			Where(user.IDEQ(inviteeID)).
			AddBalance(trialAmount)
		if applyTrialRPM {
			userUpdate = userUpdate.SetRpmLimit(service.AffiliateTrialUserRPMLimit)
		}
		if _, err := userUpdate.Save(txCtx); err != nil {
			return fmt.Errorf("credit invitee trial bonus: %w", err)
		}

		if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET register_ip = CASE WHEN $1 <> '' THEN $1 ELSE register_ip END,
    register_ip_subnet = CASE WHEN $2 <> '' THEN $2 ELSE register_ip_subnet END,
    last_active_ip = CASE WHEN $1 <> '' THEN $1 ELSE last_active_ip END,
    same_ip_warning = $3,
    risk_tags = $4,
    trial_bonus_amount = $5,
    trial_bonus_status = 'active',
    trial_skip_reason = '',
    trial_granted_at = NOW(),
    trial_expires_at = NOW() + make_interval(days => $6),
    trial_rpm_limited = $7,
    updated_at = NOW()
WHERE user_id = $8`,
			clientIP,
			ipSubnet,
			sameIPWarning,
			riskTags,
			trialAmount,
			service.AffiliateTrialExpireDays,
			applyTrialRPM,
			inviteeID,
		); err != nil {
			return fmt.Errorf("update invitee trial grant status: %w", err)
		}

		snapshot, err := queryAffiliateTransferSnapshot(txCtx, txClient, inviteeID)
		if err != nil {
			return err
		}
		opID := fmt.Sprintf("trial_grant:%d", inviteeID)
		if _, err := txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (
    user_id,
    action,
    amount,
    source_user_id,
    operation_id,
    balance_after,
    aff_quota_after,
    aff_frozen_quota_after,
    aff_history_quota_after,
    created_at,
    updated_at
)
VALUES ($1, 'trial_grant', $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
ON CONFLICT (operation_id) WHERE operation_id IS NOT NULL DO NOTHING`,
			inviteeID,
			trialAmount,
			inviterID,
			opID,
			snapshot.BalanceAfter,
			snapshot.AvailableQuotaAfter,
			snapshot.FrozenQuotaAfter,
			snapshot.HistoryQuotaAfter,
		); err != nil {
			return fmt.Errorf("insert trial grant ledger: %w", err)
		}

		outcome.Granted = true
		outcome.Amount = trialAmount
		outcome.TodayGrantedCount = todayCount + 1
		if outcome.TodayGrantedCount >= service.AffiliateTrialDailyGlobalLimit {
			outcome.CircuitBreakerTriggered = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

// RecordInviteePaidRecharge 记录被邀请人的实付充值，解除未充值体验号限速，并评估单人两步奖与阶梯奖。
func (r *affiliateRepository) RecordInviteePaidRecharge(
	ctx context.Context,
	inviteeID int64,
	paidAmount float64,
	_ *int64,
	defaultRPMLimit int,
) (*service.AffiliateRewardEvalOutcome, error) {
	if inviteeID <= 0 || paidAmount <= 0 {
		return nil, nil
	}

	outcome := &service.AffiliateRewardEvalOutcome{InviteeID: inviteeID}
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateRewardScopeTx(txCtx, txClient, inviteeID); err != nil {
			return err
		}

		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, inviteeID); err != nil {
			return err
		}

		rows, err := txClient.QueryContext(txCtx, `
SELECT inviter_id,
       COALESCE(trial_rpm_limited, false)
FROM user_affiliates
WHERE user_id = $1
FOR UPDATE`, inviteeID)
		if err != nil {
			return fmt.Errorf("lock invitee affiliate on paid recharge: %w", err)
		}
		if !rows.Next() {
			_ = rows.Close()
			return nil
		}
		var inviterID sql.NullInt64
		var trialRPMLimited bool
		if err := rows.Scan(&inviterID, &trialRPMLimited); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}

		if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET total_paid_amount = total_paid_amount + $1,
    trial_rpm_limited = false,
    updated_at = NOW()
WHERE user_id = $2`, paidAmount, inviteeID); err != nil {
			return fmt.Errorf("increment invitee total_paid_amount: %w", err)
		}

		if trialRPMLimited {
			if _, err := txClient.ExecContext(txCtx, `
UPDATE users
SET rpm_limit = $1,
    updated_at = NOW()
WHERE id = $2
  AND rpm_limit = $3`, defaultRPMLimit, inviteeID, service.AffiliateTrialUserRPMLimit); err != nil {
				return fmt.Errorf("lift trial rpm limit after paid recharge: %w", err)
			}
			outcome.TrialRPMLifted = true
		}

		if !inviterID.Valid || inviterID.Int64 <= 0 {
			return nil
		}
		outcome.InviterID = inviterID.Int64
		return evaluateInviteeAndLadderRewardsTx(txCtx, txClient, inviteeID, outcome)
	})
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

// evaluateInviteeAndLadderRewardsTx 在同一数据库事务中幂等评估并解锁单人两步奖（3元+5元）与 4 档阶梯奖。
func evaluateInviteeAndLadderRewardsTx(
	txCtx context.Context,
	txClient affiliateQueryExecer,
	inviteeID int64,
	outcome *service.AffiliateRewardEvalOutcome,
) error {
	rows, err := txClient.QueryContext(txCtx, `
SELECT inviter_id,
       COALESCE(total_paid_amount, 0)::double precision,
       COALESCE(total_consumed_amount, 0)::double precision,
       COALESCE(step1_reward_status, 'none'),
       COALESCE(step2_reward_status, 'none'),
       qualified_at,
       COALESCE(register_ip, ''),
       COALESCE(last_active_ip, ''),
       COALESCE(same_ip_warning, false),
       COALESCE(risk_tags, ''),
       created_at
FROM user_affiliates
WHERE user_id = $1
FOR UPDATE`, inviteeID)
	if err != nil {
		return fmt.Errorf("query invitee state for reward evaluation: %w", err)
	}
	if !rows.Next() {
		_ = rows.Close()
		return nil
	}
	var inviterIDNull sql.NullInt64
	var totalPaid, totalConsumed float64
	var step1Status, step2Status string
	var qualifiedAt sql.NullTime
	var regIP, activeIP string
	var sameIPWarning bool
	var riskTags string
	var createdAt time.Time
	if err := rows.Scan(
		&inviterIDNull,
		&totalPaid,
		&totalConsumed,
		&step1Status,
		&step2Status,
		&qualifiedAt,
		&regIP,
		&activeIP,
		&sameIPWarning,
		&riskTags,
		&createdAt,
	); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !inviterIDNull.Valid || inviterIDNull.Int64 <= 0 {
		return nil
	}
	inviterID := inviterIDNull.Int64
	outcome.InviterID = inviterID

	// 活动结束在途归因：注册后 30 天内达成充值与消耗门槛均可获颁奖励
	now := time.Now()
	within30Days := !now.After(createdAt.AddDate(0, 0, service.AffiliateQualifyWindowDays))
	if !within30Days {
		_, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET
            step1_reward_status = CASE WHEN step1_reward_status = 'pending_consume' THEN 'expired' ELSE step1_reward_status END,
            step2_reward_status = CASE WHEN step2_reward_status = 'pending_consume' THEN 'expired' ELSE step2_reward_status END,
            updated_at = NOW() WHERE user_id = $1`, inviteeID)
		return err
	}

	// 同 IP 加权预警检查
	inviterRows, err := txClient.QueryContext(txCtx, `
SELECT COALESCE(register_ip, ''),
       COALESCE(last_active_ip, '')
FROM user_affiliates
WHERE user_id = $1`, inviterID)
	if err != nil {
		return fmt.Errorf("query inviter ip for same-ip check: %w", err)
	}
	var inviterRegIP, inviterActiveIP string
	if inviterRows.Next() {
		if err := inviterRows.Scan(&inviterRegIP, &inviterActiveIP); err != nil {
			_ = inviterRows.Close()
			return err
		}
	}
	if err := inviterRows.Close(); err != nil {
		return err
	}

	sameIPMatched := (regIP != "" && (regIP == inviterRegIP || regIP == inviterActiveIP)) ||
		(activeIP != "" && (activeIP == inviterRegIP || activeIP == inviterActiveIP))
	if sameIPMatched && !sameIPWarning {
		sameIPWarning = true
		riskTags = appendRiskTag(riskTags, "same_ip")
		if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET same_ip_warning = true,
    risk_tags = $1,
    updated_at = NOW()
WHERE user_id = $2`, riskTags, inviteeID); err != nil {
			return err
		}
	}

	// 情况 A：已满足实付门槛（10元/50元），但累计消耗尚未满 3 元 → 更新为 pending_consume（待到账）
	if totalConsumed < service.AffiliateMinConsumed {
		newStep1 := step1Status
		newStep2 := step2Status
		if totalPaid >= service.AffiliateStep1MinPaid && step1Status == "none" {
			newStep1 = "pending_consume"
		}
		if totalPaid >= service.AffiliateStep2MinPaid && step2Status == "none" {
			newStep2 = "pending_consume"
		}
		if newStep1 != step1Status || newStep2 != step2Status {
			if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET step1_reward_status = $1::varchar,
    step1_reward_amount = CASE WHEN $1::varchar <> 'none' THEN $2 ELSE step1_reward_amount END,
    step2_reward_status = $3::varchar,
    step2_reward_amount = CASE WHEN $3::varchar <> 'none' THEN $4 ELSE step2_reward_amount END,
    updated_at = NOW()
WHERE user_id = $5`,
				newStep1, service.AffiliateStep1Bonus,
				newStep2, service.AffiliateStep2Bonus,
				inviteeID,
			); err != nil {
				return fmt.Errorf("mark invitee pending_consume: %w", err)
			}
		}
		return nil
	}

	// 情况 B：累计消耗已满 3 元，检查是否达到实付满 10 元
	if totalPaid < service.AffiliateStep1MinPaid {
		return nil
	}

	// 首次同时满足「注册 30 天内 + 累计实付满 10 元 + 累计消耗满 3 元」→ 标记为有效邀请
	if !qualifiedAt.Valid {
		if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET qualified_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1 AND qualified_at IS NULL`, inviteeID); err != nil {
			return fmt.Errorf("mark invitee qualified_at: %w", err)
		}
		outcome.QualifiedNow = true
	}

	// 判断是否命中「同 IP + 短时高频注册」或「大小号长期同 IP 高并发调用」高危转人审条件
	sameIPHighRisk := false
	if sameIPWarning {
		burstRows, err := txClient.QueryContext(txCtx, `
SELECT COUNT(*)::integer
FROM user_affiliates
WHERE inviter_id = $1
  AND same_ip_warning = true
  AND created_at >= NOW() - INTERVAL '24 hours'`, inviterID)
		if err != nil {
			return fmt.Errorf("count inviter same_ip_warning invitees: %w", err)
		}
		var sameIPCount int
		if burstRows.Next() {
			if err := burstRows.Scan(&sameIPCount); err != nil {
				_ = burstRows.Close()
				return err
			}
		}
		if err := burstRows.Close(); err != nil {
			return err
		}
		if sameIPCount >= service.AffiliateSameIPBurst24hThreshold {
			sameIPHighRisk = true
			riskTags = appendRiskTag(riskTags, "same_ip_burst")
			if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET risk_tags = $1, updated_at = NOW() WHERE user_id = $2`, riskTags, inviteeID); err != nil {
				return err
			}
		}
	}

	// 1) 解锁首档 3 元余额（累计实付满 10 元 + 累计消耗满 3 元）
	if step1Status == "none" || step1Status == "pending_consume" {
		st, err := grantOrQueueAffiliateRewardTx(
			txCtx,
			txClient,
			fmt.Sprintf("step1:%d", inviteeID),
			inviterID,
			&inviteeID,
			"step1",
			0,
			service.AffiliateStep1Bonus,
			true,
			sameIPHighRisk,
		)
		if err != nil {
			return err
		}
		if st != "" {
			if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET step1_reward_status = $1,
    step1_reward_amount = $2,
    updated_at = NOW()
WHERE user_id = $3`, st, service.AffiliateStep1Bonus, inviteeID); err != nil {
				return err
			}
			if st == "granted" {
				outcome.Step1Granted = service.AffiliateStep1Bonus
			} else if st == "pending_review" {
				outcome.Step1PendingRev = true
			}
		}
	}

	// 2) 解锁二档再加 5 元余额（累计实付满 50 元 + 累计消耗满 3 元）
	if totalPaid >= service.AffiliateStep2MinPaid && (step2Status == "none" || step2Status == "pending_consume") {
		st, err := grantOrQueueAffiliateRewardTx(
			txCtx,
			txClient,
			fmt.Sprintf("step2:%d", inviteeID),
			inviterID,
			&inviteeID,
			"step2",
			0,
			service.AffiliateStep2Bonus,
			true,
			sameIPHighRisk,
		)
		if err != nil {
			return err
		}
		if st != "" {
			if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET step2_reward_status = $1,
    step2_reward_amount = $2,
    updated_at = NOW()
WHERE user_id = $3`, st, service.AffiliateStep2Bonus, inviteeID); err != nil {
				return err
			}
			if st == "granted" {
				outcome.Step2Granted = service.AffiliateStep2Bonus
			} else if st == "pending_review" {
				outcome.Step2PendingRev = true
			}
		}
	}

	// 3) 评估邀请人的阶梯奖（5人 +10元 / 20人 +30元自动发放；50人 +60元 / 100人且实付满3000元 +100元人工审核）
	qualifiedCount, qualifiedPaidSum, err := qualifiedAffiliateTotalsTx(txCtx, txClient, inviterID)
	if err != nil {
		return err
	}

	for _, tier := range service.DefaultAffiliateLadderTiers() {
		if qualifiedCount < tier.TierTarget {
			continue
		}
		if tier.RequiresMinTotalPaid > 0 && qualifiedPaidSum < tier.RequiresMinTotalPaid {
			continue
		}
		st, err := grantOrQueueAffiliateRewardTx(
			txCtx,
			txClient,
			fmt.Sprintf("tier:%d:%d", inviterID, tier.TierTarget),
			inviterID,
			nil,
			fmt.Sprintf("tier_%d", tier.TierTarget),
			tier.TierTarget,
			tier.DeltaBonus,
			tier.AutoGrant,
			sameIPHighRisk,
		)
		if err != nil {
			return err
		}
		if st == "granted" {
			outcome.TierBonusGranted += tier.DeltaBonus
		} else if st == "pending_review" {
			outcome.TierPendingReview = true
		}
	}

	return nil
}

// grantOrQueueAffiliateRewardTx 以 operation_id 唯一约束原子写入奖励记录：
// 若已存在则直接返回既有状态（幂等防重）；
// 若属于 50/100 人阶梯档、命中同 IP 高危或邀请人今日自动发奖已满 10 笔，则进入 pending_review 人工审核列表；
// 否则自动发放到邀请人站内余额（balance，不可提现）并写入流水。
func grantOrQueueAffiliateRewardTx(
	txCtx context.Context,
	txClient affiliateQueryExecer,
	operationID string,
	inviterID int64,
	inviteeID *int64,
	rewardType string,
	tierTarget int,
	amount float64,
	allowAutoGrant bool,
	sameIPHighRisk bool,
) (string, error) {
	if err := lockAffiliateUsersTx(txCtx, txClient, inviterID); err != nil {
		return "", err
	}
	// 检查邀请人今日已自动发放的奖励笔数（单邀请人每日最多自动发 10 笔奖励，超出转人工）
	var todayAutoCount int
	if allowAutoGrant && !sameIPHighRisk {
		cntRows, err := txClient.QueryContext(txCtx, `
SELECT COUNT(*)::integer
FROM affiliate_reward_grants
WHERE inviter_id = $1
  AND granted_at IS NOT NULL
  AND reviewed_by IS NULL
  AND granted_at >= (date_trunc('day', NOW() AT TIME ZONE 'Asia/Shanghai') AT TIME ZONE 'Asia/Shanghai')`, inviterID)
		if err != nil {
			return "", fmt.Errorf("count inviter daily auto reward grants: %w", err)
		}
		if cntRows.Next() {
			if err := cntRows.Scan(&todayAutoCount); err != nil {
				_ = cntRows.Close()
				return "", err
			}
		}
		if err := cntRows.Close(); err != nil {
			return "", err
		}
	}

	status := "granted"
	reviewReason := ""
	switch {
	case !allowAutoGrant:
		status = "pending_review"
		reviewReason = "tier_manual_review"
	case sameIPHighRisk:
		status = "pending_review"
		reviewReason = "same_ip_high_risk"
	case todayAutoCount >= service.AffiliateDailyAutoRewardLimit:
		status = "pending_review"
		reviewReason = "daily_auto_limit_exceeded"
	}

	rows, err := txClient.QueryContext(txCtx, `
INSERT INTO affiliate_reward_grants (
    operation_id,
    inviter_id,
    invitee_id,
    reward_type,
    tier_target,
    amount,
    status,
    review_reason,
    granted_at,
    created_at,
    updated_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7::varchar, $8,
    CASE WHEN $7::varchar = 'granted' THEN NOW() ELSE NULL END,
    NOW(), NOW()
)
ON CONFLICT (operation_id) DO NOTHING
RETURNING id`,
		operationID,
		inviterID,
		nullableInt64Arg(inviteeID),
		rewardType,
		tierTarget,
		amount,
		status,
		reviewReason,
	)
	if err != nil {
		return "", fmt.Errorf("claim affiliate reward grant %s: %w", operationID, err)
	}
	claimed := rows.Next()
	_ = rows.Close()
	if !claimed {
		// 幂等重放：查询已存在的记录状态
		existRows, qErr := txClient.QueryContext(txCtx, `SELECT status FROM affiliate_reward_grants WHERE operation_id = $1 LIMIT 1`, operationID)
		if qErr != nil {
			return "", qErr
		}
		defer func() { _ = existRows.Close() }()
		var existingStatus string
		if existRows.Next() {
			_ = existRows.Scan(&existingStatus)
		}
		return existingStatus, nil
	}

	if status != "granted" {
		return status, nil
	}

	if err := creditInviterRewardBalanceTx(txCtx, txClient, inviterID, inviteeID, rewardType, operationID, amount); err != nil {
		return "", err
	}
	return "granted", nil
}

func creditInviterRewardBalanceTx(
	txCtx context.Context,
	txClient affiliateQueryExecer,
	inviterID int64,
	inviteeID *int64,
	rewardType string,
	operationID string,
	amount float64,
) error {
	res, err := txClient.ExecContext(txCtx, `UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE id = $2`, amount, inviterID)
	if err != nil {
		return fmt.Errorf("credit inviter reward balance: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}

	snapshot, err := queryAffiliateTransferSnapshot(txCtx, txClient, inviterID)
	if err != nil {
		return err
	}
	ledgerAction := "reward_" + rewardType
	if len(ledgerAction) > 32 {
		ledgerAction = ledgerAction[:32]
	}
	ledgerOpID := "reward_credit:" + operationID
	if _, err := txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (
    user_id,
    action,
    amount,
    source_user_id,
    operation_id,
    balance_after,
    aff_quota_after,
    aff_frozen_quota_after,
    aff_history_quota_after,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
ON CONFLICT (operation_id) WHERE operation_id IS NOT NULL DO NOTHING`,
		inviterID,
		ledgerAction,
		amount,
		nullableInt64Arg(inviteeID),
		ledgerOpID,
		snapshot.BalanceAfter,
		snapshot.AvailableQuotaAfter,
		snapshot.FrozenQuotaAfter,
		snapshot.HistoryQuotaAfter,
	); err != nil {
		return fmt.Errorf("insert reward credit ledger: %w", err)
	}
	return nil
}

// ExpireUserTrialBonus 检查并清零指定用户已过 7 天有效期仍未使用的体验金。
func (r *affiliateRepository) ExpireUserTrialBonus(ctx context.Context, userID int64) (float64, error) {
	if userID <= 0 {
		return 0, nil
	}
	var deducted float64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		var err error
		if err := lockAffiliateRewardScopeTx(txCtx, txClient, userID); err != nil {
			return err
		}
		deducted, err = expireUserTrialBonusTx(txCtx, txClient, userID)
		return err
	})
	return deducted, err
}

func expireUserTrialBonusTx(txCtx context.Context, txClient affiliateQueryExecer, userID int64) (float64, error) {
	rows, err := txClient.QueryContext(txCtx, `
SELECT COALESCE(trial_bonus_amount, 0)::double precision,
       COALESCE(trial_consumed_amount, 0)::double precision,
       COALESCE(trial_expired_amount, 0)::double precision,
       COALESCE((SELECT SUM(h.trial_reserved_amount) FROM usage_billing_batch_image_holds h WHERE h.user_id = ua.user_id AND h.status = 'reserved'), 0)::double precision
FROM user_affiliates ua
WHERE user_id = $1
  AND trial_bonus_status = 'active'
  AND trial_expires_at IS NOT NULL
  AND trial_expires_at <= NOW()
FOR UPDATE`, userID)
	if err != nil {
		return 0, err
	}
	if !rows.Next() {
		_ = rows.Close()
		return 0, nil
	}
	var trialAmount, trialConsumed, trialExpired, trialReserved float64
	if err := rows.Scan(&trialAmount, &trialConsumed, &trialExpired, &trialReserved); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}

	unused := math.Max(0, trialAmount-trialConsumed-trialExpired-trialReserved)
	if unused <= 0 && trialReserved <= 0 && trialExpired <= 0 {
		_, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET trial_bonus_status = 'used',
    trial_expired_amount = 0,
    updated_at = NOW()
WHERE user_id = $1`, userID)
		return 0, err
	}

	currentBal, err := queryUserBalance(txCtx, txClient, userID)
	if err != nil {
		return 0, err
	}
	deduct := math.Min(unused, math.Max(0, currentBal))
	deduct = math.Round(deduct*1e8) / 1e8

	if deduct > 0 {
		if _, err := txClient.ExecContext(txCtx, `
UPDATE users
SET balance = GREATEST(balance - $1, 0),
    updated_at = NOW()
WHERE id = $2`, deduct, userID); err != nil {
			return 0, fmt.Errorf("deduct expired trial bonus: %w", err)
		}
	}

	if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET trial_bonus_status = 'expired',
    trial_expired_amount = trial_expired_amount + $1,
    updated_at = NOW()
WHERE user_id = $2`, deduct, userID); err != nil {
		return 0, err
	}

	if deduct > 0 {
		if err := recordTrialExpiryLedgerTx(txCtx, txClient, userID, deduct, fmt.Sprintf("trial_expire:%d", userID)); err != nil {
			return 0, err
		}
	}

	return deduct, nil
}

// SweepExpiredTrialBonuses 批量扫描并清理到期未使用的体验金。
func (r *affiliateRepository) SweepExpiredTrialBonuses(ctx context.Context, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 200
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT user_id
FROM user_affiliates
WHERE trial_bonus_status = 'active'
  AND trial_expires_at IS NOT NULL
  AND trial_expires_at <= NOW()
ORDER BY trial_expires_at ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	userIDs := make([]int64, 0)
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			_ = rows.Close()
			return nil, err
		}
		userIDs = append(userIDs, uid)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	affected := make([]int64, 0, len(userIDs))
	for _, uid := range userIDs {
		deducted, err := r.ExpireUserTrialBonus(ctx, uid)
		if err == nil && deducted > 0 {
			affected = append(affected, uid)
		}
	}
	return affected, nil
}

// ListUserRewardGrants 查询邀请人的两步奖与阶梯奖发放记录。
func (r *affiliateRepository) ListUserRewardGrants(ctx context.Context, inviterID int64) ([]service.AffiliateRewardGrantRecord, error) {
	if inviterID <= 0 {
		return nil, nil
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT g.id,
       g.operation_id,
       g.inviter_id,
       COALESCE(inviter.email, ''),
       COALESCE(inviter.username, ''),
       g.invitee_id,
       COALESCE(invitee.email, ''),
       COALESCE(invitee.username, ''),
       g.reward_type,
       g.tier_target,
       g.amount::double precision,
       g.status,
       g.review_reason,
       g.review_note,
       g.reviewed_by,
       g.reviewed_at,
       g.granted_at,
       g.created_at
FROM affiliate_reward_grants g
JOIN users inviter ON inviter.id = g.inviter_id
LEFT JOIN users invitee ON invitee.id = g.invitee_id
WHERE g.inviter_id = $1
ORDER BY g.created_at DESC`, inviterID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanAffiliateRewardGrantRows(rows)
}

// ListAdminRewardGrants 管理端分页查询两步奖与阶梯奖发放/待审核列表。
func (r *affiliateRepository) ListAdminRewardGrants(ctx context.Context, filter service.AffiliateRecordFilter) ([]service.AffiliateRewardGrantRecord, int64, error) {
	client := clientFromContext(ctx, r.client)
	where, args := buildAffiliateRecordWhere(filter, "g.created_at", []string{
		"inviter.email", "inviter.username", "invitee.email", "invitee.username",
		"g.inviter_id::text", "g.reward_type", "g.review_reason",
	})
	if st := strings.TrimSpace(filter.Status); st != "" {
		args = append(args, st)
		cond := fmt.Sprintf("g.status = $%d", len(args))
		if where == "" {
			where = "WHERE " + cond
		} else {
			where = where + " AND " + cond
		}
	}

	baseJoin := `
FROM affiliate_reward_grants g
JOIN users inviter ON inviter.id = g.inviter_id
LEFT JOIN users invitee ON invitee.id = g.invitee_id
` + where

	total, err := queryAffiliateRecordCount(ctx, client, "SELECT COUNT(*) "+baseJoin, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := buildAffiliateRecordOrderBy(filter, map[string]string{
		"inviter":     "inviter.email",
		"reward_type": "g.reward_type",
		"amount":      "g.amount",
		"status":      "g.status",
		"created_at":  "g.created_at",
	}, "g.created_at")

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := client.QueryContext(ctx, `
SELECT g.id,
       g.operation_id,
       g.inviter_id,
       COALESCE(inviter.email, ''),
       COALESCE(inviter.username, ''),
       g.invitee_id,
       COALESCE(invitee.email, ''),
       COALESCE(invitee.username, ''),
       g.reward_type,
       g.tier_target,
       g.amount::double precision,
       g.status,
       g.review_reason,
       g.review_note,
       g.reviewed_by,
       g.reviewed_at,
       g.granted_at,
       g.created_at
`+baseJoin+`
`+orderBy+`
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanAffiliateRewardGrantRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func scanAffiliateRewardGrantRows(rows *sql.Rows) ([]service.AffiliateRewardGrantRecord, error) {
	items := make([]service.AffiliateRewardGrantRecord, 0)
	for rows.Next() {
		var item service.AffiliateRewardGrantRecord
		var inviteeID sql.NullInt64
		var reviewedBy sql.NullInt64
		var reviewedAt sql.NullTime
		var grantedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.OperationID,
			&item.InviterID,
			&item.InviterEmail,
			&item.InviterUsername,
			&inviteeID,
			&item.InviteeEmail,
			&item.InviteeUsername,
			&item.RewardType,
			&item.TierTarget,
			&item.Amount,
			&item.Status,
			&item.ReviewReason,
			&item.ReviewNote,
			&reviewedBy,
			&reviewedAt,
			&grantedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.InviteeID = nullableInt64Ptr(inviteeID)
		item.ReviewedBy = nullableInt64Ptr(reviewedBy)
		if reviewedAt.Valid {
			t := reviewedAt.Time
			item.ReviewedAt = &t
		}
		if grantedAt.Valid {
			t := grantedAt.Time
			item.GrantedAt = &t
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ApproveRewardGrant 管理员审核通过一笔待审核奖励（50/100 人阶梯奖或超日限/同 IP 预警奖励），原子发放至邀请人余额。
func (r *affiliateRepository) ApproveRewardGrant(ctx context.Context, grantID, reviewerID int64, note string) (*service.AffiliateRewardGrantRecord, error) {
	var out *service.AffiliateRewardGrantRecord
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateGrantScopeTx(txCtx, txClient, grantID); err != nil {
			return err
		}

		rows, err := txClient.QueryContext(txCtx, `
SELECT operation_id, inviter_id, invitee_id, reward_type, tier_target, amount::double precision, status
FROM affiliate_reward_grants
WHERE id = $1
FOR UPDATE`, grantID)
		if err != nil {
			return err
		}
		if !rows.Next() {
			_ = rows.Close()
			return service.ErrAffiliateRewardGrantNotFound
		}
		var opID, rewardType, status string
		var inviterID int64
		var inviteeID sql.NullInt64
		var tierTarget int
		var amount float64
		if err := rows.Scan(&opID, &inviterID, &inviteeID, &rewardType, &tierTarget, &amount, &status); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if status != "pending_review" {
			return service.ErrAffiliateRewardGrantNotPending
		}

		eligible, err := affiliateGrantStillEligibleTx(txCtx, txClient, grantID, inviterID, inviteeID, rewardType, tierTarget)
		if err != nil {
			return err
		}
		if !eligible {
			return service.ErrAffiliateRewardGrantNotPending
		}

		var reviewerPtr *int64
		if reviewerID > 0 {
			reviewerPtr = &reviewerID
		}
		if _, err := txClient.ExecContext(txCtx, `
UPDATE affiliate_reward_grants
SET status = 'granted',
    review_note = $1,
    reviewed_by = $2,
    reviewed_at = NOW(),
    granted_at = NOW(),
    updated_at = NOW()
WHERE id = $3`, note, nullableInt64Arg(reviewerPtr), grantID); err != nil {
			return err
		}

		inviteePtr := nullableInt64Ptr(inviteeID)
		if err := creditInviterRewardBalanceTx(txCtx, txClient, inviterID, inviteePtr, rewardType, opID, amount); err != nil {
			return err
		}

		if inviteePtr != nil {
			switch rewardType {
			case "step1":
				if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET step1_reward_status = 'granted', updated_at = NOW() WHERE user_id = $1`, *inviteePtr); err != nil {
					return err
				}
			case "step2":
				if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET step2_reward_status = 'granted', updated_at = NOW() WHERE user_id = $1`, *inviteePtr); err != nil {
					return err
				}
			}
		}

		now := time.Now()
		out = &service.AffiliateRewardGrantRecord{
			ID:          grantID,
			OperationID: opID,
			InviterID:   inviterID,
			InviteeID:   inviteePtr,
			RewardType:  rewardType,
			TierTarget:  tierTarget,
			Amount:      amount,
			Status:      "granted",
			ReviewNote:  note,
			ReviewedBy:  reviewerPtr,
			ReviewedAt:  &now,
			GrantedAt:   &now,
		}
		return nil
	})
	return out, err
}

// RejectRewardGrant 管理员拒绝一笔待审核奖励。
func (r *affiliateRepository) RejectRewardGrant(ctx context.Context, grantID, reviewerID int64, note string) (*service.AffiliateRewardGrantRecord, error) {
	var out *service.AffiliateRewardGrantRecord
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateGrantScopeTx(txCtx, txClient, grantID); err != nil {
			return err
		}

		rows, err := txClient.QueryContext(txCtx, `
SELECT operation_id, inviter_id, invitee_id, reward_type, tier_target, amount::double precision, status
FROM affiliate_reward_grants
WHERE id = $1
FOR UPDATE`, grantID)
		if err != nil {
			return err
		}
		if !rows.Next() {
			_ = rows.Close()
			return service.ErrAffiliateRewardGrantNotFound
		}
		var opID, rewardType, status string
		var inviterID int64
		var inviteeID sql.NullInt64
		var tierTarget int
		var amount float64
		if err := rows.Scan(&opID, &inviterID, &inviteeID, &rewardType, &tierTarget, &amount, &status); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if status != "pending_review" {
			return service.ErrAffiliateRewardGrantNotPending
		}

		var reviewerPtr *int64
		if reviewerID > 0 {
			reviewerPtr = &reviewerID
		}
		if _, err := txClient.ExecContext(txCtx, `
UPDATE affiliate_reward_grants
SET status = 'rejected',
    review_note = $1,
    reviewed_by = $2,
    reviewed_at = NOW(),
    updated_at = NOW()
WHERE id = $3`, note, nullableInt64Arg(reviewerPtr), grantID); err != nil {
			return err
		}

		inviteePtr := nullableInt64Ptr(inviteeID)
		if inviteePtr != nil {
			switch rewardType {
			case "step1":
				if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET step1_reward_status = 'rejected', updated_at = NOW() WHERE user_id = $1`, *inviteePtr); err != nil {
					return err
				}
			case "step2":
				if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET step2_reward_status = 'rejected', updated_at = NOW() WHERE user_id = $1`, *inviteePtr); err != nil {
					return err
				}
			}
		}

		now := time.Now()
		out = &service.AffiliateRewardGrantRecord{
			ID:          grantID,
			OperationID: opID,
			InviterID:   inviterID,
			InviteeID:   inviteePtr,
			RewardType:  rewardType,
			TierTarget:  tierTarget,
			Amount:      amount,
			Status:      "rejected",
			ReviewNote:  note,
			ReviewedBy:  reviewerPtr,
			ReviewedAt:  &now,
		}
		return nil
	})
	return out, err
}

// SetInviterTrialGrantPaused 管理员设置或解除邀请人的新人体验金连带风控熔断状态。
func (r *affiliateRepository) SetInviterTrialGrantPaused(ctx context.Context, inviterID int64, paused bool, reason string) error {
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, inviterID); err != nil {
			return err
		}
		if !paused {
			reason = ""
		}
		_, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET trial_grant_paused = $1,
    trial_grant_pause_reason = $2,
    updated_at = NOW()
WHERE user_id = $3`, paused, reason, inviterID)
		return err
	})
}
