package controller

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetTopUpInfoDoesNotExposeEpayMethodsWhenEpayDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalAlipayEnabled := setting.AlipayOfficialEnabled
	originalWechatEnabled := setting.WechatPayOfficialEnabled
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		setting.AlipayOfficialEnabled = originalAlipayEnabled
		setting.WechatPayOfficialEnabled = originalWechatEnabled
	})

	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay"},
		{"name": "微信", "type": "wxpay"},
	}
	setting.AlipayOfficialEnabled = false
	setting.WechatPayOfficialEnabled = false

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)

	GetTopUpInfo(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			EnableOnlineTopUp bool                `json:"enable_online_topup"`
			PayMethods        []map[string]string `json:"pay_methods"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.False(t, payload.Data.EnableOnlineTopUp)
	require.Empty(t, payload.Data.PayMethods)
}

func TestBuildOfficialTradeNoUsesAlipaySafeCharacters(t *testing.T) {
	tradeNo := buildOfficialTradeNo("ALIPAY", 42)

	require.Regexp(t, regexp.MustCompile(`^ALIPAY_42_[0-9]+_[A-Za-z0-9]+$`), tradeNo)
	require.NotContains(t, tradeNo, "-")
}

func TestGetTopUpInfoStillExposesOfficialMethodsWhenEpayDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalAlipayEnabled := setting.AlipayOfficialEnabled
	originalAlipayAppID := setting.AlipayOfficialAppID
	originalAlipayPrivateKey := setting.AlipayOfficialPrivateKey
	originalAlipayPublicKey := setting.AlipayOfficialAlipayPublicKey
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		setting.AlipayOfficialEnabled = originalAlipayEnabled
		setting.AlipayOfficialAppID = originalAlipayAppID
		setting.AlipayOfficialPrivateKey = originalAlipayPrivateKey
		setting.AlipayOfficialAlipayPublicKey = originalAlipayPublicKey
	})

	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay"},
		{"name": "微信", "type": "wxpay"},
	}
	setting.AlipayOfficialEnabled = true
	setting.AlipayOfficialAppID = "app-id"
	setting.AlipayOfficialPrivateKey = "private"
	setting.AlipayOfficialAlipayPublicKey = "public"

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)

	GetTopUpInfo(ctx)

	var payload struct {
		Data struct {
			PayMethods []map[string]string `json:"pay_methods"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Data.PayMethods, 1)
	require.Equal(t, model.PaymentMethodAlipayOfficial, payload.Data.PayMethods[0]["type"])
}
