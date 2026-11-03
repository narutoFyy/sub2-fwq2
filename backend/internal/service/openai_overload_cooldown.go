package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

const (
	openAIOverloadCooldown       = 5 * time.Minute
	openAIOverloadScopePrefix    = "openai:overload:"
	openAIOverloadCooldownReason = "openai_overload"
)

func openAIOverloadScope(canonicalModel string) string {
	model := strings.TrimSpace(canonicalModel)
	if model == "" {
		return ""
	}
	return openAIOverloadScopePrefix + model
}

// coolDownOpenAIOverload parks only the failed account/model pair. Persisting
// through the repository also updates the shared scheduler snapshot, so other
// application nodes observe the same five-minute deadline.
func (s *OpenAIGatewayService) coolDownOpenAIOverload(ctx context.Context, account *Account, canonicalModel string) bool {
	if s == nil || s.accountRepo == nil || account == nil || account.Platform != PlatformOpenAI {
		return false
	}
	scope := openAIOverloadScope(canonicalModel)
	if scope == "" {
		return false
	}
	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()
	until := time.Now().Add(openAIOverloadCooldown)
	if err := s.accountRepo.SetModelRateLimit(stateCtx, account.ID, scope, until, openAIOverloadCooldownReason); err != nil {
		slog.Warn("openai_overload_cooldown_failed", "account_id", account.ID, "model", canonicalModel, "error", err)
		return true
	}
	slog.Info("openai_overload_cooldown", "account_id", account.ID, "model", canonicalModel, "until", until)
	return true
}
