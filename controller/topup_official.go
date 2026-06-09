package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
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
	officialPaymentScenePC       = "pc"
	officialPaymentSceneH5       = "h5"
	alipayOfficialExpireInterval = time.Minute
	// 单次 tick 内最多花在过期清理上的总时间，避免单批关单失败链式重试拖死后台任务。
	alipayOfficialExpireTickBudget = 30 * time.Second
	// 单次 tick 内允许处理的批次数上限，配合 ListPendingTopUpsBefore 的 20 条/批避免长期积压。
	alipayOfficialExpireTickMaxBatches = 5
	// 支付宝侧 TradeClose/TradeQuery 均返回"交易不存在"的本地待支付订单，需要等待至少 24h
	// 再在本地标记过期：给瞬时错误和晚到的入账通知留窗口，同时打破"永远 pending → 每分钟反复扫描刷屏"的循环。
	alipayOfficialOrphanGraceDuration = 24 * time.Hour
)

var (
	alipayOfficialExpireTaskOnce    sync.Once
	alipayOfficialExpireTaskRunning sync.Mutex
)

var newWechatPayOfficialClient = defaultWechatPayOfficialClient

type OfficialPayRequest struct {
	Amount int64  `json:"amount"`
	Scene  string `json:"scene"`
}

type OfficialWechatPayStatusRequest struct {
	TradeNo string `json:"trade_no"`
}

type OfficialRefundPreviewRequest struct {
	TradeNo string `json:"trade_no"`
}

type OfficialRefundApplyRequest struct {
	TradeNo      string  `json:"trade_no"`
	RefundAmount float64 `json:"refund_amount"`
	Reason       string  `json:"reason"`
	RefundQRCode string  `json:"refund_qrcode"`
}

type AdminRefundRequestActionRequest struct {
	RequestId    int     `json:"request_id"`
	RefundAmount float64 `json:"refund_amount"`
	Reason       string  `json:"reason"`
	FullRefund   bool    `json:"full_refund"`
}

type AdminRejectRefundRequestRequest struct {
	RequestId int    `json:"request_id"`
	Reason    string `json:"reason"`
}

type wechatPayOfficialPrepayResponse struct {
	Result *service.WechatPayOfficialPrepayResult
	Scene  string
}

func RequestAlipayOfficialAmount(c *gin.Context) {
	requestOfficialAmount(c, setting.AlipayOfficialMinTopUp, setting.AlipayOfficialUnitPrice, setting.AlipayOfficialServiceFeePercent)
}

func RequestWechatPayOfficialAmount(c *gin.Context) {
	requestOfficialAmount(c, setting.WechatPayOfficialMinTopUp, setting.WechatPayOfficialUnitPrice, setting.WechatPayOfficialServiceFeePercent)
}

func requestOfficialAmount(c *gin.Context, minTopUp int, unitPrice float64, serviceFeePercent float64) {
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
	payMoney := getOfficialPayMoney(req.Amount, group, unitPrice, serviceFeePercent)
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
	payMoney := getOfficialPayMoneyBreakdown(req.Amount, group, setting.AlipayOfficialUnitPrice, setting.AlipayOfficialServiceFeePercent)
	if payMoney.TotalMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := buildOfficialTradeNo("ALIPAY", id)
	topUp := buildOfficialTopUp(id, req.Amount, payMoney.EffectiveMoney, payMoney.Fee, tradeNo, model.PaymentMethodAlipayOfficial, model.PaymentProviderAlipayOfficial)
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
		AppAuthToken:     setting.AlipayOfficialAppAuthToken,
		PrivateKey:       setting.AlipayOfficialPrivateKey,
		AppCertSN:        setting.AlipayOfficialAppCertSN,
		AlipayRootCertSN: setting.AlipayOfficialRootCertSN,
		AlipayCertSN:     setting.AlipayOfficialAlipayCertSN,
		Sandbox:          setting.AlipayOfficialSandbox,
		Method:           method,
		NotifyURL:        notifyURL,
		ReturnURL:        returnURL,
		QuitURL:          paymentReturnPath("/console/topup"),
		OutTradeNo:       tradeNo,
		TotalAmount:      formatOfficialPayMoney(payMoney.TotalMoney),
		Subject:          fmt.Sprintf("StuHelper AI 充值 %d", req.Amount),
		TimeoutExpress:   formatAlipayOfficialTimeoutExpress(setting.AlipayOfficialOrderTimeoutSec),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 生成表单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f fee=%.2f total_money=%.2f scene=%s", id, tradeNo, req.Amount, payMoney.EffectiveMoney, payMoney.Fee, payMoney.TotalMoney, scene))
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
	if scene == officialPaymentSceneH5 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前移动端不支持使用微信支付，请使用电脑端或选择其他支付方式"})
		return
	}
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
	payMoney := getOfficialPayMoneyBreakdown(req.Amount, group, setting.WechatPayOfficialUnitPrice, setting.WechatPayOfficialServiceFeePercent)
	if payMoney.TotalMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := buildWechatPayOfficialTradeNo("WX", id)
	topUp := buildOfficialTopUp(id, req.Amount, payMoney.EffectiveMoney, payMoney.Fee, tradeNo, model.PaymentMethodWechatPayOfficial, model.PaymentProviderWechatPayOfficial)
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
	prepay, err := prepayWechatPayOfficialWithNativeFallback(c.Request.Context(), client, service.WechatPayOfficialPrepayParams{
		Description: fmt.Sprintf("StuHelper AI 充值 %d", req.Amount),
		OutTradeNo:  tradeNo,
		NotifyURL:   notifyURL,
		AmountTotal: yuanToFen(payMoney.TotalMoney),
		ClientIP:    c.ClientIP(),
		WapURL:      wapURL,
		WapName:     "StuHelper AI",
		TradeType:   scene,
		TimeExpire:  formatWechatPayOfficialTimeExpire(setting.WechatPayOfficialOrderTimeoutSec),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 创建预支付订单失败 user_id=%d trade_no=%s scene=%s error=%q", id, tradeNo, scene, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f fee=%.2f total_money=%.2f scene=%s", id, tradeNo, req.Amount, payMoney.EffectiveMoney, payMoney.Fee, payMoney.TotalMoney, prepay.Scene))
	data := gin.H{
		"order_id":              tradeNo,
		"scene":                 prepay.Scene,
		"order_timeout_seconds": getWechatPayOfficialOrderTimeoutSeconds(),
	}
	if prepay.Scene == officialPaymentSceneH5 {
		data["payment_type"] = "redirect"
		data["payment_url"] = prepay.Result.H5URL
	} else {
		data["payment_type"] = "qrcode"
		data["code_url"] = prepay.Result.CodeURL
		if scene == officialPaymentSceneH5 {
			data["fallback"] = "native"
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": data})
}

func QueryWechatPayOfficialTopUpStatus(c *gin.Context) {
	if !isWechatPayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "微信支付官方支付未启用或配置不完整")
		return
	}
	var req OfficialWechatPayStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	tradeNo := strings.TrimSpace(req.TradeNo)
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		common.ApiError(c, model.ErrTopUpNotFound)
		return
	}
	if topUp.UserId != c.GetInt("id") || topUp.PaymentProvider != model.PaymentProviderWechatPayOfficial {
		common.ApiError(c, model.ErrPaymentMethodMismatch)
		return
	}
	if topUp.Status == common.TopUpStatusSuccess {
		common.ApiSuccess(c, gin.H{
			"trade_no": tradeNo,
			"status":   common.TopUpStatusSuccess,
		})
		return
	}
	if topUp.Status != common.TopUpStatusPending {
		common.ApiSuccess(c, gin.H{
			"trade_no": tradeNo,
			"status":   topUp.Status,
		})
		return
	}
	if isWechatPayOfficialOrderExpired(topUp.CreateTime) {
		if err := expireWechatPayOfficialPendingOrder(tradeNo); err != nil &&
			!errors.Is(err, model.ErrTopUpStatusInvalid) {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, gin.H{
			"trade_no": tradeNo,
			"status":   common.TopUpStatusExpired,
		})
		return
	}

	client := &service.WechatPayOfficialClient{
		AppID:             setting.WechatPayOfficialAppID,
		MchID:             setting.WechatPayOfficialMchID,
		CertificateSerial: setting.WechatPayOfficialCertificateSerial,
		APIv3Key:          setting.WechatPayOfficialAPIv3Key,
		PrivateKey:        setting.WechatPayOfficialPrivateKey,
		PlatformPublicKey: setting.WechatPayOfficialPlatformPublicKey,
		HTTPClient:        wechatPayOfficialQueryHTTPClient,
	}
	transaction, err := client.QueryTransactionByOutTradeNo(c.Request.Context(), tradeNo)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付官方 查询订单失败 user_id=%d trade_no=%s error=%q", topUp.UserId, tradeNo, err.Error()))
		common.ApiSuccess(c, gin.H{
			"trade_no": tradeNo,
			"status":   topUp.Status,
		})
		return
	}
	if transaction.OutTradeNo != tradeNo {
		common.ApiErrorMsg(c, "微信支付返回订单号不一致")
		return
	}
	if err := validateWechatPayOfficialTransactionContext(*transaction); err != nil {
		common.ApiError(c, err)
		return
	}
	if transaction.TradeState == "SUCCESS" {
		if err := reconcileWechatPayOfficialTransaction(c.Request.Context(), *transaction, c.ClientIP()); err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, gin.H{
			"trade_no":     tradeNo,
			"status":       common.TopUpStatusSuccess,
			"wechat_state": transaction.TradeState,
		})
		return
	}
	common.ApiSuccess(c, gin.H{
		"trade_no":     tradeNo,
		"status":       topUp.Status,
		"wechat_state": transaction.TradeState,
	})
}

func GetOfficialPaymentRefundPreview(c *gin.Context) {
	var req OfficialRefundPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	tradeNo := strings.TrimSpace(req.TradeNo)
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		common.ApiError(c, model.ErrTopUpNotFound)
		return
	}
	if topUp.UserId != c.GetInt("id") && c.GetInt("role") < common.RoleAdminUser {
		common.ApiError(c, model.ErrPaymentMethodMismatch)
		return
	}
	preview, err := model.CalculateOfficialPaymentRefundPreview(tradeNo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if pending := model.GetPendingTopUpRefundRequestByTradeNo(tradeNo); pending != nil {
		preview.ExistingPendingRequest = pending.Id
	}
	common.ApiSuccess(c, preview)
}

func ApplyOfficialPaymentRefund(c *gin.Context) {
	var req OfficialRefundApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		common.ApiErrorMsg(c, "请填写退款原因")
		return
	}
	request, preview, err := model.CreateTopUpRefundRequest(c.GetInt("id"), req.TradeNo, req.RefundAmount, req.Reason, req.RefundQRCode)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RecordLog(
		request.UserId,
		model.LogTypeRefund,
		fmt.Sprintf("用户提交退款申请，订单号：%s，申请金额：%.2f，原因：%s", request.TradeNo, request.RequestedAmount, strings.TrimSpace(req.Reason)),
	)
	common.ApiSuccess(c, gin.H{
		"request": request,
		"preview": preview,
	})
}

func AdminRefundAlipayOfficialTopUp(c *gin.Context) {
	if !isAlipayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "支付宝官方支付未启用或配置不完整")
		return
	}
	var req AdminRefundTopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	tradeNo := strings.TrimSpace(req.TradeNo)
	refundAmount := decimal.NewFromFloat(req.RefundAmount).Round(2)
	if !refundAmount.IsPositive() {
		common.ApiErrorMsg(c, "退款金额必须大于 0")
		return
	}
	result, err := executeOfficialPaymentRefund(c.Request.Context(), tradeNo, refundAmount.InexactFloat64(), req.Reason, c.ClientIP(), model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial, req.FullRefund)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func AdminRefundWechatPayOfficialTopUp(c *gin.Context) {
	if !isWechatPayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "微信支付官方支付未启用或配置不完整")
		return
	}
	var req AdminRefundTopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	result, err := executeOfficialPaymentRefund(c.Request.Context(), strings.TrimSpace(req.TradeNo), req.RefundAmount, req.Reason, c.ClientIP(), model.PaymentProviderWechatPayOfficial, model.PaymentMethodWechatPayOfficial, req.FullRefund)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func executeOfficialPaymentRefund(ctx context.Context, tradeNo string, refundAmountInput float64, reason string, callerIP string, paymentProvider string, paymentMethod string, allowFullRefund bool) (gin.H, error) {
	tradeNo = strings.TrimSpace(tradeNo)
	refundAmount := decimal.NewFromFloat(refundAmountInput).Round(2)
	if tradeNo == "" {
		return nil, errors.New("未提供订单号")
	}
	if !refundAmount.IsPositive() {
		return nil, errors.New("退款金额必须大于 0")
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	refund, err := model.CreateOfficialPaymentRefund(model.OfficialPaymentRefundCreateParams{
		TradeNo:         tradeNo,
		PaymentProvider: paymentProvider,
		PaymentMethod:   paymentMethod,
		RefundAmount:    refundAmount.InexactFloat64(),
		Reason:          reason,
		OutRequestNo:    buildOfficialRefundRequestNo(tradeNo),
		AllowFullRefund: allowFullRefund,
	})
	if err != nil {
		return nil, err
	}

	isSubscriptionRefund := model.GetSubscriptionOrderByTradeNo(tradeNo) != nil
	topUp := model.GetTopUpByTradeNo(tradeNo)
	var providerTradeNo string
	var rawResponse string
	switch paymentProvider {
	case model.PaymentProviderAlipayOfficial:
		response, err := requestAlipayOfficialRefund(ctx, refund, reason)
		if err != nil {
			rollbackErr := model.MarkTopUpRefundFailed(refund.OutRequestNo, err.Error())
			if rollbackErr != nil {
				logger.LogError(ctx, fmt.Sprintf("支付宝官方退款失败后回滚本地退款失败 trade_no=%s out_request_no=%s error=%q rollback_error=%q", refund.TradeNo, refund.OutRequestNo, err.Error(), rollbackErr.Error()))
			}
			return nil, err
		}
		providerTradeNo = response.TradeNo
		rawResponse = common.GetJsonString(response)
		if err := model.MarkTopUpRefundSuccess(refund.OutRequestNo, providerTradeNo, rawResponse); err != nil {
			return nil, err
		}
		if isSubscriptionRefund {
			if err := syncOfficialSubscriptionRefundAfterSuccess(tradeNo, paymentProvider, refund.RefundAmount, allowFullRefund); err != nil {
				return nil, err
			}
		}
	case model.PaymentProviderWechatPayOfficial:
		response, err := requestWechatPayOfficialRefund(ctx, refund, topUp, reason)
		if err != nil {
			rollbackErr := model.MarkTopUpRefundFailed(refund.OutRequestNo, err.Error())
			if rollbackErr != nil {
				logger.LogError(ctx, fmt.Sprintf("微信支付官方退款失败后回滚本地退款失败 trade_no=%s out_request_no=%s error=%q rollback_error=%q", refund.TradeNo, refund.OutRequestNo, err.Error(), rollbackErr.Error()))
			}
			return nil, err
		}
		providerTradeNo = response.RefundID
		rawResponse = common.GetJsonString(response)
		if response.EffectiveStatus() == "SUCCESS" {
			if err := model.MarkTopUpRefundSuccess(refund.OutRequestNo, providerTradeNo, rawResponse); err != nil {
				return nil, err
			}
			if isSubscriptionRefund {
				if err := syncOfficialSubscriptionRefundAfterSuccess(tradeNo, paymentProvider, refund.RefundAmount, allowFullRefund); err != nil {
					return nil, err
				}
			}
		}
	default:
		rollbackErr := model.MarkTopUpRefundFailed(refund.OutRequestNo, "unsupported payment provider")
		if rollbackErr != nil {
			logger.LogError(ctx, fmt.Sprintf("官方退款不支持的支付提供方回滚失败 trade_no=%s out_request_no=%s provider=%s rollback_error=%q", refund.TradeNo, refund.OutRequestNo, paymentProvider, rollbackErr.Error()))
		}
		return nil, errors.New("不支持的官方支付方式")
	}
	model.RecordOfficialPaymentRefundLog(
		refund.UserId,
		fmt.Sprintf("管理员发起官方退款成功，订单号：%s，退款金额：%.2f，退回额度：%s", refund.TradeNo, refund.RefundAmount, logger.FormatQuota(int(refund.RefundQuota))),
		callerIP,
		refund.PaymentMethod,
		refund.PaymentProvider,
	)
	return gin.H{
		"out_request_no": refund.OutRequestNo,
		"refund_amount":  refund.RefundAmount,
		"refund_quota":   refund.RefundQuota,
		"provider":       refund.PaymentProvider,
	}, nil
}

func syncOfficialSubscriptionRefundAfterSuccess(tradeNo string, paymentProvider string, refundAmount float64, allowFullRefund bool) error {
	if model.GetSubscriptionOrderByTradeNo(tradeNo) == nil {
		return nil
	}
	reloadedTopUp := model.GetTopUpByTradeNo(tradeNo)
	fullRefund := allowFullRefund ||
		(reloadedTopUp != nil && decimal.NewFromFloat(reloadedTopUp.RefundedMoney).Round(2).GreaterThanOrEqual(decimal.NewFromFloat(reloadedTopUp.Money).Round(2)))
	return model.RefundSubscriptionOrder(tradeNo, paymentProvider, refundAmount, fullRefund)
}

func syncOfficialSubscriptionRefundAfterFailure(tradeNo string, paymentProvider string) error {
	if model.GetSubscriptionOrderByTradeNo(tradeNo) == nil {
		return nil
	}
	return model.SyncSubscriptionOrderRefundState(tradeNo, paymentProvider, false)
}

func requestAlipayOfficialRefund(ctx context.Context, refund *model.TopUpRefund, reason string) (*service.AlipayOfficialOpenAPIResponse, error) {
	client := newAlipayOfficialClient()
	response, err := client.Refund(ctx, map[string]any{
		"out_trade_no":   refund.TradeNo,
		"refund_amount":  decimal.NewFromFloat(refund.RefundAmount).StringFixed(2),
		"refund_reason":  normalizeAlipayRefundReason(reason),
		"out_request_no": refund.OutRequestNo,
		"query_options":  []string{"deposit_back_info"},
	})
	if err != nil {
		return nil, err
	}
	if err := validateAlipayOfficialRefundResponse(response, refund); err != nil {
		return nil, err
	}
	if response.FundChange == "Y" {
		return response, nil
	}
	queryResponse, queryErr := queryAlipayOfficialRefund(ctx, client, refund.TradeNo, refund.OutRequestNo)
	queryRefundConfirmed := queryResponse != nil && queryResponse.RefundStatus == "REFUND_SUCCESS"
	var validateErr error
	if queryErr == nil && queryRefundConfirmed {
		validateErr = validateAlipayOfficialRefundResponse(queryResponse, refund)
	}
	if queryErr != nil || !queryRefundConfirmed || validateErr != nil {
		if queryErr != nil {
			return nil, queryErr
		}
		if validateErr != nil {
			return nil, validateErr
		}
		return nil, fmt.Errorf("支付宝退款未确认 fund_change=%s", response.FundChange)
	}
	if strings.TrimSpace(response.TradeNo) == "" {
		response.TradeNo = queryResponse.TradeNo
	}
	return queryResponse, nil
}

func requestWechatPayOfficialRefund(ctx context.Context, refund *model.TopUpRefund, topUp *model.TopUp, reason string) (*service.WechatPayOfficialRefundResponse, error) {
	if topUp == nil {
		return nil, model.ErrTopUpNotFound
	}
	client := newWechatPayOfficialClient()
	callbackAddr := service.GetCallbackAddress()
	notifyURL := callbackAddr + "/api/wechat-pay/official/notify"
	if strings.TrimSpace(setting.WechatPayOfficialNotifyURL) != "" {
		notifyURL = strings.TrimSpace(setting.WechatPayOfficialNotifyURL)
	}
	response, err := client.Refund(ctx, service.WechatPayOfficialRefundParams{
		OutTradeNo:  refund.TradeNo,
		OutRefundNo: refund.OutRequestNo,
		Reason:      normalizeWechatPayOfficialRefundReason(reason),
		NotifyURL:   notifyURL,
		RefundFen:   yuanToFen(refund.RefundAmount),
		TotalFen:    yuanToFen(topUp.PaidMoney()),
	})
	if err != nil {
		return nil, err
	}
	status := response.EffectiveStatus()
	if status == "SUCCESS" || status == "PROCESSING" {
		return response, nil
	}
	queryResponse, queryErr := client.QueryRefundByOutRefundNo(ctx, refund.OutRequestNo)
	if queryErr != nil {
		return nil, queryErr
	}
	if queryResponse.EffectiveStatus() != "SUCCESS" && queryResponse.EffectiveStatus() != "PROCESSING" {
		return nil, fmt.Errorf("微信退款未确认 status=%s", queryResponse.EffectiveStatus())
	}
	return queryResponse, nil
}

func AdminApproveOfficialPaymentRefundRequest(c *gin.Context) {
	var req AdminRefundRequestActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RequestId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	refundRequest := model.GetTopUpRefundRequestById(req.RequestId)
	if refundRequest == nil {
		common.ApiErrorMsg(c, "退款申请不存在")
		return
	}
	if refundRequest.Status != model.TopUpRefundRequestStatusPending {
		common.ApiError(c, model.ErrTopUpStatusInvalid)
		return
	}
	amount := req.RefundAmount
	if amount <= 0 {
		amount = refundRequest.RequestedAmount
	}
	var result gin.H
	var err error
	if refundRequest.PaymentProvider == model.PaymentProviderSelfServe {
		result, err = executeSelfServeManualRefund(c.Request.Context(), refundRequest.TradeNo, amount, firstNonEmptyString(req.Reason, refundRequest.Reason), c.ClientIP(), req.FullRefund)
	} else {
		result, err = executeOfficialPaymentRefund(c.Request.Context(), refundRequest.TradeNo, amount, firstNonEmptyString(req.Reason, refundRequest.Reason), c.ClientIP(), refundRequest.PaymentProvider, refundRequest.PaymentMethod, req.FullRefund)
	}
	if err != nil {
		_ = model.MarkTopUpRefundRequestFailed(refundRequest.Id, c.GetInt("id"), err.Error())
		common.ApiError(c, err)
		return
	}
	outRequestNo, _ := result["out_request_no"].(string)
	if err := model.MarkTopUpRefundRequestApproved(refundRequest.Id, c.GetInt("id"), outRequestNo, req.Reason); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func executeSelfServeManualRefund(ctx context.Context, tradeNo string, refundAmountInput float64, reason string, callerIP string, allowFullRefund bool) (gin.H, error) {
	tradeNo = strings.TrimSpace(tradeNo)
	refundAmount := decimal.NewFromFloat(refundAmountInput).Round(2)
	if tradeNo == "" {
		return nil, errors.New("未提供订单号")
	}
	if !refundAmount.IsPositive() {
		return nil, errors.New("退款金额必须大于 0")
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	refund, err := model.CreateSelfServeManualRefund(tradeNo, refundAmount.InexactFloat64(), reason, allowFullRefund)
	if err != nil {
		return nil, err
	}
	model.RecordTopupRefundLog(
		refund.UserId,
		fmt.Sprintf("管理员登记自助充值人工退款成功，订单号：%s，退款金额：%.2f，退回额度：%s", refund.TradeNo, refund.RefundAmount, logger.FormatQuota(int(refund.RefundQuota))),
		callerIP,
		refund.PaymentMethod,
		refund.PaymentProvider,
	)
	logger.LogInfo(ctx, fmt.Sprintf("自助充值人工退款登记成功 trade_no=%s out_request_no=%s refund_amount=%.2f refund_quota=%d", refund.TradeNo, refund.OutRequestNo, refund.RefundAmount, refund.RefundQuota))
	return gin.H{
		"out_request_no": refund.OutRequestNo,
		"refund_amount":  refund.RefundAmount,
		"refund_quota":   refund.RefundQuota,
		"provider":       refund.PaymentProvider,
		"manual":         true,
	}, nil
}

func AdminRejectOfficialPaymentRefundRequest(c *gin.Context) {
	var req AdminRejectRefundRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RequestId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.MarkTopUpRefundRequestRejected(req.RequestId, c.GetInt("id"), req.Reason); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminQueryAlipayOfficialTopUp(c *gin.Context) {
	if !isAlipayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "支付宝官方支付未启用或配置不完整")
		return
	}
	var req AdminTopUpTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	tradeNo := strings.TrimSpace(req.TradeNo)
	client := newAlipayOfficialClient()
	response, err := queryAlipayOfficialTrade(c.Request.Context(), client, tradeNo)
	if err != nil {
		if service.IsAlipayOfficialTradeNotFound(err) {
			topUp := model.GetTopUpByTradeNo(tradeNo)
			if topUp == nil {
				common.ApiError(c, model.ErrTopUpNotFound)
				return
			}
			if topUp.PaymentProvider != model.PaymentProviderAlipayOfficial {
				common.ApiError(c, model.ErrPaymentMethodMismatch)
				return
			}
			common.ApiSuccess(c, gin.H{
				"out_trade_no":  tradeNo,
				"trade_status":  "LOCAL_" + strings.ToUpper(topUp.Status),
				"local_status":  topUp.Status,
				"alipay_status": "TRADE_NOT_EXIST",
				"message":       "支付宝侧交易不存在，本地订单仍为待支付或已关闭状态",
			})
			return
		}
		common.ApiError(c, err)
		return
	}
	if response.TradeStatus == "TRADE_SUCCESS" || response.TradeStatus == "TRADE_FINISHED" {
		paidMoney, err := decimal.NewFromString(response.TotalAmount)
		if err != nil {
			common.ApiErrorMsg(c, "支付宝返回金额无效")
			return
		}
		LockOrder(tradeNo)
		defer UnlockOrder(tradeNo)
		payload := map[string]string{
			"out_trade_no":  tradeNo,
			"trade_no":      response.TradeNo,
			"trade_status":  response.TradeStatus,
			"total_amount":  response.TotalAmount,
			"alipay_source": "query",
		}
		if err := reconcileAlipayOfficialPaidOrder(tradeNo, payload, paidMoney, c.ClientIP()); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	common.ApiSuccess(c, response)
}

func AdminQueryWechatPayOfficialTopUp(c *gin.Context) {
	if !isWechatPayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "微信支付官方支付未启用或配置不完整")
		return
	}
	var req AdminTopUpTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	tradeNo := strings.TrimSpace(req.TradeNo)
	client := newWechatPayOfficialClient()
	transaction, err := client.QueryTransactionByOutTradeNo(c.Request.Context(), tradeNo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if transaction.OutTradeNo != tradeNo {
		common.ApiErrorMsg(c, "微信支付返回订单号不一致")
		return
	}
	if err := reconcileWechatPayOfficialTransaction(c.Request.Context(), *transaction, c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, transaction)
}

func AdminCloseAlipayOfficialTopUp(c *gin.Context) {
	if !isAlipayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "支付宝官方支付未启用或配置不完整")
		return
	}
	var req AdminTopUpTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	tradeNo := strings.TrimSpace(req.TradeNo)
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		common.ApiError(c, model.ErrTopUpNotFound)
		return
	}
	if topUp.PaymentProvider != model.PaymentProviderAlipayOfficial {
		common.ApiError(c, model.ErrPaymentMethodMismatch)
		return
	}
	if topUp.Status != common.TopUpStatusPending {
		common.ApiError(c, model.ErrTopUpStatusInvalid)
		return
	}
	client := newAlipayOfficialClient()
	response, err := client.TradeClose(c.Request.Context(), map[string]any{
		"out_trade_no": tradeNo,
		"operator_id":  "admin",
	})
	if !shouldExpireAlipayOfficialOrderAfterClose(err) {
		if service.IsAlipayOfficialTradeNotFound(err) {
			common.ApiErrorMsg(c, "支付宝侧交易不存在，无法确认关闭。请稍后重试或等待订单自动过期后再次关闭。")
			return
		}
		common.ApiError(c, err)
		return
	}
	if err := expireAlipayOfficialPendingOrder(tradeNo); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func AdminCloseWechatPayOfficialTopUp(c *gin.Context) {
	if !isWechatPayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "微信支付官方支付未启用或配置不完整")
		return
	}
	var req AdminTopUpTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	tradeNo := strings.TrimSpace(req.TradeNo)
	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		common.ApiError(c, model.ErrTopUpNotFound)
		return
	}
	if topUp.PaymentProvider != model.PaymentProviderWechatPayOfficial {
		common.ApiError(c, model.ErrPaymentMethodMismatch)
		return
	}
	if topUp.Status != common.TopUpStatusPending {
		common.ApiError(c, model.ErrTopUpStatusInvalid)
		return
	}
	client := newWechatPayOfficialClient()
	if err := client.CloseTransactionByOutTradeNo(c.Request.Context(), tradeNo); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := expireWechatPayOfficialPendingOrder(tradeNo); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"trade_no": tradeNo, "status": common.TopUpStatusExpired})
}

func AdminQueryWechatPayOfficialRefund(c *gin.Context) {
	if !isWechatPayOfficialTopUpEnabled() {
		common.ApiErrorMsg(c, "微信支付官方支付未启用或配置不完整")
		return
	}
	var req struct {
		OutRequestNo string `json:"out_request_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.OutRequestNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	client := newWechatPayOfficialClient()
	response, err := client.QueryRefundByOutRefundNo(c.Request.Context(), strings.TrimSpace(req.OutRequestNo))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	switch response.EffectiveStatus() {
	case "SUCCESS":
		if err := model.MarkTopUpRefundSuccess(response.OutRefundNo, response.RefundID, common.GetJsonString(response)); err != nil {
			common.ApiError(c, err)
			return
		}
		if err := syncOfficialSubscriptionRefundAfterSuccess(response.OutTradeNo, model.PaymentProviderWechatPayOfficial, float64(response.Amount.Refund)/100, false); err != nil {
			common.ApiError(c, err)
			return
		}
	case "REFUNDCLOSE", "ABNORMAL":
		if err := model.MarkTopUpRefundFailed(response.OutRefundNo, common.GetJsonString(response)); err != nil {
			common.ApiError(c, err)
			return
		}
		if err := syncOfficialSubscriptionRefundAfterFailure(response.OutTradeNo, model.PaymentProviderWechatPayOfficial); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	common.ApiSuccess(c, response)
}

// resolveAlipayOfficialOrphanTopUp 处理支付宝侧 TradeClose 已返回"交易不存在"的本地待支付订单：
// 当订单创建时间已超过 alipayOfficialOrphanGraceDuration、且 TradeQuery 二次确认仍为交易不存在时，
// 在本地标记为过期。这条路径专门用于打破"支付宝清掉了订单 → 本地永远 pending → 后台扫描器每分钟反复
// 关单失败刷日志"的死循环。返回 true 表示已完成本地状态更新，调用方可跳过保留 pending 的告警。
func resolveAlipayOfficialOrphanTopUp(ctx context.Context, client *service.AlipayOfficialClient, topUp *model.TopUp) bool {
	if topUp == nil || topUp.CreateTime == 0 {
		return false
	}
	graceCutoff := time.Now().Add(-alipayOfficialOrphanGraceDuration).Unix()
	if topUp.CreateTime > graceCutoff {
		return false
	}
	if _, queryErr := queryAlipayOfficialTrade(ctx, client, topUp.TradeNo); !service.IsAlipayOfficialTradeNotFound(queryErr) {
		return false
	}
	if err := expireAlipayOfficialPendingOrder(topUp.TradeNo); err != nil &&
		!errors.Is(err, model.ErrTopUpStatusInvalid) {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时孤儿订单本地状态更新失败 trade_no=%s create_time=%d error=%q", topUp.TradeNo, topUp.CreateTime, err.Error()))
		return false
	}
	logger.LogInfo(ctx, fmt.Sprintf("支付宝官方超时孤儿订单本地已标记过期 trade_no=%s create_time=%d", topUp.TradeNo, topUp.CreateTime))
	return true
}

func ExpireAlipayOfficialPendingTopUps(ctx context.Context) (int, error) {
	if !isAlipayOfficialTopUpEnabled() {
		return 0, nil
	}
	expireBefore := time.Now().Add(-time.Duration(getAlipayOfficialOrderTimeoutSeconds()) * time.Second).Unix()
	expiredTopUps, err := model.ListPendingTopUpsBefore(model.PaymentProviderAlipayOfficial, expireBefore, 20)
	if err != nil {
		return 0, err
	}
	if len(expiredTopUps) == 0 {
		return 0, nil
	}
	client := newAlipayOfficialClient()
	processed := 0
	for _, topUp := range expiredTopUps {
		if topUp == nil {
			continue
		}
		if ctx.Err() != nil {
			return processed, nil
		}
		LockOrder(topUp.TradeNo)
		currentTopUp := model.GetTopUpByTradeNo(topUp.TradeNo)
		if currentTopUp == nil ||
			currentTopUp.PaymentProvider != model.PaymentProviderAlipayOfficial ||
			currentTopUp.Status != common.TopUpStatusPending {
			UnlockOrder(topUp.TradeNo)
			continue
		}
		response, closeErr := client.TradeClose(ctx, map[string]any{
			"out_trade_no": topUp.TradeNo,
			"operator_id":  "timeout",
		})
		if closeErr != nil {
			if service.IsAlipayOfficialTradeNotFound(closeErr) {
				if !resolveAlipayOfficialOrphanTopUp(ctx, client, topUp) {
					logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单交易不存在，保留本地待支付状态以避免旧支付入口后续付款造成资金悬挂 trade_no=%s", topUp.TradeNo))
				}
				UnlockOrder(topUp.TradeNo)
				processed++
				continue
			}
			reconciled, reconcileErr := reconcileAlipayOfficialTopUpAfterCloseFailure(ctx, client, topUp.TradeNo)
			if reconcileErr != nil {
				logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单关闭失败后查询失败 trade_no=%s close_error=%q query_error=%q", topUp.TradeNo, closeErr.Error(), reconcileErr.Error()))
			} else if !reconciled {
				logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单关闭失败 trade_no=%s error=%q", topUp.TradeNo, closeErr.Error()))
			}
			UnlockOrder(topUp.TradeNo)
			processed++
			continue
		}
		if err := expireAlipayOfficialPendingOrder(topUp.TradeNo); err != nil &&
			!errors.Is(err, model.ErrTopUpStatusInvalid) {
			logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单本地状态更新失败 trade_no=%s response=%q error=%q", topUp.TradeNo, common.GetJsonString(response), err.Error()))
		}
		UnlockOrder(topUp.TradeNo)
		processed++
	}
	return processed, nil
}

func ExpireOfficialPaymentPendingTopUpsByTimeout(ctx context.Context, userId int) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	officialProviders := []struct {
		paymentProvider string
		timeoutSeconds  int
	}{
		{
			paymentProvider: model.PaymentProviderAlipayOfficial,
			timeoutSeconds:  getAlipayOfficialOrderTimeoutSeconds(),
		},
		{
			paymentProvider: model.PaymentProviderWechatPayOfficial,
			timeoutSeconds:  getWechatPayOfficialOrderTimeoutSeconds(),
		},
	}
	total := 0
	for _, provider := range officialProviders {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		timeoutSeconds := provider.timeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = 600
		}
		expireBefore := now.Add(-time.Duration(timeoutSeconds) * time.Second).Unix()
		affected, err := model.ExpireOfficialPaymentPendingTopUpsBefore(ctx, provider.paymentProvider, expireBefore, now.Unix(), userId)
		if err != nil {
			return total, err
		}
		total += int(affected)
	}
	return total, nil
}

func reconcileAlipayOfficialTopUpAfterCloseFailure(ctx context.Context, client *service.AlipayOfficialClient, tradeNo string) (bool, error) {
	response, err := queryAlipayOfficialTrade(ctx, client, tradeNo)
	if err != nil {
		return false, err
	}
	switch response.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		paidMoney, err := decimal.NewFromString(response.TotalAmount)
		if err != nil {
			return false, err
		}
		payload := map[string]string{
			"out_trade_no":  tradeNo,
			"trade_no":      response.TradeNo,
			"trade_status":  response.TradeStatus,
			"total_amount":  response.TotalAmount,
			"alipay_source": "close_reconcile",
		}
		if err := reconcileAlipayOfficialPaidOrder(tradeNo, payload, paidMoney, ""); err != nil {
			return false, err
		}
		return true, nil
	case "TRADE_CLOSED":
		if err := expireAlipayOfficialPendingOrder(tradeNo); err != nil &&
			!errors.Is(err, model.ErrTopUpStatusInvalid) {
			return false, err
		}
		return true, nil
	default:
		return false, nil
	}
}

func StartAlipayOfficialOrderExpireTask() {
	alipayOfficialExpireTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			runAlipayOfficialOrderExpireTaskOnce(context.Background())
			ticker := time.NewTicker(alipayOfficialExpireInterval)
			defer ticker.Stop()
			for range ticker.C {
				runAlipayOfficialOrderExpireTaskOnce(context.Background())
			}
		}()
	})
}

func runAlipayOfficialOrderExpireTaskOnce(ctx context.Context) {
	if !alipayOfficialExpireTaskRunning.TryLock() {
		return
	}
	defer alipayOfficialExpireTaskRunning.Unlock()
	// 给单次清理一个总预算，防止单批连环重试把整个 ticker 间隔拖垮。
	tickCtx, cancel := context.WithTimeout(ctx, alipayOfficialExpireTickBudget)
	defer cancel()
	// 单次 tick 内连续处理多批，每批 20 条，最多 alipayOfficialExpireTickMaxBatches 批；
	// 这样高峰期同分钟内 >20 条订单过期时也能及时跟进，不至于积压到下一分钟。
	for batch := 0; batch < alipayOfficialExpireTickMaxBatches; batch++ {
		if tickCtx.Err() != nil {
			return
		}
		processed, err := ExpireAlipayOfficialPendingTopUps(tickCtx)
		if err != nil {
			logger.LogWarn(tickCtx, fmt.Sprintf("支付宝官方超时订单维护失败 batch=%d error=%q", batch, err.Error()))
			return
		}
		if processed == 0 {
			return
		}
	}
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
	if err := validateAlipayOfficialNotifyContext(params); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝官方支付 webhook 应用上下文不匹配 path=%q client_ip=%s trade_no=%s error=%q", c.Request.RequestURI, c.ClientIP(), params["out_trade_no"], err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}

	if params["msg_method"] == "alipay.trade.refund.depositback.completed" {
		if err := handleAlipayOfficialRefundDepositbackCompleted(c, params); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方退款冲退完成通知处理失败 out_request_no=%s error=%q", params["out_request_no"], err.Error()))
			c.String(http.StatusOK, "fail")
			return
		}
		c.String(http.StatusOK, "success")
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
	if err := reconcileAlipayOfficialPaidOrder(tradeNo, params, paidMoney, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 订单处理失败 trade_no=%s client_ip=%s error=%q", tradeNo, c.ClientIP(), err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}
	if model.GetSubscriptionOrderByTradeNo(tradeNo) != nil {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 订阅成功 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		c.String(http.StatusOK, "success")
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 充值成功 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
	c.String(http.StatusOK, "success")
}

func queryAlipayOfficialTrade(ctx context.Context, client *service.AlipayOfficialClient, tradeNo string) (*service.AlipayOfficialOpenAPIResponse, error) {
	response, err := client.TradeQuery(ctx, map[string]any{
		"out_trade_no": tradeNo,
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.OutTradeNo) != "" && response.OutTradeNo != tradeNo {
		return nil, fmt.Errorf("支付宝返回 out_trade_no 不一致: expected=%s actual=%s", tradeNo, response.OutTradeNo)
	}
	return response, nil
}

func queryAlipayOfficialRefund(ctx context.Context, client *service.AlipayOfficialClient, tradeNo string, outRequestNo string) (*service.AlipayOfficialOpenAPIResponse, error) {
	response, err := client.RefundQuery(ctx, map[string]any{
		"out_trade_no":   tradeNo,
		"out_request_no": outRequestNo,
		"query_options":  []string{"deposit_back_info"},
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.OutTradeNo) != "" && response.OutTradeNo != tradeNo {
		return nil, fmt.Errorf("支付宝退款查询返回 out_trade_no 不一致: expected=%s actual=%s", tradeNo, response.OutTradeNo)
	}
	if strings.TrimSpace(response.OutRequestNo) != "" && response.OutRequestNo != outRequestNo {
		return nil, fmt.Errorf("支付宝退款查询返回 out_request_no 不一致: expected=%s actual=%s", outRequestNo, response.OutRequestNo)
	}
	return response, nil
}

func validateAlipayOfficialNotifyContext(params map[string]string) error {
	if params == nil {
		return fmt.Errorf("missing alipay notify params")
	}
	appID := strings.TrimSpace(params["app_id"])
	if appID != "" && strings.TrimSpace(setting.AlipayOfficialAppID) != "" && appID != setting.AlipayOfficialAppID {
		return fmt.Errorf("支付宝通知 app_id 不一致: expected=%s actual=%s", setting.AlipayOfficialAppID, appID)
	}
	return nil
}

func validateAlipayOfficialRefundResponse(response *service.AlipayOfficialOpenAPIResponse, refund *model.TopUpRefund) error {
	if response == nil || refund == nil {
		return fmt.Errorf("支付宝退款响应为空")
	}
	if strings.TrimSpace(response.OutTradeNo) != "" && response.OutTradeNo != refund.TradeNo {
		return fmt.Errorf("支付宝退款响应 out_trade_no 不一致: expected=%s actual=%s", refund.TradeNo, response.OutTradeNo)
	}
	if strings.TrimSpace(response.OutRequestNo) != "" && response.OutRequestNo != refund.OutRequestNo {
		return fmt.Errorf("支付宝退款响应 out_request_no 不一致: expected=%s actual=%s", refund.OutRequestNo, response.OutRequestNo)
	}
	refundAmountText := strings.TrimSpace(response.RefundFee)
	if refundAmountText == "" {
		refundAmountText = strings.TrimSpace(response.RefundAmount)
	}
	if refundAmountText != "" {
		responseAmount, err := decimal.NewFromString(refundAmountText)
		if err != nil {
			return fmt.Errorf("支付宝退款响应金额解析失败: %w", err)
		}
		expectedAmount := decimal.NewFromFloat(refund.RefundAmount).Round(2)
		if !responseAmount.Round(2).Equal(expectedAmount) {
			return fmt.Errorf("支付宝退款响应金额不一致: expected=%s actual=%s", expectedAmount.StringFixed(2), responseAmount.Round(2).StringFixed(2))
		}
	}
	return nil
}

func reconcileAlipayOfficialPaidOrder(tradeNo string, payload any, paidMoney decimal.Decimal, callerIp string) error {
	if err := completeAlipayOfficialSubscriptionOrderIfPresent(tradeNo, payload, paidMoney); err != nil {
		if !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
			return err
		}
		return model.RechargeOfficialPayment(tradeNo, model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial, callerIp, paidMoney.InexactFloat64())
	}
	return nil
}

func reconcileWechatPayOfficialTransaction(ctx context.Context, transaction service.WechatPayOfficialTransaction, callerIp string) error {
	if transaction.TradeState != "SUCCESS" {
		return nil
	}
	if err := validateWechatPayOfficialTransactionContext(transaction); err != nil {
		return err
	}
	LockOrder(transaction.OutTradeNo)
	defer UnlockOrder(transaction.OutTradeNo)
	if err := completeWechatPayOfficialSubscriptionOrderIfPresent(service.WechatPayOfficialNotifyEnvelope{EventType: "TRANSACTION.SUCCESS"}, transaction); err != nil {
		if !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
			return err
		}
		if err := model.RechargeOfficialPayment(transaction.OutTradeNo, model.PaymentProviderWechatPayOfficial, model.PaymentMethodWechatPayOfficial, callerIp, float64(transaction.Amount.Total)/100); err != nil {
			return err
		}
		logger.LogInfo(ctx, fmt.Sprintf("微信支付官方 查询补齐充值成功 trade_no=%s transaction_id=%s", transaction.OutTradeNo, transaction.TransactionID))
		return nil
	}
	logger.LogInfo(ctx, fmt.Sprintf("微信支付官方 查询补齐订阅成功 trade_no=%s transaction_id=%s", transaction.OutTradeNo, transaction.TransactionID))
	return nil
}

func completeAlipayOfficialSubscriptionOrderIfPresent(tradeNo string, payload any, paidMoney decimal.Decimal) error {
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		return model.ErrSubscriptionOrderNotFound
	}
	if order.PaymentProvider != model.PaymentProviderAlipayOfficial {
		return model.ErrPaymentMethodMismatch
	}
	expectedMoney := decimal.NewFromFloat(order.PaidMoney()).Round(2)
	actualMoney := paidMoney.Round(2)
	if !expectedMoney.Equal(actualMoney) {
		return errors.New("支付金额与订阅订单金额不一致")
	}
	return model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(payload), model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial)
}

func expireAlipayOfficialPendingOrder(tradeNo string) error {
	if model.GetSubscriptionOrderByTradeNo(tradeNo) != nil {
		return model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderAlipayOfficial)
	}
	return model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderAlipayOfficial, common.TopUpStatusExpired)
}

func expireWechatPayOfficialPendingOrder(tradeNo string) error {
	if model.GetSubscriptionOrderByTradeNo(tradeNo) != nil {
		return model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderWechatPayOfficial)
	}
	return model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderWechatPayOfficial, common.TopUpStatusExpired)
}

func handleAlipayOfficialRefundDepositbackCompleted(c *gin.Context, params map[string]string) error {
	bizContent := params["biz_content"]
	if strings.TrimSpace(bizContent) == "" {
		return fmt.Errorf("missing biz_content")
	}
	var payload struct {
		OutTradeNo   string `json:"out_trade_no"`
		TradeNo      string `json:"trade_no"`
		OutRequestNo string `json:"out_request_no"`
		DbackStatus  string `json:"dback_status"`
		DbackAmount  string `json:"dback_amount"`
	}
	if err := common.UnmarshalJsonStr(bizContent, &payload); err != nil {
		return fmt.Errorf("decode refund depositback biz_content: %w", err)
	}
	if payload.DbackStatus == "S" {
		if err := model.MarkTopUpRefundSuccess(payload.OutRequestNo, payload.TradeNo, bizContent); err != nil {
			return err
		}
		refundAmount, _ := decimal.NewFromString(payload.DbackAmount)
		return syncOfficialSubscriptionRefundAfterSuccess(payload.OutTradeNo, model.PaymentProviderAlipayOfficial, refundAmount.InexactFloat64(), false)
	}
	if payload.DbackStatus == "F" {
		if err := model.MarkTopUpRefundFailed(payload.OutRequestNo, bizContent); err != nil {
			return err
		}
		return syncOfficialSubscriptionRefundAfterFailure(payload.OutTradeNo, model.PaymentProviderAlipayOfficial)
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方退款冲退通知忽略未知状态 out_request_no=%s status=%s", payload.OutRequestNo, payload.DbackStatus))
	return nil
}

func handleWechatPayOfficialRefundNotify(c *gin.Context, body []byte) error {
	envelope, refund, err := service.DecodeWechatPayOfficialRefundNotify(body, setting.WechatPayOfficialAPIv3Key)
	if err != nil {
		return err
	}
	if envelope.Resource.OriginalType != "refund" {
		return nil
	}
	if strings.TrimSpace(setting.WechatPayOfficialMchID) != "" && strings.TrimSpace(refund.OutRefundNo) == "" {
		return fmt.Errorf("missing out_refund_no")
	}
	switch refund.EffectiveStatus() {
	case "SUCCESS":
		if err := model.MarkTopUpRefundSuccess(refund.OutRefundNo, refund.RefundID, common.GetJsonString(refund)); err != nil {
			return err
		}
		return syncOfficialSubscriptionRefundAfterSuccess(refund.OutTradeNo, model.PaymentProviderWechatPayOfficial, float64(refund.Amount.Refund)/100, false)
	case "REFUNDCLOSE", "ABNORMAL":
		if err := model.MarkTopUpRefundFailed(refund.OutRefundNo, common.GetJsonString(refund)); err != nil {
			return err
		}
		return syncOfficialSubscriptionRefundAfterFailure(refund.OutTradeNo, model.PaymentProviderWechatPayOfficial)
	default:
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方退款通知忽略状态 out_refund_no=%s status=%s", refund.OutRefundNo, refund.EffectiveStatus()))
		return nil
	}
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
		if envelope.Resource.OriginalType == "refund" {
			if err := handleWechatPayOfficialRefundNotify(c, body); err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方退款通知处理失败 error=%q", err.Error()))
				c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "retry"})
				return
			}
			c.Status(http.StatusNoContent)
			return
		}
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 忽略非成功状态 trade_no=%s event=%s state=%s client_ip=%s", transaction.OutTradeNo, envelope.EventType, transaction.TradeState, c.ClientIP()))
		c.Status(http.StatusNoContent)
		return
	}
	if err := validateWechatPayOfficialTransactionContext(*transaction); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付官方 webhook 商户上下文不匹配 trade_no=%s client_ip=%s error=%q", transaction.OutTradeNo, c.ClientIP(), err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "merchant context mismatch"})
		return
	}

	LockOrder(transaction.OutTradeNo)
	defer UnlockOrder(transaction.OutTradeNo)
	if err := completeWechatPayOfficialSubscriptionOrderIfPresent(*envelope, *transaction); err != nil {
		if !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 订阅处理失败 trade_no=%s transaction_id=%s client_ip=%s error=%q", transaction.OutTradeNo, transaction.TransactionID, c.ClientIP(), err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "retry"})
			return
		}
	} else {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 订阅成功 trade_no=%s transaction_id=%s client_ip=%s", transaction.OutTradeNo, transaction.TransactionID, c.ClientIP()))
		c.Status(http.StatusNoContent)
		return
	}
	if err := model.RechargeOfficialPayment(transaction.OutTradeNo, model.PaymentProviderWechatPayOfficial, model.PaymentMethodWechatPayOfficial, c.ClientIP(), float64(transaction.Amount.Total)/100); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方 充值处理失败 trade_no=%s transaction_id=%s client_ip=%s error=%q", transaction.OutTradeNo, transaction.TransactionID, c.ClientIP(), err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "retry"})
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方 充值成功 trade_no=%s transaction_id=%s client_ip=%s", transaction.OutTradeNo, transaction.TransactionID, c.ClientIP()))
	c.Status(http.StatusNoContent)
}

func completeWechatPayOfficialSubscriptionOrderIfPresent(envelope service.WechatPayOfficialNotifyEnvelope, transaction service.WechatPayOfficialTransaction) error {
	if err := validateWechatPayOfficialTransactionContext(transaction); err != nil {
		return err
	}
	order := model.GetSubscriptionOrderByTradeNo(transaction.OutTradeNo)
	if order == nil {
		return model.ErrSubscriptionOrderNotFound
	}
	if order.PaymentProvider != model.PaymentProviderWechatPayOfficial {
		return model.ErrPaymentMethodMismatch
	}
	expectedFen := yuanToFen(order.PaidMoney())
	if expectedFen != transaction.Amount.Total {
		return errors.New("支付金额与订阅订单金额不一致")
	}
	payload := struct {
		Envelope    service.WechatPayOfficialNotifyEnvelope `json:"envelope"`
		Transaction service.WechatPayOfficialTransaction    `json:"transaction"`
	}{
		Envelope:    envelope,
		Transaction: transaction,
	}
	return model.CompleteSubscriptionOrder(transaction.OutTradeNo, common.GetJsonString(payload), model.PaymentProviderWechatPayOfficial, model.PaymentMethodWechatPayOfficial)
}

func validateWechatPayOfficialTransactionContext(transaction service.WechatPayOfficialTransaction) error {
	if strings.TrimSpace(setting.WechatPayOfficialAppID) != "" && transaction.AppID != setting.WechatPayOfficialAppID {
		return fmt.Errorf("微信支付回调 appid 不一致: expected=%s actual=%s", setting.WechatPayOfficialAppID, transaction.AppID)
	}
	if strings.TrimSpace(setting.WechatPayOfficialMchID) != "" && transaction.MchID != setting.WechatPayOfficialMchID {
		return fmt.Errorf("微信支付回调 mchid 不一致: expected=%s actual=%s", setting.WechatPayOfficialMchID, transaction.MchID)
	}
	return nil
}

func prepayWechatPayOfficialWithNativeFallback(ctx context.Context, client *service.WechatPayOfficialClient, params service.WechatPayOfficialPrepayParams) (*wechatPayOfficialPrepayResponse, error) {
	result, err := client.Prepay(ctx, params)
	if err == nil {
		return &wechatPayOfficialPrepayResponse{
			Result: result,
			Scene:  params.TradeType,
		}, nil
	}
	if params.TradeType != officialPaymentSceneH5 || !service.IsWechatPayOfficialH5Unavailable(err) {
		return nil, err
	}
	nativeParams := params
	nativeParams.TradeType = officialPaymentScenePC
	nativeResult, nativeErr := client.Prepay(ctx, nativeParams)
	if nativeErr != nil {
		return nil, fmt.Errorf("wechat h5 prepay failed and native fallback failed: h5_error=%w; native_error=%v", err, nativeErr)
	}
	return &wechatPayOfficialPrepayResponse{
		Result: nativeResult,
		Scene:  officialPaymentScenePC,
	}, nil
}

func getOfficialPayMoney(amount int64, group string, unitPrice float64, serviceFeePercent float64) float64 {
	return getOfficialPayMoneyBreakdown(amount, group, unitPrice, serviceFeePercent).TotalMoney
}

func getOfficialPayMoneyBreakdown(amount int64, group string, unitPrice float64, serviceFeePercent float64) payMoneyBreakdown {
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

	payMoney := dAmount.
		Mul(decimal.NewFromFloat(unitPrice)).
		Mul(decimal.NewFromFloat(topupGroupRatio)).
		Mul(decimal.NewFromFloat(discount))

	return buildPayMoneyBreakdown(payMoney, serviceFeePercent)
}

func formatOfficialPayMoney(payMoney float64) string {
	return formatPayMoneyToCents(payMoney)
}

func getAlipayOfficialOrderTimeoutSeconds() int {
	if setting.AlipayOfficialOrderTimeoutSec > 0 {
		return setting.AlipayOfficialOrderTimeoutSec
	}
	if setting.AlipayOfficialOrderTimeoutMin > 0 {
		return setting.AlipayOfficialOrderTimeoutMin * 60
	}
	return 600
}

func getWechatPayOfficialOrderTimeoutSeconds() int {
	if setting.WechatPayOfficialOrderTimeoutSec > 0 {
		return setting.WechatPayOfficialOrderTimeoutSec
	}
	return 600
}

func isWechatPayOfficialOrderExpired(createTime int64) bool {
	if createTime <= 0 {
		return false
	}
	return createTime <= time.Now().Add(-time.Duration(getWechatPayOfficialOrderTimeoutSeconds())*time.Second).Unix()
}

func formatAlipayOfficialTimeoutExpress(timeoutSeconds int) string {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 600
	}
	timeoutMinutes := (timeoutSeconds + 59) / 60
	if timeoutMinutes <= 0 {
		timeoutMinutes = 10
	}
	return fmt.Sprintf("%dm", timeoutMinutes)
}

func formatWechatPayOfficialTimeExpire(timeoutSeconds int) string {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 600
	}
	return time.Now().Add(time.Duration(timeoutSeconds) * time.Second).UTC().Format(time.RFC3339)
}

func shouldExpireAlipayOfficialOrderAfterClose(err error) bool {
	return err == nil
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

func buildWechatPayOfficialTradeNo(prefix string, userID int) string {
	base := fmt.Sprintf("%s_%d_", prefix, userID)
	randomLength := 32 - len(base)
	if randomLength < 6 {
		randomLength = 6
	}
	tradeNo := base + randstr.String(randomLength)
	if len(tradeNo) > 32 {
		return tradeNo[:32]
	}
	return tradeNo
}

func buildOfficialRefundRequestNo(tradeNo string) string {
	return fmt.Sprintf("%s_RF_%d_%s", strings.TrimSpace(tradeNo), time.Now().UnixMilli(), randstr.String(4))
}

func normalizeAlipayRefundReason(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "管理员退款"
	}
	return strings.TrimSpace(reason)
}

func normalizeWechatPayOfficialRefundReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "管理员退款"
	}
	runes := []rune(reason)
	if len(runes) <= 80 {
		return reason
	}
	return string(runes[:80])
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func newAlipayOfficialClient() *service.AlipayOfficialClient {
	return &service.AlipayOfficialClient{
		AppID:            setting.AlipayOfficialAppID,
		AppAuthToken:     setting.AlipayOfficialAppAuthToken,
		PrivateKey:       setting.AlipayOfficialPrivateKey,
		AppCertSN:        setting.AlipayOfficialAppCertSN,
		AlipayRootCertSN: setting.AlipayOfficialRootCertSN,
		AlipayCertSN:     setting.AlipayOfficialAlipayCertSN,
		AlipayPublicKey:  setting.AlipayOfficialAlipayPublicKey,
		Sandbox:          setting.AlipayOfficialSandbox,
	}
}

func defaultWechatPayOfficialClient() *service.WechatPayOfficialClient {
	return &service.WechatPayOfficialClient{
		AppID:             setting.WechatPayOfficialAppID,
		MchID:             setting.WechatPayOfficialMchID,
		CertificateSerial: setting.WechatPayOfficialCertificateSerial,
		APIv3Key:          setting.WechatPayOfficialAPIv3Key,
		PrivateKey:        setting.WechatPayOfficialPrivateKey,
		PlatformPublicKey: setting.WechatPayOfficialPlatformPublicKey,
	}
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

func buildOfficialTopUp(userID int, amount int64, money float64, fee float64, tradeNo string, paymentMethod string, paymentProvider string) *model.TopUp {
	return &model.TopUp{
		UserId:          userID,
		Amount:          normalizeOfficialTopUpAmount(amount),
		Money:           money,
		Fee:             fee,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentMethod,
		PaymentProvider: paymentProvider,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
}
