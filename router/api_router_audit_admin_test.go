package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func setupAuditRouterTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalRedisEnabled := common.RedisEnabled
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalDB := model.DB
	originalLogDB := model.LOG_DB

	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.ReferralCommission{},
		&model.TopUp{},
		&model.SubscriptionOrder{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		common.RedisEnabled = originalRedisEnabled
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		model.DB = originalDB
		model.LOG_DB = originalLogDB
	})

	return db
}

func TestAuditAdminRouteRegistrationAndWriteGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	common.GlobalApiRateLimitEnable = false
	db := setupAuditRouterTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "audit-route-test",
		Password:    "password",
		DisplayName: "Audit Route Test",
		Role:        common.RoleAuditAdminUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}).Error)

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
		"GET /api/redemption/",
		"GET /api/redemption/search",
		"GET /api/models/",
		"GET /api/models/search",
		"GET /api/subscription/admin/plans",
		"GET /api/log/",
		"GET /api/log/stat",
		"GET /api/log/search",
		"GET /api/log/channel_affinity_usage_cache",
		"GET /api/log/channel_monitor/summary",
		"GET /api/mj/",
		"GET /api/task/",
		"GET /api/user/referrals",
		"GET /api/user/referrals/:invitee_id/commissions",
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
		{name: "channel list blocked", method: http.MethodGet, path: "/api/channel/"},
		{name: "channel search blocked", method: http.MethodGet, path: "/api/channel/search"},
		{name: "channel monitor blocked", method: http.MethodGet, path: "/api/channel/monitor/summary"},
		{name: "channel detail blocked", method: http.MethodGet, path: "/api/channel/1"},
		{name: "channel breaker reset blocked", method: http.MethodPost, path: "/api/channel/1/breaker/reset"},
		{name: "channel write blocked", method: http.MethodPost, path: "/api/channel/"},
		{name: "prefill group list blocked", method: http.MethodGet, path: "/api/prefill_group/"},
		{name: "redemption detail blocked", method: http.MethodGet, path: "/api/redemption/1"},
		{name: "model detail blocked", method: http.MethodGet, path: "/api/models/1"},
		{name: "subscription plan write blocked", method: http.MethodPost, path: "/api/subscription/admin/plans"},
		{name: "log cleanup blocked", method: http.MethodDelete, path: "/api/log/?target_timestamp=1"},
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

func TestAuditAdminCanReadLogsAndReferrals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	common.GlobalApiRateLimitEnable = false
	db := setupAuditRouterTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "audit-route-test",
		Password:    "password",
		DisplayName: "Audit Route Test",
		Role:        common.RoleAuditAdminUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}).Error)

	server := gin.New()
	server.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	server.Use(setAuditRouterTestSession(common.RoleAuditAdminUser))

	SetApiRouter(server)

	for _, tc := range []struct {
		name       string
		path       string
		wantAllow  bool
		wantSearch bool
	}{
		{name: "all logs", path: "/api/log/?p=1&page_size=10", wantAllow: true},
		{name: "log stats", path: "/api/log/stat", wantAllow: true},
		{name: "deprecated all log search", path: "/api/log/search", wantSearch: true},
		{name: "drawing logs", path: "/api/mj/?p=1&page_size=10", wantAllow: true},
		{name: "task logs", path: "/api/task/?p=1&page_size=10", wantAllow: true},
		{name: "referral records", path: "/api/user/referrals?p=1&page_size=10", wantAllow: true},
		{name: "referral commissions", path: "/api/user/referrals/1/commissions?p=1&page_size=10", wantAllow: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			server.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.NotContains(t, recorder.Body.String(), "auth.insufficient_privilege")
			if tc.wantAllow {
				require.Contains(t, recorder.Body.String(), `"success":true`)
			}
			if tc.wantSearch {
				require.Contains(t, recorder.Body.String(), "该接口已废弃")
			}
		})
	}
}
