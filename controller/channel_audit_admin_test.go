package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelAuditAdminControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	initChannelAuditAdminColumnNames(t)

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
	require.NoError(t, db.AutoMigrate(&model.Channel{}))

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

func initChannelAuditAdminColumnNames(t *testing.T) {
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

func ptrStringForChannelAuditAdminControllerTest(v string) *string {
	return &v
}

func performChannelAuditAdminSearch(t *testing.T, role int, keyword string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("role", role)
	requestURL := "/?keyword=" + url.QueryEscape(keyword) + "&p=1&page_size=20"
	ctx.Request = httptest.NewRequest(http.MethodGet, requestURL, nil)
	SearchChannels(ctx)
	return recorder
}

func decodeChannelAuditAdminControllerResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestAuditAdminChannelSearchCannotMatchHiddenSecrets(t *testing.T) {
	db := setupChannelAuditAdminControllerTestDB(t)

	secretBaseURL := "https://secret-channel.example"
	require.NoError(t, db.Create(&model.Channel{
		Type:           1,
		Key:            "sk-secret-channel-key",
		Name:           "public-channel",
		Status:         common.ChannelStatusEnabled,
		BaseURL:        &secretBaseURL,
		Models:         "gpt-4o",
		Group:          "default",
		Other:          "secret-other",
		Setting:        ptrStringForChannelAuditAdminControllerTest("secret-setting"),
		HeaderOverride: ptrStringForChannelAuditAdminControllerTest(`{"X-Secret":"value"}`),
		Remark:         ptrStringForChannelAuditAdminControllerTest("secret remark"),
	}).Error)

	for _, keyword := range []string{"sk-secret-channel-key", "secret-channel.example"} {
		recorder := performChannelAuditAdminSearch(t, common.RoleAuditAdminUser, keyword)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		response := decodeChannelAuditAdminControllerResponse(t, recorder)
		require.Equal(t, true, response["success"])
		data := response["data"].(map[string]interface{})
		require.Zero(t, int(data["total"].(float64)))
	}

	adminRecorder := performChannelAuditAdminSearch(t, common.RoleAdminUser, "sk-secret-channel-key")
	require.Equal(t, http.StatusOK, adminRecorder.Code, adminRecorder.Body.String())
	adminResponse := decodeChannelAuditAdminControllerResponse(t, adminRecorder)
	require.Equal(t, true, adminResponse["success"])
	adminData := adminResponse["data"].(map[string]interface{})
	require.Equal(t, 1, int(adminData["total"].(float64)))
}

func TestAuditAdminChannelSearchResultsHideSensitiveFields(t *testing.T) {
	db := setupChannelAuditAdminControllerTestDB(t)

	secretBaseURL := "https://secret-channel.example"
	require.NoError(t, db.Create(&model.Channel{
		Type:              1,
		Key:               "sk-secret-channel-key",
		Name:              "public-search-name",
		Status:            common.ChannelStatusEnabled,
		BaseURL:           &secretBaseURL,
		Models:            "gpt-4o",
		Group:             "default",
		Other:             "secret-other",
		OtherInfo:         "secret-other-info",
		ModelMapping:      ptrStringForChannelAuditAdminControllerTest(`{"a":"b"}`),
		StatusCodeMapping: ptrStringForChannelAuditAdminControllerTest(`{"401":"500"}`),
		Setting:           ptrStringForChannelAuditAdminControllerTest("secret-setting"),
		ParamOverride:     ptrStringForChannelAuditAdminControllerTest(`{"temperature":0}`),
		HeaderOverride:    ptrStringForChannelAuditAdminControllerTest(`{"X-Secret":"value"}`),
		Remark:            ptrStringForChannelAuditAdminControllerTest("secret remark"),
		OtherSettings:     "secret-settings",
	}).Error)

	recorder := performChannelAuditAdminSearch(t, common.RoleAuditAdminUser, "public-search-name")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	response := decodeChannelAuditAdminControllerResponse(t, recorder)
	require.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	require.Equal(t, 1, int(data["total"].(float64)))
	items := data["items"].([]interface{})
	require.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	require.Empty(t, item["key"])
	require.Nil(t, item["base_url"])
	require.Empty(t, item["other"])
	require.Empty(t, item["other_info"])
	require.Nil(t, item["model_mapping"])
	require.Nil(t, item["status_code_mapping"])
	require.Nil(t, item["setting"])
	require.Nil(t, item["param_override"])
	require.Nil(t, item["header_override"])
	require.Nil(t, item["remark"])
	require.Empty(t, item["settings"])
}
