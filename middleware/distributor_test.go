package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDistributorTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalRedisEnabled := common.RedisEnabled
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()

	gin.SetMode(gin.TestMode)
	common.MemoryCacheEnabled = false
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s_distributor?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认分组"}`))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.RedisEnabled = originalRedisEnabled
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups))
	})

	return db
}

func seedDistributorChannel(t *testing.T, db *gorm.DB, channel model.Channel) model.Channel {
	t.Helper()
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, channel.AddAbilities(nil))
	return channel
}

func TestSetupSelectedChannelWithFallbackSkipsChannelWithoutEnabledKeys(t *testing.T) {
	db := setupDistributorTestDB(t)
	autoBanDisabled := 0
	highPriority := int64(10)
	lowPriority := int64(0)

	bad := seedDistributorChannel(t, db, model.Channel{
		Id:       101,
		Type:     constant.ChannelTypeOpenAI,
		Key:      "disabled-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "disabled-key-channel",
		Models:   "gpt-4o-mini",
		Group:    "default",
		Priority: &highPriority,
		AutoBan:  &autoBanDisabled,
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:         true,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusAutoDisabled},
		},
	})
	good := seedDistributorChannel(t, db, model.Channel{
		Id:       202,
		Type:     constant.ChannelTypeOpenAI,
		Key:      "enabled-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "enabled-key-channel",
		Models:   "gpt-4o-mini",
		Group:    "default",
		Priority: &lowPriority,
		AutoBan:  &autoBanDisabled,
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroup, "auto")
	common.SetContextKey(ctx, constant.ContextKeyUsingGroup, "auto")
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")

	selected, setupErr := setupSelectedChannelWithFallback(ctx, &bad, "gpt-4o-mini")

	require.Nil(t, setupErr)
	require.NotNil(t, selected)
	require.Equal(t, good.Id, selected.Id)
	require.Equal(t, good.Id, common.GetContextKeyInt(ctx, constant.ContextKeyChannelId))
	require.Equal(t, "enabled-key", common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
	require.Equal(t, "default", common.GetContextKeyString(ctx, constant.ContextKeyUsingGroup))
}

func TestSetupSelectedChannelWithFallbackHonorsAffinitySkipRetry(t *testing.T) {
	db := setupDistributorTestDB(t)
	autoBanDisabled := 0
	highPriority := int64(10)
	lowPriority := int64(0)

	bad := seedDistributorChannel(t, db, model.Channel{
		Id:       303,
		Type:     constant.ChannelTypeOpenAI,
		Key:      "disabled-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "disabled-key-channel",
		Models:   "gpt-4o-mini",
		Group:    "default",
		Priority: &highPriority,
		AutoBan:  &autoBanDisabled,
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:         true,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusAutoDisabled},
		},
	})
	_ = seedDistributorChannel(t, db, model.Channel{
		Id:       404,
		Type:     constant.ChannelTypeOpenAI,
		Key:      "enabled-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "enabled-key-channel",
		Models:   "gpt-4o-mini",
		Group:    "default",
		Priority: &lowPriority,
		AutoBan:  &autoBanDisabled,
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("channel_affinity_skip_retry_on_failure", true)
	common.SetContextKey(ctx, constant.ContextKeyTokenGroup, "auto")
	common.SetContextKey(ctx, constant.ContextKeyUsingGroup, "auto")
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")

	selected, setupErr := setupSelectedChannelWithFallback(ctx, &bad, "gpt-4o-mini")

	require.NotNil(t, setupErr)
	require.NotNil(t, selected)
	require.Equal(t, bad.Id, selected.Id)
	require.Equal(t, bad.Id, common.GetContextKeyInt(ctx, constant.ContextKeyChannelId))
	require.Empty(t, common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
}
