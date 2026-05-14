package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetOptionsExposesOfficialPaymentSecretConfiguredFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalOptionMap := common.OptionMap
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"AlipayOfficialAppAuthToken":         "app-auth-token",
		"AlipayOfficialPrivateKey":           "alipay-private-key",
		"WechatPayOfficialAPIv3Key":          "12345678901234567890123456789012",
		"WechatPayOfficialPrivateKey":        "",
		"WechatPayOfficialPlatformPublicKey": "wechat-platform-public-key",
	}
	common.OptionMapRWMutex.Unlock()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/option/", nil)

	GetOptions(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)

	options := make(map[string]string, len(payload.Data))
	for _, item := range payload.Data {
		options[item.Key] = item.Value
	}
	require.NotContains(t, options, "AlipayOfficialPrivateKey")
	require.NotContains(t, options, "AlipayOfficialAppAuthToken")
	require.NotContains(t, options, "WechatPayOfficialAPIv3Key")
	require.NotContains(t, options, "WechatPayOfficialPrivateKey")
	require.Equal(t, "true", options["AlipayOfficialAppAuthTokenConfigured"])
	require.Equal(t, "true", options["AlipayOfficialPrivateKeyConfigured"])
	require.Equal(t, "true", options["WechatPayOfficialAPIv3KeyConfigured"])
	require.Equal(t, "false", options["WechatPayOfficialPrivateKeyConfigured"])
	require.Equal(t, "wechat-platform-public-key", options["WechatPayOfficialPlatformPublicKey"])
}
