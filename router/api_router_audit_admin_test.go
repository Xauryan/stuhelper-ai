package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setAuditRouterTestSession(role int) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 123)
		session.Set("username", "audit-route-test")
		session.Set("role", role)
		session.Set("status", common.UserStatusEnabled)
		if err := session.Save(); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Request.Header.Set("StuHelper-AI-User", "123")
		c.Next()
	}
}

func TestAuditAdminRouteRegistrationAndWriteGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	common.GlobalApiRateLimitEnable = false
	common.RedisEnabled = false

	server := gin.New()
	server.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	server.Use(setAuditRouterTestSession(common.RoleAuditAdminUser))

	require.NotPanics(t, func() {
		SetApiRouter(server)
	})

	routes := map[string]bool{}
	for _, route := range server.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"GET /api/user/",
		"GET /api/user/search",
		"GET /api/channel/",
		"GET /api/channel/search",
		"GET /api/redemption/",
		"GET /api/redemption/search",
		"GET /api/models/",
		"GET /api/models/search",
		"GET /api/subscription/admin/plans",
	} {
		require.True(t, routes[route], "expected audit-readable route %s to be registered", route)
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "user detail blocked", method: http.MethodGet, path: "/api/user/1"},
		{name: "user update blocked", method: http.MethodPut, path: "/api/user/"},
		{name: "user manage blocked", method: http.MethodPost, path: "/api/user/manage"},
		{name: "channel detail blocked", method: http.MethodGet, path: "/api/channel/1"},
		{name: "channel write blocked", method: http.MethodPost, path: "/api/channel/"},
		{name: "redemption detail blocked", method: http.MethodGet, path: "/api/redemption/1"},
		{name: "model detail blocked", method: http.MethodGet, path: "/api/models/1"},
		{name: "subscription plan write blocked", method: http.MethodPost, path: "/api/subscription/admin/plans"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			server.ServeHTTP(recorder, req)
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), `"success":false`)
			require.Contains(t, recorder.Body.String(), "auth.insufficient_privilege")
		})
	}
}
