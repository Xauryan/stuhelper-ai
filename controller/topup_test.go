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

func TestGetTopUpInfoIncludesOfficialUnitPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalAlipayEnabled := setting.AlipayOfficialEnabled
	originalAlipayAppID := setting.AlipayOfficialAppID
	originalAlipayPrivateKey := setting.AlipayOfficialPrivateKey
	originalAlipayPublicKey := setting.AlipayOfficialAlipayPublicKey
	originalAlipayUnitPrice := setting.AlipayOfficialUnitPrice
	originalEpayPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		setting.AlipayOfficialEnabled = originalAlipayEnabled
		setting.AlipayOfficialAppID = originalAlipayAppID
		setting.AlipayOfficialPrivateKey = originalAlipayPrivateKey
		setting.AlipayOfficialAlipayPublicKey = originalAlipayPublicKey
		setting.AlipayOfficialUnitPrice = originalAlipayUnitPrice
		operation_setting.PayAddress = originalEpayPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
	})

	setting.AlipayOfficialEnabled = true
	setting.AlipayOfficialAppID = "app-id"
	setting.AlipayOfficialPrivateKey = "private-key"
	setting.AlipayOfficialAlipayPublicKey = "alipay-public-key"
	setting.AlipayOfficialUnitPrice = 1.006
	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)

	GetTopUpInfo(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			PayMethods []map[string]string `json:"pay_methods"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Len(t, payload.Data.PayMethods, 1)
	require.Equal(t, model.PaymentMethodAlipayOfficial, payload.Data.PayMethods[0]["type"])
	require.Equal(t, "1.006", payload.Data.PayMethods[0]["unit_price"])
}

func TestBuildOfficialTradeNoUsesAlipaySafeCharacters(t *testing.T) {
	tradeNo := buildOfficialTradeNo("ALIPAY", 42)

	require.Regexp(t, regexp.MustCompile(`^ALIPAY_42_[0-9]+_[A-Za-z0-9]+$`), tradeNo)
	require.NotContains(t, tradeNo, "-")
}

func TestConfiguredTopUpPayMoneyCeilsToCents(t *testing.T) {
	originalPrice := operation_setting.Price
	originalQuotaDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalDiscounts := make(map[int]float64, len(operation_setting.GetPaymentSetting().AmountDiscount))
	for k, v := range operation_setting.GetPaymentSetting().AmountDiscount {
		originalDiscounts[k] = v
	}
	originalTopupGroupRatio := common.TopupGroupRatio2JSONString()
	originalStripeUnitPrice := setting.StripeUnitPrice
	originalWaffoUnitPrice := setting.WaffoUnitPrice
	originalWaffoPancakeUnitPrice := setting.WaffoPancakeUnitPrice
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalQuotaDisplayType
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(originalTopupGroupRatio))
		setting.StripeUnitPrice = originalStripeUnitPrice
		setting.WaffoUnitPrice = originalWaffoUnitPrice
		setting.WaffoPancakeUnitPrice = originalWaffoPancakeUnitPrice
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"default":1}`))
	operation_setting.Price = 7.231
	setting.StripeUnitPrice = 7.231
	setting.WaffoUnitPrice = 7.231
	setting.WaffoPancakeUnitPrice = 7.231

	testCases := []struct {
		name   string
		actual float64
	}{
		{name: "epay", actual: getPayMoney(1, "default")},
		{name: "official", actual: getOfficialPayMoney(1, "default", 7.231)},
		{name: "stripe", actual: getStripePayMoney(1, "default")},
		{name: "waffo", actual: getWaffoPayMoney(1, "default")},
		{name: "waffo pancake", actual: getWaffoPancakePayMoney(1, "default")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.InDelta(t, 7.24, tc.actual, 0.000001)
		})
	}

	require.Equal(t, "7.24", formatOfficialPayMoney(7.231))
	require.Equal(t, int64(724), yuanToFen(7.231))
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
	require.Equal(t, "支付宝", payload.Data.PayMethods[0]["name"])
}
