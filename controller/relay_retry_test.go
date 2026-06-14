package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/dto"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/Xauryan/stuhelper-ai/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
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
