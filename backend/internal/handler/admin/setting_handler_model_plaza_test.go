//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDiffSettings_DetectsModelPlazaHiddenGroupChange(t *testing.T) {
	before := &service.SystemSettings{ModelPlazaHiddenGroupIDs: []int64{3}}
	after := &service.SystemSettings{ModelPlazaHiddenGroupIDs: []int64{3, 9}}
	require.Contains(t, diffSettings(before, after, nil, nil, UpdateSettingsRequest{}), "model_plaza_hidden_group_ids")

	same := &service.SystemSettings{ModelPlazaHiddenGroupIDs: []int64{3, 9}}
	require.NotContains(t, diffSettings(after, same, nil, nil, UpdateSettingsRequest{}), "model_plaza_hidden_group_ids")
}

// 管理员在后台勾选「不在广场展示」的分组：PUT 写入、GET 读回；省略字段保持原值，传空数组才清空。
func TestSettingHandler_ModelPlazaHiddenGroups_PutGetRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{service.SettingKeyPromoCodeEnabled: "true"}}
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil)

	put := func(body map[string]any) {
		t.Helper()
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(raw))
		c.Request.Header.Set("Content-Type", "application/json")
		handler.UpdateSettings(c)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}
	get := func() any {
		t.Helper()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
		handler.GetSettings(c)
		require.Equal(t, http.StatusOK, rec.Code)
		var resp response.Response
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		return data["model_plaza_hidden_group_ids"]
	}

	// 乱序、重复的输入会被整理成升序去重后存库。
	put(map[string]any{"model_plaza_enabled": true, "model_plaza_hidden_group_ids": []int64{9, 3, 3}})
	require.JSONEq(t, "[3,9]", repo.values[service.SettingKeyModelPlazaHiddenGroupIDs])
	require.Equal(t, []any{float64(3), float64(9)}, get())

	// 别的设置项保存时没带这个字段：保持原值，不能被清空。
	put(map[string]any{"model_plaza_description": "hello"})
	require.JSONEq(t, "[3,9]", repo.values[service.SettingKeyModelPlazaHiddenGroupIDs])
	require.Equal(t, []any{float64(3), float64(9)}, get())

	// 显式传空数组 = 全部展示；读回的是 []，不是 null。
	put(map[string]any{"model_plaza_hidden_group_ids": []int64{}})
	require.JSONEq(t, "[]", repo.values[service.SettingKeyModelPlazaHiddenGroupIDs])
	require.Equal(t, []any{}, get())
}
