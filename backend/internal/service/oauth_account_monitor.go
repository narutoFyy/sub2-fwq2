package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/google/uuid"
)

const (
	SettingKeyOAuthAccountMonitorConfig = "oauth_account_monitor_config"
	SettingKeyOAuthAccountMonitorPush   = "oauth_account_monitor_pushplus"
	SettingKeyOAuthAccountMonitorEmail  = "oauth_account_monitor_email"
	oauthAccountMonitorLeaderKey        = "oauth-account-monitor:leader"
	oauthAccountMonitorLockTTL          = 10 * time.Minute
	oauthAccountMonitorQuotaTimeout     = 20 * time.Second
	oauthAccountMonitorConcurrency      = 4
	defaultOAuthMonitorIntervalMinutes  = 5
	oauthAccountOutcomeQueueSize        = 1024
	oauthAccountOutcomeWorkers          = 2
)

type OAuthAccountMonitorConfig struct {
	Enabled                     bool    `json:"enabled"`
	AccountIDs                  []int64 `json:"account_ids"`
	QuotaThreshold              float64 `json:"quota_threshold_percent"`
	IntervalMinutes             int     `json:"interval_minutes"`
	NotifyOnError               bool    `json:"notify_on_error"`
	NotifyOnQuota               bool    `json:"notify_on_quota"`
	NotifyOnRecovery            bool    `json:"notify_on_recovery"`
	NotifyEmail                 bool    `json:"notify_email"`
	NotifyPushPlus              bool    `json:"notify_pushplus"`
	RepeatEveryCheck            bool    `json:"repeat_every_check"`
	NotifyOnUnavailable         bool    `json:"notify_on_unavailable"`
	UnavailableFailureThreshold int     `json:"unavailable_failure_threshold"`
	UnavailableWindowMinutes    int     `json:"unavailable_window_minutes"`
	UnavailableReminderMinutes  int     `json:"unavailable_reminder_minutes"`
}

type OAuthMonitorEmailConfig struct {
	Enabled    bool     `json:"enabled"`
	Recipients []string `json:"recipients"`
}

type OAuthMonitorPushPlusConfig struct {
	Enabled  bool   `json:"enabled"`
	Token    string `json:"token"`
	Topic    string `json:"topic,omitempty"`
	Template string `json:"template,omitempty"`
	Channel  string `json:"channel,omitempty"`
}

type OAuthAccountMonitorState struct {
	AccountID                   int64      `json:"account_id"`
	LastCheckedAt               *time.Time `json:"last_checked_at,omitempty"`
	HealthStatus                string     `json:"health_status"`
	ErrorMessage                string     `json:"error_message,omitempty"`
	PrimaryRemainingPercent     *float64   `json:"primary_remaining_percent,omitempty"`
	SecondaryRemainingPercent   *float64   `json:"secondary_remaining_percent,omitempty"`
	QuotaStatus                 string     `json:"quota_status"`
	LastConditionKey            string     `json:"last_condition_key,omitempty"`
	LastNotifiedAt              *time.Time `json:"last_notified_at,omitempty"`
	ConsecutiveFailures         int        `json:"consecutive_failures"`
	LastSuccessAt               *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt               *time.Time `json:"last_failure_at,omitempty"`
	LastErrorCode               string     `json:"last_error_code,omitempty"`
	FailureStartedAt            *time.Time `json:"failure_started_at,omitempty"`
	LastUnavailableNotifiedAt   *time.Time `json:"last_unavailable_notified_at,omitempty"`
	FailureSequence             int64      `json:"failure_sequence"`
	LastNotifiedFailureSequence int64      `json:"last_notified_failure_sequence"`
	QuotaAlertActive            bool       `json:"quota_alert_active"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

type OAuthAccountMonitorAccountOverview struct {
	AccountID     int64                     `json:"account_id"`
	AccountName   string                    `json:"account_name"`
	AccountStatus string                    `json:"account_status"`
	Schedulable   bool                      `json:"schedulable"`
	Selected      bool                      `json:"selected"`
	Eligible      bool                      `json:"eligible"`
	Monitored     bool                      `json:"monitored"`
	Effective     bool                      `json:"effective"`
	MonitorStatus string                    `json:"monitor_status"`
	State         *OAuthAccountMonitorState `json:"state,omitempty"`
}

type OAuthAccountMonitorOverview struct {
	Config   *OAuthAccountMonitorConfig            `json:"config"`
	PushPlus *OAuthMonitorPushPlusConfig           `json:"pushplus"`
	Email    *OAuthMonitorEmailConfig              `json:"email"`
	Accounts []*OAuthAccountMonitorAccountOverview `json:"accounts"`
}

type OAuthAccountMonitorRunResult struct {
	Executed        bool   `json:"executed"`
	CheckedAccounts int    `json:"checked_accounts"`
	SkippedReason   string `json:"skipped_reason,omitempty"`
}

type OAuthAccountAvailabilityState struct {
	ConsecutiveFailures         int
	LastSuccessAt               *time.Time
	LastFailureAt               *time.Time
	LastErrorCode               string
	LastError                   string
	FailureStartedAt            *time.Time
	LastUnavailableNotifiedAt   *time.Time
	FailureSequence             int64
	LastNotifiedFailureSequence int64
}

type OAuthAccountRequestOutcome struct {
	AccountID  int64
	RequestID  string
	Success    bool
	ErrorCode  string
	Error      string
	OccurredAt time.Time
	Window     time.Duration
}

type OAuthAccountRequestOutcomeRecorder interface {
	RecordRequestOutcome(ctx context.Context, account *Account, requestID string, success bool, observedErr error)
}

type OAuthAccountMonitorOutcomeStore interface {
	RecordOutcome(ctx context.Context, outcome OAuthAccountRequestOutcome) (*OAuthAccountAvailabilityState, error)
	GetAvailabilityState(ctx context.Context, accountID int64) (*OAuthAccountAvailabilityState, error)
}

type OAuthAccountMonitorStateRepository interface {
	GetStates(ctx context.Context, accountIDs []int64) (map[int64]*OAuthAccountMonitorState, error)
	UpsertState(ctx context.Context, state *OAuthAccountMonitorState) error
}

type OAuthAccountMonitorQuotaQuerier interface {
	QueryUsage(ctx context.Context, accountID int64) (*OpenAIQuotaUsage, error)
}

type OAuthAccountMonitorService struct {
	accountRepo    AccountRepository
	stateRepo      OAuthAccountMonitorStateRepository
	settingRepo    SettingRepository
	quotaService   OAuthAccountMonitorQuotaQuerier
	opsRepo        OpsRepository
	opsService     *OpsService
	emailService   *EmailService
	outcomeStore   OAuthAccountMonitorOutcomeStore
	lockCache      LeaderLockCache
	db             *sql.DB
	parentCtx      context.Context
	parentCancel   context.CancelFunc
	instanceID     string
	wg             sync.WaitGroup
	mu             sync.Mutex
	started        bool
	stopped        bool
	httpClient     *http.Client
	outcomeQueue   chan OAuthAccountRequestOutcome
	configSnapshot atomic.Pointer[OAuthAccountMonitorConfig]
}

// ProvideOAuthAccountMonitorService starts the process-wide OAuth account monitor.
func ProvideOAuthAccountMonitorService(
	accountRepo AccountRepository,
	stateRepo OAuthAccountMonitorStateRepository,
	settingRepo SettingRepository,
	quotaService OAuthAccountMonitorQuotaQuerier,
	opsRepo OpsRepository,
	opsService *OpsService,
	emailService *EmailService,
	outcomeStore OAuthAccountMonitorOutcomeStore,
	lockCache LeaderLockCache,
	db *sql.DB,
	openAIGatewayService *OpenAIGatewayService,
) *OAuthAccountMonitorService {
	svc := NewOAuthAccountMonitorService(accountRepo, stateRepo, settingRepo, quotaService, opsRepo, opsService, emailService)
	svc.outcomeStore = outcomeStore
	svc.SetLeaderLock(lockCache, db)
	if openAIGatewayService != nil {
		openAIGatewayService.SetOAuthAccountRequestOutcomeRecorder(svc)
	}
	svc.Start()
	return svc
}

func NewOAuthAccountMonitorService(accountRepo AccountRepository, stateRepo OAuthAccountMonitorStateRepository, settingRepo SettingRepository, quotaService OAuthAccountMonitorQuotaQuerier, opsRepo OpsRepository, opsService *OpsService, emailService *EmailService) *OAuthAccountMonitorService {
	ctx, cancel := context.WithCancel(context.Background())
	return &OAuthAccountMonitorService{
		accountRepo: accountRepo, stateRepo: stateRepo, settingRepo: settingRepo, quotaService: quotaService,
		opsRepo: opsRepo, opsService: opsService, emailService: emailService,
		parentCtx: ctx, parentCancel: cancel, instanceID: uuid.NewString(),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		outcomeQueue: make(chan OAuthAccountRequestOutcome, oauthAccountOutcomeQueueSize),
	}
}

func (s *OAuthAccountMonitorService) SetLeaderLock(lockCache LeaderLockCache, db *sql.DB) {
	if s != nil {
		s.lockCache = lockCache
		s.db = db
	}
}

func defaultOAuthAccountMonitorConfig() *OAuthAccountMonitorConfig {
	return &OAuthAccountMonitorConfig{
		IntervalMinutes:             defaultOAuthMonitorIntervalMinutes,
		QuotaThreshold:              20,
		NotifyOnQuota:               true,
		NotifyOnUnavailable:         true,
		UnavailableFailureThreshold: 3,
		UnavailableWindowMinutes:    15,
		UnavailableReminderMinutes:  60,
		NotifyEmail:                 true,
		NotifyPushPlus:              true,
	}
}

func normalizeOAuthMonitorConfig(cfg *OAuthAccountMonitorConfig) {
	defaults := defaultOAuthAccountMonitorConfig()
	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = defaults.IntervalMinutes
	}
	if cfg.QuotaThreshold < 0 {
		cfg.QuotaThreshold = 0
	}
	if cfg.QuotaThreshold > 100 {
		cfg.QuotaThreshold = 100
	}
	if cfg.UnavailableFailureThreshold <= 0 {
		cfg.UnavailableFailureThreshold = defaults.UnavailableFailureThreshold
	}
	if cfg.UnavailableWindowMinutes <= 0 {
		cfg.UnavailableWindowMinutes = defaults.UnavailableWindowMinutes
	}
	if cfg.UnavailableReminderMinutes <= 0 {
		cfg.UnavailableReminderMinutes = defaults.UnavailableReminderMinutes
	}
	// Kept in the wire format for older clients, but these noisy policies are retired.
	cfg.NotifyOnError = false
	cfg.NotifyOnRecovery = false
	cfg.RepeatEveryCheck = false
	seen := make(map[int64]struct{}, len(cfg.AccountIDs))
	ids := cfg.AccountIDs[:0]
	for _, id := range cfg.AccountIDs {
		if id > 0 {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	cfg.AccountIDs = ids
}

func (s *OAuthAccountMonitorService) GetConfig(ctx context.Context) (*OAuthAccountMonitorConfig, error) {
	cfg := defaultOAuthAccountMonitorConfig()
	if s == nil || s.settingRepo == nil {
		if s != nil {
			s.storeConfigSnapshot(cfg)
		}
		return cfg, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOAuthAccountMonitorConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			s.storeConfigSnapshot(cfg)
			return cfg, nil
		}
		return nil, err
	}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, fmt.Errorf("parse oauth monitor config: %w", err)
		}
	}
	normalizeOAuthMonitorConfig(cfg)
	s.storeConfigSnapshot(cfg)
	return cfg, nil
}

func (s *OAuthAccountMonitorService) UpdateConfig(ctx context.Context, cfg *OAuthAccountMonitorConfig) (*OAuthAccountMonitorConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.QuotaThreshold < 0 || cfg.QuotaThreshold > 100 {
		return nil, fmt.Errorf("quota_threshold_percent must be between 0 and 100")
	}
	if cfg.IntervalMinutes < 1 || cfg.IntervalMinutes > 1440 {
		return nil, fmt.Errorf("interval_minutes must be between 1 and 1440")
	}
	copyCfg := *cfg
	normalizeOAuthMonitorConfig(&copyCfg)
	if copyCfg.UnavailableFailureThreshold < 1 || copyCfg.UnavailableFailureThreshold > 100 {
		return nil, fmt.Errorf("unavailable_failure_threshold must be between 1 and 100")
	}
	if copyCfg.UnavailableWindowMinutes < 1 || copyCfg.UnavailableWindowMinutes > 1440 {
		return nil, fmt.Errorf("unavailable_window_minutes must be between 1 and 1440")
	}
	if copyCfg.UnavailableReminderMinutes < 1 || copyCfg.UnavailableReminderMinutes > 10080 {
		return nil, fmt.Errorf("unavailable_reminder_minutes must be between 1 and 10080")
	}
	if s.accountRepo != nil && len(copyCfg.AccountIDs) > 0 {
		accounts, err := s.accountRepo.GetByIDs(ctx, copyCfg.AccountIDs)
		if err != nil {
			return nil, err
		}
		found := make(map[int64]*Account, len(accounts))
		for _, a := range accounts {
			if a != nil {
				found[a.ID] = a
			}
		}
		for _, id := range copyCfg.AccountIDs {
			a := found[id]
			if a == nil {
				return nil, fmt.Errorf("account %d not found", id)
			}
			if !a.IsOpenAIOAuth() {
				return nil, fmt.Errorf("account %d is not an OpenAI OAuth account", id)
			}
		}
	}
	if s.settingRepo == nil {
		return &copyCfg, nil
	}
	raw, err := json.Marshal(&copyCfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOAuthAccountMonitorConfig, string(raw)); err != nil {
		return nil, err
	}
	s.storeConfigSnapshot(&copyCfg)
	return &copyCfg, nil
}

func (s *OAuthAccountMonitorService) storeConfigSnapshot(cfg *OAuthAccountMonitorConfig) {
	if s == nil || cfg == nil {
		return
	}
	copyCfg := *cfg
	copyCfg.AccountIDs = append([]int64(nil), cfg.AccountIDs...)
	s.configSnapshot.Store(&copyCfg)
}

func (s *OAuthAccountMonitorService) RecordRequestOutcome(ctx context.Context, account *Account, requestID string, success bool, observedErr error) {
	if s == nil || account == nil || !account.IsOpenAIOAuth() {
		return
	}
	cfg := s.configSnapshot.Load()
	if cfg == nil || !cfg.Enabled || !cfg.NotifyOnUnavailable || !hasID(idSet(cfg.AccountIDs), account.ID) {
		return
	}
	errorCode, errorMessage, countFailure := classifyOAuthAccountRequestFailure(observedErr)
	if !success && !countFailure {
		return
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" && ctx != nil {
		requestID, _ = ctx.Value(ctxkey.RequestID).(string)
	}
	if requestID == "" {
		return
	}
	outcome := OAuthAccountRequestOutcome{
		AccountID:  account.ID,
		RequestID:  requestID,
		Success:    success,
		ErrorCode:  errorCode,
		Error:      truncateMonitorError(errorMessage, 240),
		OccurredAt: time.Now().UTC(),
		Window:     time.Duration(cfg.UnavailableWindowMinutes) * time.Minute,
	}
	select {
	case s.outcomeQueue <- outcome:
	default:
		slog.Warn("oauth_account_monitor_outcome_queue_full", "account_id", account.ID)
	}
}

func classifyOAuthAccountRequestFailure(err error) (code, message string, count bool) {
	if err == nil || errors.Is(err, context.Canceled) {
		return "", "", false
	}
	var failoverErr *UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		if failoverErr.RequestScopedTransient || failoverErr.Scope == GatewayFailureScopeRequest || failoverErr.Scope == GatewayFailureScopeProvider {
			return "", "", false
		}
		status := failoverErr.StatusCode
		if status == http.StatusUnauthorized || status == http.StatusPaymentRequired || status == http.StatusForbidden || status == http.StatusTooManyRequests || status >= 500 || status == 0 {
			return fmt.Sprintf("%d", status), failoverErr.Error(), true
		}
		return "", "", false
	}
	var imageErr *OpenAIImagesUpstreamError
	if errors.As(err, &imageErr) {
		status := imageErr.StatusCode
		if status == http.StatusUnauthorized || status == http.StatusPaymentRequired || status == http.StatusForbidden || status == http.StatusTooManyRequests || status >= 500 || status == 0 {
			return fmt.Sprintf("%d", status), imageErr.Error(), true
		}
		return "", "", false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", err.Error(), true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return "network", err.Error(), true
	}
	lower := strings.ToLower(err.Error())
	for _, marker := range []string{"stream disconnected", "unexpected eof", "connection reset", "broken pipe", "upstream request failed"} {
		if strings.Contains(lower, marker) {
			return "stream", err.Error(), true
		}
	}
	return "", "", false
}

func (s *OAuthAccountMonitorService) GetEmailConfig(ctx context.Context) (*OAuthMonitorEmailConfig, error) {
	cfg := &OAuthMonitorEmailConfig{Recipients: []string{}}
	if s == nil || s.settingRepo == nil {
		return cfg, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOAuthAccountMonitorEmail)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return cfg, nil
		}
		return nil, err
	}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, fmt.Errorf("parse oauth monitor email config: %w", err)
		}
	}
	cfg.Recipients = normalizeOAuthMonitorEmailRecipients(cfg.Recipients)
	return cfg, nil
}

func (s *OAuthAccountMonitorService) UpdateEmailConfig(ctx context.Context, cfg *OAuthMonitorEmailConfig) (*OAuthMonitorEmailConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("email config is required")
	}
	copyCfg := *cfg
	copyCfg.Recipients = normalizeOAuthMonitorEmailRecipients(copyCfg.Recipients)
	for _, recipient := range copyCfg.Recipients {
		address, err := mail.ParseAddress(recipient)
		if err != nil || !strings.EqualFold(address.Address, recipient) {
			return nil, fmt.Errorf("invalid email recipient %q", recipient)
		}
	}
	if copyCfg.Enabled && len(copyCfg.Recipients) == 0 {
		return nil, fmt.Errorf("at least one email recipient is required when enabled")
	}
	if s == nil || s.settingRepo == nil {
		return &copyCfg, nil
	}
	raw, err := json.Marshal(&copyCfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOAuthAccountMonitorEmail, string(raw)); err != nil {
		return nil, err
	}
	return &copyCfg, nil
}

func normalizeOAuthMonitorEmailRecipients(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *OAuthAccountMonitorService) GetPushPlusConfig(ctx context.Context) (*OAuthMonitorPushPlusConfig, error) {
	cfg := &OAuthMonitorPushPlusConfig{}
	if s == nil || s.settingRepo == nil {
		return cfg, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOAuthAccountMonitorPush)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return cfg, nil
		}
		return nil, err
	}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func (s *OAuthAccountMonitorService) UpdatePushPlusConfig(ctx context.Context, cfg *OAuthMonitorPushPlusConfig) (*OAuthMonitorPushPlusConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("pushplus config is required")
	}
	copyCfg := *cfg
	copyCfg.Token = strings.TrimSpace(copyCfg.Token)
	if copyCfg.Enabled && copyCfg.Token == "" {
		return nil, fmt.Errorf("pushplus token is required when enabled")
	}
	if s.settingRepo == nil {
		return &copyCfg, nil
	}
	raw, err := json.Marshal(&copyCfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOAuthAccountMonitorPush, string(raw)); err != nil {
		return nil, err
	}
	return &copyCfg, nil
}

func (s *OAuthAccountMonitorService) GetStates(ctx context.Context, ids []int64) (map[int64]*OAuthAccountMonitorState, error) {
	if s == nil || s.stateRepo == nil {
		return map[int64]*OAuthAccountMonitorState{}, nil
	}
	return s.stateRepo.GetStates(ctx, ids)
}

func (s *OAuthAccountMonitorService) GetOverview(ctx context.Context, selectedIDs []int64) (*OAuthAccountMonitorOverview, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	push, err := s.GetPushPlusConfig(ctx)
	if err != nil {
		return nil, err
	}

	email, err := s.GetEmailConfig(ctx)
	if err != nil {
		return nil, err
	}

	selectedIDs = uniquePositiveIDs(selectedIDs)
	allIDs := uniquePositiveIDs(append(append([]int64(nil), selectedIDs...), cfg.AccountIDs...))
	overview := &OAuthAccountMonitorOverview{
		Config:   cfg,
		PushPlus: push,
		Email:    email,
		Accounts: make([]*OAuthAccountMonitorAccountOverview, 0, len(allIDs)),
	}
	if len(allIDs) == 0 || s == nil || s.accountRepo == nil {
		return overview, nil
	}

	accounts, err := s.accountRepo.GetByIDs(ctx, allIDs)
	if err != nil {
		return nil, err
	}
	accountByID := make(map[int64]*Account, len(accounts))
	stateIDs := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		accountByID[account.ID] = account
		stateIDs = append(stateIDs, account.ID)
	}
	states, err := s.GetStates(ctx, stateIDs)
	if err != nil {
		return nil, err
	}

	selectedSet := idSet(selectedIDs)
	monitoredSet := idSet(cfg.AccountIDs)
	for _, id := range allIDs {
		account := accountByID[id]
		if account == nil {
			continue
		}
		state := states[id]
		eligible := account.IsOpenAIOAuth()
		monitored := hasID(monitoredSet, id)
		effective := cfg.Enabled && eligible && monitored
		overview.Accounts = append(overview.Accounts, &OAuthAccountMonitorAccountOverview{
			AccountID:     id,
			AccountName:   account.Name,
			AccountStatus: account.Status,
			Schedulable:   account.Schedulable,
			Selected:      hasID(selectedSet, id),
			Eligible:      eligible,
			Monitored:     monitored,
			Effective:     effective,
			MonitorStatus: oauthMonitorOverviewStatus(cfg.Enabled, eligible, monitored, state),
			State:         state,
		})
	}
	return overview, nil
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func idSet(ids []int64) map[int64]struct{} {
	result := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result
}

func hasID(ids map[int64]struct{}, id int64) bool {
	_, ok := ids[id]
	return ok
}

func oauthMonitorOverviewStatus(globalEnabled, eligible, monitored bool, state *OAuthAccountMonitorState) string {
	if !eligible {
		return "ineligible"
	}
	if !monitored {
		return "not_monitored"
	}
	if !globalEnabled {
		return "paused"
	}
	if state == nil || state.LastCheckedAt == nil {
		return "pending"
	}
	if strings.Contains(state.LastConditionKey, "quota_low:") {
		return "quota_low"
	}
	if state.HealthStatus == "error" || state.LastConditionKey != "" {
		return "error"
	}
	return "healthy"
}

func (s *OAuthAccountMonitorService) AddAccounts(ctx context.Context, ids []int64) (*OAuthAccountMonitorConfig, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	cfg.AccountIDs = append(append([]int64(nil), cfg.AccountIDs...), ids...)
	return s.UpdateConfig(ctx, cfg)
}

func (s *OAuthAccountMonitorService) RemoveAccounts(ctx context.Context, ids []int64) (*OAuthAccountMonitorConfig, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	remove := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		remove[id] = struct{}{}
	}
	kept := make([]int64, 0, len(cfg.AccountIDs))
	for _, id := range cfg.AccountIDs {
		if _, ok := remove[id]; !ok {
			kept = append(kept, id)
		}
	}
	cfg.AccountIDs = kept
	return s.UpdateConfig(ctx, cfg)
}

func (s *OAuthAccountMonitorService) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.started || s.stopped {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.wg.Add(1 + oauthAccountOutcomeWorkers)
	s.mu.Unlock()
	go s.runLoop()
	for i := 0; i < oauthAccountOutcomeWorkers; i++ {
		go s.runOutcomeWorker()
	}
}

func (s *OAuthAccountMonitorService) runOutcomeWorker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.parentCtx.Done():
			return
		case outcome := <-s.outcomeQueue:
			if s.outcomeStore == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := s.outcomeStore.RecordOutcome(ctx, outcome)
			cancel()
			if err != nil {
				slog.Warn("oauth_account_monitor_outcome_store_failed", "account_id", outcome.AccountID, "error", err)
			}
		}
	}
}

func (s *OAuthAccountMonitorService) Stop() {
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

func (s *OAuthAccountMonitorService) runLoop() {
	defer s.wg.Done()
	lastRun := time.Time{}
	result, err := s.RunCycle(s.parentCtx)
	if result != nil && result.Executed {
		lastRun = time.Now()
	}
	if err != nil {
		slog.Warn("oauth_account_monitor_cycle_failed", "error", err)
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.parentCtx.Done():
			return
		case <-ticker.C:
			cfg, err := s.GetConfig(s.parentCtx)
			if err != nil {
				slog.Warn("oauth_account_monitor_config_load_failed", "error", err)
				continue
			}
			if !cfg.Enabled || len(cfg.AccountIDs) == 0 {
				continue
			}
			interval := time.Duration(cfg.IntervalMinutes) * time.Minute
			if !lastRun.IsZero() && time.Since(lastRun) < interval {
				continue
			}
			result, runErr := s.RunCycle(s.parentCtx)
			if result != nil && result.Executed {
				lastRun = time.Now()
			}
			if runErr != nil {
				slog.Warn("oauth_account_monitor_cycle_failed", "error", runErr)
			}
		}
	}
}

func (s *OAuthAccountMonitorService) RunOnce(ctx context.Context) error {
	_, err := s.RunCycle(ctx)
	return err
}

func (s *OAuthAccountMonitorService) RunCycle(ctx context.Context) (*OAuthAccountMonitorRunResult, error) {
	result := &OAuthAccountMonitorRunResult{}
	if s == nil || s.accountRepo == nil {
		result.SkippedReason = "service_unavailable"
		return result, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return result, err
	}
	if !cfg.Enabled {
		result.SkippedReason = "monitoring_disabled"
		return result, nil
	}
	if len(cfg.AccountIDs) == 0 {
		result.SkippedReason = "no_accounts"
		return result, nil
	}
	release, ok := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, oauthAccountMonitorLeaderKey, s.instanceID, oauthAccountMonitorLockTTL)
	if !ok {
		result.SkippedReason = "leader_lock_held"
		return result, nil
	}
	defer release()
	result.Executed = true

	accounts, err := s.accountRepo.GetByIDs(ctx, cfg.AccountIDs)
	if err != nil {
		return result, err
	}
	valid := make([]int64, 0, len(accounts))
	validAccounts := make([]*Account, 0, len(accounts))
	for _, account := range accounts {
		if account == nil || !account.IsOpenAIOAuth() {
			continue
		}
		valid = append(valid, account.ID)
		validAccounts = append(validAccounts, account)
	}
	result.CheckedAccounts = len(validAccounts)

	var checkWG sync.WaitGroup
	var errorMu sync.Mutex
	checkSlots := make(chan struct{}, oauthAccountMonitorConcurrency)
	cycleErrors := make([]error, 0)
	for _, account := range validAccounts {
		account := account
		checkWG.Add(1)
		go func() {
			defer checkWG.Done()
			select {
			case checkSlots <- struct{}{}:
				defer func() { <-checkSlots }()
			case <-ctx.Done():
				errorMu.Lock()
				cycleErrors = append(cycleErrors, fmt.Errorf("check account %d: %w", account.ID, ctx.Err()))
				errorMu.Unlock()
				return
			}
			if checkErr := s.checkAccount(ctx, cfg, account); checkErr != nil {
				errorMu.Lock()
				cycleErrors = append(cycleErrors, fmt.Errorf("check account %d: %w", account.ID, checkErr))
				errorMu.Unlock()
			}
		}()
	}
	checkWG.Wait()

	if len(valid) != len(cfg.AccountIDs) {
		clean := *cfg
		clean.AccountIDs = valid
		if _, saveErr := s.UpdateConfig(ctx, &clean); saveErr != nil {
			cycleErrors = append(cycleErrors, fmt.Errorf("clean oauth monitor config: %w", saveErr))
		}
	}
	return result, errors.Join(cycleErrors...)
}

func (s *OAuthAccountMonitorService) checkAccount(ctx context.Context, cfg *OAuthAccountMonitorConfig, account *Account) error {
	if cfg == nil {
		return errors.New("oauth account monitor config is required")
	}
	effectiveConfig := *cfg
	normalizeOAuthMonitorConfig(&effectiveConfig)
	cfg = &effectiveConfig

	now := time.Now().UTC()
	var previous *OAuthAccountMonitorState
	if s.stateRepo != nil {
		states, err := s.stateRepo.GetStates(ctx, []int64{account.ID})
		if err != nil {
			return fmt.Errorf("load previous monitor state: %w", err)
		}
		if states != nil {
			previous = states[account.ID]
		}
	}
	health := "ok"
	errorMessage := ""
	availability := &OAuthAccountAvailabilityState{}
	if s.outcomeStore != nil {
		loaded, err := s.outcomeStore.GetAvailabilityState(ctx, account.ID)
		if err != nil {
			slog.Warn("oauth_account_monitor_availability_load_failed", "account_id", account.ID, "error", err)
		} else if loaded != nil {
			availability = loaded
		}
	}
	unavailable := availability.ConsecutiveFailures >= cfg.UnavailableFailureThreshold
	if unavailable {
		health = "error"
		errorMessage = strings.TrimSpace(availability.LastError)
		if errorMessage == "" {
			errorMessage = "account requests are continuously failing"
		}
	} else if account.Status != StatusActive || !account.Schedulable || strings.TrimSpace(account.ErrorMessage) != "" || (account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt)) || (account.RateLimitResetAt != nil && account.RateLimitResetAt.After(now)) || (account.OverloadUntil != nil && account.OverloadUntil.After(now)) || (account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now)) {
		// Runtime state remains visible in the drawer, but never sends an alert by itself.
		health = "error"
		errorMessage = strings.TrimSpace(account.ErrorMessage)
		if errorMessage == "" {
			errorMessage = "account is not schedulable"
		}
	}
	var primary, secondary *float64
	quotaStatus := "unknown"
	quotaLow := false
	quotaWindow := ""
	if s.quotaService != nil {
		quotaCtx, cancel := context.WithTimeout(ctx, oauthAccountMonitorQuotaTimeout)
		usage, err := s.quotaService.QueryUsage(quotaCtx, account.ID)
		cancel()
		if err == nil && usage != nil && usage.RateLimit != nil {
			quotaStatus = "ok"
			if usage.RateLimit.PrimaryWindow != nil {
				v := 100 - usage.RateLimit.PrimaryWindow.UsedPercent
				primary = &v
				if v < cfg.QuotaThreshold {
					quotaLow = true
					quotaWindow = "primary"
				}
			}
			if usage.RateLimit.SecondaryWindow != nil {
				v := 100 - usage.RateLimit.SecondaryWindow.UsedPercent
				secondary = &v
				if v < cfg.QuotaThreshold {
					quotaLow = true
					if quotaWindow == "" {
						quotaWindow = "secondary"
					}
				}
			}
		}
	}
	condition := ""
	if quotaLow {
		if condition != "" {
			condition += ";"
		}
		condition += "quota_low:" + quotaWindow
	}
	if unavailable {
		if condition != "" {
			condition += ";"
		}
		condition += "unavailable"
	}
	state := &OAuthAccountMonitorState{
		AccountID: account.ID, LastCheckedAt: &now, HealthStatus: health, ErrorMessage: errorMessage,
		PrimaryRemainingPercent: primary, SecondaryRemainingPercent: secondary, QuotaStatus: quotaStatus,
		LastConditionKey: condition, ConsecutiveFailures: availability.ConsecutiveFailures,
		LastSuccessAt: availability.LastSuccessAt, LastFailureAt: availability.LastFailureAt,
		LastErrorCode: availability.LastErrorCode, FailureStartedAt: availability.FailureStartedAt,
		LastUnavailableNotifiedAt:   availability.LastUnavailableNotifiedAt,
		FailureSequence:             availability.FailureSequence,
		LastNotifiedFailureSequence: availability.LastNotifiedFailureSequence,
		QuotaAlertActive:            quotaLow,
		UpdatedAt:                   now,
	}
	if previous != nil {
		state.LastNotifiedAt = previous.LastNotifiedAt
		if state.LastUnavailableNotifiedAt == nil {
			state.LastUnavailableNotifiedAt = previous.LastUnavailableNotifiedAt
		}
		if state.LastNotifiedFailureSequence == 0 {
			state.LastNotifiedFailureSequence = previous.LastNotifiedFailureSequence
		}
		if quotaStatus == "unknown" {
			state.QuotaAlertActive = previous.QuotaAlertActive
		}
	}
	quotaNotification := cfg.NotifyOnQuota && quotaLow && (previous == nil || !previous.QuotaAlertActive)
	unavailableNotification := cfg.NotifyOnUnavailable && unavailable && availability.FailureSequence > state.LastNotifiedFailureSequence
	if unavailableNotification && state.LastUnavailableNotifiedAt != nil && now.Sub(*state.LastUnavailableNotifiedAt) < time.Duration(cfg.UnavailableReminderMinutes)*time.Minute {
		unavailableNotification = false
	}
	var resultErrors []error
	if quotaNotification || unavailableNotification {
		alertKinds := make([]string, 0, 2)
		if quotaNotification {
			alertKinds = append(alertKinds, "quota_low")
		}
		if unavailableNotification {
			alertKinds = append(alertKinds, "unavailable")
		}
		_, notifyErr := s.notify(ctx, cfg, account, state, alertKinds)
		state.LastNotifiedAt = &now
		if unavailableNotification {
			state.LastUnavailableNotifiedAt = &now
			state.LastNotifiedFailureSequence = availability.FailureSequence
		}
		if notifyErr != nil {
			resultErrors = append(resultErrors, notifyErr)
		}
	}
	if s.stateRepo != nil {
		if err := s.stateRepo.UpsertState(ctx, state); err != nil {
			resultErrors = append(resultErrors, fmt.Errorf("persist monitor state: %w", err))
		}
	}
	return errors.Join(resultErrors...)
}

func (s *OAuthAccountMonitorService) notify(ctx context.Context, cfg *OAuthAccountMonitorConfig, account *Account, state *OAuthAccountMonitorState, alertKinds []string) (bool, error) {
	if len(alertKinds) == 0 {
		return false, nil
	}
	title := fmt.Sprintf("OAuth账号监控：%s", account.Name)
	var body strings.Builder
	if hasString(alertKinds, "quota_low") {
		body.WriteString("账号额度低于预警阈值。\n")
	}
	if hasString(alertKinds, "unavailable") {
		body.WriteString(fmt.Sprintf("账号在 %d 分钟内连续失败 %d 次。\n", cfg.UnavailableWindowMinutes, state.ConsecutiveFailures))
	}
	body.WriteString(fmt.Sprintf("账号ID: %d\n账号: %s\n状态: %s\n", account.ID, account.Name, state.HealthStatus))
	if state.ErrorMessage != "" {
		body.WriteString("错误: " + state.ErrorMessage + "\n")
	}
	if state.PrimaryRemainingPercent != nil {
		body.WriteString(fmt.Sprintf("主窗口剩余: %.1f%%\n", *state.PrimaryRemainingPercent))
	}
	if state.SecondaryRemainingPercent != nil {
		body.WriteString(fmt.Sprintf("次窗口剩余: %.1f%%\n", *state.SecondaryRemainingPercent))
	}
	body.WriteString("检查时间: " + state.LastCheckedAt.Format(time.RFC3339))
	var deliveryErrors []error
	delivered := false
	emailSent := false
	if cfg.NotifyEmail {
		emailCfg, err := s.GetEmailConfig(ctx)
		if err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("load email notification config: %w", err))
		} else if emailCfg.Enabled {
			if s.emailService == nil {
				deliveryErrors = append(deliveryErrors, errors.New("email notification service unavailable"))
			}
			recipients := 0
			for _, to := range emailCfg.Recipients {
				if strings.TrimSpace(to) == "" {
					continue
				}
				recipients++
				if s.emailService == nil {
					continue
				}
				if err := s.emailService.SendEmail(ctx, strings.TrimSpace(to), title, body.String()); err != nil {
					deliveryErrors = append(deliveryErrors, fmt.Errorf("send email notification to %s: %w", strings.TrimSpace(to), err))
				} else {
					emailSent = true
					delivered = true
				}
			}
			if recipients == 0 {
				deliveryErrors = append(deliveryErrors, errors.New("email notification has no recipients"))
			}
		}
	}
	if cfg.NotifyPushPlus {
		push, err := s.GetPushPlusConfig(ctx)
		if err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("load pushplus config: %w", err))
		} else if push.Enabled && push.Token != "" {
			if err := s.sendPushPlus(ctx, push, title, body.String()); err != nil {
				deliveryErrors = append(deliveryErrors, fmt.Errorf("send pushplus notification: %w", err))
			} else {
				delivered = true
			}
		}
	}
	if s.opsRepo != nil {
		dims := map[string]any{"account_id": account.ID, "account_name": account.Name, "platform": account.Platform, "alert_kinds": alertKinds, "condition": state.LastConditionKey, "quota_threshold_percent": cfg.QuotaThreshold, "primary_remaining_percent": state.PrimaryRemainingPercent, "secondary_remaining_percent": state.SecondaryRemainingPercent, "consecutive_failures": state.ConsecutiveFailures, "last_error_code": state.LastErrorCode, "failure_started_at": state.FailureStartedAt, "checked_at": state.LastCheckedAt}
		event := &OpsAlertEvent{Severity: "warning", Status: OpsAlertStatusFiring, Title: title, Description: body.String(), Dimensions: dims, FiredAt: time.Now().UTC(), EmailSent: emailSent}
		if _, err := s.opsRepo.CreateAlertEvent(ctx, event); err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("persist ops alert event: %w", err))
		}
	}
	return delivered, errors.Join(deliveryErrors...)
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func truncateMonitorError(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func (s *OAuthAccountMonitorService) sendPushPlus(ctx context.Context, cfg *OAuthMonitorPushPlusConfig, title, content string) error {
	payload := map[string]any{"token": cfg.Token, "title": title, "content": content}
	if cfg.Topic != "" {
		payload["topic"] = cfg.Topic
	}
	if cfg.Template != "" {
		payload["template"] = cfg.Template
	}
	if cfg.Channel != "" {
		payload["channel"] = cfg.Channel
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.pushplus.plus/send", strings.NewReader(string(raw)))
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		resp, doErr := client.Do(req)
		if doErr == nil {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var result struct {
					Code any    `json:"code"`
					Msg  string `json:"msg"`
				}
				if json.Unmarshal(body, &result) == nil && result.Code != nil {
					code := fmt.Sprint(result.Code)
					if code != "200" && code != "0" {
						lastErr = fmt.Errorf("pushplus rejected request: %s", strings.TrimSpace(result.Msg))
					} else {
						return nil
					}
				} else {
					return nil
				}
			} else if readErr != nil {
				lastErr = readErr
			} else {
				lastErr = fmt.Errorf("pushplus status %s", resp.Status)
			}
		} else {
			lastErr = doErr
		}
		if attempt < 3 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}
	return lastErr
}
