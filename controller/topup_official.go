package controller

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"

	"github.com/gin-gonic/gin"
)

const (
	officialPaymentScenePC = "pc"
	officialPaymentSceneH5 = "h5"
)

type OfficialPayRequest struct {
	Amount int64  `json:"amount"`
	Scene  string `json:"scene"`
}

func RequestAlipayOfficialAmount(c *gin.Context) {
	requestOfficialAmount(c, setting.AlipayOfficialMinTopUp, setting.AlipayOfficialUnitPrice)
}

func RequestWechatPayOfficialAmount(c *gin.Context) {
	requestOfficialAmount(c, setting.WechatPayOfficialMinTopUp, setting.WechatPayOfficialUnitPrice)
}

func requestOfficialAmount(c *gin.Context, minTopUp int, unitPrice float64) {
	var req OfficialPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < int64(minTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", minTopUp)})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getOfficialPayMoney(req.Amount, group, unitPrice)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": formatOfficialPayMoney(payMoney)})
}

func RequestAlipayOfficialPay(c *gin.Context) {
	if !isAlipayOfficialTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝官方支付未启用或配置不完整"})
		return
	}

	var req OfficialPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	scene := normalizeOfficialPaymentScene(req.Scene)
	if req.Amount < int64(setting.AlipayOfficialMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.AlipayOfficialMinTopUp)})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getOfficialPayMoney(req.Amount, group, setting.AlipayOfficialUnitPrice)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := buildOfficialTradeNo("ALIPAY", id)
	topUp := buildOfficialTopUp(id, req.Amount, payMoney, tradeNo, model.PaymentMethodAlipayOfficial, model.PaymentProviderAlipayOfficial)
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	callbackAddr := service.GetCallbackAddress()
	notifyURL := callbackAddr + "/api/alipay/official/notify"
	if strings.TrimSpace(setting.AlipayOfficialNotifyURL) != "" {
		notifyURL = strings.TrimSpace(setting.AlipayOfficialNotifyURL)
	}
	returnURL := paymentReturnPath("/console/topup?show_history=true")
	if strings.TrimSpace(setting.AlipayOfficialReturnURL) != "" {
		returnURL = strings.TrimSpace(setting.AlipayOfficialReturnURL)
	}

	method := service.AlipayOfficialPagePayMethod
	if scene == officialPaymentSceneH5 {
		method = service.AlipayOfficialWapPayMethod
	}
	form, err := service.BuildAlipayOfficialPageExecuteForm(service.AlipayOfficialBuildParams{
		AppID:            setting.AlipayOfficialAppID,
		PrivateKey:       setting.AlipayOfficialPrivateKey,
		AppCertSN:        setting.AlipayOfficialAppCertSN,
		AlipayRootCertSN: setting.AlipayOfficialRootCertSN,
		AlipayCertSN:     setting.AlipayOfficialAlipayCertSN,
		Sandbox:          setting.AlipayOfficialSandbox,
		Method:           method,
		NotifyURL:        notifyURL,
		ReturnURL:        returnURL,
		OutTradeNo:       tradeNo,
		TotalAmount:      formatOfficialPayMoney(payMoney),
		Subject:          fmt.Sprintf("StuHelper AI 充值 %d", req.Amount),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 生成表单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f scene=%s", id, tradeNo, req.Amount, payMoney, scene))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"form_html":    form,
			"payment_type": "form",
			"order_id":     tradeNo,
			"scene":        scene,
		},
	})
}

func RequestWechatPayOfficialPay(c *gin.Context) {
	if !isWechatPayOfficialTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "微信支付官方支付未启用或配置不完整"})
		return
	}

	var req OfficialPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	scene := normalizeOfficialPaymentScene(req.Scene)
	if req.Amount < int64(setting.WechatPayOfficialMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.WechatPayOfficialMinTopUp)})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getOfficialPayMoney(req.Amount, group, setting.WechatPayOfficialUnitPrice)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := buildOfficialTradeNo("WXPAY", id)
	topUp := buildOfficialTopUp(id, req.Amount, payMoney, tradeNo, model.PaymentMethodWechatPayOfficial, model.PaymentProviderWechatPayOfficial)
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	callbackAddr := service.GetCallbackAddress()
	notifyURL := callbackAddr + "/api/wechat-pay/official/notify"
	if strings.TrimSpace(setting.WechatPayOfficialNotifyURL) != "" {
		notifyURL = strings.TrimSpace(setting.WechatPayOfficialNotifyURL)
	}
	wapURL := paymentReturnPath("/console/topup")
	if strings.TrimSpace(setting.WechatPayOfficialReturnURL) != "" {
		wapURL = strings.TrimSpace(setting.WechatPayOfficialReturnURL)
	}

	client := &service.WechatPayOfficialClient{
		AppID:             setting.WechatPayOfficialAppID,
		MchID:             setting.WechatPayOfficialMchID,
		CertificateSerial: setting.WechatPayOfficialCertificateSerial,
		APIv3Key:          setting.WechatPayOfficialAPIv3Key,
		PrivateKey:        setting.WechatPayOfficialPrivateKey,
		PlatformPublicKey: setting.WechatPayOfficialPlatformPublicKey,
	}
	result, err := client.Prepay(c.Request.Context(), service.WechatPayOfficialPrepayParams{
		Description: fmt.Sprintf("StuHelper AI 充值 %d", req.Amount),
		OutTradeNo:  tradeNo,
		NotifyURL:   notifyURL,
		AmountTotal: yuanToFen(payMoney),
		ClientIP:    c.ClientIP(),
		WapURL:      wapURL,
		WapName:     "StuHelper AI",
		TradeType:   scene,
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 创建预支付订单失败 user_id=%d trade_no=%s scene=%s error=%q", id, tradeNo, scene, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f scene=%s", id, tradeNo, req.Amount, payMoney, scene))
	data := gin.H{
		"order_id": tradeNo,
		"scene":    scene,
	}
	if scene == officialPaymentSceneH5 {
		data["payment_type"] = "redirect"
		data["payment_url"] = result.H5URL
	} else {
		data["payment_type"] = "qrcode"
		data["code_url"] = result.CodeURL
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": data})
}

func AlipayOfficialNotify(c *gin.Context) {
	if !isAlipayOfficialWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝官方支付 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.String(http.StatusForbidden, "fail")
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 webhook 表单解析失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.String(http.StatusBadRequest, "fail")
		return
	}

	params := make(map[string]string, len(c.Request.Form))
	for key := range c.Request.Form {
		params[key] = c.Request.Form.Get(key)
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 webhook 收到请求 path=%q client_ip=%s params=%q", c.Request.RequestURI, c.ClientIP(), common.GetJsonString(params)))
	if !service.VerifyAlipayOfficialNotify(params, setting.AlipayOfficialAlipayPublicKey) {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝官方支付 webhook 验签失败 path=%q client_ip=%s trade_no=%s", c.Request.RequestURI, c.ClientIP(), params["out_trade_no"]))
		c.String(http.StatusOK, "fail")
		return
	}

	tradeNo := params["out_trade_no"]
	tradeStatus := params["trade_status"]
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 webhook 忽略非成功状态 trade_no=%s status=%s client_ip=%s", tradeNo, tradeStatus, c.ClientIP()))
		c.String(http.StatusOK, "success")
		return
	}
	paidMoney, err := decimal.NewFromString(params["total_amount"])
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝官方支付 webhook 金额解析失败 trade_no=%s total_amount=%s client_ip=%s", tradeNo, params["total_amount"], c.ClientIP()))
		c.String(http.StatusOK, "fail")
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)
	if err := model.RechargeOfficialPayment(tradeNo, model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial, c.ClientIP(), paidMoney.InexactFloat64()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 充值处理失败 trade_no=%s client_ip=%s error=%q", tradeNo, c.ClientIP(), err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 充值成功 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
	c.String(http.StatusOK, "success")
}

func WechatPayOfficialNotify(c *gin.Context) {
	if !isWechatPayOfficialWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.JSON(http.StatusForbidden, gin.H{"code": "FAIL", "message": "webhook disabled"})
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 读取请求体失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "bad request"})
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 收到请求 path=%q client_ip=%s body=%q", c.Request.RequestURI, c.ClientIP(), string(body)))

	if !service.VerifyWechatPayOfficialNotifySignature(
		c.GetHeader("Wechatpay-Timestamp"),
		c.GetHeader("Wechatpay-Nonce"),
		c.GetHeader("Wechatpay-Signature"),
		body,
		setting.WechatPayOfficialPlatformPublicKey,
	) {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 验签失败 path=%q client_ip=%s serial=%s", c.Request.RequestURI, c.ClientIP(), c.GetHeader("Wechatpay-Serial")))
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "invalid signature"})
		return
	}

	envelope, transaction, err := service.DecodeWechatPayOfficialNotify(body, setting.WechatPayOfficialAPIv3Key)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 解密失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "invalid payload"})
		return
	}
	if envelope.EventType != "TRANSACTION.SUCCESS" || transaction.TradeState != "SUCCESS" {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 忽略非成功状态 trade_no=%s event=%s state=%s client_ip=%s", transaction.OutTradeNo, envelope.EventType, transaction.TradeState, c.ClientIP()))
		c.Status(http.StatusNoContent)
		return
	}

	LockOrder(transaction.OutTradeNo)
	defer UnlockOrder(transaction.OutTradeNo)
	if err := model.RechargeOfficialPayment(transaction.OutTradeNo, model.PaymentProviderWechatPayOfficial, model.PaymentMethodWechatPayOfficial, c.ClientIP(), float64(transaction.Amount.Total)/100); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 充值处理失败 trade_no=%s transaction_id=%s client_ip=%s error=%q", transaction.OutTradeNo, transaction.TransactionID, c.ClientIP(), err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "retry"})
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 充值成功 trade_no=%s transaction_id=%s client_ip=%s", transaction.OutTradeNo, transaction.TransactionID, c.ClientIP()))
	c.Status(http.StatusNoContent)
}

func getOfficialPayMoney(amount int64, group string, unitPrice float64) float64 {
	dAmount := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount = dAmount.Div(decimal.NewFromFloat(common.QuotaPerUnit))
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok && ds > 0 {
		discount = ds
	}

	return dAmount.
		Mul(decimal.NewFromFloat(unitPrice)).
		Mul(decimal.NewFromFloat(topupGroupRatio)).
		Mul(decimal.NewFromFloat(discount)).
		RoundCeil(2).
		InexactFloat64()
}

func formatOfficialPayMoney(payMoney float64) string {
	return formatPayMoneyToCents(payMoney)
}

func yuanToFen(payMoney float64) int64 {
	return payMoneyYuanToFen(payMoney)
}

func normalizeOfficialPaymentScene(scene string) string {
	if strings.EqualFold(strings.TrimSpace(scene), officialPaymentSceneH5) {
		return officialPaymentSceneH5
	}
	return officialPaymentScenePC
}

func buildOfficialTradeNo(prefix string, userID int) string {
	return fmt.Sprintf("%s_%d_%d_%s", prefix, userID, time.Now().UnixMilli(), randstr.String(6))
}

func normalizeOfficialTopUpAmount(amount int64) int64 {
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return amount
	}
	normalized := decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	if normalized < 1 {
		return 1
	}
	return normalized
}

func buildOfficialTopUp(userID int, amount int64, money float64, tradeNo string, paymentMethod string, paymentProvider string) *model.TopUp {
	return &model.TopUp{
		UserId:          userID,
		Amount:          normalizeOfficialTopUpAmount(amount),
		Money:           money,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentMethod,
		PaymentProvider: paymentProvider,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
}
