package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
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

func setupUserSecurityAdminControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := setupUserAuditAdminControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PasskeyCredential{}, &model.TwoFA{}, &model.TwoFABackupCode{}))
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

func performUserAuditAdminControllerRequestWithPath(t *testing.T, role int, method string, path string, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 999)
	ctx.Set("username", "admin-test")
	ctx.Set("role", role)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	handler(ctx)

	return recorder
}

func decodeUserAuditAdminControllerResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func createAuditAdminControllerTestUser(t *testing.T, db *gorm.DB, role int) model.User {
	t.Helper()

	seq := auditAdminControllerTestUserSeq.Add(1)
	user := model.User{
		Username:                       fmt.Sprintf("role-%d-target-%d", role, seq),
		Password:                       "password123",
		DisplayName:                    fmt.Sprintf("Role %d Target", role),
		Role:                           role,
		Status:                         common.UserStatusEnabled,
		Email:                          fmt.Sprintf("role-%d-%d@example.com", role, seq),
		AffCode:                        fmt.Sprintf("aff%d", seq),
		GitHubId:                       fmt.Sprintf("github-sensitive-%d", seq),
		DiscordId:                      fmt.Sprintf("discord-sensitive-%d", seq),
		OidcId:                         fmt.Sprintf("oidc-sensitive-%d", seq),
		WeChatId:                       fmt.Sprintf("wechat-sensitive-%d", seq),
		TelegramId:                     fmt.Sprintf("telegram-sensitive-%d", seq),
		LinuxDOId:                      fmt.Sprintf("linuxdo-sensitive-%d", seq),
		Remark:                         "sensitive remark",
		StripeCustomer:                 fmt.Sprintf("cus_sensitive_%d", seq),
		ReferralCommissionPercent:      ptrFloat64ForAuditAdminControllerTest(7.5),
		InviterRewardUnlocked:          true,
		InviterRewardUnlockedByPayment: true,
	}
	user.SetAccessToken(fmt.Sprintf("sensitiveaccesstoken%012d", seq))
	user.Setting = `{"sidebar_modules":"sensitive"}`
	require.NoError(t, db.Create(&user).Error)
	return user
}

var auditAdminControllerTestUserSeq atomic.Int64

func ptrFloat64ForAuditAdminControllerTest(v float64) *float64 {
	return &v
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

func TestCreateUserRejectsDirectAuditAdminRole(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	recorder := performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		`{"username":"audit-created","password":"password123","display_name":"Audit Created","role":5}`,
		CreateUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, false, response["success"])

	var count int64
	require.NoError(t, db.Model(&model.User{}).Where("username = ?", "audit-created").Count(&count).Error)
	require.Zero(t, count)
}

func TestRootManageUserPromotesAndDemotesThroughAuditAdmin(t *testing.T) {
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
		common.RoleRootUser,
		fmt.Sprintf(`{"id":%d,"action":"promote"}`, target.Id),
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
	require.True(t, sidebarModules["admin"]["referral"])
	require.False(t, sidebarModules["admin"]["setting"])
	require.NotContains(t, sidebarModules["admin"], "deployment")

	recorder = performUserAuditAdminControllerRequest(
		t,
		common.RoleRootUser,
		fmt.Sprintf(`{"id":%d,"action":"promote"}`, target.Id),
		ManageUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response = decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, true, response["success"])

	var admin model.User
	require.NoError(t, db.First(&admin, target.Id).Error)
	require.Equal(t, common.RoleAdminUser, admin.Role)

	recorder = performUserAuditAdminControllerRequest(
		t,
		common.RoleRootUser,
		fmt.Sprintf(`{"id":%d,"action":"demote"}`, target.Id),
		ManageUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response = decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, true, response["success"])

	var auditAgain model.User
	require.NoError(t, db.First(&auditAgain, target.Id).Error)
	require.Equal(t, common.RoleAuditAdminUser, auditAgain.Role)

	recorder = performUserAuditAdminControllerRequest(
		t,
		common.RoleRootUser,
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

func TestAdminManageUserCannotDemoteAuditAdmin(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	target := model.User{
		Username:    "audit-target",
		Password:    "password123",
		DisplayName: "Audit Target",
		Role:        common.RoleAuditAdminUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&target).Error)

	recorder := performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		fmt.Sprintf(`{"id":%d,"action":"demote"}`, target.Id),
		ManageUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, false, response["success"])

	var unchanged model.User
	require.NoError(t, db.First(&unchanged, target.Id).Error)
	require.Equal(t, common.RoleAuditAdminUser, unchanged.Role)
}

func TestAdminCannotManageAuditAdminAccount(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	for _, action := range []string{"disable", "enable", "delete", "add_quota"} {
		t.Run(action, func(t *testing.T) {
			target := createAuditAdminControllerTestUser(t, db, common.RoleAuditAdminUser)
			body := fmt.Sprintf(`{"id":%d,"action":%q,"mode":"add","value":100}`, target.Id, action)

			recorder := performUserAuditAdminControllerRequest(t, common.RoleAdminUser, body, ManageUser)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			response := decodeUserAuditAdminControllerResponse(t, recorder)
			require.Equal(t, false, response["success"])

			var unchanged model.User
			require.NoError(t, db.Unscoped().First(&unchanged, target.Id).Error)
			require.Equal(t, common.RoleAuditAdminUser, unchanged.Role)
			require.Equal(t, common.UserStatusEnabled, unchanged.Status)
			require.Zero(t, unchanged.DeletedAt.Valid)
			require.Zero(t, unchanged.Quota)
		})
	}
}

func TestAdminCannotViewOrEditAuditAdminDetails(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)
	target := createAuditAdminControllerTestUser(t, db, common.RoleAuditAdminUser)

	getRecorder := performUserAuditAdminControllerRequestWithPath(
		t,
		common.RoleAdminUser,
		http.MethodGet,
		"/"+strconv.Itoa(target.Id),
		"",
		func(c *gin.Context) {
			c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}
			GetUser(c)
		},
	)
	require.Equal(t, http.StatusOK, getRecorder.Code, getRecorder.Body.String())
	getResponse := decodeUserAuditAdminControllerResponse(t, getRecorder)
	require.Equal(t, false, getResponse["success"])

	updateRecorder := performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		fmt.Sprintf(`{"id":%d,"username":"changed-audit","password":"password123","display_name":"Changed Audit","role":5}`, target.Id),
		UpdateUser,
	)
	require.Equal(t, http.StatusOK, updateRecorder.Code, updateRecorder.Body.String())
	updateResponse := decodeUserAuditAdminControllerResponse(t, updateRecorder)
	require.Equal(t, false, updateResponse["success"])

	deleteRecorder := performUserAuditAdminControllerRequestWithPath(
		t,
		common.RoleAdminUser,
		http.MethodDelete,
		"/"+strconv.Itoa(target.Id),
		"",
		func(c *gin.Context) {
			c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}
			DeleteUser(c)
		},
	)
	require.Equal(t, http.StatusOK, deleteRecorder.Code, deleteRecorder.Body.String())
	deleteResponse := decodeUserAuditAdminControllerResponse(t, deleteRecorder)
	require.Equal(t, false, deleteResponse["success"])

	var unchanged model.User
	require.NoError(t, db.Unscoped().First(&unchanged, target.Id).Error)
	require.Equal(t, target.Username, unchanged.Username)
	require.Zero(t, unchanged.DeletedAt.Valid)
}

func TestDeleteUserRejectsRootTarget(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)
	target := createAuditAdminControllerTestUser(t, db, common.RoleRootUser)

	recorder := performUserAuditAdminControllerRequestWithPath(
		t,
		common.RoleRootUser,
		http.MethodDelete,
		"/"+strconv.Itoa(target.Id),
		"",
		func(c *gin.Context) {
			c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}
			DeleteUser(c)
		},
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, false, response["success"])

	var unchanged model.User
	require.NoError(t, db.Unscoped().First(&unchanged, target.Id).Error)
	require.Equal(t, common.RoleRootUser, unchanged.Role)
	require.Zero(t, unchanged.DeletedAt.Valid)
}

func TestAdminCannotResetAuditAdminSecurityBindings(t *testing.T) {
	db := setupUserSecurityAdminControllerTestDB(t)
	target := createAuditAdminControllerTestUser(t, db, common.RoleAuditAdminUser)
	require.NoError(t, db.Create(&model.PasskeyCredential{
		UserID:          target.Id,
		CredentialID:    "YXVkaXQtY3JlZA==",
		PublicKey:       "YXVkaXQtcHVibGlj",
		AttestationType: "none",
	}).Error)
	require.NoError(t, db.Create(&model.TwoFA{
		UserId:    target.Id,
		Secret:    "SECRET",
		IsEnabled: true,
	}).Error)

	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{name: "reset passkey", handler: AdminResetPasskey},
		{name: "reset 2fa", handler: AdminDisable2FA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := performUserAuditAdminControllerRequestWithPath(
				t,
				common.RoleAdminUser,
				http.MethodDelete,
				"/"+strconv.Itoa(target.Id),
				"",
				func(c *gin.Context) {
					c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}
					tc.handler(c)
				},
			)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			response := decodeUserAuditAdminControllerResponse(t, recorder)
			require.Equal(t, false, response["success"])
			require.Contains(t, fmt.Sprint(response["message"]), "权")
		})
	}

	_, err := model.GetPasskeyByUserID(target.Id)
	require.NoError(t, err)
	require.True(t, model.IsTwoFAEnabled(target.Id))
}

func TestAdminManageUserCannotPromoteCommonUserToAuditAdmin(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	target := model.User{
		Username:    "common-target",
		Password:    "password123",
		DisplayName: "Common Target",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&target).Error)

	recorder := performUserAuditAdminControllerRequest(
		t,
		common.RoleAdminUser,
		fmt.Sprintf(`{"id":%d,"action":"promote"}`, target.Id),
		ManageUser,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, false, response["success"])

	var unchanged model.User
	require.NoError(t, db.First(&unchanged, target.Id).Error)
	require.Equal(t, common.RoleCommonUser, unchanged.Role)
}

func TestAuditSpecificManageUserActionsAreRejected(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)

	target := model.User{
		Username:    "audit-target",
		Password:    "password123",
		DisplayName: "Audit Target",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&target).Error)

	for _, action := range []string{"promote_audit", "demote_audit"} {
		recorder := performUserAuditAdminControllerRequest(
			t,
			common.RoleRootUser,
			fmt.Sprintf(`{"id":%d,"action":%q}`, target.Id, action),
			ManageUser,
		)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		response := decodeUserAuditAdminControllerResponse(t, recorder)
		require.Equal(t, false, response["success"])
	}
}

func TestAuditAdminUserListKeepsEmailAndHidesSensitiveFields(t *testing.T) {
	db := setupUserAuditAdminControllerTestDB(t)
	target := createAuditAdminControllerTestUser(t, db, common.RoleCommonUser)

	recorder := performUserAuditAdminControllerRequestWithPath(
		t,
		common.RoleAuditAdminUser,
		http.MethodGet,
		"/?p=1&page_size=10",
		"",
		GetAllUsers,
	)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeUserAuditAdminControllerResponse(t, recorder)
	require.Equal(t, true, response["success"])

	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok)
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, items)

	var got map[string]interface{}
	for _, item := range items {
		userItem, ok := item.(map[string]interface{})
		require.True(t, ok)
		if int(userItem["id"].(float64)) == target.Id {
			got = userItem
			break
		}
	}
	require.NotNil(t, got)
	require.Equal(t, target.Email, got["email"])
	require.Empty(t, got["github_id"])
	require.Empty(t, got["discord_id"])
	require.Empty(t, got["oidc_id"])
	require.Empty(t, got["wechat_id"])
	require.Empty(t, got["telegram_id"])
	require.Empty(t, got["linux_do_id"])
	require.Nil(t, got["access_token"])
	require.Empty(t, got["setting"])
	require.Empty(t, got["stripe_customer"])
	require.Nil(t, got["referral_commission_percent"])
}
