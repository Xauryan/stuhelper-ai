package controller

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/Xauryan/stuhelper-ai/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSubscriptionControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(
		&model.SubscriptionPlan{},
		&model.SubscriptionOrder{},
		&model.User{},
		&model.TopUp{},
		&model.UserSubscription{},
		&model.ReferralCommission{},
		&model.Log{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func generateSubscriptionControllerAlipayPrivateKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateBytes, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes}))
}

func generateSubscriptionControllerKeyPair(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateBytes, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}))
}

func signSubscriptionControllerWechatHeader(t *testing.T, privateKey string, timestamp string, nonce string, body []byte) string {
	t.Helper()
	block, _ := pem.Decode([]byte(privateKey))
	require.NotNil(t, block)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	require.NoError(t, err)
	rsaKey, ok := key.(*rsa.PrivateKey)
	require.True(t, ok)
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signature)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (f roundTripFunc) client() *http.Client {
	return &http.Client{Transport: f}
}

func newSubscriptionControllerContext(t *testing.T, method string, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

func TestAdminUpdateSubscriptionPlanPersistsModelLimits(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)
	plan := &model.SubscriptionPlan{
		Title:         "Before",
		PriceAmount:   1,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
	}
	require.NoError(t, db.Create(plan).Error)

	req := AdminUpsertSubscriptionPlanRequest{
		Plan: model.SubscriptionPlan{
			Title:              "After",
			PriceAmount:        2,
			Currency:           "USD",
			DurationUnit:       model.SubscriptionDurationMonth,
			DurationValue:      1,
			Enabled:            true,
			ModelLimitsEnabled: true,
			ModelLimits:        " gpt-4o,claude-3-5-sonnet,gpt-4o,, ",
		},
	}
	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPut, fmt.Sprintf("/api/subscription/admin/plans/%d", plan.Id), req)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", plan.Id)}}

	AdminUpdateSubscriptionPlan(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var reloaded model.SubscriptionPlan
	require.NoError(t, db.First(&reloaded, plan.Id).Error)
	assert.True(t, reloaded.ModelLimitsEnabled)
	assert.Equal(t, "gpt-4o,claude-3-5-sonnet", reloaded.ModelLimits)
}

func TestAdminUpdateSubscriptionPlanPersistsRecommendedFlag(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)
	plan := &model.SubscriptionPlan{
		Title:         "Before",
		PriceAmount:   1,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		Recommended:   false,
	}
	require.NoError(t, db.Create(plan).Error)

	req := AdminUpsertSubscriptionPlanRequest{
		Plan: model.SubscriptionPlan{
			Title:         "After",
			PriceAmount:   2,
			Currency:      "USD",
			DurationUnit:  model.SubscriptionDurationMonth,
			DurationValue: 1,
			Enabled:       true,
			Recommended:   true,
		},
	}
	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPut, fmt.Sprintf("/api/subscription/admin/plans/%d", plan.Id), req)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", plan.Id)}}

	AdminUpdateSubscriptionPlan(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var reloaded model.SubscriptionPlan
	require.NoError(t, db.First(&reloaded, plan.Id).Error)
	assert.True(t, reloaded.Recommended)
}

func TestSubscriptionRequestAlipayOfficialPayCreatesOfficialOrder(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	originalAlipayEnabled := setting.AlipayOfficialEnabled
	originalAlipayAppID := setting.AlipayOfficialAppID
	originalAlipayPrivateKey := setting.AlipayOfficialPrivateKey
	originalAlipayPublicKey := setting.AlipayOfficialAlipayPublicKey
	originalAlipayUnitPrice := setting.AlipayOfficialUnitPrice
	originalSystemServerAddress := system_setting.ServerAddress
	originalCallbackAddress := operation_setting.CustomCallbackAddress
	t.Cleanup(func() {
		setting.AlipayOfficialEnabled = originalAlipayEnabled
		setting.AlipayOfficialAppID = originalAlipayAppID
		setting.AlipayOfficialPrivateKey = originalAlipayPrivateKey
		setting.AlipayOfficialAlipayPublicKey = originalAlipayPublicKey
		setting.AlipayOfficialUnitPrice = originalAlipayUnitPrice
		system_setting.ServerAddress = originalSystemServerAddress
		operation_setting.CustomCallbackAddress = originalCallbackAddress
	})

	setting.AlipayOfficialEnabled = true
	setting.AlipayOfficialAppID = "app_123"
	setting.AlipayOfficialPrivateKey = generateSubscriptionControllerAlipayPrivateKey(t)
	setting.AlipayOfficialAlipayPublicKey = "public"
	setting.AlipayOfficialUnitPrice = 1.006
	system_setting.ServerAddress = "https://example.com"
	operation_setting.CustomCallbackAddress = ""

	require.NoError(t, db.Create(&model.User{Id: 77, Username: "sub-alipay-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9901,
		Title:         "Weekly",
		PriceAmount:   50,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 7,
		Enabled:       true,
		TotalAmount:   350,
	}
	require.NoError(t, db.Create(plan).Error)

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/subscription/alipay-official/pay", gin.H{
		"plan_id": plan.Id,
		"scene":   "h5",
	})
	ctx.Set("id", 77)

	SubscriptionRequestAlipayOfficialPay(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Message string `json:"message"`
		Data    struct {
			FormHTML string `json:"form_html"`
			OrderID  string `json:"order_id"`
			Scene    string `json:"scene"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "success", response.Message)
	require.Contains(t, response.Data.FormHTML, "alipay.trade.wap.pay")
	require.Contains(t, response.Data.FormHTML, "QUICK_WAP_WAY")
	require.Equal(t, "h5", response.Data.Scene)

	order := model.GetSubscriptionOrderByTradeNo(response.Data.OrderID)
	require.NotNil(t, order)
	assert.Equal(t, model.PaymentMethodAlipayOfficial, order.PaymentMethod)
	assert.Equal(t, model.PaymentProviderAlipayOfficial, order.PaymentProvider)
	assert.InDelta(t, 50.30, order.Money, 0.000001)
}

func TestSubscriptionRequestWechatPayOfficialPayCreatesOfficialOrder(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	merchantPrivateKey, _ := generateSubscriptionControllerKeyPair(t)
	platformPrivateKey, platformPublicKey := generateSubscriptionControllerKeyPair(t)

	originalWechatEnabled := setting.WechatPayOfficialEnabled
	originalWechatAppID := setting.WechatPayOfficialAppID
	originalWechatMchID := setting.WechatPayOfficialMchID
	originalWechatSerial := setting.WechatPayOfficialCertificateSerial
	originalWechatAPIv3Key := setting.WechatPayOfficialAPIv3Key
	originalWechatPrivateKey := setting.WechatPayOfficialPrivateKey
	originalWechatPlatformPublicKey := setting.WechatPayOfficialPlatformPublicKey
	originalWechatUnitPrice := setting.WechatPayOfficialUnitPrice
	originalSystemServerAddress := system_setting.ServerAddress
	originalCallbackAddress := operation_setting.CustomCallbackAddress
	originalPrepayHTTPClient := wechatPayOfficialPrepayHTTPClient
	t.Cleanup(func() {
		setting.WechatPayOfficialEnabled = originalWechatEnabled
		setting.WechatPayOfficialAppID = originalWechatAppID
		setting.WechatPayOfficialMchID = originalWechatMchID
		setting.WechatPayOfficialCertificateSerial = originalWechatSerial
		setting.WechatPayOfficialAPIv3Key = originalWechatAPIv3Key
		setting.WechatPayOfficialPrivateKey = originalWechatPrivateKey
		setting.WechatPayOfficialPlatformPublicKey = originalWechatPlatformPublicKey
		setting.WechatPayOfficialUnitPrice = originalWechatUnitPrice
		system_setting.ServerAddress = originalSystemServerAddress
		operation_setting.CustomCallbackAddress = originalCallbackAddress
		wechatPayOfficialPrepayHTTPClient = originalPrepayHTTPClient
	})

	setting.WechatPayOfficialEnabled = true
	setting.WechatPayOfficialAppID = "wx_app_123"
	setting.WechatPayOfficialMchID = "1900000109"
	setting.WechatPayOfficialCertificateSerial = "merchant-serial"
	setting.WechatPayOfficialAPIv3Key = "12345678901234567890123456789012"
	setting.WechatPayOfficialPrivateKey = merchantPrivateKey
	setting.WechatPayOfficialPlatformPublicKey = platformPublicKey
	setting.WechatPayOfficialUnitPrice = 1.006
	system_setting.ServerAddress = "https://example.com"
	operation_setting.CustomCallbackAddress = ""

	wechatPayOfficialPrepayHTTPClient = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/v3/pay/transactions/h5", req.URL.Path)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `"description":"StuHelper AI 订阅 Weekly"`)
		require.Contains(t, string(body), `"notify_url":"https://example.com/api/wechat-pay/official/notify"`)
		require.Contains(t, string(body), `"total":5030`)

		responseBody := []byte(`{"h5_url":"https://wxpay.example.com/h5"}`)
		timestamp := "1778753000"
		nonce := "wechat-subscription-response"
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader(responseBody)),
		}
		resp.Header.Set("Wechatpay-Timestamp", timestamp)
		resp.Header.Set("Wechatpay-Nonce", nonce)
		resp.Header.Set("Wechatpay-Signature", signSubscriptionControllerWechatHeader(t, platformPrivateKey, timestamp, nonce, responseBody))
		return resp, nil
	}).client()

	require.NoError(t, db.Create(&model.User{Id: 79, Username: "sub-wechat-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9903,
		Title:         "Weekly",
		PriceAmount:   50,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 7,
		Enabled:       true,
		TotalAmount:   350,
	}
	require.NoError(t, db.Create(plan).Error)

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/subscription/wechat-pay-official/pay", gin.H{
		"plan_id": plan.Id,
		"scene":   "h5",
	})
	ctx.Set("id", 79)

	SubscriptionRequestWechatPayOfficialPay(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Message string `json:"message"`
		Data    struct {
			PaymentType string `json:"payment_type"`
			PaymentURL  string `json:"payment_url"`
			OrderID     string `json:"order_id"`
			Scene       string `json:"scene"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "success", response.Message)
	require.Equal(t, "redirect", response.Data.PaymentType)
	require.Equal(t, "https://wxpay.example.com/h5", response.Data.PaymentURL)
	require.Equal(t, "h5", response.Data.Scene)

	order := model.GetSubscriptionOrderByTradeNo(response.Data.OrderID)
	require.NotNil(t, order)
	assert.Equal(t, model.PaymentMethodWechatPayOfficial, order.PaymentMethod)
	assert.Equal(t, model.PaymentProviderWechatPayOfficial, order.PaymentProvider)
	assert.InDelta(t, 50.30, order.Money, 0.000001)
}

func TestCompleteAlipayOfficialSubscriptionOrderIfPresentCompletesSubscription(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	require.NoError(t, db.Create(&model.User{Id: 78, Username: "sub-alipay-notify-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9902,
		Title:         "Notify Plan",
		PriceAmount:   12,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 7,
		Enabled:       true,
		TotalAmount:   120,
	}
	require.NoError(t, db.Create(plan).Error)
	require.NoError(t, db.Create(&model.SubscriptionOrder{
		UserId:          78,
		PlanId:          plan.Id,
		Money:           12.08,
		TradeNo:         "ALIPAYSUB_NOTIFY",
		PaymentMethod:   model.PaymentMethodAlipayOfficial,
		PaymentProvider: model.PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}).Error)

	err := completeAlipayOfficialSubscriptionOrderIfPresent(
		"ALIPAYSUB_NOTIFY",
		map[string]string{"out_trade_no": "ALIPAYSUB_NOTIFY", "total_amount": "12.08"},
		decimal.RequireFromString("12.08"),
	)
	require.NoError(t, err)

	order := model.GetSubscriptionOrderByTradeNo("ALIPAYSUB_NOTIFY")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusSuccess, order.Status)

	topUp := model.GetTopUpByTradeNo("ALIPAYSUB_NOTIFY")
	require.NotNil(t, topUp)
	assert.Equal(t, model.PaymentProviderAlipayOfficial, topUp.PaymentProvider)
	assert.Equal(t, model.PaymentMethodAlipayOfficial, topUp.PaymentMethod)
}

func TestCompleteWechatPayOfficialSubscriptionOrderIfPresentCompletesSubscription(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	require.NoError(t, db.Create(&model.User{Id: 80, Username: "sub-wechat-notify-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9904,
		Title:         "Wechat Notify Plan",
		PriceAmount:   12,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 7,
		Enabled:       true,
		TotalAmount:   120,
	}
	require.NoError(t, db.Create(plan).Error)
	require.NoError(t, db.Create(&model.SubscriptionOrder{
		UserId:          80,
		PlanId:          plan.Id,
		Money:           12.08,
		TradeNo:         "WXPAYSUB_NOTIFY",
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}).Error)

	err := completeWechatPayOfficialSubscriptionOrderIfPresent(
		service.WechatPayOfficialNotifyEnvelope{EventType: "TRANSACTION.SUCCESS"},
		service.WechatPayOfficialTransaction{
			OutTradeNo:    "WXPAYSUB_NOTIFY",
			TransactionID: "420000000000000000",
			TradeState:    "SUCCESS",
			Amount: struct {
				Total         int64  `json:"total"`
				PayerTotal    int64  `json:"payer_total"`
				Currency      string `json:"currency"`
				PayerCurrency string `json:"payer_currency"`
			}{Total: 1208, Currency: "CNY"},
		},
	)
	require.NoError(t, err)

	order := model.GetSubscriptionOrderByTradeNo("WXPAYSUB_NOTIFY")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusSuccess, order.Status)

	topUp := model.GetTopUpByTradeNo("WXPAYSUB_NOTIFY")
	require.NotNil(t, topUp)
	assert.Equal(t, model.PaymentProviderWechatPayOfficial, topUp.PaymentProvider)
	assert.Equal(t, model.PaymentMethodWechatPayOfficial, topUp.PaymentMethod)
}
