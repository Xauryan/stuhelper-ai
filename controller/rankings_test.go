package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/middleware"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func setupRankingsControllerTestDB(t *testing.T) *gorm.DB {
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
		&model.TopUp{},
		&model.Redemption{},
	))
	service.ClearUserRankingsCache()

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
		service.ClearUserRankingsCache()
	})

	return db
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

func TestGetUserRankingsRejectsInvalidMetric(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withHeaderNavModulesForRankingsTest(t, `{"rankings":{"enabled":true,"requireAuth":false}}`)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/rankings/users?metric=money", nil)

	GetUserRankings(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "invalid ranking metric")
}

func TestRankingsRequireAuthAcceptsOptionalSessionAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withHeaderNavModulesForRankingsTest(t, `{"rankings":{"enabled":true,"requireAuth":true}}`)
	db := setupRankingsControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "rankings-user",
		Password:    "password",
		DisplayName: "Rankings User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}).Error)

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

func TestGetUserRankingsIncludesMeForLoggedInUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withHeaderNavModulesForRankingsTest(t, `{"rankings":{"enabled":true,"requireAuth":false}}`)
	db := setupRankingsControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       123,
		Username: "rankings-user",
		Password: "password",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		AffCode:  "rankings-aff-123",
		Group:    "default",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId:           123,
		CreatedAt:        time.Now().Unix() - 60,
		Type:             model.LogTypeConsume,
		Quota:            300,
		PromptTokens:     40,
		CompletionTokens: 80,
	}).Error)

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
	router.GET("/api/rankings/users", middleware.TryUserAuth(), GetUserRankings)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/rankings/users?period=week", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Period      string `json:"period"`
			Consumption []struct {
				Rank         int    `json:"rank"`
				Display      string `json:"display"`
				TotalQuota   int64  `json:"total_quota"`
				TotalTokens  int64  `json:"total_tokens"`
				RequestCount int64  `json:"request_count"`
			} `json:"consumption"`
			Me *struct {
				Rank         int    `json:"rank"`
				Display      string `json:"display"`
				TotalQuota   int64  `json:"total_quota"`
				TotalTokens  int64  `json:"total_tokens"`
				RequestCount int64  `json:"request_count"`
			} `json:"me"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.True(t, payload.Success)
	require.NotNil(t, payload.Data.Me, "logged-in viewer should receive me row")
	assert.Equal(t, 1, payload.Data.Me.Rank)
	assert.Equal(t, "rankings-user", payload.Data.Me.Display)
	assert.Equal(t, int64(300), payload.Data.Me.TotalQuota)
	assert.Equal(t, int64(120), payload.Data.Me.TotalTokens)
	assert.Equal(t, int64(1), payload.Data.Me.RequestCount)

	require.Len(t, payload.Data.Consumption, 1)
	assert.Equal(t, int64(120), payload.Data.Consumption[0].TotalTokens)
	assert.Equal(t, int64(1), payload.Data.Consumption[0].RequestCount)
	// Public row stays masked even when it represents the viewer
	assert.NotEqual(t, "rankings-user", payload.Data.Consumption[0].Display)
}
