package controller

import (
	"bytes"
	"context"
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
	"net/url"
	"sort"
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
		&model.TopUpRefund{},
		&model.TopUpRefundRequest{},
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

func signSubscriptionControllerAlipayParams(privateKey string, params map[string]string) (string, error) {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || key == "sign_type" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	content := strings.Join(parts, "&")
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return "", fmt.Errorf("invalid private key")
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("private key is not RSA")
	}
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (f roundTripFunc) client() *http.Client {
	return &http.Client{Transport: f}
}

func newWechatQueryResponseClient(t *testing.T, platformPrivateKey string, body string) *http.Client {
	t.Helper()
	responseBody := []byte(body)
	timestamp := "1778753999"
	nonce := "wechat-query-response"
	signature := signSubscriptionControllerWechatHeader(t, platformPrivateKey, timestamp, nonce, responseBody)
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, req.Method)
		require.Contains(t, req.URL.Path, "/v3/pay/transactions/out-trade-no/")
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader(responseBody)),
		}
		resp.Header.Set("Wechatpay-Timestamp", timestamp)
		resp.Header.Set("Wechatpay-Nonce", nonce)
		resp.Header.Set("Wechatpay-Signature", signature)
		return resp, nil
	}).client()
}

func newWechatRefundQueryResponseClient(t *testing.T, platformPrivateKey string, body string) *http.Client {
	t.Helper()
	responseBody := []byte(body)
	timestamp := "1778754999"
	nonce := "wechat-refund-query-response"
	signature := signSubscriptionControllerWechatHeader(t, platformPrivateKey, timestamp, nonce, responseBody)
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, req.Method)
		require.Contains(t, req.URL.Path, "/v3/refund/domestic/refunds/")
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader(responseBody)),
		}
		resp.Header.Set("Wechatpay-Timestamp", timestamp)
		resp.Header.Set("Wechatpay-Nonce", nonce)
		resp.Header.Set("Wechatpay-Signature", signature)
		return resp, nil
	}).client()
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

func TestSubscriptionRequestAlipayOfficialPayRejectsPendingOrderAtPurchaseLimit(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	originalAlipayEnabled := setting.AlipayOfficialEnabled
	originalAlipayAppID := setting.AlipayOfficialAppID
	originalAlipayPrivateKey := setting.AlipayOfficialPrivateKey
	originalAlipayPublicKey := setting.AlipayOfficialAlipayPublicKey
	originalAlipayUnitPrice := setting.AlipayOfficialUnitPrice
	originalSystemServerAddress := system_setting.ServerAddress
	t.Cleanup(func() {
		setting.AlipayOfficialEnabled = originalAlipayEnabled
		setting.AlipayOfficialAppID = originalAlipayAppID
		setting.AlipayOfficialPrivateKey = originalAlipayPrivateKey
		setting.AlipayOfficialAlipayPublicKey = originalAlipayPublicKey
		setting.AlipayOfficialUnitPrice = originalAlipayUnitPrice
		system_setting.ServerAddress = originalSystemServerAddress
	})

	setting.AlipayOfficialEnabled = true
	setting.AlipayOfficialAppID = "app_123"
	setting.AlipayOfficialPrivateKey = generateSubscriptionControllerAlipayPrivateKey(t)
	setting.AlipayOfficialAlipayPublicKey = "public"
	setting.AlipayOfficialUnitPrice = 1
	system_setting.ServerAddress = "https://example.com"

	require.NoError(t, db.Create(&model.User{Id: 78, Username: "sub-alipay-limit-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:                 9902,
		Title:              "Limited Weekly",
		PriceAmount:        10,
		Currency:           "USD",
		DurationUnit:       model.SubscriptionDurationDay,
		DurationValue:      7,
		Enabled:            true,
		TotalAmount:        100,
		MaxPurchasePerUser: 1,
	}
	require.NoError(t, db.Create(plan).Error)
	require.NoError(t, (&model.SubscriptionOrder{
		UserId:          78,
		PlanId:          plan.Id,
		Money:           10,
		TradeNo:         "ALIPAYSUB_LIMIT_PENDING",
		PaymentMethod:   model.PaymentMethodAlipayOfficial,
		PaymentProvider: model.PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}).Insert())

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/subscription/alipay-official/pay", gin.H{
		"plan_id": plan.Id,
		"scene":   "h5",
	})
	ctx.Set("id", 78)

	SubscriptionRequestAlipayOfficialPay(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "已达到该套餐购买上限")
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

func TestSubscriptionRequestWechatPayOfficialPayFallsBackToNativeWhenH5Unavailable(t *testing.T) {
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

	var paths []string
	wechatPayOfficialPrepayHTTPClient = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		if req.URL.Path == "/v3/pay/transactions/h5" {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"code":"NO_AUTH","message":"商户未开通H5支付权限"}`))),
			}, nil
		}
		require.Equal(t, "/v3/pay/transactions/native", req.URL.Path)
		responseBody := []byte(`{"code_url":"weixin://wxpay/bizpayurl?pr=fallback"}`)
		timestamp := "1778753001"
		nonce := "wechat-native-fallback"
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

	require.NoError(t, db.Create(&model.User{Id: 81, Username: "sub-wechat-fallback-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9905,
		Title:         "Fallback Weekly",
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
	ctx.Set("id", 81)

	SubscriptionRequestWechatPayOfficialPay(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Message string `json:"message"`
		Data    struct {
			PaymentType string `json:"payment_type"`
			CodeURL     string `json:"code_url"`
			OrderID     string `json:"order_id"`
			Scene       string `json:"scene"`
			Fallback    string `json:"fallback"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "success", response.Message)
	require.Equal(t, []string{"/v3/pay/transactions/h5", "/v3/pay/transactions/native"}, paths)
	require.Equal(t, "qrcode", response.Data.PaymentType)
	require.Equal(t, "weixin://wxpay/bizpayurl?pr=fallback", response.Data.CodeURL)
	require.Equal(t, "pc", response.Data.Scene)
	require.Equal(t, "native", response.Data.Fallback)

	order := model.GetSubscriptionOrderByTradeNo(response.Data.OrderID)
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
	topUp := model.GetTopUpByTradeNo(response.Data.OrderID)
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
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

func TestCompleteWechatPayOfficialSubscriptionRejectsMismatchedMerchantContext(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	originalAppID := setting.WechatPayOfficialAppID
	originalMchID := setting.WechatPayOfficialMchID
	t.Cleanup(func() {
		setting.WechatPayOfficialAppID = originalAppID
		setting.WechatPayOfficialMchID = originalMchID
	})
	setting.WechatPayOfficialAppID = "wx_expected_app"
	setting.WechatPayOfficialMchID = "1900000109"

	require.NoError(t, db.Create(&model.User{Id: 83, Username: "sub-wechat-mismatch-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9907,
		Title:         "Wechat Mismatch Plan",
		PriceAmount:   12,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 7,
		Enabled:       true,
		TotalAmount:   120,
	}
	require.NoError(t, db.Create(plan).Error)
	require.NoError(t, db.Create(&model.SubscriptionOrder{
		UserId:          83,
		PlanId:          plan.Id,
		Money:           12.08,
		TradeNo:         "WXPAYSUB_MISMATCH",
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}).Error)

	err := completeWechatPayOfficialSubscriptionOrderIfPresent(
		service.WechatPayOfficialNotifyEnvelope{EventType: "TRANSACTION.SUCCESS"},
		service.WechatPayOfficialTransaction{
			AppID:         "wx_wrong_app",
			MchID:         "1900000109",
			OutTradeNo:    "WXPAYSUB_MISMATCH",
			TransactionID: "420000000000000001",
			TradeState:    "SUCCESS",
			Amount: struct {
				Total         int64  `json:"total"`
				PayerTotal    int64  `json:"payer_total"`
				Currency      string `json:"currency"`
				PayerCurrency string `json:"payer_currency"`
			}{Total: 1208, Currency: "CNY"},
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "appid")
}

func TestReconcileAlipayOfficialTopUpCompletesSubscriptionOrder(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	require.NoError(t, db.Create(&model.User{Id: 81, Username: "sub-alipay-query-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9905,
		Title:         "Query Plan",
		PriceAmount:   12,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 7,
		Enabled:       true,
		TotalAmount:   120,
	}
	require.NoError(t, db.Create(plan).Error)
	order := &model.SubscriptionOrder{
		UserId:          81,
		PlanId:          plan.Id,
		Money:           12.08,
		TradeNo:         "ALIPAYSUB_QUERY",
		PaymentMethod:   model.PaymentMethodAlipayOfficial,
		PaymentProvider: model.PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}
	require.NoError(t, order.Insert())

	err := reconcileAlipayOfficialPaidOrder("ALIPAYSUB_QUERY", map[string]string{
		"out_trade_no":  "ALIPAYSUB_QUERY",
		"trade_status":  "TRADE_SUCCESS",
		"total_amount":  "12.08",
		"alipay_source": "query",
	}, decimal.RequireFromString("12.08"), "127.0.0.1")

	require.NoError(t, err)
	reloaded := model.GetSubscriptionOrderByTradeNo("ALIPAYSUB_QUERY")
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusSuccess, reloaded.Status)

	topUp := model.GetTopUpByTradeNo("ALIPAYSUB_QUERY")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	assert.Equal(t, int64(0), topUp.Amount)
}

func TestExpireAlipayOfficialOrderExpiresSubscriptionOrder(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)

	require.NoError(t, db.Create(&model.User{Id: 82, Username: "sub-alipay-expire-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Id:            9906,
		Title:         "Expire Plan",
		PriceAmount:   12,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationDay,
		DurationValue: 7,
		Enabled:       true,
		TotalAmount:   120,
	}
	require.NoError(t, db.Create(plan).Error)
	order := &model.SubscriptionOrder{
		UserId:          82,
		PlanId:          plan.Id,
		Money:           12.08,
		TradeNo:         "ALIPAYSUB_EXPIRE",
		PaymentMethod:   model.PaymentMethodAlipayOfficial,
		PaymentProvider: model.PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}
	require.NoError(t, order.Insert())

	require.NoError(t, expireAlipayOfficialPendingOrder("ALIPAYSUB_EXPIRE"))

	reloaded := model.GetSubscriptionOrderByTradeNo("ALIPAYSUB_EXPIRE")
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusExpired, reloaded.Status)

	topUp := model.GetTopUpByTradeNo("ALIPAYSUB_EXPIRE")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusExpired, topUp.Status)
}

func TestAlipayOfficialQueryCrossChecksResponseOutTradeNo(t *testing.T) {
	client := &service.AlipayOfficialClient{
		AppID:      "app-id",
		PrivateKey: generateSubscriptionControllerAlipayPrivateKey(t),
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body := `{"alipay_trade_query_response":{"code":"10000","msg":"Success","out_trade_no":"OTHER_ORDER","trade_status":"TRADE_SUCCESS","total_amount":"12.08"},"sign":"ignored"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}).client(),
	}

	_, err := queryAlipayOfficialTrade(context.Background(), client, "ALIPAYSUB_QUERY")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "out_trade_no")
}

func TestAlipayOfficialRefundQueryCrossChecksOutTradeNo(t *testing.T) {
	client := &service.AlipayOfficialClient{
		AppID:      "app-id",
		PrivateKey: generateSubscriptionControllerAlipayPrivateKey(t),
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			form, err := url.ParseQuery(req.URL.RawQuery)
			require.NoError(t, err)
			if form.Get("method") == "" {
				bodyBytes, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				form, err = url.ParseQuery(string(bodyBytes))
				require.NoError(t, err)
			}
			require.Equal(t, service.AlipayOfficialRefundQueryMethod, form.Get("method"))
			body := `{"alipay_trade_fastpay_refund_query_response":{"code":"10000","msg":"Success","out_trade_no":"OTHER_ORDER","out_request_no":"RF_1","refund_status":"REFUND_SUCCESS","refund_amount":"12.08"},"sign":"ignored"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}).client(),
	}

	_, err := queryAlipayOfficialRefund(context.Background(), client, "ALIPAYSUB_QUERY", "RF_1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "out_trade_no")
}

func TestAlipayOfficialNotifyRejectsMismatchedAppID(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)
	privateKey, publicKey := generateSubscriptionControllerKeyPair(t)

	originalAlipayEnabled := setting.AlipayOfficialEnabled
	originalAlipayAppID := setting.AlipayOfficialAppID
	originalAlipayPrivateKey := setting.AlipayOfficialPrivateKey
	originalAlipayPublicKey := setting.AlipayOfficialAlipayPublicKey
	t.Cleanup(func() {
		setting.AlipayOfficialEnabled = originalAlipayEnabled
		setting.AlipayOfficialAppID = originalAlipayAppID
		setting.AlipayOfficialPrivateKey = originalAlipayPrivateKey
		setting.AlipayOfficialAlipayPublicKey = originalAlipayPublicKey
	})
	setting.AlipayOfficialEnabled = true
	setting.AlipayOfficialAppID = "expected_app"
	setting.AlipayOfficialPrivateKey = privateKey
	setting.AlipayOfficialAlipayPublicKey = publicKey

	require.NoError(t, db.Create(&model.User{Id: 84, Username: "alipay-notify-app-user", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.TopUp{
		UserId:          84,
		Amount:          2,
		Money:           12.08,
		TradeNo:         "ALIPAY_APP_MISMATCH",
		PaymentMethod:   model.PaymentMethodAlipayOfficial,
		PaymentProvider: model.PaymentProviderAlipayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}).Error)
	params := map[string]string{
		"app_id":       "wrong_app",
		"out_trade_no": "ALIPAY_APP_MISMATCH",
		"trade_no":     "202605150000000001",
		"trade_status": "TRADE_SUCCESS",
		"total_amount": "12.08",
		"sign_type":    "RSA2",
	}
	sign, err := signSubscriptionControllerAlipayParams(privateKey, params)
	require.NoError(t, err)
	params["sign"] = sign

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/alipay/official/notify", strings.NewReader(form.Encode()))
	ctx.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	AlipayOfficialNotify(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "fail", recorder.Body.String())
	topUp := model.GetTopUpByTradeNo("ALIPAY_APP_MISMATCH")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
}

func TestValidateAlipayOfficialRefundResponseRejectsMismatchedSuccessPayload(t *testing.T) {
	refund := &model.TopUpRefund{
		TradeNo:      "ALIPAY_REFUND_ORDER",
		OutRequestNo: "ALIPAY_REFUND_ORDER_RF_1",
		RefundAmount: 12.08,
	}

	err := validateAlipayOfficialRefundResponse(&service.AlipayOfficialOpenAPIResponse{
		OutTradeNo:   "ALIPAY_REFUND_ORDER",
		OutRequestNo: "OTHER_RF",
		FundChange:   "Y",
		RefundFee:    "12.08",
	}, refund)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "out_request_no")

	err = validateAlipayOfficialRefundResponse(&service.AlipayOfficialOpenAPIResponse{
		OutTradeNo:   "ALIPAY_REFUND_ORDER",
		OutRequestNo: "ALIPAY_REFUND_ORDER_RF_1",
		FundChange:   "Y",
		RefundFee:    "11.00",
	}, refund)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "金额不一致")
}

func TestQueryWechatPayOfficialTopUpStatusRejectsMismatchedMerchantContext(t *testing.T) {
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
	originalQueryHTTPClient := wechatPayOfficialQueryHTTPClient
	t.Cleanup(func() {
		setting.WechatPayOfficialEnabled = originalWechatEnabled
		setting.WechatPayOfficialAppID = originalWechatAppID
		setting.WechatPayOfficialMchID = originalWechatMchID
		setting.WechatPayOfficialCertificateSerial = originalWechatSerial
		setting.WechatPayOfficialAPIv3Key = originalWechatAPIv3Key
		setting.WechatPayOfficialPrivateKey = originalWechatPrivateKey
		setting.WechatPayOfficialPlatformPublicKey = originalWechatPlatformPublicKey
		wechatPayOfficialQueryHTTPClient = originalQueryHTTPClient
	})

	setting.WechatPayOfficialEnabled = true
	setting.WechatPayOfficialAppID = "wx_expected_app"
	setting.WechatPayOfficialMchID = "1900000109"
	setting.WechatPayOfficialCertificateSerial = "merchant-serial"
	setting.WechatPayOfficialAPIv3Key = "12345678901234567890123456789012"
	setting.WechatPayOfficialPrivateKey = merchantPrivateKey
	setting.WechatPayOfficialPlatformPublicKey = platformPublicKey

	require.NoError(t, db.Create(&model.User{Id: 85, Username: "wechat-status-context-user", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.TopUp{
		UserId:          85,
		Amount:          2,
		Money:           1.23,
		TradeNo:         "WX_STATUS_CONTEXT",
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}).Error)
	wechatPayOfficialQueryHTTPClient = newWechatQueryResponseClient(t, platformPrivateKey, `{"appid":"wx_wrong_app","mchid":"1900000109","out_trade_no":"WX_STATUS_CONTEXT","transaction_id":"420000000000000010","trade_state":"SUCCESS","trade_type":"NATIVE","amount":{"total":123,"currency":"CNY"}}`)

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/user/wechat-pay/official/status", gin.H{
		"trade_no": "WX_STATUS_CONTEXT",
	})
	ctx.Set("id", 85)

	QueryWechatPayOfficialTopUpStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "appid")
	topUp := model.GetTopUpByTradeNo("WX_STATUS_CONTEXT")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
}

func TestQueryWechatPayOfficialTopUpStatusCompletesTopUp(t *testing.T) {
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
	originalQuotaPerUnit := common.QuotaPerUnit
	originalQueryHTTPClient := wechatPayOfficialQueryHTTPClient
	t.Cleanup(func() {
		setting.WechatPayOfficialEnabled = originalWechatEnabled
		setting.WechatPayOfficialAppID = originalWechatAppID
		setting.WechatPayOfficialMchID = originalWechatMchID
		setting.WechatPayOfficialCertificateSerial = originalWechatSerial
		setting.WechatPayOfficialAPIv3Key = originalWechatAPIv3Key
		setting.WechatPayOfficialPrivateKey = originalWechatPrivateKey
		setting.WechatPayOfficialPlatformPublicKey = originalWechatPlatformPublicKey
		common.QuotaPerUnit = originalQuotaPerUnit
		wechatPayOfficialQueryHTTPClient = originalQueryHTTPClient
	})

	setting.WechatPayOfficialEnabled = true
	setting.WechatPayOfficialAppID = "wx_expected_app"
	setting.WechatPayOfficialMchID = "1900000109"
	setting.WechatPayOfficialCertificateSerial = "merchant-serial"
	setting.WechatPayOfficialAPIv3Key = "12345678901234567890123456789012"
	setting.WechatPayOfficialPrivateKey = merchantPrivateKey
	setting.WechatPayOfficialPlatformPublicKey = platformPublicKey
	common.QuotaPerUnit = 100

	require.NoError(t, db.Create(&model.User{Id: 86, Username: "wechat-status-success-user", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.TopUp{
		UserId:          86,
		Amount:          2,
		Money:           1.23,
		TradeNo:         "WX_STATUS_SUCCESS",
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusPending,
		CreateTime:      common.GetTimestamp(),
	}).Error)
	wechatPayOfficialQueryHTTPClient = newWechatQueryResponseClient(t, platformPrivateKey, `{"appid":"wx_expected_app","mchid":"1900000109","out_trade_no":"WX_STATUS_SUCCESS","transaction_id":"420000000000000011","trade_state":"SUCCESS","trade_type":"NATIVE","amount":{"total":123,"currency":"CNY"}}`)

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/user/wechat-pay/official/status", gin.H{
		"trade_no": "WX_STATUS_SUCCESS",
	})
	ctx.Set("id", 86)

	QueryWechatPayOfficialTopUpStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"status":"success"`)
	topUp := model.GetTopUpByTradeNo("WX_STATUS_SUCCESS")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusSuccess, topUp.Status)
	var user model.User
	require.NoError(t, db.First(&user, 86).Error)
	assert.Equal(t, 200, user.Quota)
}

func TestAdminQueryWechatPayOfficialRefundSyncsSubscriptionOrder(t *testing.T) {
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
	t.Cleanup(func() {
		setting.WechatPayOfficialEnabled = originalWechatEnabled
		setting.WechatPayOfficialAppID = originalWechatAppID
		setting.WechatPayOfficialMchID = originalWechatMchID
		setting.WechatPayOfficialCertificateSerial = originalWechatSerial
		setting.WechatPayOfficialAPIv3Key = originalWechatAPIv3Key
		setting.WechatPayOfficialPrivateKey = originalWechatPrivateKey
		setting.WechatPayOfficialPlatformPublicKey = originalWechatPlatformPublicKey
	})

	setting.WechatPayOfficialEnabled = true
	setting.WechatPayOfficialAppID = "wx_expected_app"
	setting.WechatPayOfficialMchID = "1900000109"
	setting.WechatPayOfficialCertificateSerial = "merchant-serial"
	setting.WechatPayOfficialAPIv3Key = "12345678901234567890123456789012"
	setting.WechatPayOfficialPrivateKey = merchantPrivateKey
	setting.WechatPayOfficialPlatformPublicKey = platformPublicKey

	require.NoError(t, db.Create(&model.User{Id: 87, Username: "wechat-refund-sub-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Title:         "Refund Plan",
		PriceAmount:   100,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, db.Create(plan).Error)
	order := &model.SubscriptionOrder{
		UserId:          87,
		PlanId:          plan.Id,
		Money:           100.00,
		TradeNo:         "WXSUB_REFUND_QUERY_SYNC",
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      common.GetTimestamp() - 100,
		CompleteTime:    common.GetTimestamp() - 100,
	}
	require.NoError(t, order.Insert())
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:      87,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		StartTime:   common.GetTimestamp() - 100,
		EndTime:     common.GetTimestamp() + 900,
		Status:      "active",
		Source:      "order",
	}).Error)
	refund, err := model.CreateOfficialPaymentRefund(model.OfficialPaymentRefundCreateParams{
		TradeNo:         order.TradeNo,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		RefundAmount:    10.00,
		Reason:          "query sync",
		OutRequestNo:    "WXSUB_REFUND_QUERY_SYNC_RF_1",
	})
	require.NoError(t, err)
	require.NotNil(t, refund)

	newWechatPayOfficialClient = func() *service.WechatPayOfficialClient {
		return &service.WechatPayOfficialClient{
			AppID:             setting.WechatPayOfficialAppID,
			MchID:             setting.WechatPayOfficialMchID,
			CertificateSerial: setting.WechatPayOfficialCertificateSerial,
			APIv3Key:          setting.WechatPayOfficialAPIv3Key,
			PrivateKey:        setting.WechatPayOfficialPrivateKey,
			PlatformPublicKey: setting.WechatPayOfficialPlatformPublicKey,
			HTTPClient:        newWechatRefundQueryResponseClient(t, platformPrivateKey, `{"refund_id":"503000000000000000","out_refund_no":"WXSUB_REFUND_QUERY_SYNC_RF_1","out_trade_no":"WXSUB_REFUND_QUERY_SYNC","status":"SUCCESS","amount":{"refund":1000,"total":10000,"currency":"CNY"}}`),
		}
	}
	t.Cleanup(func() {
		newWechatPayOfficialClient = defaultWechatPayOfficialClient
	})

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/user/topup/wechat-pay-official/refund-query", gin.H{
		"out_request_no": "WXSUB_REFUND_QUERY_SYNC_RF_1",
	})

	AdminQueryWechatPayOfficialRefund(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	refund = model.GetTopUpRefundByOutRequestNo("WXSUB_REFUND_QUERY_SYNC_RF_1")
	require.NotNil(t, refund)
	assert.Equal(t, model.TopUpRefundStatusSuccess, refund.Status)
	reloadedOrder := model.GetSubscriptionOrderByTradeNo("WXSUB_REFUND_QUERY_SYNC")
	require.NotNil(t, reloadedOrder)
	assert.Equal(t, common.TopUpStatusPartialRefunded, reloadedOrder.Status)
}

func TestAdminQueryWechatPayOfficialRefundClosedRollsBackSubscriptionOrder(t *testing.T) {
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
	t.Cleanup(func() {
		setting.WechatPayOfficialEnabled = originalWechatEnabled
		setting.WechatPayOfficialAppID = originalWechatAppID
		setting.WechatPayOfficialMchID = originalWechatMchID
		setting.WechatPayOfficialCertificateSerial = originalWechatSerial
		setting.WechatPayOfficialAPIv3Key = originalWechatAPIv3Key
		setting.WechatPayOfficialPrivateKey = originalWechatPrivateKey
		setting.WechatPayOfficialPlatformPublicKey = originalWechatPlatformPublicKey
		newWechatPayOfficialClient = defaultWechatPayOfficialClient
	})

	setting.WechatPayOfficialEnabled = true
	setting.WechatPayOfficialAppID = "wx_expected_app"
	setting.WechatPayOfficialMchID = "1900000109"
	setting.WechatPayOfficialCertificateSerial = "merchant-serial"
	setting.WechatPayOfficialAPIv3Key = "12345678901234567890123456789012"
	setting.WechatPayOfficialPrivateKey = merchantPrivateKey
	setting.WechatPayOfficialPlatformPublicKey = platformPublicKey

	require.NoError(t, db.Create(&model.User{Id: 88, Username: "wechat-refund-closed-sub-user", Status: common.UserStatusEnabled}).Error)
	plan := &model.SubscriptionPlan{
		Title:         "Refund Closed Plan",
		PriceAmount:   100,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, db.Create(plan).Error)
	order := &model.SubscriptionOrder{
		UserId:          88,
		PlanId:          plan.Id,
		Money:           100.00,
		TradeNo:         "WXSUB_REFUND_QUERY_CLOSED",
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      common.GetTimestamp() - 100,
		CompleteTime:    common.GetTimestamp() - 100,
	}
	require.NoError(t, order.Insert())
	require.NoError(t, db.Create(&model.UserSubscription{
		UserId:      88,
		PlanId:      plan.Id,
		AmountTotal: 1000,
		StartTime:   common.GetTimestamp() - 100,
		EndTime:     common.GetTimestamp() + 900,
		Status:      "active",
		Source:      "order",
	}).Error)
	refund, err := model.CreateOfficialPaymentRefund(model.OfficialPaymentRefundCreateParams{
		TradeNo:         order.TradeNo,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		RefundAmount:    10.00,
		Reason:          "closed rollback",
		OutRequestNo:    "WXSUB_REFUND_QUERY_CLOSED_RF_1",
	})
	require.NoError(t, err)
	require.NoError(t, model.MarkTopUpRefundSuccess(refund.OutRequestNo, "503000000000000001", `{"status":"SUCCESS"}`))
	require.NoError(t, model.SyncSubscriptionOrderRefundState(order.TradeNo, model.PaymentProviderWechatPayOfficial, false))
	reloadedOrder := model.GetSubscriptionOrderByTradeNo(order.TradeNo)
	require.NotNil(t, reloadedOrder)
	require.Equal(t, common.TopUpStatusPartialRefunded, reloadedOrder.Status)

	newWechatPayOfficialClient = func() *service.WechatPayOfficialClient {
		return &service.WechatPayOfficialClient{
			AppID:             setting.WechatPayOfficialAppID,
			MchID:             setting.WechatPayOfficialMchID,
			CertificateSerial: setting.WechatPayOfficialCertificateSerial,
			APIv3Key:          setting.WechatPayOfficialAPIv3Key,
			PrivateKey:        setting.WechatPayOfficialPrivateKey,
			PlatformPublicKey: setting.WechatPayOfficialPlatformPublicKey,
			HTTPClient:        newWechatRefundQueryResponseClient(t, platformPrivateKey, `{"refund_id":"503000000000000001","out_refund_no":"WXSUB_REFUND_QUERY_CLOSED_RF_1","out_trade_no":"WXSUB_REFUND_QUERY_CLOSED","refund_status":"CLOSED","amount":{"refund":1000,"total":10000,"currency":"CNY"}}`),
		}
	}

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/user/topup/wechat-pay-official/refund-query", gin.H{
		"out_request_no": "WXSUB_REFUND_QUERY_CLOSED_RF_1",
	})

	AdminQueryWechatPayOfficialRefund(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	refund = model.GetTopUpRefundByOutRequestNo("WXSUB_REFUND_QUERY_CLOSED_RF_1")
	require.NotNil(t, refund)
	assert.Equal(t, model.TopUpRefundStatusFailed, refund.Status)
	reloadedOrder = model.GetSubscriptionOrderByTradeNo("WXSUB_REFUND_QUERY_CLOSED")
	require.NotNil(t, reloadedOrder)
	assert.Equal(t, common.TopUpStatusSuccess, reloadedOrder.Status)
}

func TestAdminQueryWechatPayOfficialRefundClosedRollsBackBalanceTopUp(t *testing.T) {
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
	originalQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		setting.WechatPayOfficialEnabled = originalWechatEnabled
		setting.WechatPayOfficialAppID = originalWechatAppID
		setting.WechatPayOfficialMchID = originalWechatMchID
		setting.WechatPayOfficialCertificateSerial = originalWechatSerial
		setting.WechatPayOfficialAPIv3Key = originalWechatAPIv3Key
		setting.WechatPayOfficialPrivateKey = originalWechatPrivateKey
		setting.WechatPayOfficialPlatformPublicKey = originalWechatPlatformPublicKey
		common.QuotaPerUnit = originalQuotaPerUnit
		newWechatPayOfficialClient = defaultWechatPayOfficialClient
	})

	setting.WechatPayOfficialEnabled = true
	setting.WechatPayOfficialAppID = "wx_expected_app"
	setting.WechatPayOfficialMchID = "1900000109"
	setting.WechatPayOfficialCertificateSerial = "merchant-serial"
	setting.WechatPayOfficialAPIv3Key = "12345678901234567890123456789012"
	setting.WechatPayOfficialPrivateKey = merchantPrivateKey
	setting.WechatPayOfficialPlatformPublicKey = platformPublicKey
	common.QuotaPerUnit = 100

	require.NoError(t, db.Create(&model.User{Id: 89, Username: "wechat-refund-closed-topup-user", Status: common.UserStatusEnabled, Quota: 1000}).Error)
	topUp := &model.TopUp{
		UserId:          89,
		Amount:          10,
		Money:           10.00,
		TradeNo:         "WX_REFUND_QUERY_CLOSED",
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      common.GetTimestamp() - 100,
		CompleteTime:    common.GetTimestamp() - 100,
	}
	require.NoError(t, topUp.Insert())
	refund, err := model.CreateOfficialPaymentRefund(model.OfficialPaymentRefundCreateParams{
		TradeNo:         topUp.TradeNo,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		RefundAmount:    1.00,
		Reason:          "closed balance rollback",
		OutRequestNo:    "WX_REFUND_QUERY_CLOSED_RF_1",
	})
	require.NoError(t, err)

	newWechatPayOfficialClient = func() *service.WechatPayOfficialClient {
		return &service.WechatPayOfficialClient{
			AppID:             setting.WechatPayOfficialAppID,
			MchID:             setting.WechatPayOfficialMchID,
			CertificateSerial: setting.WechatPayOfficialCertificateSerial,
			APIv3Key:          setting.WechatPayOfficialAPIv3Key,
			PrivateKey:        setting.WechatPayOfficialPrivateKey,
			PlatformPublicKey: setting.WechatPayOfficialPlatformPublicKey,
			HTTPClient:        newWechatRefundQueryResponseClient(t, platformPrivateKey, `{"refund_id":"503000000000000002","out_refund_no":"WX_REFUND_QUERY_CLOSED_RF_1","out_trade_no":"WX_REFUND_QUERY_CLOSED","refund_status":"CLOSED","amount":{"refund":100,"total":1000,"currency":"CNY"}}`),
		}
	}

	ctx, recorder := newSubscriptionControllerContext(t, http.MethodPost, "/api/user/topup/wechat-pay-official/refund-query", gin.H{
		"out_request_no": refund.OutRequestNo,
	})

	AdminQueryWechatPayOfficialRefund(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	refund = model.GetTopUpRefundByOutRequestNo("WX_REFUND_QUERY_CLOSED_RF_1")
	require.NotNil(t, refund)
	assert.Equal(t, model.TopUpRefundStatusFailed, refund.Status)
	updatedTopUp := model.GetTopUpByTradeNo("WX_REFUND_QUERY_CLOSED")
	require.NotNil(t, updatedTopUp)
	assert.Equal(t, common.TopUpStatusSuccess, updatedTopUp.Status)
	assert.InDelta(t, 0.0, updatedTopUp.RefundedMoney, 0.000001)
	assert.Equal(t, int64(0), updatedTopUp.RefundedQuota)
	var user model.User
	require.NoError(t, db.First(&user, 89).Error)
	assert.Equal(t, 1000, user.Quota)
}
