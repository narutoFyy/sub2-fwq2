//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseModelPlazaHiddenGroupIDs(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []int64
	}{
		{"empty string hides nothing", "", []int64{}},
		{"blank hides nothing", "   ", []int64{}},
		{"empty array", "[]", []int64{}},
		{"invalid json hides nothing", "{oops", []int64{}},
		{"wrong shape hides nothing", `["a","b"]`, []int64{}},
		{"sorted and de-duplicated", "[7,3,7,12,3]", []int64{3, 7, 12}},
		{"non-positive ids dropped", "[0,-4,5]", []int64{5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, parseModelPlazaHiddenGroupIDs(tc.raw))
		})
	}
}

func TestGetModelPlazaRuntime_HiddenGroupIDs(t *testing.T) {
	repo := &settingRepoStub{values: map[string]string{
		SettingKeyModelPlazaEnabled:        "true",
		SettingKeyModelPlazaHiddenGroupIDs: "[3,9]",
	}}
	svc := NewSettingService(repo, &config.Config{})

	rt := svc.GetModelPlazaRuntime(context.Background())
	require.True(t, rt.Enabled)
	require.Len(t, rt.HiddenGroupIDs, 2)
	require.Contains(t, rt.HiddenGroupIDs, int64(3))
	require.Contains(t, rt.HiddenGroupIDs, int64(9))
	require.NotContains(t, rt.HiddenGroupIDs, int64(4))

	// 未设置过（老数据）：不隐藏任何分组，也不能 panic（nil map 读取安全）。
	legacy := NewSettingService(&settingRepoStub{values: map[string]string{SettingKeyModelPlazaEnabled: "true"}}, &config.Config{})
	rt = legacy.GetModelPlazaRuntime(context.Background())
	require.True(t, rt.Enabled)
	require.Empty(t, rt.HiddenGroupIDs)
}

func TestBuildSystemSettingsUpdates_ModelPlazaHiddenGroupIDs(t *testing.T) {
	svc := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})

	updates, err := svc.buildSystemSettingsUpdates(context.Background(), &SystemSettings{
		ModelPlazaEnabled:        true,
		ModelPlazaHiddenGroupIDs: []int64{9, 3, 9, -1},
	})
	require.NoError(t, err)
	require.JSONEq(t, "[3,9]", updates[SettingKeyModelPlazaHiddenGroupIDs])

	// nil 与空切片都落成 "[]"，保证再次读取得到「不隐藏」而不是 "null"。
	updates, err = svc.buildSystemSettingsUpdates(context.Background(), &SystemSettings{})
	require.NoError(t, err)
	var decoded []int64
	require.NoError(t, json.Unmarshal([]byte(updates[SettingKeyModelPlazaHiddenGroupIDs]), &decoded))
	require.Empty(t, decoded)
	require.Equal(t, "[]", updates[SettingKeyModelPlazaHiddenGroupIDs])
}
