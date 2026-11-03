package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type overloadModelRepo struct {
	AccountRepository
	account    Account
	until      time.Time
	scope      string
	contextErr error
	writes     int
}

func (r *overloadModelRepo) SetModelRateLimit(ctx context.Context, id int64, scope string, until time.Time, reason ...string) error {
	r.contextErr = ctx.Err()
	r.writes++
	r.until, r.scope = until, scope
	setAccountModelRateLimitSnapshot(&r.account, scope, until, firstNonEmpty(reason...), time.Now())
	return r.contextErr
}

func newOverloadModelFixture() (*OpenAIGatewayService, *overloadModelRepo, *Account) {
	account := &Account{ID: 1886, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}
	repo := &overloadModelRepo{account: *account}
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}}
	return svc, repo, account
}

func TestOpenAIOverloadCooldown_HTTPSharedModelGateAndExpiry(t *testing.T) {
	svc, repo, account := newOverloadModelFixture()
	// Map the public alias once; the already canonical upstream name must not
	// be mapped a second time when recording the cooldown.
	account.Credentials = map[string]any{"model_mapping": map[string]any{"public": "gpt-overloaded", "gpt-overloaded": "other-model"}}
	repo.account = *account
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	body := []byte(`{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}`)
	before := time.Now()
	blocked := svc.handleOpenAIAccountUpstreamError(ctx, account, http.StatusServiceUnavailable, nil, body, "gpt-overloaded")
	require.True(t, blocked)
	require.NoError(t, repo.contextErr, "client disconnect must not cancel shared cooldown persistence")
	require.Equal(t, "openai:overload:gpt-overloaded", repo.scope)
	require.WithinDuration(t, before.Add(5*time.Minute), repo.until, time.Second)
	require.True(t, repo.account.IsSchedulable(), "do not pause the entire account")
	require.False(t, repo.account.IsSchedulableForModel("public"))
	require.True(t, repo.account.IsSchedulableForModel("gpt-overloaded"), "mapping must be applied exactly once")
	require.True(t, repo.account.IsSchedulableForModel("gpt-other"))
	// A different gateway process needs no local failure history to reject a
	// fresh copy loaded from the shared repository/snapshot.
	other := &OpenAIGatewayService{}
	fresh := repo.account
	require.Nil(t, other.resolveFreshSchedulableOpenAIAccount(context.Background(), &fresh, PlatformOpenAI, "public", false, ""))
	failover := svc.newOpenAIAccountFailoverError(account, 503, nil, body, "", blocked, true)
	require.True(t, failover.IsOpenAICapacityShed())
	require.False(t, failover.RetryableOnSameAccount)
	require.True(t, failover.ShouldRetryNextAccount())
	// Move the persisted deadline instead of sleeping for five minutes.
	setAccountModelRateLimitSnapshot(&fresh, repo.scope, time.Now().Add(-time.Second), openAIOverloadCooldownReason, before)
	require.True(t, fresh.IsSchedulableForModel("public"))
	require.NotNil(t, other.resolveFreshSchedulableOpenAIAccount(context.Background(), &fresh, PlatformOpenAI, "public", false, ""))
}

func TestOpenAIOverloadCooldown_StreamingBeforeAndAfterOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, passthrough := range []bool{false, true} {
		for _, partialOutput := range []bool{false, true} {
			name := "native"
			if passthrough {
				name = "passthrough"
			}
			if partialOutput {
				name += "/after_output"
			} else {
				name += "/before_output"
			}
			t.Run(name, func(t *testing.T) {
				svc, repo, account := newOverloadModelFixture()
				stream := "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_overload\"}}\n\n"
				if partialOutput {
					stream += "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n"
				}
				stream += "event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_overload\",\"status\":\"failed\",\"error\":{\"code\":\"server_is_overloaded\",\"message\":\"Our servers are currently overloaded. Please try again later.\"}}}\n\n"
				resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream))}
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
				var err error
				if passthrough {
					_, err = svc.handleStreamingResponsePassthrough(c.Request.Context(), resp, c, account, time.Now(), "gpt-overloaded", "gpt-overloaded")
				} else {
					_, err = svc.handleStreamingResponse(c.Request.Context(), resp, c, account, time.Now(), "gpt-overloaded", "gpt-overloaded")
				}
				require.Error(t, err)
				require.Positive(t, repo.writes)
				require.WithinDuration(t, time.Now().Add(5*time.Minute), repo.until, time.Second)
				require.False(t, repo.account.IsSchedulableForModel("gpt-overloaded"))
				require.True(t, repo.account.IsSchedulableForModel("gpt-other"))
				var failover *UpstreamFailoverError
				if partialOutput {
					require.False(t, errors.As(err, &failover), "do not replay a stream after semantic output")
					require.Contains(t, rec.Body.String(), "partial")
				} else {
					require.ErrorAs(t, err, &failover)
					require.False(t, failover.RetryableOnSameAccount)
				}
			})
		}
	}
}

func TestOpenAIOverloadCooldown_DoesNotShortenQuotaWindow(t *testing.T) {
	svc, repo, account := newOverloadModelFixture()
	quotaUntil := time.Now().Add(time.Hour)
	setAccountModelRateLimitSnapshot(&repo.account, "gpt-overloaded", quotaUntil, "quota_exhausted", time.Now())
	require.True(t, svc.coolDownOpenAIOverload(context.Background(), account, "gpt-overloaded"))
	require.WithinDuration(t, quotaUntil, *repo.account.modelRateLimitResetAt("gpt-overloaded"), time.Second)
	setAccountModelRateLimitSnapshot(&repo.account, repo.scope, time.Now().Add(-time.Second), openAIOverloadCooldownReason, time.Now())
	require.False(t, repo.account.IsSchedulableForModel("gpt-overloaded"))
}

func TestOpenAIOverloadCooldown_WebSocketTerminal(t *testing.T) {
	for _, payload := range []string{
		`{"type":"response.failed","response":{"error":{"code":"server_is_overloaded"}}}`,
		`{"type":"response.failed","response":{"error":{"message":"Our servers are currently overloaded. Please try again later."}}}`,
	} {
		svc, repo, account := newOverloadModelFixture()
		require.True(t, svc.handleOpenAIWSFailureAccountSideEffects(context.Background(), account, "gpt-overloaded", nil, []byte(payload)))
		require.Equal(t, 1, repo.writes)
		require.WithinDuration(t, time.Now().Add(5*time.Minute), repo.until, time.Second)
		require.False(t, repo.account.IsSchedulableForModel("gpt-overloaded"))
	}
}

func TestOpenAIOverloadCooldown_IgnoresUnrelatedErrorsAndOtherPlatforms(t *testing.T) {
	for _, body := range []string{
		`{"error":{"code":"server_error","message":"upstream unavailable"}}`,
		`{"error":{"message":"upstream unavailable"},"echo":"Our servers are currently overloaded"}`,
	} {
		svc, repo, account := newOverloadModelFixture()
		require.False(t, svc.handleOpenAIAccountUpstreamError(context.Background(), account, 503, nil, []byte(body), "gpt-overloaded"))
		require.Zero(t, repo.writes)
	}
	svc, repo, account := newOverloadModelFixture()
	account.Platform = PlatformAnthropic
	require.False(t, svc.coolDownOpenAIOverload(context.Background(), account, "claude"))
	require.Zero(t, repo.writes)
	account.Platform = PlatformOpenAI
	require.False(t, svc.coolDownOpenAIOverload(context.Background(), account, ""))
	require.Zero(t, repo.writes, "missing model must not pause unrelated models")
}
