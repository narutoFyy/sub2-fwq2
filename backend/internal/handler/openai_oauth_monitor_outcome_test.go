package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOAuthMonitorRequestIDIsStableAndTurnScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/openai/v1/responses", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.ClientRequestID, "client-request-7"))
	c.Request = req

	require.Equal(t, "client-request-7", oauthMonitorRequestID(c, 0))
	require.Equal(t, "client-request-7:turn:1", oauthMonitorRequestID(c, 1))
	require.Equal(t, "client-request-7:turn:2", oauthMonitorRequestID(c, 2))
	require.Equal(t, "client-request-7", oauthMonitorRequestID(c, 0))
}
