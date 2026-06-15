package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelMonitorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := DB
	originalLogDB := LOG_DB
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	initCol()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	LOG_DB = db
	require.NoError(t, db.AutoMigrate(&Log{}, &Channel{}))

	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		initCol()
	})

	return db
}

func TestGetChannelMonitorStatsSeparatesLogProbeAndSLA(t *testing.T) {
	db := setupChannelMonitorTestDB(t)
	now := common.GetTimestamp()

	require.NoError(t, db.Create(&Channel{Id: 11, Name: "primary"}).Error)
	require.NoError(t, db.Create(&[]Log{
		{
			CreatedAt: now - 60,
			Type:      LogTypeConsume,
			Content:   "ok",
			ModelName: "gpt-4o",
			ChannelId: 11,
			UseTime:   2,
			Group:     "default",
			Other:     common.MapToJsonStr(map[string]interface{}{"request_path": "/v1/chat/completions"}),
		},
		{
			CreatedAt: now - 50,
			Type:      LogTypeError,
			Content:   "status_code=503, upstream timeout",
			ModelName: "gpt-4o",
			ChannelId: 11,
			Group:     "default",
			Other: common.MapToJsonStr(map[string]interface{}{
				"status_code": 503,
				"error_code":  "bad_response_status_code",
				"error_type":  "openai_error",
			}),
		},
		{
			CreatedAt: now - 40,
			Type:      LogTypeError,
			Content:   "status_code=400, invalid request",
			ModelName: "gpt-4o",
			ChannelId: 11,
			Group:     "default",
			Other: common.MapToJsonStr(map[string]interface{}{
				"status_code": 400,
				"error_code":  "invalid_request",
				"error_type":  "invalid_request_error",
			}),
		},
		{
			CreatedAt: now - 30,
			Type:      LogTypeConsume,
			Content:   "模型测试",
			TokenName: "模型测试",
			ModelName: "gpt-4o",
			ChannelId: 11,
			UseTime:   1,
			Group:     "default",
			Other: common.MapToJsonStr(map[string]interface{}{
				"monitor_source": "probe",
				"probe":          true,
				"probe_status":   "success",
			}),
		},
		{
			CreatedAt: now - 20,
			Type:      LogTypeError,
			Content:   "invalid api key",
			TokenName: "模型测试",
			ModelName: "gpt-4o",
			ChannelId: 11,
			Group:     "default",
			Other: common.MapToJsonStr(map[string]interface{}{
				"monitor_source": "probe",
				"probe":          true,
				"probe_status":   "failed",
				"status_code":    401,
				"error_code":     "channel:invalid_key",
				"error_type":     "new_api_error",
			}),
		},
		{
			CreatedAt: now - 10,
			Type:      LogTypeError,
			Content:   "unsupported test channel",
			TokenName: "模型测试",
			ModelName: "gpt-4o",
			ChannelId: 11,
			Group:     "default",
			Other: common.MapToJsonStr(map[string]interface{}{
				"monitor_source": "probe",
				"probe":          true,
				"probe_status":   "local_error",
			}),
		},
	}).Error)

	stats, err := GetChannelMonitorStats(ChannelMonitorStatsParams{
		WindowSeconds: 600,
		ChannelID:     11,
		ModelName:     "gpt-4o",
		Group:         "default",
		ErrorLimit:    10,
	})
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, int64(1), stats.Log.Success)
	assert.Equal(t, int64(1), stats.Log.TransientFailures)
	assert.Equal(t, int64(1), stats.Log.Ignored)
	assert.InDelta(t, 0.5, stats.Log.SLA, 0.0001)
	assert.Equal(t, int64(1), stats.Probe.Success)
	assert.Equal(t, int64(1), stats.Probe.ChannelFailures)
	assert.Equal(t, int64(1), stats.Probe.Ignored)
	assert.InDelta(t, 0.5, stats.Probe.SLA, 0.0001)
	assert.Equal(t, int64(2), stats.Combined.Success)
	assert.Equal(t, int64(2), stats.Combined.Failures)
	assert.Equal(t, int64(2), stats.Combined.Ignored)
	assert.InDelta(t, 0.5, stats.Combined.SLA, 0.0001)

	require.Len(t, stats.Errors, 4)
	assert.Equal(t, "probe", stats.Errors[0].Source)
	assert.True(t, stats.Errors[0].Ignored)
	assert.Equal(t, "primary", stats.Errors[0].ChannelName)
	assert.Equal(t, "probe", stats.Errors[1].Source)
	assert.False(t, stats.Errors[1].Ignored)
	assert.Equal(t, "log", stats.Errors[2].Source)
	assert.True(t, stats.Errors[2].Ignored)
	assert.Equal(t, "log", stats.Errors[3].Source)
	assert.False(t, stats.Errors[3].Ignored)
}
