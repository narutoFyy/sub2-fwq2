package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

type modelRadarGroupRepoStub struct {
	GroupRepository
	groups []Group
}

func (s *modelRadarGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return s.groups, nil
}

type modelRadarRepoStub struct {
	ModelRadarRepository
	configs []*ModelRadarConfig
	results []*ModelRadarResult
}

func (s *modelRadarRepoStub) ListConfigs(context.Context) ([]*ModelRadarConfig, error) {
	return s.configs, nil
}

func (s *modelRadarRepoStub) ListResults(_ context.Context, _ *int64, testType string, _ int) ([]*ModelRadarResult, error) {
	out := make([]*ModelRadarResult, 0, len(s.results))
	for _, result := range s.results {
		if testType == "" || result.TestType == testType {
			out = append(out, result)
		}
	}
	return out, nil
}

func TestModelRadarPublicOverviewExposesResultsAndRedactsInternalDetails(t *testing.T) {
	reviewedBy := int64(77)
	repo := &modelRadarRepoStub{configs: []*ModelRadarConfig{{GroupID: 10, Enabled: true}}, results: []*ModelRadarResult{{
		ID: 2, GroupID: 10, ModelID: "gpt-5.4", TestType: ModelRadarTestTypeCiallo,
		Status: ModelRadarStatusPassed, ResponseText: "Ciallo～(∠・ω< )⌒☆", ErrorMessage: "secret error",
		LatencyMS: 120, DetectedAt: time.Now(), ReviewedBy: &reviewedBy,
	}, {
		ID: 1, GroupID: 10, ModelID: "gpt-5.4", TestType: ModelRadarTestTypeLogic,
		Status: ModelRadarStatusPassed, ResponseText: "Answer: 21", DetectedAt: time.Now(),
	}}}
	groups := &modelRadarGroupRepoStub{groups: []Group{{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive}}}
	svc := NewModelRadarService(repo, nil, groups, nil)

	overview, err := svc.PublicOverview(context.Background())
	require.NoError(t, err)
	require.Len(t, overview.Groups, 1)
	require.Equal(t, "ciallo", overview.Prompt)
	require.NotNil(t, overview.Groups[0].Ciallo)
	require.Equal(t, "Ciallo～(∠・ω< )⌒☆", overview.Groups[0].Ciallo.ResponseText)
	require.Empty(t, overview.Groups[0].Ciallo.ErrorMessage)
	require.Nil(t, overview.Groups[0].Ciallo.ReviewedBy)
	// 旧数列题的历史结果不进时间线。
	require.Len(t, overview.Groups[0].Timeline, 1)
	require.Empty(t, overview.Groups[0].Timeline[0].ErrorMessage)
}

func TestModelRadarRejectsNonOpenAIConfig(t *testing.T) {
	groups := &modelRadarGroupRepoStub{groups: []Group{{ID: 10, Name: "Claude", Platform: PlatformAnthropic, Status: StatusActive}}}
	svc := NewModelRadarService(&modelRadarRepoStub{}, nil, groups, nil)

	_, err := svc.UpdateConfigs(context.Background(), []*ModelRadarConfig{{GroupID: 10, Enabled: true, ReasoningEffort: "medium"}})
	require.ErrorIs(t, err, ErrModelRadarOpenAIOnly)
}

func TestIsCialloReply(t *testing.T) {
	// 以下回复均为 2026-11-15 实测所得。
	for _, reply := range []string{
		"Ciallo～(∠・ω< )⌒☆",
		"Ciallo～(∠・ω< )⌒★ 有什么我可以帮你的吗？",
		"ciallo~ (≧▽≦)",
		"Ciallo！(^_^)",
	} {
		require.True(t, IsCialloReply(reply), reply)
	}
	for _, reply := range []string{
		"Ciallo! 👋 What can I help you with?",
		"Ciallo～ 👋 今天想聊点什么？",
		"Hello! How can I help? (∠・ω< )⌒☆",
		"Ciallo (hello) there",
		"",
	} {
		require.False(t, IsCialloReply(reply), reply)
	}
}

func TestRadarProbePayloadCanCarryModelMatchedInstructions(t *testing.T) {
	generic := createOpenAITestPayload("gpt-6-astra", true, openAIAccountTestPayloadOptions{Prompt: "ciallo"})
	require.Equal(t, openai.DefaultInstructions, generic["instructions"])

	// 后台里填的模型名可能带空格和大写（线上就是 "gpt-6 Astra"），也要匹配到 Astra 的提示词。
	for _, model := range []string{"gpt-6-astra", "gpt-6 Astra"} {
		matched := openai.CodexBaseInstructionsForModel(model)
		require.NotEqual(t, openai.DefaultInstructions, matched, model)
		payload := createOpenAITestPayload(model, true, openAIAccountTestPayloadOptions{Prompt: "ciallo", ReasoningEffort: "high", Instructions: matched})
		require.Equal(t, matched, payload["instructions"])
	}
}
