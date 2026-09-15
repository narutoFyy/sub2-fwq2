package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

const (
	SettingKeyOpenAILowCostProbe = "openai_low_cost_probe_settings"

	OpenAILowCostProbeStatusUnknown     = "unknown"
	OpenAILowCostProbeStatusHealthy     = "healthy"
	OpenAILowCostProbeStatusDegraded    = "degraded"
	OpenAILowCostProbeStatusCircuitOpen = "circuit_open"
	OpenAILowCostProbeStatusRecovering  = "recovering"

	OpenAILowCostProbeEstimatedCostUSD = 0.001

	openAILowCostProbeCycleInterval = 5 * time.Second
	openAILowCostProbeLeaderLockKey = "jobs:openai-low-cost-probe"
	openAILowCostProbeLeaderLockTTL = 2 * time.Minute
	openAILowCostProbeMaxBatchSize  = 50
	openAILowCostProbeMaxErrorBytes = 500
	openAILowCostPolicyCacheTTL     = time.Second
)

var ErrOpenAILowCostProbeUnavailable = infraerrors.ServiceUnavailable(
	"OPENAI_LOW_COST_PROBE_UNAVAILABLE",
	"OpenAI low-cost probe service is unavailable",
)

type OpenAILowCostProbeSettings struct {
	Enabled                      bool    `json:"enabled"`
	MaxGroupRateMultiplier       float64 `json:"max_group_rate_multiplier"`
	MaxProbeLatencyMS            int64   `json:"max_probe_latency_ms"`
	HealthyProbeIntervalSeconds  int     `json:"healthy_probe_interval_seconds"`
	RecoveryProbeIntervalSeconds int     `json:"recovery_probe_interval_seconds"`
	FailureThreshold             int     `json:"failure_threshold"`
	RecoverySuccesses            int     `json:"recovery_successes"`
	ProbeModel                   string  `json:"probe_model"`
	MaxProbeSpendUSDPer24H       float64 `json:"max_probe_spend_usd_per_24h"`
}

type OpenAILowCostProbeState struct {
	GroupID              int64      `json:"group_id"`
	AccountID            int64      `json:"account_id"`
	Status               string     `json:"status"`
	LastProbeAt          *time.Time `json:"last_probe_at,omitempty"`
	LastSuccessAt        *time.Time `json:"last_success_at,omitempty"`
	LastError            string     `json:"last_error,omitempty"`
	LatencyMS            *int64     `json:"latency_ms,omitempty"`
	ConsecutiveFailures  int        `json:"consecutive_failures"`
	ConsecutiveSuccesses int        `json:"consecutive_successes"`
	DisabledUntil        *time.Time `json:"disabled_until,omitempty"`
	EstimatedSpendUSD    float64    `json:"estimated_spend_usd"`
	UpdatedAt            time.Time  `json:"updated_at"`

	GroupName             string  `json:"group_name,omitempty"`
	AccountName           string  `json:"account_name,omitempty"`
	GroupRateMultiplier   float64 `json:"group_rate_multiplier,omitempty"`
	AccountRateMultiplier float64 `json:"account_rate_multiplier,omitempty"`
}

type OpenAILowCostProbeScope struct {
	GroupID   int64
	AccountID int64
}

type OpenAILowCostProbeStateRepository interface {
	ListStates(ctx context.Context) ([]OpenAILowCostProbeState, error)
	ListStatesByGroup(ctx context.Context, groupID int64) ([]OpenAILowCostProbeState, error)
	UpsertState(ctx context.Context, state *OpenAILowCostProbeState) error
	DeleteOutsideScope(ctx context.Context, scopes []OpenAILowCostProbeScope) error
	ReserveBudget(ctx context.Context, now time.Time, accountID int64, estimatedSpendUSD, maxSpendUSD float64) (usedUSD float64, reserved bool, err error)
	GetBudgetUsage(ctx context.Context, now time.Time) (float64, error)
}

type OpenAILowCostProbeStatesResponse struct {
	States             []OpenAILowCostProbeState `json:"states"`
	BudgetUsedUSD      float64                   `json:"budget_used_usd"`
	BudgetRemainingUSD float64                   `json:"budget_remaining_usd"`
	EstimatedCallUSD   float64                   `json:"estimated_call_usd"`
	LastProbeAt        *time.Time                `json:"last_probe_at,omitempty"`
}

type OpenAILowCostProbeRunItem struct {
	AccountID int64  `json:"account_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
}

type OpenAILowCostProbeRunResult struct {
	Executed           int                         `json:"executed"`
	Succeeded          int                         `json:"succeeded"`
	Failed             int                         `json:"failed"`
	Skipped            int                         `json:"skipped"`
	BudgetUsedUSD      float64                     `json:"budget_used_usd"`
	BudgetRemainingUSD float64                     `json:"budget_remaining_usd"`
	Items              []OpenAILowCostProbeRunItem `json:"items"`
}

type openAILowCostProbeTarget struct {
	account *Account
	groups  map[int64]*Group
}

type openAILowCostSchedulingPolicy struct {
	fetchedAt time.Time
	enabled   bool
	inScope   bool
	states    map[int64]string
}

type OpenAILowCostProbeService struct {
	accountRepo        AccountRepository
	groupRepo          GroupRepository
	stateRepo          OpenAILowCostProbeStateRepository
	settingRepo        SettingRepository
	accountTestService *AccountTestService
	opsRepo            OpsRepository
	lockCache          LeaderLockCache
	db                 *sql.DB
	instanceID         string

	probeFn func(context.Context, int64, string) (*ScheduledTestResult, error)
	now     func() time.Time

	parentCtx    context.Context
	parentCancel context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.Mutex
	cycleMu      sync.Mutex
	policyMu     sync.Mutex
	policyCache  map[int64]openAILowCostSchedulingPolicy
	started      bool
	stopped      bool
}

func DefaultOpenAILowCostProbeSettings() *OpenAILowCostProbeSettings {
	return &OpenAILowCostProbeSettings{
		Enabled:                      false,
		MaxGroupRateMultiplier:       1.0,
		MaxProbeLatencyMS:            6000,
		HealthyProbeIntervalSeconds:  60,
		RecoveryProbeIntervalSeconds: 15,
		FailureThreshold:             1,
		RecoverySuccesses:            2,
		ProbeModel:                   "codex-auto-review",
		MaxProbeSpendUSDPer24H:       0,
	}
}

func normalizeOpenAILowCostProbeSettings(settings *OpenAILowCostProbeSettings) {
	if settings == nil {
		return
	}
	settings.ProbeModel = strings.TrimSpace(settings.ProbeModel)
}

func validateOpenAILowCostProbeSettings(settings *OpenAILowCostProbeSettings) error {
	if settings == nil {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "settings are required")
	}
	if settings.MaxGroupRateMultiplier < 0 || math.IsNaN(settings.MaxGroupRateMultiplier) || math.IsInf(settings.MaxGroupRateMultiplier, 0) {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "max_group_rate_multiplier must be finite and non-negative")
	}
	if settings.MaxProbeLatencyMS < 100 || settings.MaxProbeLatencyMS > 120000 {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "max_probe_latency_ms must be between 100 and 120000")
	}
	if settings.HealthyProbeIntervalSeconds < 15 || settings.HealthyProbeIntervalSeconds > 86400 {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "healthy_probe_interval_seconds must be between 15 and 86400")
	}
	if settings.RecoveryProbeIntervalSeconds < 5 || settings.RecoveryProbeIntervalSeconds > 3600 {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "recovery_probe_interval_seconds must be between 5 and 3600")
	}
	if settings.FailureThreshold < 1 || settings.FailureThreshold > 10 {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "failure_threshold must be between 1 and 10")
	}
	if settings.RecoverySuccesses < 1 || settings.RecoverySuccesses > 10 {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "recovery_successes must be between 1 and 10")
	}
	if settings.ProbeModel == "" || len(settings.ProbeModel) > 128 {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "probe_model is required and must not exceed 128 characters")
	}
	if settings.MaxProbeSpendUSDPer24H < 0 || settings.MaxProbeSpendUSDPer24H > 1000 || math.IsNaN(settings.MaxProbeSpendUSDPer24H) || math.IsInf(settings.MaxProbeSpendUSDPer24H, 0) {
		return infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_SETTINGS", "max_probe_spend_usd_per_24h must be between 0 and 1000")
	}
	return nil
}

func NewOpenAILowCostProbeService(
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	stateRepo OpenAILowCostProbeStateRepository,
	settingRepo SettingRepository,
	accountTestService *AccountTestService,
	opsRepo OpsRepository,
) *OpenAILowCostProbeService {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &OpenAILowCostProbeService{
		accountRepo: accountRepo, groupRepo: groupRepo, stateRepo: stateRepo,
		settingRepo: settingRepo, accountTestService: accountTestService, opsRepo: opsRepo,
		instanceID: uuid.NewString(), now: time.Now, parentCtx: ctx, parentCancel: cancel,
	}
	if accountTestService != nil {
		svc.probeFn = accountTestService.RunOpenAILowCostProbe
	}
	return svc
}

func ProvideOpenAILowCostProbeService(
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	stateRepo OpenAILowCostProbeStateRepository,
	settingRepo SettingRepository,
	accountTestService *AccountTestService,
	opsRepo OpsRepository,
	lockCache LeaderLockCache,
	db *sql.DB,
	openAIGatewayService *OpenAIGatewayService,
) *OpenAILowCostProbeService {
	svc := NewOpenAILowCostProbeService(accountRepo, groupRepo, stateRepo, settingRepo, accountTestService, opsRepo)
	svc.SetLeaderLock(lockCache, db)
	if openAIGatewayService != nil {
		openAIGatewayService.SetOpenAILowCostProbeService(svc)
	}
	svc.Start()
	return svc
}

func (s *OpenAILowCostProbeService) SetLeaderLock(lockCache LeaderLockCache, db *sql.DB) {
	if s == nil {
		return
	}
	s.lockCache = lockCache
	s.db = db
}

func (s *OpenAILowCostProbeService) GetSettings(ctx context.Context) (*OpenAILowCostProbeSettings, error) {
	defaults := DefaultOpenAILowCostProbeSettings()
	if s == nil || s.settingRepo == nil {
		return defaults, nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAILowCostProbe)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaults, nil
		}
		return nil, err
	}
	if strings.TrimSpace(value) == "" {
		return defaults, nil
	}
	if err := json.Unmarshal([]byte(value), defaults); err != nil {
		return nil, fmt.Errorf("decode OpenAI low-cost probe settings: %w", err)
	}
	normalizeOpenAILowCostProbeSettings(defaults)
	if err := validateOpenAILowCostProbeSettings(defaults); err != nil {
		return nil, err
	}
	return defaults, nil
}

func (s *OpenAILowCostProbeService) UpdateSettings(ctx context.Context, settings *OpenAILowCostProbeSettings) (*OpenAILowCostProbeSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, ErrOpenAILowCostProbeUnavailable
	}
	if settings == nil {
		return nil, validateOpenAILowCostProbeSettings(nil)
	}
	copySettings := *settings
	normalizeOpenAILowCostProbeSettings(&copySettings)
	if err := validateOpenAILowCostProbeSettings(&copySettings); err != nil {
		return nil, err
	}
	data, err := json.Marshal(&copySettings)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOpenAILowCostProbe, string(data)); err != nil {
		return nil, err
	}
	s.invalidateSchedulingPolicyCache(0)
	return &copySettings, nil
}

// ApplySchedulingPolicy filters accounts whose probe circuit is open and
// decorates healthy accounts with a request-local priority tier. Repository
// failures are returned so the gateway can fail open without dropping traffic.
func (s *OpenAILowCostProbeService) ApplySchedulingPolicy(ctx context.Context, groupID *int64, platform string, accounts []Account) ([]Account, error) {
	if s == nil || groupID == nil || *groupID <= 0 || NormalizeOpenAICompatiblePlatform(platform) != PlatformOpenAI || len(accounts) == 0 {
		return accounts, nil
	}
	policy, err := s.loadSchedulingPolicy(ctx, *groupID)
	if err != nil || !policy.enabled || !policy.inScope {
		return accounts, err
	}
	now := s.currentTime()
	filtered := make([]Account, 0, len(accounts))
	for i := range accounts {
		account := accounts[i]
		status := policy.states[account.ID]
		if status == OpenAILowCostProbeStatusCircuitOpen || status == OpenAILowCostProbeStatusRecovering {
			continue
		}
		if status == OpenAILowCostProbeStatusHealthy {
			account.openAILowCostProbePreferred = true
			account.openAILowCostProbeRate = OpenAILowCostAccountRate(&account, now)
		}
		filtered = append(filtered, account)
	}
	return filtered, nil
}

// IsAccountSchedulingAllowed applies the same group-scoped circuit gate to
// sticky and direct paths that do not traverse a candidate list.
func (s *OpenAILowCostProbeService) IsAccountSchedulingAllowed(ctx context.Context, groupID *int64, platform string, accountID int64) (bool, error) {
	if s == nil || groupID == nil || *groupID <= 0 || accountID <= 0 || NormalizeOpenAICompatiblePlatform(platform) != PlatformOpenAI {
		return true, nil
	}
	policy, err := s.loadSchedulingPolicy(ctx, *groupID)
	if err != nil || !policy.enabled || !policy.inScope {
		return true, err
	}
	status := policy.states[accountID]
	return status != OpenAILowCostProbeStatusCircuitOpen && status != OpenAILowCostProbeStatusRecovering, nil
}

func (s *OpenAILowCostProbeService) loadSchedulingPolicy(ctx context.Context, groupID int64) (openAILowCostSchedulingPolicy, error) {
	now := s.currentTime()
	s.policyMu.Lock()
	if cached, ok := s.policyCache[groupID]; ok && now.Sub(cached.fetchedAt) < openAILowCostPolicyCacheTTL {
		s.policyMu.Unlock()
		return cached, nil
	}
	s.policyMu.Unlock()

	settings, err := s.GetSettings(ctx)
	if err != nil {
		return openAILowCostSchedulingPolicy{}, err
	}
	policy := openAILowCostSchedulingPolicy{fetchedAt: now, enabled: settings.Enabled}
	if !settings.Enabled {
		s.storeSchedulingPolicy(groupID, policy)
		return policy, nil
	}
	if s.groupRepo == nil || s.stateRepo == nil {
		return openAILowCostSchedulingPolicy{}, ErrOpenAILowCostProbeUnavailable
	}
	group, err := s.groupRepo.GetByIDLite(ctx, groupID)
	if err != nil {
		return openAILowCostSchedulingPolicy{}, err
	}
	policy.inScope = group != nil && group.Status == StatusActive && group.Platform == PlatformOpenAI && group.RateMultiplier <= settings.MaxGroupRateMultiplier
	if !policy.inScope {
		s.storeSchedulingPolicy(groupID, policy)
		return policy, nil
	}
	states, err := s.stateRepo.ListStatesByGroup(ctx, groupID)
	if err != nil {
		return openAILowCostSchedulingPolicy{}, err
	}
	policy.states = make(map[int64]string, len(states))
	for i := range states {
		policy.states[states[i].AccountID] = states[i].Status
	}
	s.storeSchedulingPolicy(groupID, policy)
	return policy, nil
}

func (s *OpenAILowCostProbeService) storeSchedulingPolicy(groupID int64, policy openAILowCostSchedulingPolicy) {
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	if s.policyCache == nil {
		s.policyCache = make(map[int64]openAILowCostSchedulingPolicy)
	}
	s.policyCache[groupID] = policy
}

func (s *OpenAILowCostProbeService) invalidateSchedulingPolicyCache(groupID int64) {
	if s == nil {
		return
	}
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	if groupID > 0 {
		delete(s.policyCache, groupID)
		return
	}
	clear(s.policyCache)
}

func (s *OpenAILowCostProbeService) GetStates(ctx context.Context) (*OpenAILowCostProbeStatesResponse, error) {
	if s == nil || s.stateRepo == nil {
		return nil, ErrOpenAILowCostProbeUnavailable
	}
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	states, err := s.stateRepo.ListStates(ctx)
	if err != nil {
		return nil, err
	}
	used, err := s.stateRepo.GetBudgetUsage(ctx, s.currentTime())
	if err != nil {
		return nil, err
	}
	var last *time.Time
	for i := range states {
		if states[i].LastProbeAt != nil && (last == nil || states[i].LastProbeAt.After(*last)) {
			value := *states[i].LastProbeAt
			last = &value
		}
	}
	remaining := math.Max(0, settings.MaxProbeSpendUSDPer24H-used)
	return &OpenAILowCostProbeStatesResponse{States: states, BudgetUsedUSD: used, BudgetRemainingUSD: remaining, EstimatedCallUSD: OpenAILowCostProbeEstimatedCostUSD, LastProbeAt: last}, nil
}

func (s *OpenAILowCostProbeService) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.started || s.stopped {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.wg.Add(1)
	s.mu.Unlock()
	go s.runLoop()
}

func (s *OpenAILowCostProbeService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.parentCancel()
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *OpenAILowCostProbeService) runLoop() {
	defer s.wg.Done()
	_ = s.RunDue(s.parentCtx)
	ticker := time.NewTicker(openAILowCostProbeCycleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.parentCtx.Done():
			return
		case <-ticker.C:
			if err := s.RunDue(s.parentCtx); err != nil {
				logger.LegacyPrintf("service.openai_low_cost_probe", "run_due_failed: err=%v", err)
			}
		}
	}
}

func (s *OpenAILowCostProbeService) RunDue(ctx context.Context) error {
	if s == nil || s.stateRepo == nil || s.accountRepo == nil || s.groupRepo == nil {
		return nil
	}
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()
	release, acquired := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, openAILowCostProbeLeaderLockKey, s.instanceID, openAILowCostProbeLeaderLockTTL)
	if !acquired {
		return nil
	}
	defer release()

	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}
	targets, states, err := s.collectTargets(ctx, settings)
	if err != nil {
		return err
	}
	if !settings.Enabled || settings.MaxProbeSpendUSDPer24H <= 0 {
		return nil
	}
	now := s.currentTime()
	due := make([]int64, 0, len(targets))
	for accountID, target := range targets {
		if targetDue(target, states, settings, now) {
			due = append(due, accountID)
		}
	}
	sort.Slice(due, func(i, j int) bool { return due[i] < due[j] })
	if len(due) > openAILowCostProbeMaxBatchSize {
		due = due[:openAILowCostProbeMaxBatchSize]
	}
	_, err = s.runTargets(ctx, settings, targets, states, due)
	return err
}

func (s *OpenAILowCostProbeService) RunManual(ctx context.Context, accountIDs []int64) (*OpenAILowCostProbeRunResult, error) {
	if s == nil || s.stateRepo == nil || s.accountRepo == nil || s.groupRepo == nil {
		return nil, ErrOpenAILowCostProbeUnavailable
	}
	if len(accountIDs) > openAILowCostProbeMaxBatchSize {
		return nil, infraerrors.BadRequest("INVALID_OPENAI_LOW_COST_PROBE_ACCOUNTS", "account_ids must not exceed 50 items")
	}
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()
	release, acquired := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, openAILowCostProbeLeaderLockKey, s.instanceID, openAILowCostProbeLeaderLockTTL)
	if !acquired {
		return &OpenAILowCostProbeRunResult{}, nil
	}
	defer release()
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	targets, states, err := s.collectTargets(ctx, settings)
	if err != nil {
		return nil, err
	}
	ids := normalizeProbeAccountIDs(accountIDs, targets)
	return s.runTargets(ctx, settings, targets, states, ids)
}

func (s *OpenAILowCostProbeService) collectTargets(ctx context.Context, settings *OpenAILowCostProbeSettings) (map[int64]*openAILowCostProbeTarget, map[OpenAILowCostProbeScope]*OpenAILowCostProbeState, error) {
	groups, err := s.groupRepo.ListActiveByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil, nil, err
	}
	targets := make(map[int64]*openAILowCostProbeTarget)
	scopes := make([]OpenAILowCostProbeScope, 0)
	now := s.currentTime()
	for i := range groups {
		group := groups[i]
		if group.RateMultiplier > settings.MaxGroupRateMultiplier {
			continue
		}
		accounts, listErr := s.accountRepo.ListByGroup(ctx, group.ID)
		if listErr != nil {
			return nil, nil, listErr
		}
		for j := range accounts {
			account := accounts[j]
			if !eligibleOpenAILowCostProbeAccount(&account, now) {
				continue
			}
			target := targets[account.ID]
			if target == nil {
				accountCopy := account
				target = &openAILowCostProbeTarget{account: &accountCopy, groups: make(map[int64]*Group)}
				targets[account.ID] = target
			}
			groupCopy := group
			target.groups[group.ID] = &groupCopy
			scopes = append(scopes, OpenAILowCostProbeScope{GroupID: group.ID, AccountID: account.ID})
		}
	}
	if err := s.stateRepo.DeleteOutsideScope(ctx, scopes); err != nil {
		return nil, nil, err
	}
	s.invalidateSchedulingPolicyCache(0)
	stored, err := s.stateRepo.ListStates(ctx)
	if err != nil {
		return nil, nil, err
	}
	states := make(map[OpenAILowCostProbeScope]*OpenAILowCostProbeState, len(stored))
	for i := range stored {
		state := stored[i]
		states[OpenAILowCostProbeScope{GroupID: state.GroupID, AccountID: state.AccountID}] = &state
	}
	return targets, states, nil
}

func eligibleOpenAILowCostProbeAccount(account *Account, now time.Time) bool {
	if account == nil || account.Platform != PlatformOpenAI || !account.IsOAuth() || account.Status != StatusActive || !account.Schedulable {
		return false
	}
	return !account.AutoPauseOnExpired || account.ExpiresAt == nil || now.Before(*account.ExpiresAt)
}

func targetDue(target *openAILowCostProbeTarget, states map[OpenAILowCostProbeScope]*OpenAILowCostProbeState, settings *OpenAILowCostProbeSettings, now time.Time) bool {
	for groupID := range target.groups {
		state := states[OpenAILowCostProbeScope{GroupID: groupID, AccountID: target.account.ID}]
		if state == nil || state.LastProbeAt == nil {
			return true
		}
		if state.DisabledUntil != nil {
			if !now.Before(*state.DisabledUntil) {
				return true
			}
			continue
		}
		interval := settings.HealthyProbeIntervalSeconds
		if state.Status == OpenAILowCostProbeStatusCircuitOpen || state.Status == OpenAILowCostProbeStatusRecovering {
			interval = settings.RecoveryProbeIntervalSeconds
		}
		if !now.Before(state.LastProbeAt.Add(time.Duration(interval) * time.Second)) {
			return true
		}
	}
	return false
}

func normalizeProbeAccountIDs(requested []int64, targets map[int64]*openAILowCostProbeTarget) []int64 {
	if len(requested) == 0 {
		ids := make([]int64, 0, len(targets))
		for id := range targets {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		if len(ids) > openAILowCostProbeMaxBatchSize {
			ids = ids[:openAILowCostProbeMaxBatchSize]
		}
		return ids
	}
	seen := make(map[int64]struct{}, len(requested))
	ids := make([]int64, 0, len(requested))
	for _, id := range requested {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func (s *OpenAILowCostProbeService) runTargets(ctx context.Context, settings *OpenAILowCostProbeSettings, targets map[int64]*openAILowCostProbeTarget, states map[OpenAILowCostProbeScope]*OpenAILowCostProbeState, accountIDs []int64) (*OpenAILowCostProbeRunResult, error) {
	result := &OpenAILowCostProbeRunResult{Items: make([]OpenAILowCostProbeRunItem, 0, len(accountIDs))}
	budgetEventWritten := false
	for _, accountID := range accountIDs {
		target := targets[accountID]
		if target == nil {
			result.Skipped++
			result.Items = append(result.Items, OpenAILowCostProbeRunItem{AccountID: accountID, Status: "skipped", Error: "account is not eligible for the configured low-cost groups"})
			continue
		}
		if settings.MaxProbeSpendUSDPer24H <= 0 || s.probeFn == nil {
			result.Skipped++
			result.Items = append(result.Items, OpenAILowCostProbeRunItem{AccountID: accountID, Status: "skipped", Error: "probe budget is disabled"})
			continue
		}
		now := s.currentTime()
		used, reserved, err := s.stateRepo.ReserveBudget(ctx, now, accountID, OpenAILowCostProbeEstimatedCostUSD, settings.MaxProbeSpendUSDPer24H)
		if err != nil {
			return nil, err
		}
		if !reserved {
			result.Skipped++
			result.Items = append(result.Items, OpenAILowCostProbeRunItem{AccountID: accountID, Status: "budget_exhausted", Error: "24-hour probe budget exhausted"})
			if !budgetEventWritten {
				s.emitBudgetEvent(ctx, settings, used, now)
				budgetEventWritten = true
			}
			continue
		}
		timeout := time.Duration(settings.MaxProbeLatencyMS+2000) * time.Millisecond
		if timeout < 3*time.Second {
			timeout = 3 * time.Second
		}
		if timeout > 30*time.Second {
			timeout = 30 * time.Second
		}
		probeCtx, cancel := context.WithTimeout(ctx, timeout)
		probeResult, probeErr := s.probeFn(probeCtx, accountID, settings.ProbeModel)
		cancel()
		result.Executed++
		success, latency, errorSummary := classifyOpenAILowCostProbe(probeResult, probeErr, settings.MaxProbeLatencyMS)
		if success {
			result.Succeeded++
			result.Items = append(result.Items, OpenAILowCostProbeRunItem{AccountID: accountID, Status: OpenAILowCostProbeStatusHealthy, LatencyMS: latency})
		} else {
			result.Failed++
			result.Items = append(result.Items, OpenAILowCostProbeRunItem{AccountID: accountID, Status: OpenAILowCostProbeStatusCircuitOpen, Error: errorSummary, LatencyMS: latency})
		}
		for groupID, group := range target.groups {
			key := OpenAILowCostProbeScope{GroupID: groupID, AccountID: accountID}
			previous := states[key]
			state := nextOpenAILowCostProbeState(previous, settings, now, success, latency, errorSummary)
			state.GroupID = groupID
			state.AccountID = accountID
			state.EstimatedSpendUSD += OpenAILowCostProbeEstimatedCostUSD
			state.GroupName = group.Name
			state.AccountName = target.account.Name
			state.GroupRateMultiplier = group.RateMultiplier
			state.AccountRateMultiplier = OpenAILowCostAccountRate(target.account, now)
			if err := s.stateRepo.UpsertState(ctx, state); err != nil {
				return nil, err
			}
			s.invalidateSchedulingPolicyCache(group.ID)
			states[key] = state
			s.emitStateEvent(ctx, settings, target.account, group, previous, state, now)
		}
	}
	used, err := s.stateRepo.GetBudgetUsage(ctx, s.currentTime())
	if err != nil {
		return nil, err
	}
	result.BudgetUsedUSD = used
	result.BudgetRemainingUSD = math.Max(0, settings.MaxProbeSpendUSDPer24H-used)
	return result, nil
}

func classifyOpenAILowCostProbe(result *ScheduledTestResult, err error, maxLatencyMS int64) (bool, int64, string) {
	if err != nil {
		return false, 0, safeOpenAILowCostProbeError(err.Error())
	}
	if result == nil {
		return false, 0, "probe returned no result"
	}
	if result.Status != "success" {
		message := result.ErrorMessage
		if strings.TrimSpace(message) == "" {
			message = "upstream probe failed"
		}
		return false, result.LatencyMs, safeOpenAILowCostProbeError(message)
	}
	if result.LatencyMs > maxLatencyMS {
		return false, result.LatencyMs, fmt.Sprintf("probe latency %dms exceeds limit %dms", result.LatencyMs, maxLatencyMS)
	}
	return true, result.LatencyMs, ""
}

func nextOpenAILowCostProbeState(previous *OpenAILowCostProbeState, settings *OpenAILowCostProbeSettings, now time.Time, success bool, latency int64, errorSummary string) *OpenAILowCostProbeState {
	state := &OpenAILowCostProbeState{Status: OpenAILowCostProbeStatusUnknown, LastProbeAt: lowCostProbeTimePtr(now), LatencyMS: lowCostProbeInt64Ptr(latency), UpdatedAt: now}
	if previous != nil {
		*state = *previous
		state.LastProbeAt = lowCostProbeTimePtr(now)
		state.LatencyMS = lowCostProbeInt64Ptr(latency)
		state.UpdatedAt = now
	}
	if success {
		state.LastSuccessAt = lowCostProbeTimePtr(now)
		state.LastError = ""
		state.ConsecutiveFailures = 0
		state.ConsecutiveSuccesses++
		if previous != nil && (previous.Status == OpenAILowCostProbeStatusCircuitOpen || previous.Status == OpenAILowCostProbeStatusRecovering) && state.ConsecutiveSuccesses < settings.RecoverySuccesses {
			state.Status = OpenAILowCostProbeStatusRecovering
			next := now.Add(time.Duration(settings.RecoveryProbeIntervalSeconds) * time.Second)
			state.DisabledUntil = &next
		} else {
			state.Status = OpenAILowCostProbeStatusHealthy
			state.DisabledUntil = nil
		}
		return state
	}
	state.LastError = errorSummary
	state.ConsecutiveFailures++
	state.ConsecutiveSuccesses = 0
	if state.ConsecutiveFailures >= settings.FailureThreshold || (previous != nil && (previous.Status == OpenAILowCostProbeStatusCircuitOpen || previous.Status == OpenAILowCostProbeStatusRecovering)) {
		state.Status = OpenAILowCostProbeStatusCircuitOpen
		next := now.Add(time.Duration(settings.RecoveryProbeIntervalSeconds) * time.Second)
		state.DisabledUntil = &next
	} else {
		state.Status = OpenAILowCostProbeStatusDegraded
		state.DisabledUntil = nil
	}
	return state
}

func OpenAILowCostAccountRate(account *Account, now time.Time) float64 {
	if account == nil {
		return 1
	}
	if snapshot := decodeUpstreamBillingProbeSnapshot(account.Extra); snapshot != nil && snapshot.Status == UpstreamBillingProbeStatusOK && snapshot.FreshUntil != nil && now.Before(*snapshot.FreshUntil) {
		if rate, ok := upstreamBillingRateAt(snapshot.Data, now); ok {
			return rate
		}
	}
	return account.BillingRateMultiplier()
}

func compareOpenAILowCostProbePreference(left, right *Account) int {
	if left == nil || right == nil {
		return 0
	}
	if left.openAILowCostProbePreferred != right.openAILowCostProbePreferred {
		if left.openAILowCostProbePreferred {
			return -1
		}
		return 1
	}
	if !left.openAILowCostProbePreferred {
		return 0
	}
	if left.openAILowCostProbeRate < right.openAILowCostProbeRate {
		return -1
	}
	if left.openAILowCostProbeRate > right.openAILowCostProbeRate {
		return 1
	}
	return 0
}

func copyOpenAILowCostProbePreference(destination, source *Account) {
	if destination == nil || source == nil {
		return
	}
	destination.openAILowCostProbePreferred = source.openAILowCostProbePreferred
	destination.openAILowCostProbeRate = source.openAILowCostProbeRate
}

func (s *OpenAILowCostProbeService) emitStateEvent(ctx context.Context, settings *OpenAILowCostProbeSettings, account *Account, group *Group, previous, state *OpenAILowCostProbeState, now time.Time) {
	if s.opsRepo == nil || state == nil || account == nil || group == nil {
		return
	}
	recovery := state.Status == OpenAILowCostProbeStatusHealthy && previous != nil && (previous.Status == OpenAILowCostProbeStatusCircuitOpen || previous.Status == OpenAILowCostProbeStatusRecovering)
	failed := state.LastError != ""
	if !failed && !recovery {
		return
	}
	status := OpsAlertStatusFiring
	title := fmt.Sprintf("OpenAI低价账号探活失败：%s", account.Name)
	description := state.LastError
	if recovery {
		status = OpsAlertStatusResolved
		title = fmt.Sprintf("OpenAI低价账号探活恢复：%s", account.Name)
		description = "账号已通过连续恢复探活并重新参与调度"
	}
	dimensions := map[string]any{
		"event_type": "openai_low_cost_probe", "group_id": group.ID, "group_name": group.Name,
		"account_id": account.ID, "account_name": account.Name,
		"account_rate_multiplier": OpenAILowCostAccountRate(account, now), "group_rate_multiplier": group.RateMultiplier,
		"status": state.Status, "latency_ms": state.LatencyMS, "error_summary": state.LastError,
		"probe_model": settings.ProbeModel, "checked_at": state.LastProbeAt,
	}
	event := &OpsAlertEvent{Severity: "warning", Status: status, Title: title, Description: description, Dimensions: dimensions, FiredAt: now.UTC()}
	if recovery {
		resolvedAt := now.UTC()
		event.ResolvedAt = &resolvedAt
	}
	_, _ = s.opsRepo.CreateAlertEvent(ctx, event)
}

func (s *OpenAILowCostProbeService) emitBudgetEvent(ctx context.Context, settings *OpenAILowCostProbeSettings, used float64, now time.Time) {
	if s.opsRepo == nil {
		return
	}
	_, _ = s.opsRepo.CreateAlertEvent(ctx, &OpsAlertEvent{
		Severity: "warning", Status: OpsAlertStatusFiring, Title: "OpenAI低价账号探活预算已耗尽",
		Description: "24小时探活预算已达到上限，真实上游探活已暂停",
		Dimensions:  map[string]any{"event_type": "openai_low_cost_probe_budget", "used_usd": used, "limit_usd": settings.MaxProbeSpendUSDPer24H, "checked_at": now.UTC()},
		FiredAt:     now.UTC(),
	})
}

func safeOpenAILowCostProbeError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > openAILowCostProbeMaxErrorBytes {
		value = value[:openAILowCostProbeMaxErrorBytes]
	}
	return value
}

func (s *OpenAILowCostProbeService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func lowCostProbeTimePtr(value time.Time) *time.Time { return &value }
func lowCostProbeInt64Ptr(value int64) *int64        { return &value }
