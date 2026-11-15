package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/requestmodel"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// selectAPIKeyGroupForRequest 处理「一把 Key 绑定多个分组」：按本次请求的模型，
// 在 主分组 + 附加分组 里选一个，把 apiKey.Group / GroupID 换成它。
// 之后的订阅检查、白名单、调度、计费都照常读 apiKey.Group，不需要感知多分组。
//
// 返回 false 表示已写出错误响应并 Abort。没有附加分组的 Key 直接放行，零开销。
func selectAPIKeyGroupForRequest(c *gin.Context, apiKey *service.APIKey) bool {
	if !apiKey.HasExtraGroups() || apiKey.User == nil {
		return true
	}

	candidates := make([]*service.Group, 0, 1+len(apiKey.ExtraGroups))
	if apiKey.Group != nil {
		candidates = append(candidates, apiKey.Group)
	}
	candidates = append(candidates, apiKey.ExtraGroups...)
	// 专属分组被收回权限时，对应候选直接剔除（订阅分组由订阅检查把关）。
	allowed := candidates[:0:0]
	for _, g := range candidates {
		if g == nil {
			continue
		}
		if !g.IsSubscriptionType() && !apiKey.User.CanBindGroup(g.ID, g.IsExclusive) {
			continue
		}
		allowed = append(allowed, g)
	}

	model, hint := requestModelForGroupSelection(c)
	selected := service.SelectGroupForModel(allowed, model)
	if selected != nil && model == "" && hint != "" {
		// 拿不到模型（如 Responses WebSocket 的模型在帧里）时按入口协议给个倾向。
		if g := firstGroupOfPlatform(allowed, hint); g != nil {
			selected = g
		}
	}
	if selected == nil {
		if len(allowed) == 0 {
			// 全部候选都不可用：沿用主分组，交给后面的可用性检查给出原有错误。
			return true
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalModelConfiguration)
		MarkIngressRejected(c, IngressRejectModelNotAllowed)
		groupModelAllowlistErrorWriter(c)(c, http.StatusNotFound, fmt.Sprintf("Model %q is not available for this API key", model))
		c.Abort()
		return false
	}

	if apiKey.Group == nil || selected.ID != apiKey.Group.ID {
		id := selected.ID
		apiKey.Group = selected
		apiKey.GroupID = &id
		// 快照里的 RPM 覆盖值是主分组的；换组后清空，让限流回退到按实际分组查询。
		apiKey.User.UserGroupRPMOverride = nil
	}
	return true
}

func firstGroupOfPlatform(groups []*service.Group, platform string) *service.Group {
	for _, g := range groups {
		if g.Platform == platform || g.Platform == service.PlatformComposite {
			return g
		}
	}
	return nil
}

// requestModelForGroupSelection 取本次请求的模型名（路径参数 → 请求体 → 查询参数）。
// 第二个返回值是取不到模型时的平台倾向（目前只有 Responses 入口）。
// 读取请求体失败时返回空模型，由后续中间件按原有口径报错。
func requestModelForGroupSelection(c *gin.Context) (string, string) {
	if c.Request == nil {
		return "", ""
	}
	if model := groupModelAllowlistModelFromParams(c); model != "" {
		return model, ""
	}
	switch c.Request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		candidates := groupSelectionModelFromBody(c)
		for _, m := range candidates {
			if m = strings.TrimSpace(m); m != "" {
				return m, ""
			}
		}
	}
	if model := strings.TrimSpace(c.Query("model")); model != "" {
		return model, ""
	}
	if isResponsesWebSocketRoute(c) {
		return "", service.PlatformOpenAI
	}
	return "", ""
}

// groupSelectionModelFromBody 与白名单中间件同口径读取请求体并回填，读失败不 Abort。
func groupSelectionModelFromBody(c *gin.Context) []string {
	body, err := httputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		return nil
	}
	requestmodel.ResetRequestBody(c.Request, body)
	return requestmodel.FromBodyCandidates(c.FullPath(), c.GetHeader("Content-Type"), body)
}
