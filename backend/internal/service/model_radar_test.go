package service

import (
	"context"
	"testing"
	"time"

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

func (s *modelRadarRepoStub) ListResults(context.Context, *int64, string, int) ([]*ModelRadarResult, error) {
	return s.results, nil
}

func TestModelRadarPublicOverviewRedactsResponseDetails(t *testing.T) {
	reviewedBy := int64(77)
	repo := &modelRadarRepoStub{results: []*ModelRadarResult{{
		ID: 1, GroupID: 10, ModelID: "gpt-5.4", TestType: ModelRadarTestTypeLogic,
		Status: ModelRadarStatusPassed, ResponseText: "secret upstream answer", ErrorMessage: "secret error",
		LatencyMS: 120, DetectedAt: time.Now(), ReviewedBy: &reviewedBy,
	}}}
	groups := &modelRadarGroupRepoStub{groups: []Group{{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive}}}
	svc := NewModelRadarService(repo, nil, groups, nil)

	overview, err := svc.PublicOverview(context.Background())
	require.NoError(t, err)
	require.Len(t, overview.Groups, 1)
	require.NotNil(t, overview.Groups[0].Logic)
	require.Empty(t, overview.Groups[0].Logic.ResponseText)
	require.Empty(t, overview.Groups[0].Logic.ErrorMessage)
	require.Nil(t, overview.Groups[0].Logic.ReviewedBy)
}

func TestModelRadarRejectsNonOpenAIConfig(t *testing.T) {
	groups := &modelRadarGroupRepoStub{groups: []Group{{ID: 10, Name: "Claude", Platform: PlatformAnthropic, Status: StatusActive}}}
	svc := NewModelRadarService(&modelRadarRepoStub{}, nil, groups, nil)

	_, err := svc.UpdateConfigs(context.Background(), []*ModelRadarConfig{{GroupID: 10, Enabled: true, ReasoningEffort: "medium"}})
	require.ErrorIs(t, err, ErrModelRadarOpenAIOnly)
}

func TestModelRadarLogicAnswerMatcher(t *testing.T) {
	require.True(t, logicAnswerPattern.MatchString("Answer: 21"))
	require.True(t, logicAnswerPattern.MatchString("答案：21"))
	require.False(t, logicAnswerPattern.MatchString("Answer: 20"))
}
