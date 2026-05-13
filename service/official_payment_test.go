package service

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
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
	require.Contains(t, form, `FAST_INSTANT_PAY_PAY`)
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
	require.Contains(t, wapForm, `QUICK_WAP_PAY`)
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
