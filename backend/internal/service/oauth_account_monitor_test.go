package service

import (
	"context"
	"testing"
	"time"
)

type monitorStateRepoFake struct{ state *OAuthAccountMonitorState }
func (f *monitorStateRepoFake) GetStates(context.Context, []int64) (map[int64]*OAuthAccountMonitorState, error) { if f.state == nil { return map[int64]*OAuthAccountMonitorState{}, nil }; return map[int64]*OAuthAccountMonitorState{f.state.AccountID: f.state}, nil }
func (f *monitorStateRepoFake) UpsertState(_ context.Context, state *OAuthAccountMonitorState) error { f.state = state; return nil }
type monitorQuotaFake struct{ usage *OpenAIQuotaUsage; err error }
func (f monitorQuotaFake) QueryUsage(context.Context, int64) (*OpenAIQuotaUsage, error) { return f.usage, f.err }

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
	previous := &OAuthAccountMonitorState{AccountID: 7, LastConditionKey: "quota_low:primary", LastCheckedAt: monitorPtrTime(time.Now())}
	stateRepo := &monitorStateRepoFake{state: previous}
	svc := NewOAuthAccountMonitorService(nil, stateRepo, nil, monitorQuotaFake{err: context.DeadlineExceeded}, nil, nil, nil)
	svc.checkAccount(context.Background(), &OAuthAccountMonitorConfig{QuotaThreshold: 20, NotifyOnQuota: true, NotifyOnRecovery: true}, &Account{ID: 7, Name: "test", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true})
	if stateRepo.state.LastConditionKey != "" || stateRepo.state.QuotaStatus != "unknown" {
		t.Fatalf("unexpected state: %+v", stateRepo.state)
	}
	if stateRepo.state.LastNotifiedAt != nil { t.Fatal("unknown quota must not emit recovery notification") }
}

func monitorPtrTime(v time.Time) *time.Time { return &v }

func TestOAuthMonitorConfigNormalization(t *testing.T) {
	cfg := &OAuthAccountMonitorConfig{AccountIDs: []int64{3, 0, 3, 2}, QuotaThreshold: -1}
	normalizeOAuthMonitorConfig(cfg)
	if cfg.IntervalMinutes != 5 || cfg.QuotaThreshold != 0 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.AccountIDs) != 2 || cfg.AccountIDs[0] != 3 || cfg.AccountIDs[1] != 2 {
		t.Fatalf("unexpected account ids: %+v", cfg.AccountIDs)
	}
}
