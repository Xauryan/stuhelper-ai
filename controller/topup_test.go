package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/service"
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

func TestGetTopUpInfoDoesNotExposeWaffoPancake(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalWaffoPancakeEnabled := setting.WaffoPancakeEnabled
	originalWaffoPancakeMerchantID := setting.WaffoPancakeMerchantID
	originalWaffoPancakePrivateKey := setting.WaffoPancakePrivateKey
	originalWaffoPancakeStoreID := setting.WaffoPancakeStoreID
	originalWaffoPancakeProductID := setting.WaffoPancakeProductID
	originalWaffoPancakeWebhookPublicKey := setting.WaffoPancakeWebhookPublicKey
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		setting.WaffoPancakeEnabled = originalWaffoPancakeEnabled
		setting.WaffoPancakeMerchantID = originalWaffoPancakeMerchantID
		setting.WaffoPancakePrivateKey = originalWaffoPancakePrivateKey
		setting.WaffoPancakeStoreID = originalWaffoPancakeStoreID
		setting.WaffoPancakeProductID = originalWaffoPancakeProductID
		setting.WaffoPancakeWebhookPublicKey = originalWaffoPancakeWebhookPublicKey
	})

	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "pid"
	operation_setting.EpayKey = "key"
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay"},
		{"name": "Waffo Pancake", "type": model.PaymentMethodWaffoPancake},
	}
	setting.WaffoPancakeEnabled = true
	setting.WaffoPancakeMerchantID = "merchant"
	setting.WaffoPancakePrivateKey = "private"
	setting.WaffoPancakeStoreID = "store"
	setting.WaffoPancakeProductID = "product"
	setting.WaffoPancakeWebhookPublicKey = "webhook"

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)

	GetTopUpInfo(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			PayMethods              []map[string]string `json:"pay_methods"`
			EnableWaffoPancakeTopUp *bool               `json:"enable_waffo_pancake_topup"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Nil(t, payload.Data.EnableWaffoPancakeTopUp)
	require.Len(t, payload.Data.PayMethods, 1)
	require.Equal(t, "alipay", payload.Data.PayMethods[0]["type"])
}

func TestGetTopUpInfoIncludesOfficialUnitPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalAlipayEnabled := setting.AlipayOfficialEnabled
	originalAlipayAppID := setting.AlipayOfficialAppID
	originalAlipayPrivateKey := setting.AlipayOfficialPrivateKey
	originalAlipayPublicKey := setting.AlipayOfficialAlipayPublicKey
	originalAlipayUnitPrice := setting.AlipayOfficialUnitPrice
	originalAlipayServiceFeePercent := setting.AlipayOfficialServiceFeePercent
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
		setting.AlipayOfficialServiceFeePercent = originalAlipayServiceFeePercent
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
	setting.AlipayOfficialServiceFeePercent = 0.6
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
	require.Equal(t, "0.6", payload.Data.PayMethods[0]["service_fee_percent"])
}

func TestGetTopUpInfoIncludesSelfServeUnitPriceAndTopupGroupRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalStripeAPISecret := setting.StripeApiSecret
	originalStripeWebhookSecret := setting.StripeWebhookSecret
	originalStripePriceID := setting.StripePriceId
	originalCreemAPIKey := setting.CreemApiKey
	originalCreemProducts := setting.CreemProducts
	originalWaffoEnabled := setting.WaffoEnabled
	originalAlipayOfficialEnabled := setting.AlipayOfficialEnabled
	originalWechatPayOfficialEnabled := setting.WechatPayOfficialEnabled
	originalSelfServeTopUpEnabled := setting.SelfServeTopUpEnabled
	originalSelfServeAlipayEnabled := setting.SelfServeAlipayEnabled
	originalSelfServeWechatPayEnabled := setting.SelfServeWechatPayEnabled
	originalSelfServeAlipayQRCode := setting.SelfServeAlipayQRCode
	originalSelfServeWechatPayQRCode := setting.SelfServeWechatPayQRCode
	originalSelfServeTopUpUnitPrice := setting.SelfServeTopUpUnitPrice
	originalSelfServeSingleMax := setting.SelfServeTopUpSingleMaxAmount
	originalSelfServeDailyMax := setting.SelfServeTopUpDailyMaxAmount
	originalTopupGroupRatio := common.TopupGroupRatio2JSONString()
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		setting.StripeApiSecret = originalStripeAPISecret
		setting.StripeWebhookSecret = originalStripeWebhookSecret
		setting.StripePriceId = originalStripePriceID
		setting.CreemApiKey = originalCreemAPIKey
		setting.CreemProducts = originalCreemProducts
		setting.WaffoEnabled = originalWaffoEnabled
		setting.AlipayOfficialEnabled = originalAlipayOfficialEnabled
		setting.WechatPayOfficialEnabled = originalWechatPayOfficialEnabled
		setting.SelfServeTopUpEnabled = originalSelfServeTopUpEnabled
		setting.SelfServeAlipayEnabled = originalSelfServeAlipayEnabled
		setting.SelfServeWechatPayEnabled = originalSelfServeWechatPayEnabled
		setting.SelfServeAlipayQRCode = originalSelfServeAlipayQRCode
		setting.SelfServeWechatPayQRCode = originalSelfServeWechatPayQRCode
		setting.SelfServeTopUpUnitPrice = originalSelfServeTopUpUnitPrice
		setting.SelfServeTopUpSingleMaxAmount = originalSelfServeSingleMax
		setting.SelfServeTopUpDailyMaxAmount = originalSelfServeDailyMax
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(originalTopupGroupRatio))
	})

	operation_setting.PayAddress = ""
	operation_setting.EpayId = ""
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{}
	setting.StripeApiSecret = ""
	setting.StripeWebhookSecret = ""
	setting.StripePriceId = ""
	setting.CreemApiKey = ""
	setting.CreemProducts = "[]"
	setting.WaffoEnabled = false
	setting.AlipayOfficialEnabled = false
	setting.WechatPayOfficialEnabled = false
	setting.SelfServeTopUpEnabled = true
	setting.SelfServeAlipayEnabled = true
	setting.SelfServeWechatPayEnabled = false
	setting.SelfServeAlipayQRCode = "data:image/png;base64,Zm9v"
	setting.SelfServeWechatPayQRCode = ""
	setting.SelfServeTopUpUnitPrice = 1.23
	setting.SelfServeTopUpSingleMaxAmount = 199.99
	setting.SelfServeTopUpDailyMaxAmount = 499.99
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"default":1,"vip":1.2}`))

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/topup/info", nil)
	ctx.Set("group", "vip")

	GetTopUpInfo(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			EnableSelfServeTopUp     bool                `json:"enable_self_serve_topup"`
			SelfServeTopupGroupRatio float64             `json:"self_serve_topup_group_ratio"`
			PayMethods               []map[string]string `json:"pay_methods"`
			SelfServeLimits          struct {
				SingleMaxMoney  float64 `json:"single_max_money"`
				DailyMaxMoney   float64 `json:"daily_max_money"`
				UnitPrice       float64 `json:"unit_price"`
				TopupGroupRatio float64 `json:"topup_group_ratio"`
			} `json:"self_serve_limits"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.True(t, payload.Data.EnableSelfServeTopUp)
	require.InDelta(t, 1.2, payload.Data.SelfServeTopupGroupRatio, 0.000001)
	require.InDelta(t, 1.23, payload.Data.SelfServeLimits.UnitPrice, 0.000001)
	require.InDelta(t, 1.2, payload.Data.SelfServeLimits.TopupGroupRatio, 0.000001)
	require.InDelta(t, 199.99, payload.Data.SelfServeLimits.SingleMaxMoney, 0.000001)
	require.InDelta(t, 499.99, payload.Data.SelfServeLimits.DailyMaxMoney, 0.000001)
	require.Len(t, payload.Data.PayMethods, 1)
	require.Equal(t, model.PaymentMethodAlipaySelfServe, payload.Data.PayMethods[0]["type"])
	require.Equal(t, "1.23", payload.Data.PayMethods[0]["unit_price"])
}

func TestBuildOfficialTradeNoUsesAlipaySafeCharacters(t *testing.T) {
	tradeNo := buildOfficialTradeNo("ALIPAY", 42)

	require.Regexp(t, regexp.MustCompile(`^ALIPAY_42_[0-9]+_[A-Za-z0-9]+$`), tradeNo)
	require.NotContains(t, tradeNo, "-")
}

func TestBuildWechatPayOfficialTradeNoUsesWechatLengthLimit(t *testing.T) {
	tradeNo := buildWechatPayOfficialTradeNo("WXSUB", 1234567890)

	require.LessOrEqual(t, len(tradeNo), 32)
	require.Regexp(t, regexp.MustCompile(`^WXSUB_[0-9]+_[A-Za-z0-9]+$`), tradeNo)
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
	originalWaffoPancakeServiceFeePercent := setting.WaffoPancakeServiceFeePercent
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalQuotaDisplayType
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(originalTopupGroupRatio))
		setting.StripeUnitPrice = originalStripeUnitPrice
		setting.WaffoUnitPrice = originalWaffoUnitPrice
		setting.WaffoPancakeUnitPrice = originalWaffoPancakeUnitPrice
		setting.WaffoPancakeServiceFeePercent = originalWaffoPancakeServiceFeePercent
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"default":1}`))
	operation_setting.Price = 7.231
	setting.StripeUnitPrice = 7.231
	setting.WaffoUnitPrice = 7.231
	setting.WaffoPancakeUnitPrice = 7.231
	setting.WaffoPancakeServiceFeePercent = 0

	testCases := []struct {
		name   string
		actual float64
	}{
		{name: "epay", actual: getPayMoney(1, "default", "alipay")},
		{name: "official", actual: getOfficialPayMoney(1, "default", 7.231, 0)},
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

func TestConfiguredTopUpPayMoneySplitsServiceFee(t *testing.T) {
	originalPrice := operation_setting.Price
	originalQuotaDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalPayMethods := operation_setting.PayMethods
	originalDiscounts := make(map[int]float64, len(operation_setting.GetPaymentSetting().AmountDiscount))
	for k, v := range operation_setting.GetPaymentSetting().AmountDiscount {
		originalDiscounts[k] = v
	}
	originalTopupGroupRatio := common.TopupGroupRatio2JSONString()
	originalWaffoUnitPrice := setting.WaffoUnitPrice
	originalWaffoPancakeUnitPrice := setting.WaffoPancakeUnitPrice
	originalWaffoPancakeServiceFeePercent := setting.WaffoPancakeServiceFeePercent
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalQuotaDisplayType
		operation_setting.PayMethods = originalPayMethods
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(originalTopupGroupRatio))
		setting.WaffoUnitPrice = originalWaffoUnitPrice
		setting.WaffoPancakeUnitPrice = originalWaffoPancakeUnitPrice
		setting.WaffoPancakeServiceFeePercent = originalWaffoPancakeServiceFeePercent
	})

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay", "service_fee_percent": "0.6"},
	}
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"default":1}`))
	operation_setting.Price = 1
	setting.WaffoUnitPrice = 1
	setting.WaffoPancakeUnitPrice = 1
	setting.WaffoPancakeServiceFeePercent = 0.6

	testCases := []struct {
		name      string
		breakdown payMoneyBreakdown
	}{
		{name: "epay", breakdown: getPayMoneyBreakdown(10, "default", "alipay")},
		{name: "official", breakdown: getOfficialPayMoneyBreakdown(10, "default", 1, 0.6)},
		{name: "waffo", breakdown: getWaffoPayMoneyBreakdown(10, "default", 0.6)},
		{name: "waffo pancake", breakdown: getWaffoPancakePayMoneyBreakdown(10, "default")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.InDelta(t, 10.00, tc.breakdown.EffectiveMoney, 0.000001)
			require.InDelta(t, 0.06, tc.breakdown.Fee, 0.000001)
			require.InDelta(t, 10.06, tc.breakdown.TotalMoney, 0.000001)
		})
	}
}

func TestAlipayOfficialCloseDoesNotExpireOrderWhenTradeIsNotFound(t *testing.T) {
	require.False(t, shouldExpireAlipayOfficialOrderAfterClose(service.ErrAlipayOfficialTradeNotFound))
	require.False(t, shouldExpireAlipayOfficialOrderAfterClose(errors.New("temporary close failure")))
	require.True(t, shouldExpireAlipayOfficialOrderAfterClose(nil))
}

func TestFormatAlipayOfficialTimeoutExpressUsesConfiguredSeconds(t *testing.T) {
	require.Equal(t, "10m", formatAlipayOfficialTimeoutExpress(600))
	require.Equal(t, "2m", formatAlipayOfficialTimeoutExpress(61))
	require.Equal(t, "10m", formatAlipayOfficialTimeoutExpress(0))
	require.Equal(t, "10m", formatAlipayOfficialTimeoutExpress(-1))
}

func TestOfficialPaymentOrderTimeoutSecondsFallbackAndWechatExpiry(t *testing.T) {
	originalAlipayTimeoutSec := setting.AlipayOfficialOrderTimeoutSec
	originalAlipayTimeoutMin := setting.AlipayOfficialOrderTimeoutMin
	originalWechatTimeoutSec := setting.WechatPayOfficialOrderTimeoutSec
	t.Cleanup(func() {
		setting.AlipayOfficialOrderTimeoutSec = originalAlipayTimeoutSec
		setting.AlipayOfficialOrderTimeoutMin = originalAlipayTimeoutMin
		setting.WechatPayOfficialOrderTimeoutSec = originalWechatTimeoutSec
	})

	setting.AlipayOfficialOrderTimeoutSec = 0
	setting.AlipayOfficialOrderTimeoutMin = 15
	require.Equal(t, 900, getAlipayOfficialOrderTimeoutSeconds())
	setting.AlipayOfficialOrderTimeoutMin = 0
	require.Equal(t, 600, getAlipayOfficialOrderTimeoutSeconds())

	setting.WechatPayOfficialOrderTimeoutSec = 120
	require.Equal(t, 120, getWechatPayOfficialOrderTimeoutSeconds())
	require.True(t, isWechatPayOfficialOrderExpired(time.Now().Add(-121*time.Second).Unix()))
	require.False(t, isWechatPayOfficialOrderExpired(time.Now().Add(-119*time.Second).Unix()))
}

func TestGetAlipayOfficialSubscriptionPayMoneyUsesConfiguredUnitPrice(t *testing.T) {
	originalAlipayUnitPrice := setting.AlipayOfficialUnitPrice
	originalServiceFeePercent := setting.AlipayOfficialServiceFeePercent
	t.Cleanup(func() {
		setting.AlipayOfficialUnitPrice = originalAlipayUnitPrice
		setting.AlipayOfficialServiceFeePercent = originalServiceFeePercent
	})

	setting.AlipayOfficialUnitPrice = 7.231
	setting.AlipayOfficialServiceFeePercent = 0

	require.InDelta(t, 7.24, getAlipayOfficialSubscriptionPayMoney(1), 0.000001)
	require.Equal(t, "7.24", formatOfficialPayMoney(getAlipayOfficialSubscriptionPayMoney(1)))
}

func TestGetWechatPayOfficialSubscriptionPayMoneyUsesConfiguredUnitPrice(t *testing.T) {
	originalWechatUnitPrice := setting.WechatPayOfficialUnitPrice
	originalServiceFeePercent := setting.WechatPayOfficialServiceFeePercent
	t.Cleanup(func() {
		setting.WechatPayOfficialUnitPrice = originalWechatUnitPrice
		setting.WechatPayOfficialServiceFeePercent = originalServiceFeePercent
	})

	setting.WechatPayOfficialUnitPrice = 1.006
	setting.WechatPayOfficialServiceFeePercent = 0

	require.InDelta(t, 50.30, getWechatPayOfficialSubscriptionPayMoney(50), 0.000001)
	require.Equal(t, int64(5030), yuanToFen(getWechatPayOfficialSubscriptionPayMoney(50)))
}

func TestGetEpaySubscriptionPayMoneyUsesConfiguredUnitPrice(t *testing.T) {
	originalPrice := operation_setting.Price
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		operation_setting.PayMethods = originalPayMethods
	})

	operation_setting.Price = 1.006
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay", "unit_price": "1.2"},
	}

	require.InDelta(t, 50.30, getEpaySubscriptionPayMoney(50, "wxpay"), 0.000001)
	require.Equal(t, "50.30", formatPayMoneyToCents(getEpaySubscriptionPayMoney(50, "wxpay")))
	require.InDelta(t, 60.00, getEpaySubscriptionPayMoney(50, "alipay"), 0.000001)
	require.Equal(t, "60.00", formatPayMoneyToCents(getEpaySubscriptionPayMoney(50, "alipay")))
}

func TestSubscriptionPayMoneyAppliesServiceFee(t *testing.T) {
	originalPrice := operation_setting.Price
	originalPayMethods := operation_setting.PayMethods
	originalAlipayUnitPrice := setting.AlipayOfficialUnitPrice
	originalWechatUnitPrice := setting.WechatPayOfficialUnitPrice
	originalAlipayServiceFeePercent := setting.AlipayOfficialServiceFeePercent
	originalWechatServiceFeePercent := setting.WechatPayOfficialServiceFeePercent
	t.Cleanup(func() {
		operation_setting.Price = originalPrice
		operation_setting.PayMethods = originalPayMethods
		setting.AlipayOfficialUnitPrice = originalAlipayUnitPrice
		setting.WechatPayOfficialUnitPrice = originalWechatUnitPrice
		setting.AlipayOfficialServiceFeePercent = originalAlipayServiceFeePercent
		setting.WechatPayOfficialServiceFeePercent = originalWechatServiceFeePercent
	})

	operation_setting.Price = 1
	operation_setting.PayMethods = []map[string]string{
		{"name": "微信", "type": "wxpay", "service_fee_percent": "0.6"},
		{"name": "支付宝", "type": "alipay", "unit_price": "1.2", "service_fee_percent": "0.6"},
	}
	setting.AlipayOfficialUnitPrice = 1
	setting.WechatPayOfficialUnitPrice = 1
	setting.AlipayOfficialServiceFeePercent = 0.6
	setting.WechatPayOfficialServiceFeePercent = 0.6

	epayDefault := getEpaySubscriptionPayMoneyBreakdown(50, "wxpay")
	require.InDelta(t, 50.00, epayDefault.EffectiveMoney, 0.000001)
	require.InDelta(t, 0.30, epayDefault.Fee, 0.000001)
	require.InDelta(t, 50.30, epayDefault.TotalMoney, 0.000001)

	epayMethod := getEpaySubscriptionPayMoneyBreakdown(50, "alipay")
	require.InDelta(t, 60.00, epayMethod.EffectiveMoney, 0.000001)
	require.InDelta(t, 0.36, epayMethod.Fee, 0.000001)
	require.InDelta(t, 60.36, epayMethod.TotalMoney, 0.000001)

	alipay := getAlipayOfficialSubscriptionPayMoneyBreakdown(50)
	require.InDelta(t, 50.00, alipay.EffectiveMoney, 0.000001)
	require.InDelta(t, 0.30, alipay.Fee, 0.000001)
	require.InDelta(t, 50.30, alipay.TotalMoney, 0.000001)

	wechat := getWechatPayOfficialSubscriptionPayMoneyBreakdown(50)
	require.InDelta(t, 50.00, wechat.EffectiveMoney, 0.000001)
	require.InDelta(t, 0.30, wechat.Fee, 0.000001)
	require.InDelta(t, 50.30, wechat.TotalMoney, 0.000001)
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
