package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserAuditAdminControllerTestDB(t *testing.T) *gorm.DB {
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

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))

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

func performUserAuditAdminControllerRequest(t *testing.T, role int, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 999)
	ctx.Set("username", "admin-test")
	ctx.Set("role", role)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	handler(ctx)

	return recorder
}

func decodeUserAuditAdminControllerResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestCreateUserRejectsInvalidAuditAdminRoleValues(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	recorder := performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		`{"username":"invalid-role","password":"password123","display_name":"Invalid Role","role":9}`,
		CreateUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, false, response["success"])

	var count int64
	require.NoError(t, db.Model(&model.User{}).Where("username = ?", "invalid-role").Count(&count).Error)
	require.Zero(t, count)
}

func TestCreateUserAllowsAuditAdminRole(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	recorder := performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		`{"username":"audit-created","password":"password123","display_name":"Audit Created","role":5}`,
		CreateUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, true, response["success"])

	var created model.User
	require.NoError(t, db.Where("username = ?", "audit-created").First(&created).Error)
	require.Equal(t, common.RoleAuditAdminUser, created.Role)
	createdSetting := created.GetSetting()
	require.NotEmpty(t, createdSetting.SidebarModules)
	require.Contains(t, createdSetting.SidebarModules, `"admin"`)
	require.Contains(t, createdSetting.SidebarModules, `"setting":false`)
}

func TestManageUserCanPromoteAndDemoteAuditAdmin(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	target := model.User{
		Username:    "audit-target",
		Password:    "password123",
		DisplayName: "Audit Target",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&target).Error)

	recorder := performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		fmt.Sprintf(`{"id":%d,"action":"promote_audit"}`, target.Id),
		ManageUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, true, response["success"])

	var promoted model.User
	require.NoError(t, db.First(&promoted, target.Id).Error)
	require.Equal(t, common.RoleAuditAdminUser, promoted.Role)
	promotedSetting := promoted.GetSetting()
	require.NotEmpty(t, promotedSetting.SidebarModules)
	sidebarModules := map[string]map[string]bool{}
	require.NoError(t, common.UnmarshalJsonStr(promotedSetting.SidebarModules, &sidebarModules))
	require.Contains(t, sidebarModules, "admin")
	require.True(t, sidebarModules["admin"]["channel"])
	require.False(t, sidebarModules["admin"]["setting"])
	require.False(t, sidebarModules["admin"]["deployment"])

	recorder = performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		fmt.Sprintf(`{"id":%d,"action":"demote"}`, target.Id),
		ManageUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response = decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, true, response["success"])

	var demoted model.User
	require.NoError(t, db.First(&demoted, target.Id).Error)
	require.Equal(t, common.RoleCommonUser, demoted.Role)
	demotedSetting := demoted.GetSetting()
	sidebarModules = map[string]map[string]bool{}
	require.NoError(t, common.UnmarshalJsonStr(demotedSetting.SidebarModules, &sidebarModules))
	require.NotContains(t, sidebarModules, "admin")
}
