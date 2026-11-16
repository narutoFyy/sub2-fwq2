package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ModelRadarHandler exposes the shared OpenAI group radar. Public reads are
// served by the same handler but always use the redacted service view.
type ModelRadarHandler struct{ radar *service.ModelRadarService }

func NewModelRadarHandler(radar *service.ModelRadarService) *ModelRadarHandler {
	return &ModelRadarHandler{radar: radar}
}

func (h *ModelRadarHandler) PublicOverview(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	data, err := h.radar.PublicOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ModelRadarHandler) AdminOverview(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	data, err := h.radar.AdminOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

// visible 用指针接：旧页面不带这个字段时按「展示」处理，不会把分组悄悄藏掉。
type modelRadarConfigItem struct {
	service.ModelRadarConfig
	Visible *bool `json:"visible"`
}

type modelRadarConfigsRequest struct {
	Configs []*modelRadarConfigItem `json:"configs"`
}

func (h *ModelRadarHandler) UpdateConfigs(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	var req modelRadarConfigsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid model radar config: "+err.Error())
		return
	}
	configs := make([]*service.ModelRadarConfig, 0, len(req.Configs))
	for _, item := range req.Configs {
		if item == nil {
			continue
		}
		cfg := item.ModelRadarConfig
		cfg.Visible = item.Visible == nil || *item.Visible
		configs = append(configs, &cfg)
	}
	data, err := h.radar.UpdateConfigs(c.Request.Context(), configs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"configs": data})
}

func (h *ModelRadarHandler) RunNow(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	var req struct {
		GroupIDs []int64 `json:"group_ids"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid group_ids: "+err.Error())
			return
		}
	}
	count, err := h.radar.RunNowInBackground(c.Request.Context(), req.GroupIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"groups_started": count})
}

func (h *ModelRadarHandler) ReviewResult(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid result id")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid review status: "+err.Error())
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	result, err := h.radar.ReviewResult(c.Request.Context(), id, req.Status, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ModelRadarHandler) Results(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	data, err := h.radar.AdminOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ModelRadarHandler) UpdateSettings(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	req := service.DefaultModelRadarSettings()
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid model radar settings: "+err.Error())
		return
	}
	data, err := h.radar.UpdateSettings(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ModelRadarHandler) DeleteResult(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid result id")
		return
	}
	if err := h.radar.DeleteResult(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": 1})
}

func (h *ModelRadarHandler) ClearGroupResults(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid group id")
		return
	}
	count, err := h.radar.ClearGroupResults(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": count})
}

func (h *ModelRadarHandler) ClearAllResults(c *gin.Context) {
	if h == nil || h.radar == nil {
		response.ErrorFrom(c, service.ErrModelRadarUnavailable)
		return
	}
	count, err := h.radar.ClearAllResults(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": count})
}
