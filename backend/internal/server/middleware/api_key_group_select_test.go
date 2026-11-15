//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func multiGroupTestRouter(t *testing.T, extras []*service.Group) (*gin.Engine, *int64) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	claude := &service.Group{ID: 1, Name: "claude", Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	user := &service.User{ID: 7, Role: service.RoleUser, Status: service.StatusActive, Balance: 10, Concurrency: 3}
	apiKey := &service.APIKey{
		ID: 100, UserID: user.ID, Key: "multi-key", Status: service.StatusActive,
		User: user, Group: claude, ExtraGroups: extras,
	}
	apiKey.GroupID = &claude.ID
	for _, g := range extras {
		apiKey.ExtraGroupIDs = append(apiKey.ExtraGroupIDs, g.ID)
	}

	repo := &stubApiKeyRepo{getByKey: func(ctx context.Context, key string) (*service.APIKey, error) {
		if key != apiKey.Key {
			return nil, service.ErrAPIKeyNotFound
		}
		clone := *apiKey
		return &clone, nil
	}}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	svc := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)

	var seen int64
	r := gin.New()
	r.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(svc, nil, cfg)))
	handler := func(c *gin.Context) {
		k, _ := GetAPIKeyFromContext(c)
		seen = *k.GroupID
		body, _ := c.GetRawData()
		c.JSON(http.StatusOK, gin.H{"group_id": *k.GroupID, "group_ctx_platform": k.Group.Platform, "body_len": len(body)})
	}
	r.POST("/v1/chat/completions", handler)
	r.POST("/v1/messages", handler)
	r.GET("/v1/models", handler)
	r.POST("/v1beta/models/:modelAction", handler)
	return r, &seen
}

func doMultiGroupRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer multi-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAPIKeyAuthSelectsGroupByModel(t *testing.T) {
	openai := &service.Group{ID: 2, Name: "gpt", Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	gemini := &service.Group{ID: 3, Name: "gemini", Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	r, seen := multiGroupTestRouter(t, []*service.Group{openai, gemini})

	cases := []struct {
		name, method, path, body string
		wantGroup                int64
	}{
		{"gpt 走 OpenAI 分组", http.MethodPost, "/v1/chat/completions", `{"model":"gpt-5.5","messages":[]}`, 2},
		{"claude 走主分组", http.MethodPost, "/v1/messages", `{"model":"claude-sonnet-4-5","messages":[]}`, 1},
		{"Claude Code 路径调 gpt 也走 OpenAI 分组", http.MethodPost, "/v1/messages", `{"model":"gpt-5.5","messages":[]}`, 2},
		{"Gemini 原生路径取路径里的模型", http.MethodPost, "/v1beta/models/gemini-2.5-pro:generateContent", `{"contents":[]}`, 3},
		{"认不出的别名走主分组", http.MethodPost, "/v1/chat/completions", `{"model":"my-alias"}`, 1},
		{"没有模型的 GET 走主分组", http.MethodGet, "/v1/models", ``, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			*seen = 0
			w := doMultiGroupRequest(r, tc.method, tc.path, tc.body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, tc.wantGroup, *seen)
			// 选组读过请求体后必须回填，下游还能读到完整内容
			if tc.body != "" {
				require.Contains(t, w.Body.String(), `"body_len":`+strconv.Itoa(len(tc.body)))
			}
		})
	}
}

func TestAPIKeyAuthRejectsModelWithoutMatchingGroup(t *testing.T) {
	openai := &service.Group{ID: 2, Name: "gpt", Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	r, _ := multiGroupTestRouter(t, []*service.Group{openai})

	w := doMultiGroupRequest(r, http.MethodPost, "/v1/chat/completions", `{"model":"gemini-2.5-pro"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "gemini-2.5-pro")
}

func TestAPIKeyAuthSingleGroupKeyUnaffected(t *testing.T) {
	r, seen := multiGroupTestRouter(t, nil)
	w := doMultiGroupRequest(r, http.MethodPost, "/v1/chat/completions", `{"model":"gpt-5.5"}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.EqualValues(t, 1, *seen, "没有附加分组的 Key 不做选组，行为与以前一致")
}
