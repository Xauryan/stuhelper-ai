package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/dto"
	"github.com/Xauryan/stuhelper-ai/model"
	relaycommon "github.com/Xauryan/stuhelper-ai/relay/common"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/Xauryan/stuhelper-ai/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func buildChannelAffinityRetryContextForTest() *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("channel_affinity_skip_retry_on_failure", true)
	return ctx
}

func withChannelAffinityTemporaryFallbackForTest(t *testing.T, enabled bool, ranges string) {
	t.Helper()
	setting := operation_setting.GetChannelAffinitySetting()
	originalEnabled := setting.FallbackOnTemporaryError
	originalRanges := setting.TemporaryErrorStatusCodes
	setting.FallbackOnTemporaryError = enabled
	setting.TemporaryErrorStatusCodes = ranges
	t.Cleanup(func() {
		setting.FallbackOnTemporaryError = originalEnabled
		setting.TemporaryErrorStatusCodes = originalRanges
	})
}

func withAutoGroupRetrySettingsForTest(t *testing.T) {
	t.Helper()
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUserUsableGroups := setting.UserUsableGroups2JSONString()
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["default","backup"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认分组","backup":"备用分组"}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUserUsableGroups))
	})
}

func buildAutoGroupRetryContextForTest(t *testing.T, crossGroupRetry bool) *gin.Context {
	t.Helper()
	withAutoGroupRetrySettingsForTest(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	common.SetContextKey(ctx, constant.ContextKeyUsingGroup, "auto")
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, crossGroupRetry)
	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "default")
	return ctx
}

func setupRelayRetryTestDB(t *testing.T) *gorm.DB {
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

	dsn := fmt.Sprintf("file:%s_relay_retry?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Log{}))
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

func seedRelayRetryChannel(t *testing.T, db *gorm.DB, channel model.Channel) model.Channel {
	t.Helper()
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, channel.AddAbilities(nil))
	return channel
}

func TestShouldRetryAllowsChannelAffinityFallbackForTemporaryStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withChannelAffinityTemporaryFallbackForTest(t, true, "429,500,502-503")

	err := types.NewErrorWithStatusCode(
		errors.New("Service temporarily unavailable"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusServiceUnavailable,
	)

	require.True(t, shouldRetry(buildChannelAffinityRetryContextForTest(), err, 1))
}

func TestShouldRetryKeepsChannelAffinityForNonTemporaryStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withChannelAffinityTemporaryFallbackForTest(t, true, "429,500,502-503")

	err := types.NewErrorWithStatusCode(
		errors.New("invalid api key"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusUnauthorized,
	)

	require.False(t, shouldRetry(buildChannelAffinityRetryContextForTest(), err, 1))
}

func TestShouldRetryKeepsAlwaysSkipStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withChannelAffinityTemporaryFallbackForTest(t, true, "429,500,502-504")

	err := types.NewErrorWithStatusCode(
		errors.New("gateway timeout"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusGatewayTimeout,
	)

	require.False(t, shouldRetry(buildChannelAffinityRetryContextForTest(), err, 1))
}

func TestShouldRetryKeepsCloudflareGatewayTimeoutAlwaysSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withChannelAffinityTemporaryFallbackForTest(t, true, "429,500,502-503,524")

	err := types.NewErrorWithStatusCode(
		errors.New("bad response status code 524"),
		types.ErrorCodeBadResponseStatusCode,
		524,
	)

	require.False(t, shouldRetry(buildChannelAffinityRetryContextForTest(), err, 1))
}

func TestShouldRetryDoesNotBypassExplicitSkipRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withChannelAffinityTemporaryFallbackForTest(t, true, "429,500,502-503")

	err := types.NewErrorWithStatusCode(
		errors.New("Service temporarily unavailable"),
		types.ErrorCodeInvalidRequest,
		http.StatusServiceUnavailable,
		types.ErrOptionWithSkipRetry(),
	)

	require.False(t, shouldRetry(buildChannelAffinityRetryContextForTest(), err, 1))
}

func TestShouldRetryTaskRelayAllowsChannelAffinityFallbackForTemporaryStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withChannelAffinityTemporaryFallbackForTest(t, true, "429,500,502-503")

	taskErr := &dto.TaskError{
		StatusCode: http.StatusServiceUnavailable,
		Message:    "Service temporarily unavailable",
	}

	require.True(t, shouldRetryTaskRelay(buildChannelAffinityRetryContextForTest(), 1, taskErr, 1))
}

func TestTaskRelayErrorForAccountingClassifiesUpstreamFailures(t *testing.T) {
	transient := taskRelayErrorForAccounting(&dto.TaskError{
		StatusCode: http.StatusServiceUnavailable,
		Message:    "service unavailable",
	})
	require.NotNil(t, transient)
	transientClass := service.ClassifyRelayError(transient)
	require.Equal(t, service.RetryClassTransient, transientClass.Class)
	require.True(t, transientClass.Transient)

	channelSide := taskRelayErrorForAccounting(&dto.TaskError{
		StatusCode: http.StatusUnauthorized,
		Message:    "invalid api key",
	})
	require.NotNil(t, channelSide)
	channelClass := service.ClassifyRelayError(channelSide)
	require.Equal(t, service.RetryClassChannel, channelClass.Class)
	require.True(t, channelClass.ChannelSide)

	local := taskRelayErrorForAccounting(&dto.TaskError{
		StatusCode: http.StatusBadRequest,
		Message:    "bad local request",
		LocalError: true,
	})
	require.Nil(t, local)
}

func TestGetChannelSkipsSetupChannelWithoutEnabledKeys(t *testing.T) {
	db := setupRelayRetryTestDB(t)
	autoBanDisabled := 0
	highPriority := int64(10)
	lowPriority := int64(0)

	bad := seedRelayRetryChannel(t, db, model.Channel{
		Id:       301,
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
	good := seedRelayRetryChannel(t, db, model.Channel{
		Id:       302,
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
	common.SetContextKey(ctx, constant.ContextKeyAutoGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyChannelId, bad.Id)
	common.SetContextKey(ctx, constant.ContextKeyChannelType, bad.Type)
	common.SetContextKey(ctx, constant.ContextKeyChannelName, bad.Name)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o-mini",
		TokenGroup:      "auto",
		UserGroup:       "default",
		UsingGroup:      "default",
		ChannelMeta:     &relaycommon.ChannelMeta{},
	}
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-4o-mini",
		Retry:      common.GetPointer(0),
	}

	selected, channelErr := getChannel(ctx, info, retryParam)

	require.Nil(t, channelErr)
	require.NotNil(t, selected)
	require.Equal(t, good.Id, selected.Id)
	require.Equal(t, good.Id, common.GetContextKeyInt(ctx, constant.ContextKeyChannelId))
	require.Equal(t, "enabled-key", common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
}

func TestPrepareAutoGroupRetryAllowsCrossGroupWhenGlobalRetryExhausted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := buildAutoGroupRetryContextForTest(t, true)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	err := types.NewErrorWithStatusCode(
		errors.New("bad response status code 503"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusServiceUnavailable,
	)

	require.True(t, prepareAutoGroupRetryAfterRelayError(ctx, err, retryParam))
	require.Equal(t, 1, common.GetContextKeyInt(ctx, constant.ContextKeyAutoGroupIndex))

	retryParam.IncreaseRetry()
	require.Equal(t, 0, retryParam.GetRetry())
}

func TestPrepareAutoGroupRetryRequiresCrossGroupEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := buildAutoGroupRetryContextForTest(t, false)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	err := types.NewErrorWithStatusCode(
		errors.New("bad response status code 503"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusServiceUnavailable,
	)

	require.False(t, prepareAutoGroupRetryAfterRelayError(ctx, err, retryParam))
}

func TestPrepareAutoGroupRetryKeepsAlwaysSkipStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := buildAutoGroupRetryContextForTest(t, true)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	err := types.NewErrorWithStatusCode(
		errors.New("gateway timeout"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusGatewayTimeout,
	)

	require.False(t, prepareAutoGroupRetryAfterRelayError(ctx, err, retryParam))
	_, exists := common.GetContextKey(ctx, constant.ContextKeyAutoGroupIndex)
	require.False(t, exists)
}

func TestPrepareAutoGroupRetryAllowsCloudflareGatewayTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := buildAutoGroupRetryContextForTest(t, true)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	err := types.NewErrorWithStatusCode(
		errors.New("bad response status code 524"),
		types.ErrorCodeBadResponseStatusCode,
		524,
	)

	require.True(t, prepareAutoGroupRetryAfterRelayError(ctx, err, retryParam))
	require.Equal(t, 1, common.GetContextKeyInt(ctx, constant.ContextKeyAutoGroupIndex))
}

func TestPrepareAutoGroupRetryDoesNotBypassExplicitSkipRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := buildAutoGroupRetryContextForTest(t, true)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	err := types.NewErrorWithStatusCode(
		errors.New("bad response status code 524"),
		types.ErrorCodeBadResponseStatusCode,
		524,
		types.ErrOptionWithSkipRetry(),
	)

	require.False(t, prepareAutoGroupRetryAfterRelayError(ctx, err, retryParam))
}

func TestPrepareAutoGroupRetryAllowsStructuredCloudflareGatewayTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := buildAutoGroupRetryContextForTest(t, true)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	err := types.NewErrorWithStatusCode(
		errors.New("upstream timeout"),
		types.ErrorCode("server_error"),
		524,
	)

	require.True(t, prepareAutoGroupRetryAfterRelayError(ctx, err, retryParam))
	require.Equal(t, 1, common.GetContextKeyInt(ctx, constant.ContextKeyAutoGroupIndex))
}

func TestPrepareAutoGroupRetryDoesNotBypassChannelAffinityForCloudflareGatewayTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withChannelAffinityTemporaryFallbackForTest(t, true, "429,500,502-503,524")
	ctx := buildAutoGroupRetryContextForTest(t, true)
	ctx.Set("channel_affinity_skip_retry_on_failure", true)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	err := types.NewErrorWithStatusCode(
		errors.New("bad response status code 524"),
		types.ErrorCodeBadResponseStatusCode,
		524,
	)

	require.False(t, prepareAutoGroupRetryAfterRelayError(ctx, err, retryParam))
	_, exists := common.GetContextKey(ctx, constant.ContextKeyAutoGroupIndex)
	require.False(t, exists)
}

func TestPrepareAutoGroupRetryAllowsTaskCloudflareGatewayTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := buildAutoGroupRetryContextForTest(t, true)
	retryParam := &service.RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5",
		Retry:      common.GetPointer(0),
	}
	taskErr := &dto.TaskError{
		StatusCode: 524,
		Message:    "bad response status code 524",
	}

	require.True(t, prepareAutoGroupRetryAfterTaskError(ctx, taskErr, retryParam))
	require.Equal(t, 1, common.GetContextKeyInt(ctx, constant.ContextKeyAutoGroupIndex))
}
