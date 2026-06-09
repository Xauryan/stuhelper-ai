package controller

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestStripeWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	originalAPISecret := setting.StripeApiSecret
	originalWebhookSecret := setting.StripeWebhookSecret
	originalPriceID := setting.StripePriceId
	t.Cleanup(func() {
		setting.StripeApiSecret = originalAPISecret
		setting.StripeWebhookSecret = originalWebhookSecret
		setting.StripePriceId = originalPriceID
	})

	setting.StripeWebhookSecret = ""
	setting.StripeApiSecret = "sk_test_123"
	setting.StripePriceId = "price_123"
	require.False(t, isStripeWebhookEnabled())

	setting.StripeWebhookSecret = "whsec_test"
	require.True(t, isStripeWebhookEnabled())

	setting.StripePriceId = ""
	require.False(t, isStripeWebhookEnabled())
}

func TestCreemWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	originalAPIKey := setting.CreemApiKey
	originalProducts := setting.CreemProducts
	originalWebhookSecret := setting.CreemWebhookSecret
	t.Cleanup(func() {
		setting.CreemApiKey = originalAPIKey
		setting.CreemProducts = originalProducts
		setting.CreemWebhookSecret = originalWebhookSecret
	})

	setting.CreemWebhookSecret = ""
	setting.CreemApiKey = "creem_api_key"
	setting.CreemProducts = `[{"productId":"prod_123"}]`
	require.False(t, isCreemWebhookEnabled())

	setting.CreemWebhookSecret = "creem_secret"
	require.True(t, isCreemWebhookEnabled())

	setting.CreemProducts = "[]"
	require.False(t, isCreemWebhookEnabled())
}

func TestWaffoWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	originalEnabled := setting.WaffoEnabled
	originalSandbox := setting.WaffoSandbox
	originalAPIKey := setting.WaffoApiKey
	originalPrivateKey := setting.WaffoPrivateKey
	originalPublicCert := setting.WaffoPublicCert
	originalSandboxAPIKey := setting.WaffoSandboxApiKey
	originalSandboxPrivateKey := setting.WaffoSandboxPrivateKey
	originalSandboxPublicCert := setting.WaffoSandboxPublicCert
	t.Cleanup(func() {
		setting.WaffoEnabled = originalEnabled
		setting.WaffoSandbox = originalSandbox
		setting.WaffoApiKey = originalAPIKey
		setting.WaffoPrivateKey = originalPrivateKey
		setting.WaffoPublicCert = originalPublicCert
		setting.WaffoSandboxApiKey = originalSandboxAPIKey
		setting.WaffoSandboxPrivateKey = originalSandboxPrivateKey
		setting.WaffoSandboxPublicCert = originalSandboxPublicCert
	})

	setting.WaffoEnabled = true
	setting.WaffoSandbox = false
	setting.WaffoApiKey = ""
	setting.WaffoPrivateKey = "private"
	setting.WaffoPublicCert = "public"
	require.False(t, isWaffoWebhookEnabled())

	setting.WaffoApiKey = "api"
	require.True(t, isWaffoWebhookEnabled())

	setting.WaffoEnabled = false
	require.False(t, isWaffoWebhookEnabled())

	setting.WaffoEnabled = true
	setting.WaffoSandbox = true
	setting.WaffoSandboxApiKey = ""
	setting.WaffoSandboxPrivateKey = "sandbox_private"
	setting.WaffoSandboxPublicCert = "sandbox_public"
	require.False(t, isWaffoWebhookEnabled())

	setting.WaffoSandboxApiKey = "sandbox_api"
	require.True(t, isWaffoWebhookEnabled())
}

func TestWaffoPancakeWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	originalEnabled := setting.WaffoPancakeEnabled
	originalSandbox := setting.WaffoPancakeSandbox
	originalMerchantID := setting.WaffoPancakeMerchantID
	originalPrivateKey := setting.WaffoPancakePrivateKey
	originalWebhookPublicKey := setting.WaffoPancakeWebhookPublicKey
	originalWebhookTestKey := setting.WaffoPancakeWebhookTestKey
	originalStoreID := setting.WaffoPancakeStoreID
	originalProductID := setting.WaffoPancakeProductID
	t.Cleanup(func() {
		setting.WaffoPancakeEnabled = originalEnabled
		setting.WaffoPancakeSandbox = originalSandbox
		setting.WaffoPancakeMerchantID = originalMerchantID
		setting.WaffoPancakePrivateKey = originalPrivateKey
		setting.WaffoPancakeWebhookPublicKey = originalWebhookPublicKey
		setting.WaffoPancakeWebhookTestKey = originalWebhookTestKey
		setting.WaffoPancakeStoreID = originalStoreID
		setting.WaffoPancakeProductID = originalProductID
	})

	setting.WaffoPancakeEnabled = true
	setting.WaffoPancakeSandbox = false
	setting.WaffoPancakeMerchantID = "merchant"
	setting.WaffoPancakePrivateKey = "private"
	setting.WaffoPancakeStoreID = "store"
	setting.WaffoPancakeProductID = "product"
	setting.WaffoPancakeWebhookPublicKey = ""
	require.False(t, isWaffoPancakeWebhookEnabled())

	setting.WaffoPancakeWebhookPublicKey = "public"
	require.True(t, isWaffoPancakeWebhookEnabled())

	setting.WaffoPancakeEnabled = false
	require.False(t, isWaffoPancakeWebhookEnabled())

	setting.WaffoPancakeEnabled = true
	setting.WaffoPancakeSandbox = true
	setting.WaffoPancakeWebhookTestKey = ""
	require.False(t, isWaffoPancakeWebhookEnabled())

	setting.WaffoPancakeWebhookTestKey = "test_public"
	require.True(t, isWaffoPancakeWebhookEnabled())
}

func TestEpayWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
	})

	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "epay_id"
	operation_setting.EpayKey = ""
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}}
	require.False(t, isEpayWebhookEnabled())

	operation_setting.EpayKey = "epay_key"
	require.True(t, isEpayWebhookEnabled())

	operation_setting.PayMethods = nil
	require.False(t, isEpayWebhookEnabled())
}

func TestAlipayOfficialWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	originalEnabled := setting.AlipayOfficialEnabled
	originalAppID := setting.AlipayOfficialAppID
	originalPrivateKey := setting.AlipayOfficialPrivateKey
	originalPublicKey := setting.AlipayOfficialAlipayPublicKey
	t.Cleanup(func() {
		setting.AlipayOfficialEnabled = originalEnabled
		setting.AlipayOfficialAppID = originalAppID
		setting.AlipayOfficialPrivateKey = originalPrivateKey
		setting.AlipayOfficialAlipayPublicKey = originalPublicKey
	})

	setting.AlipayOfficialEnabled = true
	setting.AlipayOfficialAppID = "app-id"
	setting.AlipayOfficialPrivateKey = "private"
	setting.AlipayOfficialAlipayPublicKey = ""
	require.False(t, isAlipayOfficialWebhookEnabled())

	setting.AlipayOfficialAlipayPublicKey = "public"
	require.True(t, isAlipayOfficialWebhookEnabled())

	setting.AlipayOfficialEnabled = false
	require.False(t, isAlipayOfficialWebhookEnabled())
}

func TestWechatPayOfficialWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
	originalEnabled := setting.WechatPayOfficialEnabled
	originalAppID := setting.WechatPayOfficialAppID
	originalMchID := setting.WechatPayOfficialMchID
	originalSerial := setting.WechatPayOfficialCertificateSerial
	originalAPIv3Key := setting.WechatPayOfficialAPIv3Key
	originalPrivateKey := setting.WechatPayOfficialPrivateKey
	originalPlatformPublicKey := setting.WechatPayOfficialPlatformPublicKey
	t.Cleanup(func() {
		setting.WechatPayOfficialEnabled = originalEnabled
		setting.WechatPayOfficialAppID = originalAppID
		setting.WechatPayOfficialMchID = originalMchID
		setting.WechatPayOfficialCertificateSerial = originalSerial
		setting.WechatPayOfficialAPIv3Key = originalAPIv3Key
		setting.WechatPayOfficialPrivateKey = originalPrivateKey
		setting.WechatPayOfficialPlatformPublicKey = originalPlatformPublicKey
	})

	setting.WechatPayOfficialEnabled = true
	setting.WechatPayOfficialAppID = "wx-app"
	setting.WechatPayOfficialMchID = "mch"
	setting.WechatPayOfficialCertificateSerial = "serial"
	setting.WechatPayOfficialAPIv3Key = "12345678901234567890123456789012"
	setting.WechatPayOfficialPrivateKey = ""
	setting.WechatPayOfficialPlatformPublicKey = "platform_public"
	require.False(t, isWechatPayOfficialWebhookEnabled())

	setting.WechatPayOfficialPrivateKey = "private"
	require.True(t, isWechatPayOfficialWebhookEnabled())

	setting.WechatPayOfficialPlatformPublicKey = ""
	require.False(t, isWechatPayOfficialWebhookEnabled())

	setting.WechatPayOfficialEnabled = false
	require.False(t, isWechatPayOfficialWebhookEnabled())
}

func TestSelfServeTopUpEnabledRequiresPricingLimitsAndQRCode(t *testing.T) {
	originalEnabled := setting.SelfServeTopUpEnabled
	originalAlipayEnabled := setting.SelfServeAlipayEnabled
	originalWechatPayEnabled := setting.SelfServeWechatPayEnabled
	originalAlipayQRCode := setting.SelfServeAlipayQRCode
	originalWechatPayQRCode := setting.SelfServeWechatPayQRCode
	originalUnitPrice := setting.SelfServeTopUpUnitPrice
	originalSingleMax := setting.SelfServeTopUpSingleMaxAmount
	originalDailyMax := setting.SelfServeTopUpDailyMaxAmount
	t.Cleanup(func() {
		setting.SelfServeTopUpEnabled = originalEnabled
		setting.SelfServeAlipayEnabled = originalAlipayEnabled
		setting.SelfServeWechatPayEnabled = originalWechatPayEnabled
		setting.SelfServeAlipayQRCode = originalAlipayQRCode
		setting.SelfServeWechatPayQRCode = originalWechatPayQRCode
		setting.SelfServeTopUpUnitPrice = originalUnitPrice
		setting.SelfServeTopUpSingleMaxAmount = originalSingleMax
		setting.SelfServeTopUpDailyMaxAmount = originalDailyMax
	})

	setting.SelfServeTopUpEnabled = true
	setting.SelfServeAlipayEnabled = true
	setting.SelfServeWechatPayEnabled = false
	setting.SelfServeAlipayQRCode = "data:image/png;base64,Zm9v"
	setting.SelfServeWechatPayQRCode = ""
	setting.SelfServeTopUpUnitPrice = 1.23
	setting.SelfServeTopUpSingleMaxAmount = 199.99
	setting.SelfServeTopUpDailyMaxAmount = 499.99
	require.True(t, isSelfServeTopUpEnabled())
	require.True(t, isSelfServeAlipayTopUpEnabled())

	setting.SelfServeTopUpUnitPrice = 0
	require.False(t, isSelfServeTopUpEnabled())
	require.False(t, isSelfServeAlipayTopUpEnabled())

	setting.SelfServeTopUpUnitPrice = 1.23
	setting.SelfServeTopUpSingleMaxAmount = 0
	require.False(t, isSelfServeTopUpEnabled())
	require.False(t, isSelfServeAlipayTopUpEnabled())

	setting.SelfServeTopUpSingleMaxAmount = 199.99
	setting.SelfServeAlipayQRCode = ""
	require.False(t, isSelfServeTopUpEnabled())
	require.False(t, isSelfServeAlipayTopUpEnabled())
}
