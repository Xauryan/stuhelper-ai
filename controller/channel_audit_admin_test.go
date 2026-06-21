package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/middleware"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelAuditAdminControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalRedisEnabled := common.RedisEnabled
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalDB := model.DB
	originalLogDB := model.LOG_DB

	gin.SetMode(gin.TestMode)
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initModelListColumnNames(t)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.User{}, &model.Log{}))

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

func setChannelAuditAdminTestSession(role int) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 123)
		session.Set("username", "channel-audit-test")
		session.Set("role", role)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		if err := session.Save(); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Request.Header.Set("StuHelper-AI-User", "123")
		c.Next()
	}
}

func setupChannelAuditAdminTestRouter(t *testing.T, role int) *gin.Engine {
	t.Helper()
	db := setupChannelAuditAdminControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "channel-audit-test",
		Password:    "password",
		DisplayName: "Channel Audit Test",
		Role:        role,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Id:     11,
		Type:   1,
		Name:   "private-channel-name",
		Key:    "sk-secret",
		Status: common.ChannelStatusEnabled,
		Models: "gpt-4o",
		Group:  "default",
	}).Error)

	server := gin.New()
	server.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	server.Use(setChannelAuditAdminTestSession(role))
	channelRoute := server.Group("/api/channel")
	channelRoute.Use(middleware.AdminAuth())
	{
		channelRoute.GET("/", GetAllChannels)
		channelRoute.GET("/search", SearchChannels)
		channelRoute.GET("/models", ChannelListModels)
		channelRoute.GET("/models_enabled", EnabledListModels)
		channelRoute.GET("/ops", GetChannelOps)
		channelRoute.GET("/monitor/summary", GetChannelMonitorSummary)
	}
	return server
}

func TestAuditAdminCannotAccessChannelManagement(t *testing.T) {
	common.GlobalApiRateLimitEnable = false
	server := setupChannelAuditAdminTestRouter(t, common.RoleAuditAdminUser)

	for _, path := range []string{
		"/api/channel/?p=1&page_size=20",
		"/api/channel/search?keyword=private-channel-name&p=1&page_size=20",
		"/api/channel/models",
		"/api/channel/models_enabled",
		"/api/channel/ops",
		"/api/channel/monitor/summary",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			server.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), `"success":false`)
			require.Contains(t, recorder.Body.String(), "auth.insufficient_privilege")
			require.NotContains(t, recorder.Body.String(), "private-channel-name")
			require.NotContains(t, recorder.Body.String(), "sk-secret")
		})
	}
}

func TestAdminCanAccessChannelManagement(t *testing.T) {
	common.GlobalApiRateLimitEnable = false
	server := setupChannelAuditAdminTestRouter(t, common.RoleAdminUser)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/channel/?p=1&page_size=20", nil)
	server.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), "private-channel-name")
	require.NotContains(t, recorder.Body.String(), "sk-secret")
}
