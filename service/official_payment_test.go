package service

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/require"
)

func generateOfficialPaymentTestKey(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateBytes, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	privatePEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes}))
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}))
	return privatePEM, publicPEM
}

func TestBuildAlipayOfficialPageExecuteFormUsesOfficialProductCodes(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)

	form, err := BuildAlipayOfficialPageExecuteForm(AlipayOfficialBuildParams{
		AppID:       "app_123",
		PrivateKey:  privateKey,
		Method:      AlipayOfficialPagePayMethod,
		NotifyURL:   "https://example.com/api/alipay/notify",
		ReturnURL:   "https://example.com/console/topup",
		OutTradeNo:  "ORDER123",
		TotalAmount: "1.23",
		Subject:     "StuHelper AI recharge",
	})
	require.NoError(t, err)
	require.Contains(t, form, `alipay.trade.page.pay`)
	require.Contains(t, form, `FAST_INSTANT_TRADE_PAY`)
	require.Contains(t, form, `name="sign"`)

	wapForm, err := BuildAlipayOfficialPageExecuteForm(AlipayOfficialBuildParams{
		AppID:       "app_123",
		PrivateKey:  privateKey,
		Method:      AlipayOfficialWapPayMethod,
		NotifyURL:   "https://example.com/api/alipay/notify",
		ReturnURL:   "https://example.com/console/topup",
		OutTradeNo:  "ORDER124",
		TotalAmount: "1.23",
		Subject:     "StuHelper AI recharge",
	})
	require.NoError(t, err)
	require.Contains(t, wapForm, `alipay.trade.wap.pay`)
	require.Contains(t, wapForm, `QUICK_WAP_WAY`)
}

func TestBuildAlipayOfficialSignedValuesUsesRecognizedProductCodes(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)

	pageValues, err := buildAlipayOfficialSignedValues(AlipayOfficialBuildParams{
		AppID:        "app_123",
		PrivateKey:   privateKey,
		AppAuthToken: "app_auth_token_123",
		Method:       AlipayOfficialPagePayMethod,
		NotifyURL:    "https://example.com/api/alipay/notify",
		ReturnURL:    "https://example.com/console/topup",
		OutTradeNo:   "ORDER_123",
		TotalAmount:  "1.23",
		Subject:      "StuHelper AI recharge",
	})
	require.NoError(t, err)
	require.Contains(t, pageValues.Get("biz_content"), `"product_code":"FAST_INSTANT_TRADE_PAY"`)
	require.Equal(t, "app_auth_token_123", pageValues.Get("app_auth_token"))
	require.Contains(t, buildAlipaySignContent(valuesToMap(pageValues), false), "app_auth_token=app_auth_token_123")

	wapValues, err := buildAlipayOfficialSignedValues(AlipayOfficialBuildParams{
		AppID:       "app_123",
		PrivateKey:  privateKey,
		Method:      AlipayOfficialWapPayMethod,
		NotifyURL:   "https://example.com/api/alipay/notify",
		ReturnURL:   "https://example.com/console/topup",
		OutTradeNo:  "ORDER_124",
		TotalAmount: "1.23",
		Subject:     "StuHelper AI recharge",
	})
	require.NoError(t, err)
	require.Contains(t, wapValues.Get("biz_content"), `"product_code":"QUICK_WAP_WAY"`)
}

func TestBuildAlipayOfficialWapPayIncludesQuitURLInBizContent(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)

	wapValues, err := buildAlipayOfficialSignedValues(AlipayOfficialBuildParams{
		AppID:       "app_123",
		PrivateKey:  privateKey,
		Method:      AlipayOfficialWapPayMethod,
		NotifyURL:   "https://example.com/api/alipay/notify",
		ReturnURL:   "https://example.com/console/topup?show_history=true",
		QuitURL:     "https://example.com/console/topup",
		OutTradeNo:  "ORDER_125",
		TotalAmount: "1.23",
		Subject:     "StuHelper AI recharge",
	})
	require.NoError(t, err)

	var bizContent map[string]string
	require.NoError(t, common.Unmarshal([]byte(wapValues.Get("biz_content")), &bizContent))
	require.Equal(t, AlipayOfficialWapPayProductCode, bizContent["product_code"])
	require.Equal(t, "https://example.com/console/topup", bizContent["quit_url"])
	require.Equal(t, "https://example.com/console/topup?show_history=true", wapValues.Get("return_url"))
}

func TestAlipayOfficialClientRefundBuildsSignedOpenAPIRequest(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)
	client := &AlipayOfficialClient{
		AppID:        "app_123",
		AppAuthToken: "app_auth_token_123",
		PrivateKey:   privateKey,
	}

	values, responseKey, err := client.buildOpenAPIRequestValues(AlipayOfficialRefundMethod, map[string]any{
		"out_trade_no":   "ORDER_REFUND",
		"refund_amount":  "0.50",
		"refund_reason":  "partial refund",
		"out_request_no": "ORDER_REFUND_RF_1",
		"query_options":  []string{"deposit_back_info"},
	})
	require.NoError(t, err)
	require.Equal(t, "alipay_trade_refund_response", responseKey)
	require.Equal(t, "app_123", values.Get("app_id"))
	require.Equal(t, AlipayOfficialRefundMethod, values.Get("method"))
	require.Equal(t, "app_auth_token_123", values.Get("app_auth_token"))
	require.Equal(t, "RSA2", values.Get("sign_type"))
	require.NotEmpty(t, values.Get("sign"))
	require.Contains(t, buildAlipaySignContent(valuesToMap(values), false), "app_auth_token=app_auth_token_123")

	var bizContent map[string]any
	require.NoError(t, common.Unmarshal([]byte(values.Get("biz_content")), &bizContent))
	require.Equal(t, "ORDER_REFUND", bizContent["out_trade_no"])
	require.Equal(t, "0.50", bizContent["refund_amount"])
	require.Equal(t, "ORDER_REFUND_RF_1", bizContent["out_request_no"])
	require.Equal(t, []any{"deposit_back_info"}, bizContent["query_options"])
	require.NotContains(t, values.Encode(), "refund_amount=0.50")
}

func TestAlipayOfficialClientTradeCloseUsesV3RestAPI(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)
	var capturedPath string
	var capturedBody string
	var capturedAuthorization string
	var capturedAppAuthToken string
	client := &AlipayOfficialClient{
		AppID:        "app_123",
		AppAuthToken: "app_auth_token_123",
		PrivateKey:   privateKey,
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedPath = req.URL.Path
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			capturedBody = string(body)
			capturedAuthorization = req.Header.Get("Authorization")
			capturedAppAuthToken = req.Header.Get("alipay-app-auth-token")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(`{"trade_no":"202605142200000000","out_trade_no":"ALIPAY_1"}`)),
			}, nil
		}).client(),
	}

	response, err := client.TradeClose(t.Context(), map[string]any{
		"out_trade_no": "ALIPAY_1",
	})

	require.NoError(t, err)
	require.Equal(t, "/v3/alipay/trade/close", capturedPath)
	require.JSONEq(t, `{"out_trade_no":"ALIPAY_1"}`, capturedBody)
	require.Contains(t, capturedAuthorization, "ALIPAY-SHA256withRSA ")
	require.Contains(t, capturedAuthorization, "app_id=app_123")
	require.Contains(t, capturedAuthorization, "nonce=")
	require.Contains(t, capturedAuthorization, "timestamp=")
	require.Contains(t, capturedAuthorization, "sign=")
	require.Equal(t, "app_auth_token_123", capturedAppAuthToken)
	require.Equal(t, "ALIPAY_1", response.OutTradeNo)
	require.Equal(t, "202605142200000000", response.TradeNo)
}

func TestAlipayOfficialClientTradeCloseV3DetectsTradeNotFound(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)
	client := &AlipayOfficialClient{
		AppID:      "app_123",
		PrivateKey: privateKey,
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{},
				Body: io.NopCloser(strings.NewReader(`{
					"code":"ACQ.TRADE_NOT_EXIST",
					"message":"交易不存在"
				}`)),
			}, nil
		}).client(),
	}

	_, err := client.TradeClose(t.Context(), map[string]any{
		"out_trade_no": "ALIPAY_NOT_EXIST",
	})

	require.Error(t, err)
	require.True(t, IsAlipayOfficialTradeNotFound(err))
}

func TestBuildAlipayOfficialSignedValuesIncludesTimeoutExpress(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)

	values, err := buildAlipayOfficialSignedValues(AlipayOfficialBuildParams{
		AppID:          "app_123",
		PrivateKey:     privateKey,
		Method:         AlipayOfficialPagePayMethod,
		NotifyURL:      "https://example.com/api/alipay/notify",
		OutTradeNo:     "ORDER_TIMEOUT",
		TotalAmount:    "1.23",
		Subject:        "StuHelper AI recharge",
		TimeoutExpress: "10m",
	})

	require.NoError(t, err)
	var bizContent map[string]string
	require.NoError(t, common.Unmarshal([]byte(values.Get("biz_content")), &bizContent))
	require.Equal(t, "10m", bizContent["timeout_express"])
}

func TestParseAlipayOfficialOpenAPIResponseRequiresSuccessNode(t *testing.T) {
	body := []byte(`{"alipay_trade_refund_response":{"code":"10000","msg":"Success","fund_change":"Y","refund_fee":"0.50"},"sign":"ignored"}`)

	response, err := parseAlipayOfficialOpenAPIResponse(body, "alipay_trade_refund_response")
	require.NoError(t, err)
	require.Equal(t, "10000", response.Code)
	require.Equal(t, "Y", response.FundChange)
	require.Equal(t, "0.50", response.RefundFee)

	_, err = parseAlipayOfficialOpenAPIResponse([]byte(`{"error_response":{"code":"40004","sub_msg":"Business Failed"}}`), "alipay_trade_refund_response")
	require.Error(t, err)
}

func TestParseAlipayOfficialOpenAPIResponseDetectsTradeNotFound(t *testing.T) {
	response, err := parseAlipayOfficialOpenAPIResponse(
		[]byte(`{"alipay_trade_query_response":{"code":"40004","msg":"Business Failed","sub_code":"ACQ.TRADE_NOT_EXIST","sub_msg":"交易不存在"},"sign":"ignored"}`),
		"alipay_trade_query_response",
	)

	require.Error(t, err)
	require.NotNil(t, response)
	require.True(t, IsAlipayOfficialTradeNotFound(err))

	response, err = parseAlipayOfficialOpenAPIResponse(
		[]byte(`{"error_response":{"code":"40004","msg":"Business Failed","sub_msg":"交易不存在"},"sign":"ignored"}`),
		"alipay_trade_query_response",
	)
	require.Error(t, err)
	require.Nil(t, response)
	require.True(t, IsAlipayOfficialTradeNotFound(err))
}

func TestVerifyAlipayOfficialOpenAPIResponseSignatureUsesRawResponseNode(t *testing.T) {
	privateKey, publicKey := generateOfficialPaymentTestKey(t)
	rawResponse := `{"code":"10000","msg":"Success","fund_change":"Y","refund_fee":"0.50"}`
	sign, err := signRSA2(rawResponse, privateKey)
	require.NoError(t, err)
	body, err := common.Marshal(map[string]any{
		"alipay_trade_refund_response": jsonRawMessageForTest(t, rawResponse),
		"sign":                         sign,
	})
	require.NoError(t, err)

	client := &AlipayOfficialClient{AlipayPublicKey: publicKey}
	require.NoError(t, client.verifyOpenAPIResponseSignature(body, "alipay_trade_refund_response"))

	tampered := []byte(`{"alipay_trade_refund_response":{"code":"10000","msg":"Success","fund_change":"N","refund_fee":"0.50"},"sign":"` + sign + `"}`)
	require.Error(t, client.verifyOpenAPIResponseSignature(tampered, "alipay_trade_refund_response"))
}

func TestVerifyAlipayOfficialNotifyExcludesSignAndSignType(t *testing.T) {
	privateKey, publicKey := generateOfficialPaymentTestKey(t)
	params := map[string]string{
		"app_id":       "app_123",
		"out_trade_no": "ORDER123",
		"trade_no":     "202405132200000000",
		"trade_status": "TRADE_SUCCESS",
		"total_amount": "1.23",
		"sign_type":    "RSA2",
	}
	sign, err := signRSA2(buildAlipaySignContent(params, true), privateKey)
	require.NoError(t, err)
	params["sign"] = sign

	require.True(t, VerifyAlipayOfficialNotify(params, publicKey))
	params["total_amount"] = "2.00"
	require.False(t, VerifyAlipayOfficialNotify(params, publicKey))
}

func TestBuildWechatPaySignatureMessageUsesRequiredTrailingNewline(t *testing.T) {
	message := buildWechatPaySignatureMessage("post", "/v3/pay/transactions/native", "1710000000", "nonce", []byte(`{"a":1}`))
	require.Equal(t, "POST\n/v3/pay/transactions/native\n1710000000\nnonce\n{\"a\":1}\n", message)
}

func TestVerifyWechatPayOfficialNotifySignatureUsesWechatHeaderMessage(t *testing.T) {
	platformKey, platformPublicKey := generateOfficialPaymentTestKeyPair(t)
	timestamp := "1710000000"
	nonce := "wechat-notify-nonce"
	body := []byte(`{"id":"notify-id","resource":{"algorithm":"AEAD_AES_256_GCM"}}`)
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, platformKey, crypto.SHA256, digest[:])
	require.NoError(t, err)

	require.True(t, VerifyWechatPayOfficialNotifySignature(
		timestamp,
		nonce,
		base64.StdEncoding.EncodeToString(signature),
		body,
		platformPublicKey,
	))
	require.False(t, VerifyWechatPayOfficialNotifySignature(
		timestamp,
		nonce,
		base64.StdEncoding.EncodeToString(signature),
		[]byte(`{"id":"tampered"}`),
		platformPublicKey,
	))
}

func TestVerifyWechatPayOfficialResponseSignatureUsesWechatHeaderMessage(t *testing.T) {
	platformKey, platformPublicKey := generateOfficialPaymentTestKeyPair(t)
	timestamp := "1710000000"
	nonce := "wechat-response-nonce"
	body := []byte(`{"code_url":"weixin://wxpay/bizpayurl?pr=test"}`)
	signature := signWechatPayHeaderMessage(t, platformKey, timestamp, nonce, body)

	require.NoError(t, VerifyWechatPayOfficialResponseSignature(
		timestamp,
		nonce,
		signature,
		body,
		platformPublicKey,
	))
	require.Error(t, VerifyWechatPayOfficialResponseSignature(
		timestamp,
		nonce,
		signature,
		[]byte(`{"code_url":"tampered"}`),
		platformPublicKey,
	))
}

func TestWechatPayOfficialPrepayVerifiesPlatformResponseSignature(t *testing.T) {
	merchantPrivateKey, _ := generateOfficialPaymentTestKey(t)
	platformKey, platformPublicKey := generateOfficialPaymentTestKeyPair(t)
	responseBody := []byte(`{"code_url":"weixin://wxpay/bizpayurl?pr=test"}`)
	timestamp := "1710000000"
	nonce := "response-nonce"
	signature := signWechatPayHeaderMessage(t, platformKey, timestamp, nonce, responseBody)

	client := &WechatPayOfficialClient{
		AppID:             "wx123",
		MchID:             "1900000109",
		CertificateSerial: "merchant-serial",
		PrivateKey:        merchantPrivateKey,
		PlatformPublicKey: platformPublicKey,
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(string(responseBody))),
			}
			resp.Header.Set("Wechatpay-Timestamp", timestamp)
			resp.Header.Set("Wechatpay-Nonce", nonce)
			resp.Header.Set("Wechatpay-Signature", signature)
			return resp, nil
		}).client(),
	}

	result, err := client.Prepay(t.Context(), WechatPayOfficialPrepayParams{
		Description: "StuHelper AI recharge",
		OutTradeNo:  "WXPAY-1",
		NotifyURL:   "https://example.com/api/wechat-pay/official/notify",
		AmountTotal: 100,
		TradeType:   "pc",
	})
	require.NoError(t, err)
	require.Equal(t, "weixin://wxpay/bizpayurl?pr=test", result.CodeURL)

	client.HTTPClient = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(string(responseBody))),
		}
		resp.Header.Set("Wechatpay-Timestamp", timestamp)
		resp.Header.Set("Wechatpay-Nonce", nonce)
		resp.Header.Set("Wechatpay-Signature", signature)
		resp.Body = io.NopCloser(strings.NewReader(`{"code_url":"tampered"}`))
		return resp, nil
	}).client()

	_, err = client.Prepay(t.Context(), WechatPayOfficialPrepayParams{
		Description: "StuHelper AI recharge",
		OutTradeNo:  "WXPAY-2",
		NotifyURL:   "https://example.com/api/wechat-pay/official/notify",
		AmountTotal: 100,
		TradeType:   "pc",
	})
	require.Error(t, err)
}

func TestWechatPayOfficialPrepayDetectsH5Unavailable(t *testing.T) {
	merchantPrivateKey, _ := generateOfficialPaymentTestKey(t)
	client := &WechatPayOfficialClient{
		AppID:             "wx123",
		MchID:             "1900000109",
		CertificateSerial: "merchant-serial",
		PrivateKey:        merchantPrivateKey,
		PlatformPublicKey: "unused",
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "/v3/pay/transactions/h5", req.URL.Path)
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(`{"code":"NO_AUTH","message":"商户未开通H5支付权限"}`)),
			}, nil
		}).client(),
	}

	_, err := client.Prepay(t.Context(), WechatPayOfficialPrepayParams{
		Description: "StuHelper AI recharge",
		OutTradeNo:  "WXPAY-H5-NOAUTH",
		NotifyURL:   "https://example.com/api/wechat-pay/official/notify",
		AmountTotal: 100,
		ClientIP:    "127.0.0.1",
		WapURL:      "https://example.com/console/topup",
		WapName:     "StuHelper AI",
		TradeType:   "h5",
	})
	require.ErrorIs(t, err, ErrWechatPayOfficialH5Unavailable)
}

func TestWechatPayOfficialQueryTransactionByOutTradeNo(t *testing.T) {
	merchantPrivateKey, _ := generateOfficialPaymentTestKey(t)
	platformKey, platformPublicKey := generateOfficialPaymentTestKeyPair(t)
	responseBody := []byte(`{"appid":"wx123","mchid":"1900000109","out_trade_no":"WX_1_QUERY","transaction_id":"420000000000000000","trade_state":"SUCCESS","trade_type":"NATIVE","amount":{"total":123,"currency":"CNY"}}`)
	timestamp := "1710000001"
	nonce := "query-response-nonce"
	signature := signWechatPayHeaderMessage(t, platformKey, timestamp, nonce, responseBody)

	client := &WechatPayOfficialClient{
		AppID:             "wx123",
		MchID:             "1900000109",
		CertificateSerial: "merchant-serial",
		PrivateKey:        merchantPrivateKey,
		PlatformPublicKey: platformPublicKey,
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "/v3/pay/transactions/out-trade-no/WX_1_QUERY", req.URL.Path)
			require.Equal(t, "mchid=1900000109", req.URL.RawQuery)
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader(responseBody)),
			}
			resp.Header.Set("Wechatpay-Timestamp", timestamp)
			resp.Header.Set("Wechatpay-Nonce", nonce)
			resp.Header.Set("Wechatpay-Signature", signature)
			return resp, nil
		}).client(),
	}

	transaction, err := client.QueryTransactionByOutTradeNo(t.Context(), "WX_1_QUERY")

	require.NoError(t, err)
	require.NotNil(t, transaction)
	require.Equal(t, "WX_1_QUERY", transaction.OutTradeNo)
	require.Equal(t, "SUCCESS", transaction.TradeState)
	require.Equal(t, int64(123), transaction.Amount.Total)
}

func TestWechatPayOfficialCloseTransactionByOutTradeNo(t *testing.T) {
	merchantPrivateKey, _ := generateOfficialPaymentTestKey(t)
	var capturedBody string
	client := &WechatPayOfficialClient{
		AppID:             "wx123",
		MchID:             "1900000109",
		CertificateSerial: "merchant-serial",
		PrivateKey:        merchantPrivateKey,
		PlatformPublicKey: "unused",
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodPost, req.Method)
			require.Equal(t, "/v3/pay/transactions/out-trade-no/WX_1_CLOSE/close", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			capturedBody = string(body)
			require.Contains(t, req.Header.Get("Authorization"), `mchid="1900000109"`)
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}).client(),
	}

	err := client.CloseTransactionByOutTradeNo(t.Context(), "WX_1_CLOSE")

	require.NoError(t, err)
	require.JSONEq(t, `{"mchid":"1900000109"}`, capturedBody)
}

func TestWechatPayOfficialRefundBuildsV3Request(t *testing.T) {
	merchantPrivateKey, _ := generateOfficialPaymentTestKey(t)
	platformKey, platformPublicKey := generateOfficialPaymentTestKeyPair(t)
	responseBody := []byte(`{"refund_id":"503000000000000000","out_refund_no":"WX_1_RF_1","out_trade_no":"WX_1_REFUND","transaction_id":"420000000000000000","status":"PROCESSING","amount":{"refund":123,"total":1000,"currency":"CNY"}}`)
	timestamp := "1710000002"
	nonce := "refund-response-nonce"
	signature := signWechatPayHeaderMessage(t, platformKey, timestamp, nonce, responseBody)
	var capturedBody string
	client := &WechatPayOfficialClient{
		AppID:             "wx123",
		MchID:             "1900000109",
		CertificateSerial: "merchant-serial",
		PrivateKey:        merchantPrivateKey,
		PlatformPublicKey: platformPublicKey,
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodPost, req.Method)
			require.Equal(t, "/v3/refund/domestic/refunds", req.URL.Path)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			capturedBody = string(body)
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader(responseBody)),
			}
			resp.Header.Set("Wechatpay-Timestamp", timestamp)
			resp.Header.Set("Wechatpay-Nonce", nonce)
			resp.Header.Set("Wechatpay-Signature", signature)
			return resp, nil
		}).client(),
	}

	refund, err := client.Refund(t.Context(), WechatPayOfficialRefundParams{
		OutTradeNo:  "WX_1_REFUND",
		OutRefundNo: "WX_1_RF_1",
		Reason:      "user request",
		NotifyURL:   "https://example.com/api/wechat-pay/official/notify",
		RefundFen:   123,
		TotalFen:    1000,
	})

	require.NoError(t, err)
	require.JSONEq(t, `{
		"out_trade_no":"WX_1_REFUND",
		"out_refund_no":"WX_1_RF_1",
		"reason":"user request",
		"notify_url":"https://example.com/api/wechat-pay/official/notify",
		"amount":{"refund":123,"total":1000,"currency":"CNY"}
	}`, capturedBody)
	require.Equal(t, "WX_1_RF_1", refund.OutRefundNo)
	require.Equal(t, "PROCESSING", refund.EffectiveStatus())
	require.Equal(t, int64(123), refund.Amount.Refund)
}

func TestWechatPayOfficialQueryRefundByOutRefundNo(t *testing.T) {
	merchantPrivateKey, _ := generateOfficialPaymentTestKey(t)
	platformKey, platformPublicKey := generateOfficialPaymentTestKeyPair(t)
	responseBody := []byte(`{"refund_id":"503000000000000000","out_refund_no":"WX_1_RF_QUERY","out_trade_no":"WX_1_REFUND","status":"SUCCESS","amount":{"refund":123,"total":1000,"currency":"CNY"}}`)
	timestamp := "1710000003"
	nonce := "refund-query-response-nonce"
	signature := signWechatPayHeaderMessage(t, platformKey, timestamp, nonce, responseBody)
	client := &WechatPayOfficialClient{
		AppID:             "wx123",
		MchID:             "1900000109",
		CertificateSerial: "merchant-serial",
		PrivateKey:        merchantPrivateKey,
		PlatformPublicKey: platformPublicKey,
		HTTPClient: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "/v3/refund/domestic/refunds/WX_1_RF_QUERY", req.URL.Path)
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewReader(responseBody)),
			}
			resp.Header.Set("Wechatpay-Timestamp", timestamp)
			resp.Header.Set("Wechatpay-Nonce", nonce)
			resp.Header.Set("Wechatpay-Signature", signature)
			return resp, nil
		}).client(),
	}

	refund, err := client.QueryRefundByOutRefundNo(t.Context(), "WX_1_RF_QUERY")

	require.NoError(t, err)
	require.Equal(t, "WX_1_RF_QUERY", refund.OutRefundNo)
	require.Equal(t, "SUCCESS", refund.EffectiveStatus())
}

func TestDecodeWechatPayOfficialNotifyDecryptsTransaction(t *testing.T) {
	apiV3Key := "12345678901234567890123456789012"
	nonce := "nonce1234567"
	associatedData := "transaction"
	plain := []byte(`{"appid":"wx123","mchid":"1900000109","out_trade_no":"WX-1","trade_state":"SUCCESS","trade_type":"NATIVE","amount":{"total":123,"currency":"CNY"}}`)

	block, err := aes.NewCipher([]byte(apiV3Key))
	require.NoError(t, err)
	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)
	ciphertext := aead.Seal(nil, []byte(nonce), plain, []byte(associatedData))

	body, err := common.Marshal(map[string]any{
		"id":            "notify-id",
		"create_time":   "2026-05-13T12:00:00+08:00",
		"event_type":    "TRANSACTION.SUCCESS",
		"resource_type": "encrypt-resource",
		"summary":       "支付成功",
		"resource": map[string]any{
			"original_type":   "transaction",
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      base64.StdEncoding.EncodeToString(ciphertext),
			"associated_data": associatedData,
			"nonce":           nonce,
		},
	})
	require.NoError(t, err)

	envelope, transaction, err := DecodeWechatPayOfficialNotify(body, apiV3Key)
	require.NoError(t, err)
	require.Equal(t, "TRANSACTION.SUCCESS", envelope.EventType)
	require.Equal(t, "WX-1", transaction.OutTradeNo)
	require.Equal(t, "SUCCESS", transaction.TradeState)
	require.Equal(t, int64(123), transaction.Amount.Total)
}

func TestDecodeWechatPayOfficialRefundNotifyAcceptsClosedEvent(t *testing.T) {
	apiV3Key := "12345678901234567890123456789012"
	nonce := "refundnonce1"
	associatedData := "refund"
	plain := []byte(`{"mchid":"1900000109","out_trade_no":"WX-REFUND-CLOSED","out_refund_no":"WX-REFUND-CLOSED-RF-1","refund_id":"503000000000000001","refund_status":"CLOSED","amount":{"refund":123,"total":1000,"currency":"CNY"}}`)

	block, err := aes.NewCipher([]byte(apiV3Key))
	require.NoError(t, err)
	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)
	ciphertext := aead.Seal(nil, []byte(nonce), plain, []byte(associatedData))

	body, err := common.Marshal(map[string]any{
		"id":            "refund-notify-id",
		"create_time":   "2026-05-15T12:00:00+08:00",
		"event_type":    "REFUND.CLOSED",
		"resource_type": "encrypt-resource",
		"summary":       "退款关闭",
		"resource": map[string]any{
			"original_type":   "refund",
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      base64.StdEncoding.EncodeToString(ciphertext),
			"associated_data": associatedData,
			"nonce":           nonce,
		},
	})
	require.NoError(t, err)

	envelope, transaction, err := DecodeWechatPayOfficialNotify(body, apiV3Key)
	require.NoError(t, err)
	require.Equal(t, "REFUND.CLOSED", envelope.EventType)
	require.Equal(t, "refund", envelope.Resource.OriginalType)
	require.Empty(t, transaction.OutTradeNo)

	refundEnvelope, refund, err := DecodeWechatPayOfficialRefundNotify(body, apiV3Key)
	require.NoError(t, err)
	require.Equal(t, "REFUND.CLOSED", refundEnvelope.EventType)
	require.Equal(t, "WX-REFUND-CLOSED-RF-1", refund.OutRefundNo)
	require.Equal(t, "REFUNDCLOSE", refund.EffectiveStatus())
}

func TestNormalizeRSAKeyAcceptsBase64DER(t *testing.T) {
	privateKey, _ := generateOfficialPaymentTestKey(t)
	block, _ := pem.Decode([]byte(privateKey))
	require.NotNil(t, block)

	normalized, err := normalizeRSAPrivateKey(base64.StdEncoding.EncodeToString(block.Bytes))
	require.NoError(t, err)
	require.True(t, strings.Contains(normalized, "BEGIN PRIVATE KEY"))
}

func TestParseRSAPublicKeyAcceptsCertificatePEM(t *testing.T) {
	platformKey, _ := generateOfficialPaymentTestKeyPair(t)
	certificatePEM := generateOfficialPaymentTestCertificate(t, platformKey)

	parsed, err := parseRSAPublicKey(certificatePEM)
	require.NoError(t, err)
	require.Equal(t, platformKey.PublicKey.N.String(), parsed.N.String())
}

func generateOfficialPaymentTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}))
	return key, publicPEM
}

func signWechatPayHeaderMessage(t *testing.T, privateKey *rsa.PrivateKey, timestamp string, nonce string, body []byte) string {
	t.Helper()
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signature)
}

func jsonRawMessageForTest(t *testing.T, value string) json.RawMessage {
	t.Helper()
	return json.RawMessage(value)
}

func generateOfficialPaymentTestCertificate(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (f roundTripFunc) client() *http.Client {
	return &http.Client{Transport: f}
}
