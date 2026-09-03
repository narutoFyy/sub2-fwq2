package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type OAuthAccountMonitorHandler struct {
	service *service.OAuthAccountMonitorService
}

func NewOAuthAccountMonitorHandler(svc *service.OAuthAccountMonitorService) *OAuthAccountMonitorHandler {
	return &OAuthAccountMonitorHandler{service: svc}
}

func (h *OAuthAccountMonitorHandler) GetOverview(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	selectedIDs, err := parseOAuthMonitorAccountIDs(c.QueryArray("account_ids"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	overview, err := h.service.GetOverview(c.Request.Context(), selectedIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	overview.PushPlus = maskedOAuthMonitorPushPlus(overview.PushPlus)
	response.Success(c, overview)
}

func (h *OAuthAccountMonitorHandler) GetConfig(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	cfg, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) UpdateConfig(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	var req service.OAuthAccountMonitorConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	cfg, err := h.service.UpdateConfig(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) AddAccounts(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	var req struct {
		AccountIDs []int64 `json:"account_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	cfg, err := h.service.AddAccounts(c.Request.Context(), req.AccountIDs)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) RemoveAccounts(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	var req struct {
		AccountIDs []int64 `json:"account_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	cfg, err := h.service.RemoveAccounts(c.Request.Context(), req.AccountIDs)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) GetStates(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	cfg, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	states, err := h.service.GetStates(c.Request.Context(), cfg.AccountIDs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, states)
}

func (h *OAuthAccountMonitorHandler) Run(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	result, err := h.service.RunCycle(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *OAuthAccountMonitorHandler) GetPushPlus(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	cfg, err := h.service.GetPushPlusConfig(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, maskedOAuthMonitorPushPlus(cfg))
}

func (h *OAuthAccountMonitorHandler) UpdatePushPlus(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	var req struct {
		Enabled  bool   `json:"enabled"`
		Token    string `json:"token"`
		Topic    string `json:"topic"`
		Template string `json:"template"`
		Channel  string `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if len(req.Token) >= 4 && req.Token[len(req.Token)-4:] == "****" {
		existing, err := h.service.GetPushPlusConfig(c.Request.Context())
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		req.Token = existing.Token
	}
	cfg, err := h.service.UpdatePushPlusConfig(c.Request.Context(), &service.OAuthMonitorPushPlusConfig{Enabled: req.Enabled, Token: req.Token, Topic: req.Topic, Template: req.Template, Channel: req.Channel})
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, maskedOAuthMonitorPushPlus(cfg))
}

func (h *OAuthAccountMonitorHandler) GetEmail(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	cfg, err := h.service.GetEmailConfig(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) UpdateEmail(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable")
		return
	}
	var req service.OAuthMonitorEmailConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	cfg, err := h.service.UpdateEmailConfig(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, cfg)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maskedOAuthMonitorPushPlus(cfg *service.OAuthMonitorPushPlusConfig) *service.OAuthMonitorPushPlusConfig {
	if cfg == nil {
		return &service.OAuthMonitorPushPlusConfig{}
	}
	masked := *cfg
	if masked.Token != "" {
		masked.Token = masked.Token[:minInt(4, len(masked.Token))] + "****"
	}
	return &masked
}

func parseOAuthMonitorAccountIDs(values []string) ([]int64, error) {
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid account_id %q", part)
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}
