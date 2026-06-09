package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/dto"
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
