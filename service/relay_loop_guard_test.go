package service

import (
	"errors"
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

func setupRelayLoopGuardTestDB(t *testing.T) *gorm.DB {
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

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default","backup"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认分组","backup":"备用分组"}`))

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

func seedRelayLoopGuardChannel(t *testing.T, db *gorm.DB, id int, group string, modelName string) {
	t.Helper()

	channel := model.Channel{
		Id:       id,
		Type:     constant.ChannelTypeOpenAI,
		Key:      fmt.Sprintf("key-%d", id),
		Status:   common.ChannelStatusEnabled,
		Name:     fmt.Sprintf("channel-%d", id),
		Models:   modelName,
		Group:    group,
		Priority: common.GetPointer(int64(0)),
	}
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: id,
		Enabled:   true,
		Priority:  common.GetPointer(int64(0)),
		Weight:    100,
	}).Error)
}

func seedRelayLoopGuardChannelWithAbilities(t *testing.T, db *gorm.DB, channel model.Channel) {
	t.Helper()

	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, channel.AddAbilities(nil))
}

func newRelayLoopGuardContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return c, w
}

func TestRelayLoopGuardCapturesSignedPathAndRejectsRepeatedChannel(t *testing.T) {
	c, _ := newRelayLoopGuardContext()
	upstreamReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	ApplyRelayLoopHeaders(c, upstreamReq, 12)
	c.Request.Header = upstreamReq.Header.Clone()
	CaptureRelayLoopPath(c)

	require.Equal(t, []int{12}, GetRelayLoopChannelIDs(c))
	relayErr := CheckRelayLoopForChannel(c, 12)
	require.NotNil(t, relayErr)
	require.Equal(t, http.StatusLoopDetected, relayErr.StatusCode)
	require.Equal(t, "channel:relay_loop", string(relayErr.GetErrorCode()))
}

func TestRelayLoopGuardIgnoresUnsignedPath(t *testing.T) {
	c, _ := newRelayLoopGuardContext()
	c.Request.Header.Set(RelayLoopPathHeader, "12")
	c.Request.Header.Set(RelayLoopSignatureHeader, "invalid")

	CaptureRelayLoopPath(c)

	require.Empty(t, GetRelayLoopChannelIDs(c))
	require.Nil(t, CheckRelayLoopForChannel(c, 12))
}

func TestAutoSelectionExcludesRelayLoopChannels(t *testing.T) {
	db := setupRelayLoopGuardTestDB(t)
	seedRelayLoopGuardChannel(t, db, 11, "default", "gpt-4o-mini")
	seedRelayLoopGuardChannel(t, db, 22, "backup", "gpt-4o-mini")

	c, _ := newRelayLoopGuardContext()
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(c, constant.ContextKeyTokenCrossGroupRetry, true)
	common.SetContextKey(c, constant.ContextKeyRelayLoopChannelIds, []int{11})

	channel, selectGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      common.GetPointer(0),
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 22, channel.Id)
	require.Equal(t, "backup", selectGroup)
}

func TestAutoSelectionUsesTokenPriorityBeforeSystemDefault(t *testing.T) {
	db := setupRelayLoopGuardTestDB(t)
	seedRelayLoopGuardChannel(t, db, 11, "default", "gpt-4o-mini")
	seedRelayLoopGuardChannel(t, db, 22, "backup", "gpt-4o-mini")

	c, _ := newRelayLoopGuardContext()
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(c, constant.ContextKeyTokenAutoGroups, []string{"backup"})

	channel, selectGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      common.GetPointer(0),
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 22, channel.Id)
	require.Equal(t, "backup", selectGroup)
}

func TestAutoSelectionUsesModelMappingSourceAlias(t *testing.T) {
	db := setupRelayLoopGuardTestDB(t)
	modelMapping := `{"claude-sonnet-4-6":"anthropic.claude-sonnet-4-6"}`
	seedRelayLoopGuardChannelWithAbilities(t, db, model.Channel{
		Id:           33,
		Type:         constant.ChannelTypeOpenAI,
		Key:          "key-33",
		Status:       common.ChannelStatusEnabled,
		Name:         "mapped-channel",
		Models:       "anthropic.claude-sonnet-4-6",
		Group:        "default",
		ModelMapping: &modelMapping,
		Priority:     common.GetPointer(int64(0)),
		Weight:       common.GetPointer(uint(100)),
	})

	c, _ := newRelayLoopGuardContext()
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")

	channel, selectGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "claude-sonnet-4-6",
		Retry:      common.GetPointer(0),
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 33, channel.Id)
	require.Equal(t, "default", selectGroup)
}

func TestPrepareAutoGroupRetrySelectsNextConcreteGroup(t *testing.T) {
	db := setupRelayLoopGuardTestDB(t)
	seedRelayLoopGuardChannel(t, db, 11, "default", "gpt-4o-mini")
	seedRelayLoopGuardChannel(t, db, 22, "backup", "gpt-4o-mini")

	c, _ := newRelayLoopGuardContext()
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "auto")
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(c, constant.ContextKeyTokenCrossGroupRetry, true)

	retry := common.GetPointer(0)
	channel, selectGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      retry,
	})
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 11, channel.Id)
	require.Equal(t, "default", selectGroup)

	require.True(t, PrepareAutoGroupRetry(c))

	channel, selectGroup, err = CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      retry,
	})
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 22, channel.Id)
	require.Equal(t, "backup", selectGroup)
}

func TestAutoSelectionAllowsTokenPriorityWhenSystemAutoGroupsEmpty(t *testing.T) {
	db := setupRelayLoopGuardTestDB(t)
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`[]`))
	seedRelayLoopGuardChannel(t, db, 22, "backup", "gpt-4o-mini")

	c, _ := newRelayLoopGuardContext()
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(c, constant.ContextKeyTokenAutoGroups, []string{"backup"})

	channel, selectGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      common.GetPointer(0),
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 22, channel.Id)
	require.Equal(t, "backup", selectGroup)
}

func TestAutoSelectionReturnsClearErrorWhenNoUsableAutoGroups(t *testing.T) {
	_ = setupRelayLoopGuardTestDB(t)
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"vip":"VIP分组"}`))

	c, _ := newRelayLoopGuardContext()
	common.SetContextKey(c, constant.ContextKeyUserGroup, "vip")

	channel, selectGroup, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      common.GetPointer(0),
	})

	require.Nil(t, channel)
	require.Equal(t, "auto", selectGroup)
	require.ErrorContains(t, err, "auto groups has no usable groups")
}

func TestAutoSelectionReturnsRelayLoopErrorWhenAllChannelsExcluded(t *testing.T) {
	db := setupRelayLoopGuardTestDB(t)
	seedRelayLoopGuardChannel(t, db, 11, "default", "gpt-4o-mini")

	c, _ := newRelayLoopGuardContext()
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(c, constant.ContextKeyRelayLoopChannelIds, []int{11})

	channel, _, err := CacheGetRandomSatisfiedChannel(&RetryParam{
		Ctx:        c,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      common.GetPointer(0),
	})

	require.Nil(t, channel)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRelayLoopNoAvailableChannel))
}
