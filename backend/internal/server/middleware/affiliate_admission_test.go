//go:build unit

package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type affiliateAdmissionUserRepo struct {
	service.UserRepository
	state service.AffiliateUserAdmission
	err   error
	ip    string
}

func (r *affiliateAdmissionUserRepo) PrepareAffiliateUserForRequest(_ context.Context, _ int64, clientIP string) (*service.AffiliateUserAdmission, error) {
	r.ip = clientIP
	return &r.state, r.err
}

func TestAffiliateExpiryAdmissionAndTrustedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, google := range []bool{false, true} {
		for _, failing := range []bool{false, true} {
			name := "standard"
			if google {
				name = "google"
			}
			if failing {
				name += "_database_failure"
			}
			t.Run(name, func(t *testing.T) {
				cached := &service.APIKey{ID: 1, UserID: 2, Key: "admission-key", Status: service.StatusActive, User: &service.User{ID: 2, Role: service.RoleUser, Status: service.StatusActive, Balance: 1}}
				keys := &stubApiKeyRepo{getByKey: func(context.Context, string) (*service.APIKey, error) { return cached, nil }}
				users := &affiliateAdmissionUserRepo{state: service.AffiliateUserAdmission{Balance: 0, ExpiredAmount: 1}}
				if failing {
					users.err = errors.New("expiry write unavailable")
				}
				cfg := &config.Config{}
				svc := service.NewAPIKeyService(keys, users, nil, nil, nil, nil, cfg)
				router := gin.New()
				require.NoError(t, router.SetTrustedProxies(nil))
				if google {
					router.Use(APIKeyAuthGoogle(svc, cfg))
				} else {
					router.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(svc, nil, cfg)))
				}
				called := false
				router.POST("/v1/test", func(c *gin.Context) { called = true; c.Status(200) })
				req := httptest.NewRequest(http.MethodPost, "/v1/test", nil)
				req.RemoteAddr = "203.0.113.20:12345"
				req.Header.Set("Authorization", "Bearer admission-key")
				req.Header.Set("X-Forwarded-For", "198.51.100.99")
				req.Header.Set("CF-Connecting-IP", "198.51.100.98")
				out := httptest.NewRecorder()
				router.ServeHTTP(out, req)
				expected := 403
				if failing {
					expected = 503
				}
				require.Equal(t, expected, out.Code, out.Body.String())
				require.False(t, called, "expired funds must be removed before any upstream handler")
				require.Equal(t, "203.0.113.20", users.ip, "untrusted forwarding headers must not supply affiliate risk IP")
				require.Equal(t, 1.0, cached.User.Balance, "shared auth cache snapshot must remain immutable")
			})
		}
	}
}
