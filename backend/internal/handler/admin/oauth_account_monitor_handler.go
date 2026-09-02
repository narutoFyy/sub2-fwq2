package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type OAuthAccountMonitorHandler struct { service *service.OAuthAccountMonitorService }

func NewOAuthAccountMonitorHandler(svc *service.OAuthAccountMonitorService) *OAuthAccountMonitorHandler {
	return &OAuthAccountMonitorHandler{service: svc}
}

func (h *OAuthAccountMonitorHandler) GetConfig(c *gin.Context) {
	if h == nil || h.service == nil { response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable"); return }
	cfg, err := h.service.GetConfig(c.Request.Context()); if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) UpdateConfig(c *gin.Context) {
	if h == nil || h.service == nil { response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable"); return }
	var req service.OAuthAccountMonitorConfig
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "invalid request body"); return }
	cfg, err := h.service.UpdateConfig(c.Request.Context(), &req); if err != nil { response.Error(c, http.StatusBadRequest, err.Error()); return }
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) AddAccounts(c *gin.Context) {
	if h == nil || h.service == nil { response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable"); return }
	var req struct { AccountIDs []int64 `json:"account_ids"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "invalid request body"); return }
	cfg, err := h.service.AddAccounts(c.Request.Context(), req.AccountIDs); if err != nil { response.Error(c, http.StatusBadRequest, err.Error()); return }
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) RemoveAccounts(c *gin.Context) {
	if h == nil || h.service == nil { response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable"); return }
	var req struct { AccountIDs []int64 `json:"account_ids"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "invalid request body"); return }
	cfg, err := h.service.RemoveAccounts(c.Request.Context(), req.AccountIDs); if err != nil { response.Error(c, http.StatusBadRequest, err.Error()); return }
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) GetStates(c *gin.Context) {
	if h == nil || h.service == nil { response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable"); return }
	cfg, err := h.service.GetConfig(c.Request.Context()); if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	states, err := h.service.GetStates(c.Request.Context(), cfg.AccountIDs); if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, states)
}

func (h *OAuthAccountMonitorHandler) GetPushPlus(c *gin.Context) {
	if h == nil || h.service == nil { response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable"); return }
	cfg, err := h.service.GetPushPlusConfig(c.Request.Context()); if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	if cfg.Token != "" { cfg.Token = cfg.Token[:minInt(4, len(cfg.Token))] + "****" }
	response.Success(c, cfg)
}

func (h *OAuthAccountMonitorHandler) UpdatePushPlus(c *gin.Context) {
	if h == nil || h.service == nil { response.Error(c, http.StatusServiceUnavailable, "monitor service unavailable"); return }
	var req struct { Enabled bool `json:"enabled"`; Token string `json:"token"`; Topic string `json:"topic"`; Template string `json:"template"`; Channel string `json:"channel"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "invalid request body"); return }
	if len(req.Token) >= 4 && req.Token[len(req.Token)-4:] == "****" {
		existing, err := h.service.GetPushPlusConfig(c.Request.Context())
		if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
		req.Token = existing.Token
	}
	cfg, err := h.service.UpdatePushPlusConfig(c.Request.Context(), &service.OAuthMonitorPushPlusConfig{Enabled: req.Enabled, Token: req.Token, Topic: req.Topic, Template: req.Template, Channel: req.Channel}); if err != nil { response.Error(c, http.StatusBadRequest, err.Error()); return }
	if cfg.Token != "" { cfg.Token = cfg.Token[:minInt(4, len(cfg.Token))] + "****" }
	response.Success(c, cfg)
}

func minInt(a, b int) int { if a < b { return a }; return b }
