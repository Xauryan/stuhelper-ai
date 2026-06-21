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

func setupLogAuditAdminControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalRedisEnabled := common.RedisEnabled
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalDB := model.DB
	originalLogDB := model.LOG_DB

	gin.SetMode(gin.TestMode)
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initModelListColumnNames(t)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Log{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		common.RedisEnabled = originalRedisEnabled
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		model.DB = originalDB
		model.LOG_DB = originalLogDB
	})

	return db
}

func performLogAuditAdminRequest(t *testing.T, role int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("role", role)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?p=1&page_size=20&type=0", nil)
	GetAllLogs(ctx)
	return recorder
}

func TestAuditAdminLogsDoNotExposeChannelName(t *testing.T) {
	db := setupLogAuditAdminControllerTestDB(t)
	require.NoError(t, db.Create(&model.Channel{
		Id:     11,
		Type:   1,
		Name:   "private-channel-name",
		Status: common.ChannelStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		Id:        101,
		CreatedAt: common.GetTimestamp(),
		Type:      model.LogTypeManage,
		Content:   "Updated channel private-channel-name (ID: 11)",
		UserId:    1,
		Username:  "alice",
		ModelName: "gpt-4o",
		ChannelId: 11,
		Other: common.MapToJsonStr(map[string]interface{}{
			"channel_name": "private-channel-name",
			"admin_info": map[string]interface{}{
				"use_channel":      []int{11},
				"channel_name":     "private-channel-name",
				"channel_affinity": map[string]interface{}{"selected_group": "private-group", "key_hint": "private-key-hint"},
			},
			"op": map[string]interface{}{
				"action": "channel.update",
				"params": map[string]interface{}{
					"id":             11,
					"name":           "private-channel-name",
					"tag":            "private-tag",
					"type":           1,
					"base_url":       "https://private.example.com",
					"changed_fields": []string{"name", "base_url"},
				},
			},
		}),
	}).Error)

	recorder := performLogAuditAdminRequest(t, common.RoleAuditAdminUser)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.NotContains(t, recorder.Body.String(), "private-channel-name")
	require.NotContains(t, recorder.Body.String(), "private-tag")
	require.NotContains(t, recorder.Body.String(), "private.example.com")
	require.NotContains(t, recorder.Body.String(), "private-key-hint")
	require.NotContains(t, recorder.Body.String(), "private-group")
	require.NotContains(t, recorder.Body.String(), `"type":1`)
	require.NotContains(t, recorder.Body.String(), `\"type\":1`)
	require.Contains(t, recorder.Body.String(), `"channel":11`)
	require.Contains(t, recorder.Body.String(), "Updated channel #11")
	require.Contains(t, recorder.Body.String(), "changed_fields")

	adminRecorder := performLogAuditAdminRequest(t, common.RoleAdminUser)
	require.Equal(t, http.StatusOK, adminRecorder.Code, adminRecorder.Body.String())
	require.Contains(t, adminRecorder.Body.String(), `"success":true`)
	require.Contains(t, adminRecorder.Body.String(), "private-channel-name")
	require.Contains(t, adminRecorder.Body.String(), "private-tag")
	require.Contains(t, adminRecorder.Body.String(), "private.example.com")
	require.Contains(t, adminRecorder.Body.String(), `\"type\":1`)
}

func performChannelMonitorAuditAdminRequest(t *testing.T, role int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("role", role)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?window_seconds=600&error_limit=10", nil)
	GetChannelMonitorSummary(ctx)
	return recorder
}

func TestAuditAdminChannelMonitorUsesChannelNumbersOnly(t *testing.T) {
	db := setupLogAuditAdminControllerTestDB(t)
	now := common.GetTimestamp()
	require.NoError(t, db.Create(&model.Channel{
		Id:     11,
		Type:   1,
		Name:   "private-channel-name",
		Status: common.ChannelStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		Id:        102,
		CreatedAt: now,
		Type:      model.LogTypeError,
		Content:   "private-channel-name status_code=503, upstream timeout",
		UserId:    1,
		Username:  "alice",
		ModelName: "gpt-4o",
		ChannelId: 11,
		Group:     "default",
		Other: common.MapToJsonStr(map[string]interface{}{
			"channel_name": "private-channel-name",
			"status_code":  503,
			"error_code":   "bad_response_status_code",
		}),
	}).Error)

	recorder := performChannelMonitorAuditAdminRequest(t, common.RoleAuditAdminUser)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.Contains(t, recorder.Body.String(), `"channel_id":11`)
	require.Contains(t, recorder.Body.String(), "#11 status_code=503")
	require.NotContains(t, recorder.Body.String(), "private-channel-name")

	adminRecorder := performChannelMonitorAuditAdminRequest(t, common.RoleAdminUser)
	require.Equal(t, http.StatusOK, adminRecorder.Code, adminRecorder.Body.String())
	require.Contains(t, adminRecorder.Body.String(), `"success":true`)
	require.Contains(t, adminRecorder.Body.String(), "private-channel-name")
}
