package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) GetOpenAILowCostProbeSettings(c *gin.Context) {
	if h == nil || h.openAILowCostProbe == nil {
		response.ErrorFrom(c, service.ErrOpenAILowCostProbeUnavailable)
		return
	}
	settings, err := h.openAILowCostProbe.GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *AccountHandler) UpdateOpenAILowCostProbeSettings(c *gin.Context) {
	if h == nil || h.openAILowCostProbe == nil {
		response.ErrorFrom(c, service.ErrOpenAILowCostProbeUnavailable)
		return
	}
	var req service.OpenAILowCostProbeSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings, err := h.openAILowCostProbe.UpdateSettings(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *AccountHandler) GetOpenAILowCostProbeStates(c *gin.Context) {
	if h == nil || h.openAILowCostProbe == nil {
		response.ErrorFrom(c, service.ErrOpenAILowCostProbeUnavailable)
		return
	}
	states, err := h.openAILowCostProbe.GetStates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, states)
}

func (h *AccountHandler) RunOpenAILowCostProbe(c *gin.Context) {
	if h == nil || h.openAILowCostProbe == nil {
		response.ErrorFrom(c, service.ErrOpenAILowCostProbeUnavailable)
		return
	}
	var req struct {
		AccountIDs []int64 `json:"account_ids"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}
	for _, accountID := range req.AccountIDs {
		if accountID <= 0 {
			response.BadRequest(c, "account_ids must contain positive IDs")
			return
		}
	}
	result, err := h.openAILowCostProbe.RunManual(c.Request.Context(), req.AccountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
