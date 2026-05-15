package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setTestSessionUser(role int) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 123)
		session.Set("username", "role-test")
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

func TestAuditAdminAuthAllowsAuditAdminAndRejectsCommonUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name string
		role int
		want int
	}{
		{name: "common user rejected", role: common.RoleCommonUser, want: http.StatusOK},
		{name: "audit admin allowed", role: common.RoleAuditAdminUser, want: http.StatusOK},
		{name: "admin allowed", role: common.RoleAdminUser, want: http.StatusOK},
		{name: "root allowed", role: common.RoleRootUser, want: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupSessionAuthTestDB(t)
			require.NoError(t, db.Create(&model.User{
				Id:          123,
				Username:    "role-test",
				Password:    "password",
				DisplayName: "Role Test",
				Role:        tc.role,
				Status:      common.UserStatusEnabled,
				Group:       "default",
			}).Error)

			router := gin.New()
			router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
			router.Use(setTestSessionUser(tc.role))
			router.GET("/", AuditAdminAuth(), func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			router.ServeHTTP(recorder, req)

			require.Equal(t, tc.want, recorder.Code, recorder.Body.String())
			if tc.role == common.RoleCommonUser {
				require.Contains(t, recorder.Body.String(), `"success":false`)
				require.Contains(t, recorder.Body.String(), "auth.insufficient_privilege")
				return
			}
			require.Equal(t, "ok", recorder.Body.String())
		})
	}
}

func setupSessionAuthTestDB(t *testing.T) *gorm.DB {
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

	dsn := fmt.Sprintf("file:%s_session?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))

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

func setStaleTestSessionUser(role int, status int) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 123)
		session.Set("username", "stale-session-user")
		session.Set("role", role)
		session.Set("status", status)
		if err := session.Save(); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Request.Header.Set("StuHelper-AI-User", "123")
		c.Next()
	}
}

func TestSessionAuthRevalidatesCurrentRoleAndStatus(t *testing.T) {
	db := setupSessionAuthTestDB(t)

	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "stale-session-user",
		Password:    "password",
		DisplayName: "Stale Session User",
		Role:        common.RoleAuditAdminUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(setStaleTestSessionUser(common.RoleAdminUser, common.UserStatusEnabled))
	router.GET("/", AdminAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), "auth.insufficient_privilege")

	require.NoError(t, db.Model(&model.User{}).Where("id = ?", 123).Update("status", common.UserStatusDisabled).Error)

	router = gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(setStaleTestSessionUser(common.RoleCommonUser, common.UserStatusEnabled))
	router.GET("/", UserAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), "auth.user_banned")
}

func TestTryUserAuthIgnoresDisabledSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 123)
		session.Set("username", "disabled-user")
		session.Set("role", common.RoleCommonUser)
		session.Set("status", common.UserStatusDisabled)
		require.NoError(t, session.Save())
		c.Next()
	})
	router.GET("/", TryUserAuth(), func(c *gin.Context) {
		assert.Zero(t, c.GetInt("id"))
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "ok", recorder.Body.String())
}

func TestTryUserAuthIgnoresStaleEnabledSessionForDisabledUser(t *testing.T) {
	db := setupSessionAuthTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "disabled-db-user",
		Password:    "password",
		DisplayName: "Disabled DB User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusDisabled,
		Group:       "default",
	}).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(setStaleTestSessionUser(common.RoleCommonUser, common.UserStatusEnabled))
	router.GET("/", TryUserAuth(), func(c *gin.Context) {
		assert.Zero(t, c.GetInt("id"))
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "ok", recorder.Body.String())
}

func TestTokenOrUserAuthRejectsStaleEnabledSessionForDisabledUser(t *testing.T) {
	db := setupSessionAuthTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "disabled-db-user",
		Password:    "password",
		DisplayName: "Disabled DB User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusDisabled,
		Group:       "default",
	}).Error)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(setStaleTestSessionUser(common.RoleCommonUser, common.UserStatusEnabled))
	router.GET("/", TokenOrUserAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	require.NotEqual(t, "ok", recorder.Body.String())
}

func TestAdminAuthRejectsDeletedSessionUserAsLoggedOut(t *testing.T) {
	setupSessionAuthTestDB(t)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	router.Use(setStaleTestSessionUser(common.RoleAdminUser, common.UserStatusEnabled))
	router.GET("/", AdminAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), "auth.not_logged_in")
}

func TestRequireAdminRoleRejectsAuditAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", common.RoleAuditAdminUser)
		c.Next()
	})
	router.POST("/", RequireAdminRole(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), "auth.insufficient_privilege")
}

func TestRequireAuditOrAdminRoleUsesAuthenticatedRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		role, _ := strconv.Atoi(c.GetHeader("X-Test-Role"))
		c.Set("role", role)
		c.Next()
	})
	router.GET("/", RequireAuditOrAdminRole(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for _, tc := range []struct {
		name string
		role int
		ok   bool
	}{
		{name: "audit admin allowed", role: common.RoleAuditAdminUser, ok: true},
		{name: "admin allowed", role: common.RoleAdminUser, ok: true},
		{name: "common rejected", role: common.RoleCommonUser, ok: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Test-Role", strconv.Itoa(tc.role))
			router.ServeHTTP(recorder, req)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			if tc.ok {
				require.Equal(t, "ok", recorder.Body.String())
				return
			}
			require.Contains(t, recorder.Body.String(), `"success":false`)
			require.Contains(t, recorder.Body.String(), "auth.insufficient_privilege")
		})
	}
}

func setupTokenAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	initTokenAuthColumnNames(t)

	originalRedisEnabled := common.RedisEnabled
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalDB := model.DB
	originalLogDB := model.LOG_DB

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))

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

func initTokenAuthColumnNames(t *testing.T) {
	t.Helper()

	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.UsingSQLite = false
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	require.NoError(t, os.Setenv("SQL_DSN", "local"))

	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

func TestTokenAuthAllowsAutoPseudoGroup(t *testing.T) {
	db := setupTokenAuthTestDB(t)
	createTokenAuthTestToken(t, db, "autotokentestkey", "auto")

	router := gin.New()
	router.Use(TokenAuth())
	router.GET("/v1/models", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"using_group": common.GetContextKeyString(c, constant.ContextKeyUsingGroup),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-autotokentestkey")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"using_group":"auto"`)
}

func TestTokenAuthRejectsUnavailableExplicitGroup(t *testing.T) {
	db := setupTokenAuthTestDB(t)
	createTokenAuthTestToken(t, db, "blockedgrouptestkey", "blocked")

	router := gin.New()
	router.Use(TokenAuth())
	router.GET("/v1/models", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-blockedgrouptestkey")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "无权访问 blocked 分组")
}

func createTokenAuthTestToken(t *testing.T, db *gorm.DB, key string, group string) {
	t.Helper()

	user := model.User{
		Username:    key + "-user",
		Password:    "password",
		DisplayName: "Token Auth Test User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	token := model.Token{
		UserId:         user.Id,
		Key:            key,
		Status:         common.TokenStatusEnabled,
		Name:           key + "-token",
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		Group:          group,
	}
	require.NoError(t, db.Create(&token).Error)
}
