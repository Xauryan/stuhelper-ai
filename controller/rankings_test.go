package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withHeaderNavModulesForRankingsTest(t *testing.T, value string) {
	t.Helper()
	originalOptionMap := common.OptionMap
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{"HeaderNavModules": value}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})
}

func TestRankingsConfigParsesEnabledAndRequireAuth(t *testing.T) {
	withHeaderNavModulesForRankingsTest(t, `{"rankings":{"enabled":false,"requireAuth":true}}`)

	config := getRankingsAccessConfig()

	assert.False(t, config.enabled)
	assert.True(t, config.requireAuth)
}

func TestGetUserRankingsRejectsDisabledRankings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withHeaderNavModulesForRankingsTest(t, `{"rankings":{"enabled":false,"requireAuth":false}}`)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/rankings/users", nil)

	GetUserRankings(ctx)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "rankings is disabled")
}

func TestGetUserRankingsRequiresAuthenticatedUserWhenConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withHeaderNavModulesForRankingsTest(t, `{"rankings":{"enabled":true,"requireAuth":true}}`)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/rankings/users", nil)

	GetUserRankings(ctx)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "login required")
}

func TestRankingsRequireAuthAcceptsOptionalSessionAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withHeaderNavModulesForRankingsTest(t, `{"rankings":{"enabled":true,"requireAuth":true}}`)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("rankings-test-secret"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 123)
		session.Set("username", "rankings-user")
		session.Set("role", common.RoleCommonUser)
		session.Set("status", common.UserStatusEnabled)
		require.NoError(t, session.Save())
		c.Next()
	})
	router.GET("/api/rankings/users", middleware.TryUserAuth(), func(c *gin.Context) {
		if !enforceRankingsAccess(c) {
			return
		}
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/rankings/users", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Equal(t, "ok", recorder.Body.String())
}
