package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func selTestGroup(id int64, platform string) *Group {
	return &Group{ID: id, Platform: platform, Status: StatusActive}
}

func TestSelectGroupForModel(t *testing.T) {
	claude := selTestGroup(1, PlatformAnthropic)
	gpt := selTestGroup(2, PlatformOpenAI)
	gpt2 := selTestGroup(3, PlatformOpenAI)
	gemini := selTestGroup(4, PlatformGemini)
	composite := selTestGroup(5, PlatformComposite)
	disabled := selTestGroup(6, PlatformGrok)
	disabled.Status = "disabled"

	pick := func(groups []*Group, model string) int64 {
		g := SelectGroupForModel(groups, model)
		if g == nil {
			return 0
		}
		return g.ID
	}

	t.Run("按模型所属平台选分组", func(t *testing.T) {
		groups := []*Group{claude, gpt, gemini}
		require.EqualValues(t, 1, pick(groups, "claude-sonnet-4-5"))
		require.EqualValues(t, 2, pick(groups, "gpt-5.5"))
		require.EqualValues(t, 2, pick(groups, "o3-mini"))
		require.EqualValues(t, 4, pick(groups, "gemini-2.5-pro"))
		require.EqualValues(t, 1, pick(groups, "anthropic/claude-opus-4"))
	})

	t.Run("同平台多个分组取排在前面的", func(t *testing.T) {
		require.EqualValues(t, 2, pick([]*Group{claude, gpt, gpt2}, "gpt-5"))
		require.EqualValues(t, 3, pick([]*Group{claude, gpt2, gpt}, "gpt-5"))
	})

	t.Run("同平台分组被白名单挡住时让给放行的那个", func(t *testing.T) {
		strict := selTestGroup(7, PlatformOpenAI)
		strict.ModelAllowlist = GroupModelAllowlist{Enabled: true, Models: []string{"gpt-4o"}}
		require.EqualValues(t, 3, pick([]*Group{claude, strict, gpt2}, "gpt-5"))
		require.EqualValues(t, 7, pick([]*Group{claude, strict, gpt2}, "gpt-4o"))
		// 都不放行：取第一个，让白名单中间件按它的口径拒绝
		require.EqualValues(t, 7, pick([]*Group{claude, strict}, "gpt-5"))
	})

	t.Run("没有同平台分组时落到 composite，再没有就返回空", func(t *testing.T) {
		require.EqualValues(t, 5, pick([]*Group{claude, composite}, "gpt-5"))
		require.EqualValues(t, 0, pick([]*Group{claude, gpt}, "gemini-2.5-pro"))
	})

	t.Run("判断不了平台的模型走主分组", func(t *testing.T) {
		require.EqualValues(t, 1, pick([]*Group{claude, gpt}, "my-private-alias"))
		require.EqualValues(t, 1, pick([]*Group{claude, gpt}, ""))
	})

	t.Run("别名被某个分组白名单显式放行时由它接手", func(t *testing.T) {
		aliasGroup := selTestGroup(8, PlatformOpenAI)
		aliasGroup.ModelAllowlist = GroupModelAllowlist{Enabled: true, Models: []string{"my-private-alias"}}
		require.EqualValues(t, 8, pick([]*Group{claude, aliasGroup}, "my-private-alias"))
	})

	t.Run("单个可用分组直接用，停用的分组被忽略", func(t *testing.T) {
		require.EqualValues(t, 1, pick([]*Group{claude}, "gpt-5"))
		require.EqualValues(t, 2, pick([]*Group{disabled, gpt}, "grok-4"))
		require.EqualValues(t, 0, pick([]*Group{disabled}, "grok-4"))
		require.EqualValues(t, 0, pick(nil, "gpt-5"))
	})
}

func TestAPIKeyAllGroupIDs(t *testing.T) {
	one, two := int64(1), int64(2)
	k := &APIKey{GroupID: &one, ExtraGroupIDs: []int64{2, 1, 3, 2}}
	require.Equal(t, []int64{1, 2, 3}, k.AllGroupIDs())
	require.Equal(t, []int64{2}, (&APIKey{GroupID: &two}).AllGroupIDs())
	require.Empty(t, (&APIKey{}).AllGroupIDs())
	require.False(t, k.HasExtraGroups(), "HasExtraGroups 看的是已加载的分组对象")
}

// 附加分组必须经认证快照（含 L2 JSON 往返）原样还原。
func TestAPIKeyAuthSnapshotExtraGroupsRoundtrip(t *testing.T) {
	svc := &APIKeyService{}
	apiKey := profitAuthTestAPIKey()
	apiKey.ExtraGroupIDs = []int64{61, 62}
	apiKey.ExtraGroups = []*Group{
		{ID: 61, Name: "claude-pool", Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true, RateMultiplier: 1.5,
			ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"claude-*"}}},
		{ID: 62, Name: "grok-pool", Platform: PlatformGrok, Status: StatusActive, Hydrated: true, ProfitControlEnabled: true, ProfitMinMargin: 0.3},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)
	require.NotNil(t, snapshot)
	payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: snapshot})
	require.NoError(t, err)
	var restored APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal(payload, &restored))

	got, used, err := svc.applyAuthCacheEntry("sk-x", &restored)
	require.NoError(t, err)
	require.True(t, used)
	require.Equal(t, []int64{61, 62}, got.ExtraGroupIDs)
	require.Len(t, got.ExtraGroups, 2)
	require.True(t, got.HasExtraGroups())
	require.Equal(t, int64(61), got.ExtraGroups[0].ID)
	require.Equal(t, PlatformAnthropic, got.ExtraGroups[0].Platform)
	require.True(t, got.ExtraGroups[0].Hydrated)
	require.InDelta(t, 1.5, got.ExtraGroups[0].RateMultiplier, 1e-12)
	require.True(t, got.ExtraGroups[0].ModelAllowlist.Enabled)
	require.True(t, got.ExtraGroups[1].ProfitControlEnabled)
	require.InDelta(t, 0.3, got.ExtraGroups[1].ProfitMinMargin, 1e-12)
	// 主分组不受影响
	require.NotNil(t, got.Group)
	require.True(t, got.Group.ProfitControlEnabled)
}
