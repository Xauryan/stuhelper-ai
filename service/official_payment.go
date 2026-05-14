package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
)

var ErrAlipayOfficialTradeNotFound = errors.New("alipay trade not found")

const (
	AlipayOfficialPagePayMethod      = "alipay.trade.page.pay"
	AlipayOfficialWapPayMethod       = "alipay.trade.wap.pay"
	AlipayOfficialRefundMethod       = "alipay.trade.refund"
	AlipayOfficialRefundQueryMethod  = "alipay.trade.fastpay.refund.query"
	AlipayOfficialTradeQueryMethod   = "alipay.trade.query"
	AlipayOfficialTradeCloseMethod   = "alipay.trade.close"
	AlipayOfficialPagePayProductCode = "FAST_INSTANT_TRADE_PAY"
	AlipayOfficialWapPayProductCode  = "QUICK_WAP_WAY"
	alipayOfficialProductionGateway  = "https://openapi.alipay.com/gateway.do"
	alipayOfficialSandboxGateway     = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	alipayOfficialProductionV3Base   = "https://openapi.alipay.com"
	alipayOfficialSandboxV3Base      = "https://openapi-sandbox.dl.alipaydev.com"
	alipayOfficialTradeCloseV3Path   = "/v3/alipay/trade/close"
	alipayOfficialOpenAPITimeout     = 15 * time.Second
)

type AlipayOfficialBuildParams struct {
	AppID            string
	PrivateKey       string
	AppCertSN        string
	AlipayRootCertSN string
	AlipayCertSN     string
	Sandbox          bool
	Method           string
	NotifyURL        string
	ReturnURL        string
	QuitURL          string
	OutTradeNo       string
	TotalAmount      string
	Subject          string
	TimeoutExpress   string
}

type AlipayOfficialClient struct {
	AppID            string
	PrivateKey       string
	AppCertSN        string
	AlipayRootCertSN string
	AlipayCertSN     string
	AlipayPublicKey  string
	Sandbox          bool
	HTTPClient       *http.Client
}

type AlipayOfficialOpenAPIResponse struct {
	Code         string `json:"code"`
	Msg          string `json:"msg"`
	SubCode      string `json:"sub_code"`
	SubMsg       string `json:"sub_msg"`
	OutTradeNo   string `json:"out_trade_no"`
	TradeNo      string `json:"trade_no"`
	TradeStatus  string `json:"trade_status"`
	TotalAmount  string `json:"total_amount"`
	FundChange   string `json:"fund_change"`
	RefundFee    string `json:"refund_fee"`
	RefundAmount string `json:"refund_amount"`
	RefundStatus string `json:"refund_status"`
}

type AlipayOfficialError struct {
	Code     string
	Msg      string
	SubCode  string
	SubMsg   string
	Response *AlipayOfficialOpenAPIResponse
}

func (e *AlipayOfficialError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("Alipay %s failed: %s", e.Code, firstNonEmpty(e.SubMsg, e.Msg))
}

func (e *AlipayOfficialError) Unwrap() error {
	if e == nil || (!isAlipayTradeNotFoundPayload(e.SubCode, e.SubMsg) && !isAlipayTradeNotFoundPayload(e.Code, e.Msg)) {
		return nil
	}
	return ErrAlipayOfficialTradeNotFound
}

func IsAlipayOfficialTradeNotFound(err error) bool {
	return errors.Is(err, ErrAlipayOfficialTradeNotFound)
}

func isAlipayTradeNotFoundPayload(subCode string, subMsg string) bool {
	if strings.EqualFold(strings.TrimSpace(subCode), "ACQ.TRADE_NOT_EXIST") {
		return true
	}
	return strings.Contains(strings.TrimSpace(subMsg), "交易不存在")
}

type WechatPayOfficialClient struct {
	AppID             string
	MchID             string
	CertificateSerial string
	APIv3Key          string
	PrivateKey        string
	PlatformPublicKey string
	HTTPClient        *http.Client
}

type WechatPayOfficialPrepayParams struct {
	Description string
	OutTradeNo  string
	NotifyURL   string
	AmountTotal int64
	ClientIP    string
	WapURL      string
	WapName     string
	TradeType   string
}

type WechatPayOfficialPrepayResult struct {
	CodeURL string `json:"code_url,omitempty"`
	H5URL   string `json:"h5_url,omitempty"`
}

type WechatPayOfficialNotifyEnvelope struct {
	ID           string `json:"id"`
	CreateTime   string `json:"create_time"`
	EventType    string `json:"event_type"`
	ResourceType string `json:"resource_type"`
	Summary      string `json:"summary"`
	Resource     struct {
		OriginalType   string `json:"original_type"`
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

type WechatPayOfficialTransaction struct {
	AppID         string `json:"appid"`
	MchID         string `json:"mchid"`
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeType     string `json:"trade_type"`
	TradeState    string `json:"trade_state"`
	Attach        string `json:"attach"`
	Amount        struct {
		Total         int64  `json:"total"`
		PayerTotal    int64  `json:"payer_total"`
		Currency      string `json:"currency"`
		PayerCurrency string `json:"payer_currency"`
	} `json:"amount"`
}

func BuildAlipayOfficialPageExecuteForm(params AlipayOfficialBuildParams) (string, error) {
	values, err := buildAlipayOfficialSignedValues(params)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(`<form id="alipaysubmit" name="alipaysubmit" action="`)
	builder.WriteString(htmlEscape(resolveAlipayOfficialGateway(params.Sandbox)))
	builder.WriteString(`?charset=utf-8" method="POST">`)
	keys := sortedValueKeys(values)
	for _, key := range keys {
		builder.WriteString(`<input type="hidden" name="`)
		builder.WriteString(htmlEscape(key))
		builder.WriteString(`" value="`)
		builder.WriteString(htmlEscape(values.Get(key)))
		builder.WriteString(`"/>`)
	}
	builder.WriteString(`<input type="submit" value="ok" style="display:none;"></form>`)
	builder.WriteString(`<script>document.forms['alipaysubmit'].submit();</script>`)
	return builder.String(), nil
}

func VerifyAlipayOfficialNotify(params map[string]string, publicKey string) bool {
	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return false
	}
	signature, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}
	content := buildAlipaySignContent(params, true)
	publicKeyPEM, err := normalizeRSAPublicKey(publicKey)
	if err != nil {
		return false
	}
	pub, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return false
	}
	digest := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], signature) == nil
}

func buildAlipayOfficialSignedValues(params AlipayOfficialBuildParams) (url.Values, error) {
	method := strings.TrimSpace(params.Method)
	if method == "" {
		return nil, fmt.Errorf("missing Alipay method")
	}
	productCode := AlipayOfficialPagePayProductCode
	if method == AlipayOfficialWapPayMethod {
		productCode = AlipayOfficialWapPayProductCode
	}

	bizContent := map[string]string{
		"out_trade_no": params.OutTradeNo,
		"total_amount": params.TotalAmount,
		"subject":      params.Subject,
		"product_code": productCode,
	}
	if strings.TrimSpace(params.TimeoutExpress) != "" {
		bizContent["timeout_express"] = strings.TrimSpace(params.TimeoutExpress)
	}
	if method == AlipayOfficialWapPayMethod && strings.TrimSpace(params.QuitURL) != "" {
		bizContent["quit_url"] = strings.TrimSpace(params.QuitURL)
	}
	bizContentBytes, err := common.Marshal(bizContent)
	if err != nil {
		return nil, fmt.Errorf("marshal Alipay biz_content: %w", err)
	}

	values := url.Values{}
	values.Set("app_id", strings.TrimSpace(params.AppID))
	values.Set("method", method)
	values.Set("format", "JSON")
	values.Set("charset", "utf-8")
	values.Set("sign_type", "RSA2")
	values.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	values.Set("version", "1.0")
	values.Set("notify_url", strings.TrimSpace(params.NotifyURL))
	if strings.TrimSpace(params.ReturnURL) != "" {
		values.Set("return_url", strings.TrimSpace(params.ReturnURL))
	}
	if strings.TrimSpace(params.AppCertSN) != "" {
		values.Set("app_cert_sn", strings.TrimSpace(params.AppCertSN))
	}
	if strings.TrimSpace(params.AlipayRootCertSN) != "" {
		values.Set("alipay_root_cert_sn", strings.TrimSpace(params.AlipayRootCertSN))
	}
	if strings.TrimSpace(params.AlipayCertSN) != "" {
		values.Set("alipay_cert_sn", strings.TrimSpace(params.AlipayCertSN))
	}
	values.Set("biz_content", string(bizContentBytes))

	signContent := buildAlipaySignContent(valuesToMap(values), false)
	signature, err := signRSA2(signContent, params.PrivateKey)
	if err != nil {
		return nil, err
	}
	values.Set("sign", signature)
	return values, nil
}

func (c *AlipayOfficialClient) Refund(ctx context.Context, bizContent map[string]any) (*AlipayOfficialOpenAPIResponse, error) {
	return c.DoOpenAPI(ctx, AlipayOfficialRefundMethod, bizContent)
}

func (c *AlipayOfficialClient) RefundQuery(ctx context.Context, bizContent map[string]any) (*AlipayOfficialOpenAPIResponse, error) {
	return c.DoOpenAPI(ctx, AlipayOfficialRefundQueryMethod, bizContent)
}

func (c *AlipayOfficialClient) TradeQuery(ctx context.Context, bizContent map[string]any) (*AlipayOfficialOpenAPIResponse, error) {
	return c.DoOpenAPI(ctx, AlipayOfficialTradeQueryMethod, bizContent)
}

func (c *AlipayOfficialClient) TradeClose(ctx context.Context, bizContent map[string]any) (*AlipayOfficialOpenAPIResponse, error) {
	return c.DoOpenAPIV3(ctx, http.MethodPost, alipayOfficialTradeCloseV3Path, bizContent)
}

func (c *AlipayOfficialClient) DoOpenAPI(ctx context.Context, method string, bizContent map[string]any) (*AlipayOfficialOpenAPIResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("missing Alipay client")
	}
	values, responseKey, err := c.buildOpenAPIRequestValues(method, bizContent)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resolveAlipayOfficialGateway(c.Sandbox)+"?charset=utf-8", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build Alipay request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	req.Header.Set("Accept", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: alipayOfficialOpenAPITimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Alipay %s: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Alipay response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("Alipay %s failed with status %d: %s", method, resp.StatusCode, string(body))
	}
	if strings.TrimSpace(c.AlipayPublicKey) != "" {
		if err := c.verifyOpenAPIResponseSignature(body, responseKey); err != nil {
			return nil, err
		}
	}
	return parseAlipayOfficialOpenAPIResponse(body, responseKey)
}

func (c *AlipayOfficialClient) DoOpenAPIV3(ctx context.Context, method string, path string, body map[string]any) (*AlipayOfficialOpenAPIResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("missing Alipay client")
	}
	bodyBytes, err := common.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal Alipay V3 request body: %w", err)
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return nil, fmt.Errorf("missing Alipay V3 method")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequestWithContext(ctx, method, resolveAlipayOfficialV3Base(c.Sandbox)+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build Alipay V3 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Alipay-Request-Id", newAlipayOfficialV3Nonce())
	if strings.TrimSpace(c.AlipayRootCertSN) != "" {
		req.Header.Set("Alipay-Root-Cert-Sn", strings.TrimSpace(c.AlipayRootCertSN))
	}
	auth, err := c.BuildOpenAPIV3Authorization(method, path, bodyBytes)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: alipayOfficialOpenAPITimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Alipay V3 %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Alipay V3 response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, parseAlipayOfficialV3Error(responseBody, resp.StatusCode)
	}
	if strings.TrimSpace(c.AlipayPublicKey) != "" {
		if err := verifyAlipayOfficialV3ResponseSignature(resp.Header, responseBody, c.AlipayPublicKey); err != nil {
			return nil, err
		}
	}
	return parseAlipayOfficialV3Response(responseBody)
}

func (c *AlipayOfficialClient) BuildOpenAPIV3Authorization(method string, path string, body []byte) (string, error) {
	if c == nil {
		return "", fmt.Errorf("missing Alipay client")
	}
	authString := "app_id=" + strings.TrimSpace(c.AppID)
	if strings.TrimSpace(c.AppCertSN) != "" {
		authString += ",app_cert_sn=" + strings.TrimSpace(c.AppCertSN)
	}
	authString += ",nonce=" + newAlipayOfficialV3Nonce()
	authString += ",timestamp=" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	authString += ",expired_seconds=600"
	signString := authString + "\n" + strings.ToUpper(strings.TrimSpace(method)) + "\n" + path + "\n" + string(body) + "\n"
	signature, err := signRSA2(signString, c.PrivateKey)
	if err != nil {
		return "", err
	}
	return "ALIPAY-SHA256withRSA " + authString + ",sign=" + signature, nil
}

func (c *AlipayOfficialClient) buildOpenAPIRequestValues(method string, bizContent map[string]any) (url.Values, string, error) {
	if c == nil {
		return nil, "", fmt.Errorf("missing Alipay client")
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return nil, "", fmt.Errorf("missing Alipay method")
	}
	bizContentBytes, err := common.Marshal(bizContent)
	if err != nil {
		return nil, "", fmt.Errorf("marshal Alipay biz_content: %w", err)
	}

	values := url.Values{}
	values.Set("app_id", strings.TrimSpace(c.AppID))
	values.Set("method", method)
	values.Set("format", "JSON")
	values.Set("charset", "utf-8")
	values.Set("sign_type", "RSA2")
	values.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	values.Set("version", "1.0")
	if strings.TrimSpace(c.AppCertSN) != "" {
		values.Set("app_cert_sn", strings.TrimSpace(c.AppCertSN))
	}
	if strings.TrimSpace(c.AlipayRootCertSN) != "" {
		values.Set("alipay_root_cert_sn", strings.TrimSpace(c.AlipayRootCertSN))
	}
	if strings.TrimSpace(c.AlipayCertSN) != "" {
		values.Set("alipay_cert_sn", strings.TrimSpace(c.AlipayCertSN))
	}
	values.Set("biz_content", string(bizContentBytes))

	signContent := buildAlipaySignContent(valuesToMap(values), false)
	signature, err := signRSA2(signContent, c.PrivateKey)
	if err != nil {
		return nil, "", err
	}
	values.Set("sign", signature)
	return values, alipayOpenAPIResponseKey(method), nil
}

func parseAlipayOfficialV3Response(body []byte) (*AlipayOfficialOpenAPIResponse, error) {
	var response AlipayOfficialOpenAPIResponse
	if err := common.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode Alipay V3 response: %w", err)
	}
	if response.Code == "" {
		response.Code = "10000"
		response.Msg = "Success"
	}
	return &response, nil
}

func parseAlipayOfficialV3Error(body []byte, statusCode int) error {
	var apiError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
		SubCode string `json:"sub_code"`
		SubMsg  string `json:"sub_msg"`
	}
	if err := common.Unmarshal(body, &apiError); err != nil {
		return fmt.Errorf("Alipay V3 failed with status %d: %s", statusCode, string(body))
	}
	message := firstNonEmpty(apiError.Message, apiError.Msg, apiError.SubMsg)
	subCode := firstNonEmpty(apiError.SubCode, apiError.Code)
	return &AlipayOfficialError{
		Code:    apiError.Code,
		Msg:     message,
		SubCode: subCode,
		SubMsg:  message,
	}
}

func alipayOpenAPIResponseKey(method string) string {
	return strings.ReplaceAll(method, ".", "_") + "_response"
}

func parseAlipayOfficialOpenAPIResponse(body []byte, responseKey string) (*AlipayOfficialOpenAPIResponse, error) {
	var envelope map[string]json.RawMessage
	if err := common.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode Alipay response envelope: %w", err)
	}
	rawResponse, ok := envelope[responseKey]
	if !ok {
		if rawError, hasError := envelope["error_response"]; hasError {
			var apiError AlipayOfficialOpenAPIResponse
			_ = common.Unmarshal(rawError, &apiError)
			return nil, &AlipayOfficialError{
				Code:     apiError.Code,
				Msg:      apiError.Msg,
				SubCode:  apiError.SubCode,
				SubMsg:   apiError.SubMsg,
				Response: &apiError,
			}
		}
		return nil, fmt.Errorf("missing Alipay response node %s", responseKey)
	}
	var response AlipayOfficialOpenAPIResponse
	if err := common.Unmarshal(rawResponse, &response); err != nil {
		return nil, fmt.Errorf("decode Alipay %s: %w", responseKey, err)
	}
	if response.Code != "10000" {
		return &response, &AlipayOfficialError{
			Code:     response.Code,
			Msg:      response.Msg,
			SubCode:  response.SubCode,
			SubMsg:   response.SubMsg,
			Response: &response,
		}
	}
	return &response, nil
}

func (c *AlipayOfficialClient) verifyOpenAPIResponseSignature(body []byte, responseKey string) error {
	var envelope map[string]json.RawMessage
	if err := common.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode Alipay response for signature: %w", err)
	}
	rawSign, ok := envelope["sign"]
	if !ok {
		return fmt.Errorf("missing Alipay response signature")
	}
	sign := common.JsonRawMessageToString(rawSign)
	if strings.TrimSpace(sign) == "" {
		return fmt.Errorf("empty Alipay response signature")
	}
	rawResponse, ok := envelope[responseKey]
	if !ok {
		return fmt.Errorf("missing Alipay signed response node %s", responseKey)
	}
	if !verifyRSA2RawPayload(string(rawResponse), sign, c.AlipayPublicKey) {
		return fmt.Errorf("invalid Alipay response signature")
	}
	return nil
}

func verifyRSA2RawPayload(content string, sign string, publicKey string) bool {
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sign))
	if err != nil {
		return false
	}
	publicKeyPEM, err := normalizeRSAPublicKey(publicKey)
	if err != nil {
		return false
	}
	pub, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return false
	}
	digest := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], signature) == nil
}

func verifyAlipayOfficialV3ResponseSignature(header http.Header, body []byte, publicKey string) error {
	timestamp := strings.TrimSpace(header.Get("Alipay-Timestamp"))
	nonce := strings.TrimSpace(header.Get("Alipay-Nonce"))
	signature := strings.TrimSpace(header.Get("Alipay-Signature"))
	if timestamp == "" || nonce == "" || signature == "" {
		return fmt.Errorf("missing Alipay V3 response signature")
	}
	signString := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	if !verifyRSA2RawPayload(signString, signature, publicKey) {
		return fmt.Errorf("invalid Alipay V3 response signature")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func newAlipayOfficialV3Nonce() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(bytes[:])
}

func signRSA2(content string, rawPrivateKey string) (string, error) {
	privateKeyPEM, err := normalizeRSAPrivateKey(rawPrivateKey)
	if err != nil {
		return "", err
	}
	privateKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign RSA2 payload: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func buildAlipaySignContent(params map[string]string, excludeSignType bool) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || value == "" || (excludeSignType && key == "sign_type") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func valuesToMap(values url.Values) map[string]string {
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	return params
}

func sortedValueKeys(values url.Values) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func resolveAlipayOfficialGateway(sandbox bool) string {
	if sandbox {
		return alipayOfficialSandboxGateway
	}
	return alipayOfficialProductionGateway
}

func resolveAlipayOfficialV3Base(sandbox bool) string {
	if sandbox {
		return alipayOfficialSandboxV3Base
	}
	return alipayOfficialProductionV3Base
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`"`, `&quot;`,
		`'`, `&#39;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
	)
	return replacer.Replace(value)
}

func (c *WechatPayOfficialClient) Prepay(ctx context.Context, params WechatPayOfficialPrepayParams) (*WechatPayOfficialPrepayResult, error) {
	if c == nil {
		return nil, fmt.Errorf("missing WeChat Pay client")
	}

	path := "/v3/pay/transactions/native"
	if params.TradeType == "h5" {
		path = "/v3/pay/transactions/h5"
	}
	body := map[string]any{
		"appid":        c.AppID,
		"mchid":        c.MchID,
		"description":  params.Description,
		"out_trade_no": params.OutTradeNo,
		"notify_url":   params.NotifyURL,
		"amount": map[string]any{
			"total":    params.AmountTotal,
			"currency": "CNY",
		},
	}
	if params.TradeType == "h5" {
		body["scene_info"] = map[string]any{
			"payer_client_ip": params.ClientIP,
			"h5_info": map[string]any{
				"type":     "Wap",
				"wap_url":  params.WapURL,
				"wap_name": params.WapName,
			},
		}
	}

	bodyBytes, err := common.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal WeChat Pay prepay payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mch.weixin.qq.com"+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build WeChat Pay request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	auth, err := c.BuildAuthorization(http.MethodPost, path, bodyBytes)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request WeChat Pay prepay: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read WeChat Pay prepay response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("WeChat Pay prepay failed with status %d: %s", resp.StatusCode, string(responseBody))
	}
	if err := VerifyWechatPayOfficialResponseSignature(
		resp.Header.Get("Wechatpay-Timestamp"),
		resp.Header.Get("Wechatpay-Nonce"),
		resp.Header.Get("Wechatpay-Signature"),
		responseBody,
		c.PlatformPublicKey,
	); err != nil {
		return nil, err
	}

	var result WechatPayOfficialPrepayResult
	if err := common.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode WeChat Pay prepay response: %w", err)
	}
	if params.TradeType == "h5" {
		if strings.TrimSpace(result.H5URL) == "" {
			return nil, fmt.Errorf("WeChat Pay returned empty h5_url")
		}
	} else if strings.TrimSpace(result.CodeURL) == "" {
		return nil, fmt.Errorf("WeChat Pay returned empty code_url")
	}
	return &result, nil
}

func (c *WechatPayOfficialClient) BuildAuthorization(method string, canonicalURL string, body []byte) (string, error) {
	if c == nil {
		return "", fmt.Errorf("missing WeChat Pay client")
	}
	nonce := common.GetRandomString(32)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := buildWechatPaySignatureMessage(method, canonicalURL, timestamp, nonce, body)
	signature, err := signRSA2(message, c.PrivateKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		c.MchID,
		nonce,
		signature,
		timestamp,
		c.CertificateSerial,
	), nil
}

func buildWechatPaySignatureMessage(method string, canonicalURL string, timestamp string, nonce string, body []byte) string {
	return strings.Join([]string{
		strings.ToUpper(method),
		canonicalURL,
		timestamp,
		nonce,
		string(body),
		"",
	}, "\n")
}

func VerifyWechatPayOfficialNotifySignature(timestamp string, nonce string, signature string, body []byte, platformPublicKey string) bool {
	return verifyWechatPayHeaderSignature(timestamp, nonce, signature, body, platformPublicKey) == nil
}

func VerifyWechatPayOfficialResponseSignature(timestamp string, nonce string, signature string, body []byte, platformPublicKey string) error {
	if err := verifyWechatPayHeaderSignature(timestamp, nonce, signature, body, platformPublicKey); err != nil {
		return fmt.Errorf("verify WeChat Pay response signature: %w", err)
	}
	return nil
}

func verifyWechatPayHeaderSignature(timestamp string, nonce string, signature string, body []byte, platformPublicKey string) error {
	if strings.TrimSpace(timestamp) == "" || strings.TrimSpace(nonce) == "" || strings.TrimSpace(signature) == "" {
		return fmt.Errorf("missing WeChat Pay signature header")
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return fmt.Errorf("decode WeChat Pay signature: %w", err)
	}
	publicKeyPEM, err := normalizeRSAPublicKey(platformPublicKey)
	if err != nil {
		return err
	}
	pub, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return err
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], signatureBytes); err != nil {
		return fmt.Errorf("invalid WeChat Pay signature: %w", err)
	}
	return nil
}

func DecodeWechatPayOfficialNotify(body []byte, apiV3Key string) (*WechatPayOfficialNotifyEnvelope, *WechatPayOfficialTransaction, error) {
	var envelope WechatPayOfficialNotifyEnvelope
	if err := common.Unmarshal(body, &envelope); err != nil {
		return nil, nil, fmt.Errorf("decode WeChat Pay notify envelope: %w", err)
	}
	if envelope.Resource.Algorithm != "AEAD_AES_256_GCM" {
		return nil, nil, fmt.Errorf("unsupported WeChat Pay notify algorithm: %s", envelope.Resource.Algorithm)
	}
	plain, err := decryptWechatPayResource(apiV3Key, envelope.Resource.AssociatedData, envelope.Resource.Nonce, envelope.Resource.Ciphertext)
	if err != nil {
		return nil, nil, err
	}
	var transaction WechatPayOfficialTransaction
	if err := common.Unmarshal(plain, &transaction); err != nil {
		return nil, nil, fmt.Errorf("decode WeChat Pay transaction: %w", err)
	}
	return &envelope, &transaction, nil
}

func decryptWechatPayResource(apiV3Key string, associatedData string, nonce string, ciphertext string) ([]byte, error) {
	key := []byte(apiV3Key)
	if len(key) != 32 {
		return nil, fmt.Errorf("WeChat Pay APIv3 key must be 32 bytes")
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode WeChat Pay ciphertext: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	plain, err := aead.Open(nil, []byte(nonce), cipherBytes, []byte(associatedData))
	if err != nil {
		return nil, fmt.Errorf("decrypt WeChat Pay resource: %w", err)
	}
	return plain, nil
}

func normalizeRSAPrivateKey(raw string) (string, error) {
	return normalizePEMKey(raw, "PRIVATE KEY", "RSA PRIVATE KEY")
}

func normalizeRSAPublicKey(raw string) (string, error) {
	return normalizePEMKey(raw, "PUBLIC KEY", "RSA PUBLIC KEY")
}

func normalizePEMKey(raw string, pkcs8Type string, pkcs1Type string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("%s is empty", strings.ToLower(pkcs8Type))
	}

	normalized := strings.TrimSpace(strings.ReplaceAll(raw, `\n`, "\n"))
	if strings.Contains(normalized, "BEGIN ") {
		block, _ := pem.Decode([]byte(normalized))
		if block == nil {
			return "", fmt.Errorf("invalid PEM encoded %s", strings.ToLower(pkcs8Type))
		}
		return string(pem.EncodeToMemory(block)), nil
	}

	der, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(normalized, "\n", ""))
	if err != nil {
		return "", fmt.Errorf("invalid base64 encoded %s: %w", strings.ToLower(pkcs8Type), err)
	}

	pemType := pkcs8Type
	if pkcs8Type == "PRIVATE KEY" {
		if _, err := x509.ParsePKCS8PrivateKey(der); err != nil {
			if _, err := x509.ParsePKCS1PrivateKey(der); err == nil {
				pemType = pkcs1Type
			} else {
				return "", fmt.Errorf("invalid RSA private key")
			}
		}
	} else {
		if _, err := x509.ParsePKIXPublicKey(der); err != nil {
			if _, err := x509.ParsePKCS1PublicKey(der); err == nil {
				pemType = pkcs1Type
			} else {
				return "", fmt.Errorf("invalid RSA public key")
			}
		}
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: der})), nil
}

func parseRSAPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid RSA private key PEM")
	}

	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
		}
		parsed, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return parsed, nil
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}
}

func parseRSAPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid RSA public key PEM")
	}

	switch block.Type {
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKIX public key: %w", err)
		}
		parsed, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not RSA")
		}
		return parsed, nil
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate: %w", err)
		}
		parsed, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("certificate public key is not RSA")
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("unsupported public key type: %s", block.Type)
	}
}
