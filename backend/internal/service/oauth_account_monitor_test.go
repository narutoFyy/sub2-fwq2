package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type monitorStateRepoFake struct{ state *OAuthAccountMonitorState }

func (f *monitorStateRepoFake) GetStates(context.Context, []int64) (map[int64]*OAuthAccountMonitorState, error) {
	if f.state == nil {
		return map[int64]*OAuthAccountMonitorState{}, nil
	}
	return map[int64]*OAuthAccountMonitorState{f.state.AccountID: f.state}, nil
}
func (f *monitorStateRepoFake) UpsertState(_ context.Context, state *OAuthAccountMonitorState) error {
	f.state = state
	return nil
}

type monitorQuotaFake struct {
	usage *OpenAIQuotaUsage
	err   error
}

func (f monitorQuotaFake) QueryUsage(context.Context, int64) (*OpenAIQuotaUsage, error) {
	return f.usage, f.err
}

type monitorOutcomeStoreFake struct {
	state *OAuthAccountAvailabilityState
	err   error
}

func (f *monitorOutcomeStoreFake) RecordOutcome(context.Context, OAuthAccountRequestOutcome) (*OAuthAccountAvailabilityState, error) {
	return f.state, f.err
}

func (f *monitorOutcomeStoreFake) GetAvailabilityState(context.Context, int64) (*OAuthAccountAvailabilityState, error) {
	return f.state, f.err
}

func TestOAuthMonitorConfigValidation(t *testing.T) {
	svc := NewOAuthAccountMonitorService(nil, nil, nil, nil, nil, nil, nil)
	if _, err := svc.UpdateConfig(nil, &OAuthAccountMonitorConfig{QuotaThreshold: 101, IntervalMinutes: 5}); err == nil {
		t.Fatal("expected quota threshold validation error")
	}
	if _, err := svc.UpdateConfig(nil, &OAuthAccountMonitorConfig{QuotaThreshold: 20, IntervalMinutes: 1441}); err == nil {
		t.Fatal("expected interval validation error")
	}
	if _, err := svc.UpdateConfig(nil, &OAuthAccountMonitorConfig{QuotaThreshold: 20, IntervalMinutes: 0}); err == nil {
		t.Fatal("expected zero interval validation error")
	}
}

func TestOAuthMonitorCheckAccountTracksQuotaEvenWhenNotificationsDisabled(t *testing.T) {
	stateRepo := &monitorStateRepoFake{}
	remaining := 12.0
	svc := NewOAuthAccountMonitorService(nil, stateRepo, nil, monitorQuotaFake{usage: &OpenAIQuotaUsage{RateLimit: &OpenAIRateLimit{PrimaryWindow: &OpenAIRateLimitWindow{UsedPercent: 88}}}}, nil, nil, nil)
	svc.checkAccount(context.Background(), &OAuthAccountMonitorConfig{QuotaThreshold: 20, NotifyOnQuota: false}, &Account{ID: 7, Name: "test", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true})
	if stateRepo.state == nil || stateRepo.state.QuotaStatus != "ok" || stateRepo.state.PrimaryRemainingPercent == nil || *stateRepo.state.PrimaryRemainingPercent != remaining {
		t.Fatalf("quota state not recorded: %+v", stateRepo.state)
	}
}

func TestOAuthMonitorDoesNotTreatUnknownQuotaAsRecovery(t *testing.T) {
	previous := &OAuthAccountMonitorState{AccountID: 7, LastConditionKey: "quota_low:primary", LastCheckedAt: monitorPtrTime(time.Now()), QuotaAlertActive: true}
	stateRepo := &monitorStateRepoFake{state: previous}
	svc := NewOAuthAccountMonitorService(nil, stateRepo, nil, monitorQuotaFake{err: context.DeadlineExceeded}, nil, nil, nil)
	svc.checkAccount(context.Background(), &OAuthAccountMonitorConfig{QuotaThreshold: 20, NotifyOnQuota: true, NotifyOnRecovery: true}, &Account{ID: 7, Name: "test", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true})
	if stateRepo.state.LastConditionKey != "" || stateRepo.state.QuotaStatus != "unknown" {
		t.Fatalf("unexpected state: %+v", stateRepo.state)
	}
	if stateRepo.state.LastNotifiedAt != nil {
		t.Fatal("unknown quota must not emit recovery notification")
	}
	if !stateRepo.state.QuotaAlertActive {
		t.Fatal("unknown quota must preserve the active quota alert cycle")
	}
}

func TestOAuthMonitorQuotaNotifiesOnlyOnLowTransition(t *testing.T) {
	stateRepo := &monitorStateRepoFake{}
	svc := NewOAuthAccountMonitorService(nil, stateRepo, nil, monitorQuotaFake{usage: &OpenAIQuotaUsage{RateLimit: &OpenAIRateLimit{PrimaryWindow: &OpenAIRateLimitWindow{UsedPercent: 90}}}}, nil, nil, nil)
	config := &OAuthAccountMonitorConfig{QuotaThreshold: 20, NotifyOnQuota: true}
	account := &Account{ID: 7, Name: "test", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}

	if err := svc.checkAccount(context.Background(), config, account); err != nil {
		t.Fatalf("first quota check failed: %v", err)
	}
	if stateRepo.state.LastNotifiedAt == nil || !stateRepo.state.QuotaAlertActive {
		t.Fatalf("first low quota transition was not recorded: %+v", stateRepo.state)
	}
	firstNotification := *stateRepo.state.LastNotifiedAt

	if err := svc.checkAccount(context.Background(), config, account); err != nil {
		t.Fatalf("second quota check failed: %v", err)
	}
	if stateRepo.state.LastNotifiedAt == nil || !stateRepo.state.LastNotifiedAt.Equal(firstNotification) {
		t.Fatalf("continuous low quota emitted another notification: %+v", stateRepo.state)
	}
}

func TestOAuthMonitorSustainedUnavailableNotificationCadence(t *testing.T) {
	stateRepo := &monitorStateRepoFake{}
	outcomes := &monitorOutcomeStoreFake{state: &OAuthAccountAvailabilityState{}}
	svc := NewOAuthAccountMonitorService(nil, stateRepo, nil, nil, nil, nil, nil)
	svc.outcomeStore = outcomes
	config := &OAuthAccountMonitorConfig{
		NotifyOnUnavailable:         true,
		UnavailableFailureThreshold: 3,
		UnavailableWindowMinutes:    15,
		UnavailableReminderMinutes:  60,
	}
	account := &Account{ID: 7, Name: "test", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}

	for failures := 1; failures <= 2; failures++ {
		outcomes.state = &OAuthAccountAvailabilityState{ConsecutiveFailures: failures, FailureSequence: int64(failures)}
		if err := svc.checkAccount(context.Background(), config, account); err != nil {
			t.Fatalf("failure %d check failed: %v", failures, err)
		}
		if stateRepo.state.LastUnavailableNotifiedAt != nil {
			t.Fatalf("failure %d unexpectedly notified: %+v", failures, stateRepo.state)
		}
	}

	outcomes.state = &OAuthAccountAvailabilityState{ConsecutiveFailures: 3, FailureSequence: 3, LastErrorCode: "502", LastError: "upstream failed"}
	if err := svc.checkAccount(context.Background(), config, account); err != nil {
		t.Fatalf("third failure check failed: %v", err)
	}
	if stateRepo.state.LastUnavailableNotifiedAt == nil || stateRepo.state.LastNotifiedFailureSequence != 3 {
		t.Fatalf("third consecutive failure did not notify: %+v", stateRepo.state)
	}
	firstNotification := *stateRepo.state.LastUnavailableNotifiedAt

	outcomes.state = &OAuthAccountAvailabilityState{ConsecutiveFailures: 4, FailureSequence: 4, LastErrorCode: "502", LastError: "upstream failed"}
	if err := svc.checkAccount(context.Background(), config, account); err != nil {
		t.Fatalf("failure inside reminder window check failed: %v", err)
	}
	if stateRepo.state.LastUnavailableNotifiedAt == nil || !stateRepo.state.LastUnavailableNotifiedAt.Equal(firstNotification) || stateRepo.state.LastNotifiedFailureSequence != 3 {
		t.Fatalf("failure inside reminder window unexpectedly notified: %+v", stateRepo.state)
	}

	agedNotification := time.Now().Add(-61 * time.Minute)
	stateRepo.state.LastUnavailableNotifiedAt = monitorPtrTime(agedNotification)
	outcomes.state = &OAuthAccountAvailabilityState{ConsecutiveFailures: 5, FailureSequence: 5, LastErrorCode: "502", LastError: "upstream failed"}
	if err := svc.checkAccount(context.Background(), config, account); err != nil {
		t.Fatalf("failure after reminder window check failed: %v", err)
	}
	if stateRepo.state.LastUnavailableNotifiedAt == nil || !stateRepo.state.LastUnavailableNotifiedAt.After(agedNotification) || stateRepo.state.LastNotifiedFailureSequence != 5 {
		t.Fatalf("new failure after reminder window did not notify: %+v", stateRepo.state)
	}
}

func monitorPtrTime(v time.Time) *time.Time { return &v }

func TestOAuthMonitorConfigNormalization(t *testing.T) {
	cfg := &OAuthAccountMonitorConfig{AccountIDs: []int64{3, 0, 3, 2}, QuotaThreshold: -1, NotifyOnError: true, NotifyOnRecovery: true, RepeatEveryCheck: true}
	normalizeOAuthMonitorConfig(cfg)
	if cfg.IntervalMinutes != 5 || cfg.QuotaThreshold != 0 || cfg.UnavailableFailureThreshold != 3 || cfg.UnavailableWindowMinutes != 15 || cfg.UnavailableReminderMinutes != 60 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.NotifyOnError || cfg.NotifyOnRecovery || cfg.RepeatEveryCheck {
		t.Fatalf("legacy noisy notification flags must be disabled: %+v", cfg)
	}
	if len(cfg.AccountIDs) != 2 || cfg.AccountIDs[0] != 3 || cfg.AccountIDs[1] != 2 {
		t.Fatalf("unexpected account ids: %+v", cfg.AccountIDs)
	}
}

func TestOAuthMonitorEmailConfigValidationAndNormalization(t *testing.T) {
	svc := NewOAuthAccountMonitorService(nil, nil, nil, nil, nil, nil, nil)
	cfg, err := svc.UpdateEmailConfig(context.Background(), &OAuthMonitorEmailConfig{
		Enabled:    true,
		Recipients: []string{" Admin@Example.com ", "admin@example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected email config error: %v", err)
	}
	if len(cfg.Recipients) != 1 || cfg.Recipients[0] != "admin@example.com" {
		t.Fatalf("unexpected normalized recipients: %+v", cfg.Recipients)
	}
	if _, err := svc.UpdateEmailConfig(context.Background(), &OAuthMonitorEmailConfig{Enabled: true}); err == nil {
		t.Fatal("expected enabled email config without recipients to fail")
	}
	if _, err := svc.UpdateEmailConfig(context.Background(), &OAuthMonitorEmailConfig{Recipients: []string{"invalid"}}); err == nil {
		t.Fatal("expected invalid recipient to fail")
	}
}

func TestOpsEmailNormalizationDisablesAlertsAndPreservesReports(t *testing.T) {
	defaults := defaultOpsEmailNotificationConfig()
	if defaults.Alert.Enabled {
		t.Fatal("global Ops realtime alert email must default to disabled")
	}

	cfg := &OpsEmailNotificationConfig{
		Alert:  OpsEmailAlertConfig{Enabled: true, Recipients: []string{"ops@example.com"}},
		Report: OpsEmailReportConfig{Enabled: true, Recipients: []string{"report@example.com"}},
	}
	normalizeOpsEmailNotificationConfig(cfg)
	if cfg.Alert.Enabled {
		t.Fatal("global Ops realtime alert email must stay disabled")
	}
	if !cfg.Report.Enabled || len(cfg.Report.Recipients) != 1 || cfg.Report.Recipients[0] != "report@example.com" {
		t.Fatalf("Ops report configuration changed unexpectedly: %+v", cfg.Report)
	}
}

func TestClassifyOAuthAccountRequestFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unauthorized", err: &UpstreamFailoverError{StatusCode: http.StatusUnauthorized, Scope: GatewayFailureScopeAccount}, want: true},
		{name: "rate limited", err: &UpstreamFailoverError{StatusCode: http.StatusTooManyRequests, Scope: GatewayFailureScopeAccount}, want: true},
		{name: "server error", err: &UpstreamFailoverError{StatusCode: http.StatusBadGateway}, want: true},
		{name: "images server error", err: &OpenAIImagesUpstreamError{StatusCode: http.StatusBadGateway, Message: "upstream failed"}, want: true},
		{name: "images bad parameter", err: &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, Message: "bad request"}, want: false},
		{name: "timeout", err: context.DeadlineExceeded, want: true},
		{name: "stream disconnect", err: errors.New("stream disconnected before completion"), want: true},
		{name: "request scoped", err: &UpstreamFailoverError{StatusCode: http.StatusBadGateway, Scope: GatewayFailureScopeRequest}, want: false},
		{name: "provider scoped", err: &UpstreamFailoverError{StatusCode: http.StatusBadGateway, Scope: GatewayFailureScopeProvider}, want: false},
		{name: "bad parameter", err: &UpstreamFailoverError{StatusCode: http.StatusBadRequest, Scope: GatewayFailureScopeRequest}, want: false},
		{name: "client canceled", err: context.Canceled, want: false},
		{name: "local error", err: errors.New("account is busy"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, got := classifyOAuthAccountRequestFailure(tt.err)
			if got != tt.want {
				t.Fatalf("classify failure = %v, want %v", got, tt.want)
			}
		})
	}
}
