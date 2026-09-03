package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const oauthAccountOutcomeStateTTL = 30 * 24 * time.Hour

type oauthAccountMonitorOutcomeStore struct {
	rdb *redis.Client
}

func NewOAuthAccountMonitorOutcomeStore(rdb *redis.Client) service.OAuthAccountMonitorOutcomeStore {
	return &oauthAccountMonitorOutcomeStore{rdb: rdb}
}

var recordOAuthAccountOutcomeScript = redis.NewScript(`
local previous = redis.call('GET', KEYS[2])
if previous == ARGV[1] then
  return 0
end
if previous == 'success' and ARGV[1] == 'failure' then
  return 0
end

redis.call('SET', KEYS[2], ARGV[1], 'PX', ARGV[7])
local now = tonumber(ARGV[2])
local last_failure = tonumber(redis.call('HGET', KEYS[1], 'last_failure_at') or '0')
local last_success = tonumber(redis.call('HGET', KEYS[1], 'last_success_at') or '0')
if ARGV[1] == 'success' then
	if last_success > now or last_failure > now then
		redis.call('PEXPIRE', KEYS[1], ARGV[6])
		return 1
	end
  redis.call('HSET', KEYS[1],
    'consecutive_failures', 0,
    'last_success_at', ARGV[2],
    'failure_started_at', '',
    'last_error_code', '',
    'last_error', '')
  redis.call('PEXPIRE', KEYS[1], ARGV[6])
  return 1
end

if last_success >= now then
	redis.call('PEXPIRE', KEYS[1], ARGV[6])
	return 1
end
local count = tonumber(redis.call('HGET', KEYS[1], 'consecutive_failures') or '0')
local started = redis.call('HGET', KEYS[1], 'failure_started_at') or ''
local started_number = tonumber(started) or 0
if last_failure == 0 or now - last_failure > tonumber(ARGV[3]) or started_number == 0 or now - started_number > tonumber(ARGV[3]) or last_success > last_failure then
  count = 1
  started = ARGV[2]
else
  count = count + 1
end
local sequence = redis.call('HINCRBY', KEYS[1], 'failure_sequence', 1)
redis.call('HSET', KEYS[1],
  'consecutive_failures', count,
  'last_failure_at', ARGV[2],
  'failure_started_at', started,
  'last_error_code', ARGV[4],
  'last_error', ARGV[5],
  'failure_sequence', sequence)
redis.call('PEXPIRE', KEYS[1], ARGV[6])
return 1
`)

func (s *oauthAccountMonitorOutcomeStore) RecordOutcome(ctx context.Context, outcome service.OAuthAccountRequestOutcome) (*service.OAuthAccountAvailabilityState, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("oauth account outcome store unavailable")
	}
	if outcome.AccountID <= 0 || strings.TrimSpace(outcome.RequestID) == "" {
		return nil, errors.New("invalid oauth account request outcome")
	}
	if outcome.OccurredAt.IsZero() {
		outcome.OccurredAt = time.Now().UTC()
	}
	if outcome.Window <= 0 {
		outcome.Window = 15 * time.Minute
	}
	outcomeName := "failure"
	if outcome.Success {
		outcomeName = "success"
	}
	stateKey := oauthAccountOutcomeStateKey(outcome.AccountID)
	dedupeKey := oauthAccountOutcomeDedupeKey(outcome.AccountID, outcome.RequestID)
	if _, err := recordOAuthAccountOutcomeScript.Run(
		ctx,
		s.rdb,
		[]string{stateKey, dedupeKey},
		outcomeName,
		outcome.OccurredAt.UnixMilli(),
		outcome.Window.Milliseconds(),
		strings.TrimSpace(outcome.ErrorCode),
		strings.TrimSpace(outcome.Error),
		oauthAccountOutcomeStateTTL.Milliseconds(),
		(24 * time.Hour).Milliseconds(),
	).Result(); err != nil {
		return nil, err
	}
	return s.GetAvailabilityState(ctx, outcome.AccountID)
}

func (s *oauthAccountMonitorOutcomeStore) GetAvailabilityState(ctx context.Context, accountID int64) (*service.OAuthAccountAvailabilityState, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("oauth account outcome store unavailable")
	}
	values, err := s.rdb.HGetAll(ctx, oauthAccountOutcomeStateKey(accountID)).Result()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return &service.OAuthAccountAvailabilityState{}, nil
	}
	state := &service.OAuthAccountAvailabilityState{
		ConsecutiveFailures:         parseOAuthOutcomeInt(values["consecutive_failures"]),
		LastErrorCode:               strings.TrimSpace(values["last_error_code"]),
		LastError:                   strings.TrimSpace(values["last_error"]),
		FailureSequence:             parseOAuthOutcomeInt64(values["failure_sequence"]),
		LastNotifiedFailureSequence: parseOAuthOutcomeInt64(values["last_notified_failure_sequence"]),
	}
	state.LastSuccessAt = parseOAuthOutcomeTime(values["last_success_at"])
	state.LastFailureAt = parseOAuthOutcomeTime(values["last_failure_at"])
	state.FailureStartedAt = parseOAuthOutcomeTime(values["failure_started_at"])
	state.LastUnavailableNotifiedAt = parseOAuthOutcomeTime(values["last_unavailable_notified_at"])
	return state, nil
}

func oauthAccountOutcomeStateKey(accountID int64) string {
	return fmt.Sprintf("oauth-account-monitor:availability:%d", accountID)
}

func oauthAccountOutcomeDedupeKey(accountID int64, requestID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(requestID)))
	return fmt.Sprintf("oauth-account-monitor:outcome:%d:%s", accountID, hex.EncodeToString(sum[:12]))
}

func parseOAuthOutcomeInt(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func parseOAuthOutcomeInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed
}

func parseOAuthOutcomeTime(value string) *time.Time {
	millis, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || millis <= 0 {
		return nil
	}
	parsed := time.UnixMilli(millis).UTC()
	return &parsed
}
