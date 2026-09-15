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
	ModelRadarTestTypeLogic    = "logic"
	ModelRadarTestTypeDrawing  = "drawing"
	ModelRadarStatusPassed     = "passed"
	ModelRadarStatusFailed     = "failed"
	ModelRadarStatusRequest    = "request_failed"
	ModelRadarStatusPending    = "pending_review"
	ModelRadarReviewPassed     = "passed"
	ModelRadarReviewFailed     = "failed"
	modelRadarInterval         = 30 * time.Minute
	modelRadarCycleInterval    = time.Minute
	modelRadarMaxResponseBytes = 2 << 20
)

var (
	ErrModelRadarUnavailable = errors.New("model radar service is unavailable")
	ErrModelRadarOpenAIOnly  = errors.New("model radar supports OpenAI groups only")
	ErrModelRadarGroup       = errors.New("OpenAI group not found")
	ErrModelRadarReview      = errors.New("invalid model radar review status")
	logicAnswerPattern       = regexp.MustCompile(`(?i)(?:答案|answer|结果|result)\s*[:：]?\s*(?:是\s*)?(21)\b|\b(21)\b`)
)

type ModelRadarConfig struct {
	ID              int64      `json:"id"`
	GroupID         int64      `json:"group_id"`
	ModelID         string     `json:"model_id"`
	ReasoningEffort string     `json:"reasoning_effort"`
	Enabled         bool       `json:"enabled"`
	NextRunAt       *time.Time `json:"next_run_at,omitempty"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
}

type ModelRadarResult struct {
	ID           int64      `json:"id"`
	GroupID      int64      `json:"group_id"`
	ModelID      string     `json:"model_id"`
	TestType     string     `json:"test_type"`
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
	NextRunAt       *time.Time         `json:"next_run_at,omitempty"`
	LastRunAt       *time.Time         `json:"last_run_at,omitempty"`
	Logic           *ModelRadarResult  `json:"logic,omitempty"`
	Drawing         *ModelRadarResult  `json:"drawing,omitempty"`
	Timeline        []ModelRadarResult `json:"timeline"`
}

type ModelRadarOverview struct {
	Groups          []ModelRadarGroupOverview `json:"groups"`
	IntervalMins    int                       `json:"interval_minutes"`
	IQStatus        string                    `json:"iq_status"`
	RecommendStatus string                    `json:"recommend_status"`
}

type ModelRadarRepository interface {
	ListConfigs(context.Context) ([]*ModelRadarConfig, error)
	UpsertConfig(context.Context, *ModelRadarConfig) (*ModelRadarConfig, error)
	ListResults(context.Context, *int64, string, int) ([]*ModelRadarResult, error)
	CreateResult(context.Context, *ModelRadarResult) (*ModelRadarResult, error)
	ReviewResult(context.Context, int64, string, int64) (*ModelRadarResult, error)
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
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				_ = s.runDue(ctx, now)
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
	results, err := s.repo.ListResults(ctx, nil, "", 300)
	if err != nil {
		return nil, err
	}
	resultByGroup := make(map[int64][]*ModelRadarResult)
	for _, result := range results {
		resultByGroup[result.GroupID] = append(resultByGroup[result.GroupID], result)
	}
	out := &ModelRadarOverview{Groups: make([]ModelRadarGroupOverview, 0, len(groups)), IntervalMins: 30, IQStatus: "未开发", RecommendStatus: "未开发"}
	for _, group := range groups {
		item := ModelRadarGroupOverview{GroupID: group.ID, GroupName: group.Name, ModelID: DefaultRadarModel(&group), ReasoningEffort: "medium", Timeline: []ModelRadarResult{}}
		if cfg := configByGroup[group.ID]; cfg != nil {
			item.ModelID, item.ReasoningEffort, item.Enabled, item.NextRunAt, item.LastRunAt = cfg.ModelID, cfg.ReasoningEffort, cfg.Enabled, cfg.NextRunAt, cfg.LastRunAt
			if item.ModelID == "" {
				item.ModelID = DefaultRadarModel(&group)
			}
		}
		for _, result := range resultByGroup[group.ID] {
			copy := *result
			if !admin {
				copy.ResponseText, copy.ErrorMessage, copy.ReviewedBy = "", "", nil
			}
			item.Timeline = append(item.Timeline, copy)
			if result.TestType == ModelRadarTestTypeLogic && item.Logic == nil {
				item.Logic = &copy
			}
			if result.TestType == ModelRadarTestTypeDrawing && item.Drawing == nil {
				item.Drawing = &copy
			}
		}
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
		if cfg.Enabled && cfg.NextRunAt == nil {
			now := s.now()
			cfg.NextRunAt = &now
		}
		if !cfg.Enabled {
			cfg.NextRunAt = nil
		}
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
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *ModelRadarService) runDue(ctx context.Context, now time.Time) error {
	configs, err := s.repo.ListConfigs(ctx)
	if err != nil {
		return err
	}
	for _, cfg := range configs {
		if cfg.Enabled && cfg.NextRunAt != nil && !cfg.NextRunAt.After(now) {
			if err := s.runGroup(ctx, cfg.GroupID); err != nil {
				continue
			}
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
		cfg = &ModelRadarConfig{GroupID: groupID, ModelID: DefaultRadarModel(group), ReasoningEffort: "medium", Enabled: true}
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, groupID, PlatformOpenAI)
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return s.finishRun(ctx, cfg)
	}
	accountID := accounts[0].ID
	probes := []struct{ typ, prompt string }{{ModelRadarTestTypeLogic, radarLogicPrompt}, {ModelRadarTestTypeDrawing, radarDrawingPrompt}}
	for _, probe := range probes {
		result, callErr := s.accountTest.RunTestBackgroundWithPromptAndReasoning(ctx, accountID, cfg.ModelID, probe.prompt, cfg.ReasoningEffort)
		stored := &ModelRadarResult{GroupID: groupID, ModelID: cfg.ModelID, TestType: probe.typ, DetectedAt: s.now()}
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
			if probe.typ == ModelRadarTestTypeLogic {
				if logicAnswerPattern.MatchString(result.ResponseText) {
					stored.Status = ModelRadarStatusPassed
				} else {
					stored.Status = ModelRadarStatusFailed
				}
			} else {
				if strings.Contains(strings.ToLower(result.ResponseText), "<svg") {
					stored.Status = ModelRadarStatusPending
					stored.ReviewStatus = "pending"
				} else {
					stored.Status = ModelRadarStatusFailed
				}
			}
		}
		if _, err := s.repo.CreateResult(ctx, stored); err != nil {
			return err
		}
	}
	return s.finishRun(ctx, cfg)
}

func truncateRadarResponse(value string) string {
	if len(value) <= modelRadarMaxResponseBytes {
		return value
	}
	return value[:modelRadarMaxResponseBytes]
}

func (s *ModelRadarService) finishRun(ctx context.Context, cfg *ModelRadarConfig) error {
	now := s.now()
	cfg.LastRunAt = &now
	if cfg.Enabled {
		next := now.Add(modelRadarInterval)
		cfg.NextRunAt = &next
	} else {
		cfg.NextRunAt = nil
	}
	_, err := s.repo.UpsertConfig(ctx, cfg)
	return err
}

func (s *ModelRadarService) ReviewResult(ctx context.Context, resultID int64, status string, reviewerID int64) (*ModelRadarResult, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != ModelRadarReviewPassed && status != ModelRadarReviewFailed {
		return nil, ErrModelRadarReview
	}
	return s.repo.ReviewResult(ctx, resultID, status, reviewerID)
}

const radarLogicPrompt = `Solve this logic question. What is the next number in the sequence 1, 3, 6, 10, 15, ? Explain briefly, then finish with exactly "Answer: 21".`
const radarDrawingPrompt = `Return a complete standalone SVG document only. Draw a simple animated radar sweep: a circular radar grid, one green sweep line, and a pulsing dot. Use valid SVG with a short CSS or SMIL animation. Do not use markdown fences.`

func (s *ModelRadarService) PublicOverview(ctx context.Context) (*ModelRadarOverview, error) {
	return s.GetOverview(ctx, false)
}
func (s *ModelRadarService) AdminOverview(ctx context.Context) (*ModelRadarOverview, error) {
	return s.GetOverview(ctx, true)
}
