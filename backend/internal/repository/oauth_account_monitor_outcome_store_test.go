package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestOAuthAccountMonitorOutcomeStoreDeduplicatesAndResets(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewOAuthAccountMonitorOutcomeStore(client)
	ctx := context.Background()
	start := time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC)

	record := func(requestID string, at time.Time, success bool) *service.OAuthAccountAvailabilityState {
		state, err := store.RecordOutcome(ctx, service.OAuthAccountRequestOutcome{
			AccountID: 7, RequestID: requestID, Success: success,
			ErrorCode: "502", Error: "upstream failed", OccurredAt: at, Window: 15 * time.Minute,
		})
		require.NoError(t, err)
		return state
	}

	require.Equal(t, 1, record("req-1", start, false).ConsecutiveFailures)
	require.Equal(t, 1, record("req-1", start.Add(time.Second), false).ConsecutiveFailures)
	require.Equal(t, 2, record("req-2", start.Add(2*time.Minute), false).ConsecutiveFailures)
	require.Equal(t, 3, record("req-3", start.Add(4*time.Minute), false).ConsecutiveFailures)

	state := record("req-3", start.Add(5*time.Minute), true)
	require.Zero(t, state.ConsecutiveFailures)
	require.NotNil(t, state.LastSuccessAt)
	require.Nil(t, state.FailureStartedAt)

	state = record("req-3", start.Add(6*time.Minute), false)
	require.Zero(t, state.ConsecutiveFailures, "a late failure cannot override a success for the same request")
	state = record("req-4", start.Add(7*time.Minute), false)
	require.Equal(t, 1, state.ConsecutiveFailures)
}

func TestOAuthAccountMonitorOutcomeStoreRestartsOutsideWindow(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewOAuthAccountMonitorOutcomeStore(client)
	ctx := context.Background()
	start := time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC)

	_, err := store.RecordOutcome(ctx, service.OAuthAccountRequestOutcome{AccountID: 9, RequestID: "a", OccurredAt: start, Window: 15 * time.Minute})
	require.NoError(t, err)
	state, err := store.RecordOutcome(ctx, service.OAuthAccountRequestOutcome{AccountID: 9, RequestID: "b", OccurredAt: start.Add(16 * time.Minute), Window: 15 * time.Minute})
	require.NoError(t, err)
	require.Equal(t, 1, state.ConsecutiveFailures)
	require.Equal(t, int64(2), state.FailureSequence)
}

func TestOAuthAccountMonitorOutcomeStoreIgnoresOutOfOrderResults(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewOAuthAccountMonitorOutcomeStore(client)
	ctx := context.Background()
	start := time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC)

	record := func(requestID string, at time.Time, success bool) *service.OAuthAccountAvailabilityState {
		state, err := store.RecordOutcome(ctx, service.OAuthAccountRequestOutcome{
			AccountID: 11, RequestID: requestID, Success: success,
			ErrorCode: "502", Error: "upstream failed", OccurredAt: at, Window: 15 * time.Minute,
		})
		require.NoError(t, err)
		return state
	}

	require.Equal(t, 1, record("failure-before-success", start, false).ConsecutiveFailures)
	require.Zero(t, record("newer-success", start.Add(2*time.Minute), true).ConsecutiveFailures)
	require.Zero(t, record("late-old-failure", start.Add(time.Minute), false).ConsecutiveFailures)

	require.Equal(t, 1, record("newer-failure", start.Add(4*time.Minute), false).ConsecutiveFailures)
	require.Equal(t, 1, record("late-old-success", start.Add(3*time.Minute), true).ConsecutiveFailures)
}
