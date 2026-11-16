package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// ModelRadarTestTypeLogic 是旧的数列题，只为读历史数据保留，不再发起也不再展示。
	ModelRadarTestTypeLogic    = "logic"
	ModelRadarTestTypeCiallo   = "ciallo"
	ModelRadarTestTypeDrawing  = "drawing"
	ModelRadarStatusPassed     = "passed"
	ModelRadarStatusFailed     = "failed"
	ModelRadarStatusRequest    = "request_failed"
	ModelRadarStatusPending    = "pending_review"
	ModelRadarReviewPassed     = "passed"
	ModelRadarReviewFailed     = "failed"
	modelRadarCycleInterval    = time.Minute
	modelRadarMaxResponseBytes = 2 << 20
	modelRadarDefaultEffort    = "high"
	modelRadarTimelineSize     = 24
	// 正常模型也不是每次都带颜文字（实测 gpt-6-astra high 大约一半到八成），
	// 降智的则一次都不带，所以一轮里多问几次，有一次带就算正常。
	modelRadarCialloAttempts = 3
	// 判定结论：watch 是「最近没带颜文字，但还没连续到判定降智的轮数」。
	ModelRadarVerdictOK     = "ok"
	ModelRadarVerdictNerfed = "nerfed"
	ModelRadarVerdictWatch  = "watch"
	ModelRadarVerdictError  = "error"
	ModelRadarVerdictNone   = "none"
	modelRadarCleanupEvery  = time.Hour
)

var (
	ErrModelRadarUnavailable = errors.New("model radar service is unavailable")
	ErrModelRadarOpenAIOnly  = errors.New("model radar supports OpenAI groups only")
	ErrModelRadarGroup       = errors.New("OpenAI group not found")
	ErrModelRadarReview      = errors.New("invalid model radar review status")
	cialloWordPattern        = regexp.MustCompile(`(?i)ciallo`)
	// 颜文字：括号里带表情符号的一小段，或 Ciallo 招牌的「⌒☆」尾巴。emoji（如 👋）不算。
	cialloKaomojiPattern = regexp.MustCompile(`[(（][^()（）\n]{0,24}[ω∀▽∇・･´｀≧≦﹏∠°□Дᴗ•╹◡‿ㅂ╥＾￣＿＞＜][^()（）\n]{0,24}[)）]|[(（]\s*[\^>;T]\s*[_.o\-wv]+\s*[\^<;T]\s*[)）]|⌒\s*[☆★]`)
)

type ModelRadarConfig struct {
	ID              int64  `json:"id"`
	GroupID         int64  `json:"group_id"`
	ModelID         string `json:"model_id"`
	ReasoningEffort string `json:"reasoning_effort"`
	Enabled         bool   `json:"enabled"`
	// IntervalMinutes 为 0 表示跟随全局间隔。
	IntervalMinutes int `json:"interval_minutes"`
	// Visible 为 false 时照常检测，但不出现在用户的降智雷达里。
	Visible   bool       `json:"visible"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
}

// ModelRadarSettings 是雷达的全局设置，库里只有一行。
type ModelRadarSettings struct {
	IntervalMinutes int `json:"interval_minutes"`
	// CialloAttempts：一轮里最多问几次，有一次带颜文字就算正常。
	CialloAttempts int `json:"ciallo_attempts"`
	// NerfConfirmRounds：连续几轮不带颜文字才判定「疑似降智」，不足时显示「观察中」。
	NerfConfirmRounds int  `json:"nerf_confirm_rounds"`
	DrawingEnabled    bool `json:"drawing_enabled"`
	// RetentionDays 为 0 表示不自动清理历史结果。
	RetentionDays int `json:"retention_days"`
}

func DefaultModelRadarSettings() ModelRadarSettings {
	return ModelRadarSettings{IntervalMinutes: 30, CialloAttempts: modelRadarCialloAttempts, NerfConfirmRounds: 1, DrawingEnabled: true}
}

func clampRadarInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func normalizeRadarSettings(in ModelRadarSettings) ModelRadarSettings {
	in.IntervalMinutes = clampRadarInt(in.IntervalMinutes, 5, 1440)
	in.CialloAttempts = clampRadarInt(in.CialloAttempts, 1, 5)
	in.NerfConfirmRounds = clampRadarInt(in.NerfConfirmRounds, 1, 5)
	in.RetentionDays = clampRadarInt(in.RetentionDays, 0, 365)
	return in
}

func radarInterval(cfg *ModelRadarConfig, settings ModelRadarSettings) time.Duration {
	minutes := settings.IntervalMinutes
	if cfg != nil && cfg.IntervalMinutes > 0 {
		minutes = cfg.IntervalMinutes
	}
	return time.Duration(minutes) * time.Minute
}

type ModelRadarResult struct {
	ID           int64      `json:"id"`
	GroupID      int64      `json:"group_id"`
	ModelID      string     `json:"model_id"`
	TestType     string     `json:"test_type"`
	Prompt       string     `json:"prompt"`
	Status       string     `json:"status"`
	ResponseText string     `json:"response_text,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	LatencyMS    int64      `json:"latency_ms"`
	DetectedAt   time.Time  `json:"detected_at"`
	ReviewStatus string     `json:"review_status,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy   *int64     `json:"reviewed_by,omitempty"`
}

type ModelRadarGroupOverview struct {
	GroupID         int64              `json:"group_id"`
	GroupName       string             `json:"group_name"`
	ModelID         string             `json:"model_id"`
	ReasoningEffort string             `json:"reasoning_effort"`
	Enabled         bool               `json:"enabled"`
	IntervalMinutes int                `json:"interval_minutes"`
	Visible         bool               `json:"visible"`
	Verdict         string             `json:"verdict"`
	NextRunAt       *time.Time         `json:"next_run_at,omitempty"`
	LastRunAt       *time.Time         `json:"last_run_at,omitempty"`
	Ciallo          *ModelRadarResult  `json:"ciallo,omitempty"`
	Drawing         *ModelRadarResult  `json:"drawing,omitempty"`
	Timeline        []ModelRadarResult `json:"timeline"`
}

type ModelRadarOverview struct {
	Groups       []ModelRadarGroupOverview `json:"groups"`
	IntervalMins int                       `json:"interval_minutes"`
	Prompt       string                    `json:"prompt"`
	ExpectedText string                    `json:"expected_reply"`
	Settings     ModelRadarSettings        `json:"settings"`
}

type ModelRadarRepository interface {
	ListConfigs(context.Context) ([]*ModelRadarConfig, error)
	UpsertConfig(context.Context, *ModelRadarConfig) (*ModelRadarConfig, error)
	ListResults(context.Context, *int64, string, int) ([]*ModelRadarResult, error)
	CreateResult(context.Context, *ModelRadarResult) (*ModelRadarResult, error)
	ReviewResult(context.Context, int64, string, int64) (*ModelRadarResult, error)
	GetSettings(context.Context) (*ModelRadarSettings, error)
	UpdateSettings(context.Context, ModelRadarSettings) (*ModelRadarSettings, error)
	// ClaimDue 把到期分组的下次时间原子地推后；多台节点同时到点时只有一台拿到 true。
	ClaimDue(ctx context.Context, groupID int64, now, next time.Time) (bool, error)
	// MarkRun 只更新运行时间，不碰管理员可编辑的字段。
	MarkRun(ctx context.Context, groupID int64, last, next time.Time) error
	DeleteResult(ctx context.Context, resultID int64) error
	DeleteGroupResults(ctx context.Context, groupID int64) (int64, error)
	DeleteResultsBefore(ctx context.Context, before time.Time) (int64, error)
}

type ModelRadarService struct {
	repo         ModelRadarRepository
	accountRepo  AccountRepository
	groupRepo    GroupRepository
	accountTest  *AccountTestService
	now          func() time.Time
	parentCtx    context.Context
	parentCancel context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.Mutex
	running      map[int64]bool
	started      bool
}

func NewModelRadarService(repo ModelRadarRepository, accountRepo AccountRepository, groupRepo GroupRepository, accountTest *AccountTestService) *ModelRadarService {
	return &ModelRadarService{repo: repo, accountRepo: accountRepo, groupRepo: groupRepo, accountTest: accountTest, now: time.Now, running: make(map[int64]bool)}
}

func (s *ModelRadarService) Start() {
	if s == nil || s.repo == nil || s.accountTest == nil {
		return
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.parentCtx, s.parentCancel = context.WithCancel(context.Background())
	ctx := s.parentCtx
	s.wg.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(modelRadarCycleInterval)
		defer ticker.Stop()
		var lastCleanup time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				_ = s.runDue(ctx, now)
				if now.Sub(lastCleanup) >= modelRadarCleanupEvery {
					lastCleanup = now
					s.cleanup(ctx, now)
				}
			}
		}
	}()
}

func (s *ModelRadarService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.parentCancel != nil {
		s.parentCancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// loadSettings 读全局设置；读不到（比如迁移还没跑）就用默认值，不让雷达停摆。
func (s *ModelRadarService) loadSettings(ctx context.Context) ModelRadarSettings {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil || settings == nil {
		return DefaultModelRadarSettings()
	}
	return normalizeRadarSettings(*settings)
}

func (s *ModelRadarService) UpdateSettings(ctx context.Context, in ModelRadarSettings) (*ModelRadarSettings, error) {
	if s == nil || s.repo == nil {
		return nil, ErrModelRadarUnavailable
	}
	return s.repo.UpdateSettings(ctx, normalizeRadarSettings(in))
}

func (s *ModelRadarService) cleanup(ctx context.Context, now time.Time) {
	settings := s.loadSettings(ctx)
	if settings.RetentionDays <= 0 {
		return
	}
	_, _ = s.repo.DeleteResultsBefore(ctx, now.AddDate(0, 0, -settings.RetentionDays))
}

func (s *ModelRadarService) DeleteResult(ctx context.Context, resultID int64) error {
	if s == nil || s.repo == nil {
		return ErrModelRadarUnavailable
	}
	return s.repo.DeleteResult(ctx, resultID)
}

func (s *ModelRadarService) ClearGroupResults(ctx context.Context, groupID int64) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, ErrModelRadarUnavailable
	}
	return s.repo.DeleteGroupResults(ctx, groupID)
}

// ClearAllResults 清空全部检测记录（配置不动）。
func (s *ModelRadarService) ClearAllResults(ctx context.Context) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, ErrModelRadarUnavailable
	}
	return s.repo.DeleteResultsBefore(ctx, s.now().Add(24*time.Hour))
}

// radarVerdict 由某分组的 ciallo 结果（新到旧）得出结论。最新一次带颜文字就是正常；
// 最新一次请求失败是「请求失败」；否则数连续不带颜文字的轮数（中间的请求失败跳过不算），
// 够 rounds 轮才判降智，不够是观察中。
func radarVerdict(results []*ModelRadarResult, rounds int) string {
	if len(results) == 0 {
		return ModelRadarVerdictNone
	}
	switch results[0].Status {
	case ModelRadarStatusPassed, ModelRadarStatusPending:
		return ModelRadarVerdictOK
	case ModelRadarStatusFailed:
	default:
		return ModelRadarVerdictError
	}
	streak := 0
	for _, result := range results {
		if result.Status == ModelRadarStatusFailed {
			streak++
			if streak >= rounds {
				return ModelRadarVerdictNerfed
			}
			continue
		}
		if result.Status == ModelRadarStatusRequest {
			continue
		}
		break
	}
	return ModelRadarVerdictWatch
}

func (s *ModelRadarService) GetOverview(ctx context.Context, admin bool) (*ModelRadarOverview, error) {
	if s == nil || s.repo == nil || s.groupRepo == nil {
		return nil, ErrModelRadarUnavailable
	}
	groups, err := s.groupRepo.ListActiveByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	configs, err := s.repo.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	configByGroup := make(map[int64]*ModelRadarConfig, len(configs))
	for _, cfg := range configs {
		configByGroup[cfg.GroupID] = cfg
	}
	// 两类结果分开取：ciallo 回复很短，可以多取做时间线；画图回复很大，只要每组最新一条。
	results, err := s.repo.ListResults(ctx, nil, ModelRadarTestTypeCiallo, 1000)
	if err != nil {
		return nil, err
	}
	drawings, err := s.repo.ListResults(ctx, nil, ModelRadarTestTypeDrawing, 60)
	if err != nil {
		return nil, err
	}
	results = append(results, drawings...)
	resultByGroup := make(map[int64][]*ModelRadarResult)
	for _, result := range results {
		resultByGroup[result.GroupID] = append(resultByGroup[result.GroupID], result)
	}
	settings := s.loadSettings(ctx)
	out := &ModelRadarOverview{Groups: make([]ModelRadarGroupOverview, 0, len(groups)), IntervalMins: settings.IntervalMinutes, Prompt: radarCialloPrompt, ExpectedText: radarCialloExpected, Settings: settings}
	for _, group := range groups {
		cfg := configByGroup[group.ID]
		if !admin && (group.IsExclusive || cfg == nil || !cfg.Enabled || !cfg.Visible) {
			continue
		}
		item := ModelRadarGroupOverview{GroupID: group.ID, GroupName: group.Name, ModelID: DefaultRadarModel(&group), ReasoningEffort: modelRadarDefaultEffort, Visible: true, Timeline: []ModelRadarResult{}}
		cialloResults := make([]*ModelRadarResult, 0, modelRadarTimelineSize)
		if cfg != nil {
			item.ModelID, item.ReasoningEffort, item.Enabled, item.NextRunAt, item.LastRunAt = cfg.ModelID, cfg.ReasoningEffort, cfg.Enabled, cfg.NextRunAt, cfg.LastRunAt
			item.IntervalMinutes, item.Visible = cfg.IntervalMinutes, cfg.Visible
			if item.ModelID == "" {
				item.ModelID = DefaultRadarModel(&group)
			}
		}
		for _, result := range resultByGroup[group.ID] {
			current := *result
			current.Prompt = modelRadarPrompt(result.TestType)
			if current.TestType == ModelRadarTestTypeDrawing && current.Status == ModelRadarStatusPending && containsRadarSVG(current.ResponseText) {
				current.Status = ModelRadarStatusPassed
				current.ReviewStatus = ""
			}
			if !admin {
				current.ErrorMessage, current.ReviewedBy = "", nil
			}
			if result.TestType == ModelRadarTestTypeCiallo {
				cialloResults = append(cialloResults, result)
				if item.Ciallo == nil {
					item.Ciallo = &current
				}
				if len(item.Timeline) < modelRadarTimelineSize {
					timeline := current
					timeline.ErrorMessage, timeline.ReviewedBy = "", nil
					item.Timeline = append(item.Timeline, timeline)
				}
			}
			if result.TestType == ModelRadarTestTypeDrawing && item.Drawing == nil && settings.DrawingEnabled {
				item.Drawing = &current
			}
		}
		item.Verdict = radarVerdict(cialloResults, settings.NerfConfirmRounds)
		out.Groups = append(out.Groups, item)
	}
	return out, nil
}

func DefaultRadarModel(group *Group) string {
	if group != nil && strings.TrimSpace(group.DefaultMappedModel) != "" {
		return strings.TrimSpace(group.DefaultMappedModel)
	}
	return "gpt-5.4"
}

func (s *ModelRadarService) UpdateConfigs(ctx context.Context, configs []*ModelRadarConfig) ([]*ModelRadarConfig, error) {
	if s == nil || s.repo == nil || s.groupRepo == nil {
		return nil, ErrModelRadarUnavailable
	}
	groups, err := s.groupRepo.ListActiveByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	allowed := make(map[int64]Group, len(groups))
	for _, group := range groups {
		allowed[group.ID] = group
	}
	settings := s.loadSettings(ctx)
	out := make([]*ModelRadarConfig, 0, len(configs))
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		group, ok := allowed[cfg.GroupID]
		if !ok || group.Platform != PlatformOpenAI || !group.IsActive() {
			return nil, ErrModelRadarOpenAIOnly
		}
		cfg.ModelID = strings.TrimSpace(cfg.ModelID)
		if cfg.ModelID == "" {
			cfg.ModelID = DefaultRadarModel(&group)
		}
		cfg.ReasoningEffort = normalizeRadarEffort(cfg.ReasoningEffort)
		if cfg.ReasoningEffort == "" {
			return nil, fmt.Errorf("invalid reasoning effort")
		}
		if cfg.IntervalMinutes != 0 {
			cfg.IntervalMinutes = clampRadarInt(cfg.IntervalMinutes, 5, 1440)
		}
		// 下次时间由仓库层决定：新开启的立刻排上，已排的不往后拖，只在间隔调短时提前。
		latest := s.now().Add(radarInterval(cfg, settings))
		cfg.NextRunAt = &latest
		updated, err := s.repo.UpsertConfig(ctx, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, updated)
	}
	return out, nil
}

func normalizeRadarEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "minimal", "low", "medium", "high", "xhigh", "max":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func (s *ModelRadarService) RunNow(ctx context.Context, groupIDs []int64) (int, error) {
	if len(groupIDs) == 0 {
		groups, err := s.groupRepo.ListActiveByPlatform(ctx, PlatformOpenAI)
		if err != nil {
			return 0, err
		}
		for _, group := range groups {
			groupIDs = append(groupIDs, group.ID)
		}
	}
	count := 0
	for _, groupID := range groupIDs {
		if err := s.runGroup(ctx, groupID); err != nil {
			continue
		}
		count++
	}
	return count, nil
}

// RunNowInBackground 供后台「立即检测」用：一轮要跑文字题加画图题，可能要几分钟，
// 所以立刻返回要检测的分组数，结果由页面轮询总览拿。
func (s *ModelRadarService) RunNowInBackground(ctx context.Context, groupIDs []int64) (int, error) {
	if s == nil || s.repo == nil || s.groupRepo == nil || s.accountTest == nil {
		return 0, ErrModelRadarUnavailable
	}
	if len(groupIDs) == 0 {
		groups, err := s.groupRepo.ListActiveByPlatform(ctx, PlatformOpenAI)
		if err != nil {
			return 0, err
		}
		for _, group := range groups {
			groupIDs = append(groupIDs, group.ID)
		}
	}
	s.mu.Lock()
	parent := s.parentCtx
	if parent == nil {
		parent = context.Background()
	}
	s.wg.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wg.Done()
		runCtx, cancel := context.WithTimeout(parent, 15*time.Minute)
		defer cancel()
		_, _ = s.RunNow(runCtx, groupIDs)
	}()
	return len(groupIDs), nil
}

func (s *ModelRadarService) runDue(ctx context.Context, now time.Time) error {
	configs, err := s.repo.ListConfigs(ctx)
	if err != nil {
		return err
	}
	settings := s.loadSettings(ctx)
	for _, cfg := range configs {
		if cfg.Enabled && cfg.NextRunAt != nil && !cfg.NextRunAt.After(now) {
			// 线上有两台节点各跑一个调度器，先抢到的那台检测，另一台跳过。
			claimed, err := s.repo.ClaimDue(ctx, cfg.GroupID, now, now.Add(radarInterval(cfg, settings)))
			if err != nil || !claimed {
				continue
			}
			_ = s.runGroup(ctx, cfg.GroupID)
		}
	}
	return nil
}

func (s *ModelRadarService) runGroup(ctx context.Context, groupID int64) error {
	s.mu.Lock()
	if s.running[groupID] {
		s.mu.Unlock()
		return nil
	}
	s.running[groupID] = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.running, groupID); s.mu.Unlock() }()
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil || group == nil || group.Platform != PlatformOpenAI || !group.IsActive() {
		return ErrModelRadarOpenAIOnly
	}
	configs, err := s.repo.ListConfigs(ctx)
	if err != nil {
		return err
	}
	var cfg *ModelRadarConfig
	for _, candidate := range configs {
		if candidate.GroupID == groupID {
			cfg = candidate
			break
		}
	}
	if cfg == nil {
		cfg = &ModelRadarConfig{GroupID: groupID, ModelID: DefaultRadarModel(group), ReasoningEffort: modelRadarDefaultEffort}
	}
	settings := s.loadSettings(ctx)
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, groupID, PlatformOpenAI)
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return s.finishRun(ctx, cfg, settings)
	}
	accountID := accounts[0].ID
	probes := []struct{ typ, prompt string }{{ModelRadarTestTypeCiallo, radarCialloPrompt}}
	if settings.DrawingEnabled {
		probes = append(probes, struct{ typ, prompt string }{ModelRadarTestTypeDrawing, radarDrawingPrompt})
	}
	for _, probe := range probes {
		result, callErr := s.accountTest.RunTestBackgroundWithPromptAndReasoning(ctx, accountID, cfg.ModelID, probe.prompt, cfg.ReasoningEffort)
		if probe.typ == ModelRadarTestTypeCiallo {
			for attempt := 1; attempt < settings.CialloAttempts && callErr == nil && result != nil && result.Status == "success" && !IsCialloReply(result.ResponseText); attempt++ {
				result, callErr = s.accountTest.RunTestBackgroundWithPromptAndReasoning(ctx, accountID, cfg.ModelID, probe.prompt, cfg.ReasoningEffort)
			}
		}
		stored := &ModelRadarResult{GroupID: groupID, ModelID: cfg.ModelID, TestType: probe.typ, Prompt: probe.prompt, DetectedAt: s.now()}
		if callErr != nil || result == nil || result.Status != "success" {
			stored.Status = ModelRadarStatusRequest
			if callErr != nil {
				stored.ErrorMessage = callErr.Error()
			}
			if result != nil {
				stored.ResponseText, stored.ErrorMessage, stored.LatencyMS = truncateRadarResponse(result.ResponseText), truncateRadarResponse(result.ErrorMessage), result.LatencyMs
			}
		} else {
			stored.ResponseText, stored.LatencyMS = truncateRadarResponse(result.ResponseText), result.LatencyMs
			if probe.typ == ModelRadarTestTypeCiallo {
				if IsCialloReply(result.ResponseText) {
					stored.Status = ModelRadarStatusPassed
				} else {
					stored.Status = ModelRadarStatusFailed
				}
			} else {
				if containsRadarSVG(result.ResponseText) {
					stored.Status = ModelRadarStatusPassed
				} else {
					stored.Status = ModelRadarStatusFailed
				}
			}
		}
		if _, err := s.repo.CreateResult(ctx, stored); err != nil {
			return err
		}
	}
	return s.finishRun(ctx, cfg, settings)
}

func truncateRadarResponse(value string) string {
	if len(value) <= modelRadarMaxResponseBytes {
		return value
	}
	return value[:modelRadarMaxResponseBytes]
}

func containsRadarSVG(value string) bool {
	return strings.Contains(strings.ToLower(value), "<svg")
}

// IsCialloReply 判定 ciallo 题：没降智的模型认得这个梗，会回「Ciallo」并带上颜文字；
// 只回一句普通问候（哪怕带 emoji）算未通过。
func IsCialloReply(reply string) bool {
	return cialloWordPattern.MatchString(reply) && cialloKaomojiPattern.MatchString(reply)
}

func modelRadarPrompt(testType string) string {
	if testType == ModelRadarTestTypeDrawing {
		return radarDrawingPrompt
	}
	return radarCialloPrompt
}

func (s *ModelRadarService) finishRun(ctx context.Context, cfg *ModelRadarConfig, settings ModelRadarSettings) error {
	now := s.now()
	return s.repo.MarkRun(ctx, cfg.GroupID, now, now.Add(radarInterval(cfg, settings)))
}

func (s *ModelRadarService) ReviewResult(ctx context.Context, resultID int64, status string, reviewerID int64) (*ModelRadarResult, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != ModelRadarReviewPassed && status != ModelRadarReviewFailed {
		return nil, ErrModelRadarReview
	}
	return s.repo.ReviewResult(ctx, resultID, status, reviewerID)
}

// 实测（2026-11-15，gpt-6-astra / gpt-6.1-sol）：high 强度稳定回「Ciallo～(∠・ω< )⌒★」，
// medium 及以下只回「Ciallo～ 👋」，所以默认强度用 high。
const radarCialloPrompt = `ciallo`
const radarCialloExpected = `Ciallo～(∠・ω< )⌒☆`
const radarDrawingPrompt = `创建一个HTML，内容是SVG绘制一个鹈鹕骑自行车的2D动画`

func (s *ModelRadarService) PublicOverview(ctx context.Context) (*ModelRadarOverview, error) {
	return s.GetOverview(ctx, false)
}
func (s *ModelRadarService) AdminOverview(ctx context.Context) (*ModelRadarOverview, error) {
	return s.GetOverview(ctx, true)
}
