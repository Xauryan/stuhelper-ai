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
)

var (
	alipayOfficialExpireTaskOnce    sync.Once
	alipayOfficialExpireTaskRunning sync.Mutex
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
		TotalAmount:      formatOfficialPayMoney(payMoney),
		Subject:          fmt.Sprintf("StuHelper AI 充值 %d", req.Amount),
		TimeoutExpress:   formatAlipayOfficialTimeoutExpress(setting.AlipayOfficialOrderTimeoutMin),
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

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	refund, err := model.CreateOfficialPaymentRefund(model.OfficialPaymentRefundCreateParams{
		TradeNo:         tradeNo,
		PaymentProvider: model.PaymentProviderAlipayOfficial,
		PaymentMethod:   model.PaymentMethodAlipayOfficial,
		RefundAmount:    refundAmount.InexactFloat64(),
		Reason:          req.Reason,
		OutRequestNo:    buildOfficialRefundRequestNo(tradeNo),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	client := newAlipayOfficialClient()
	response, err := client.Refund(c.Request.Context(), map[string]any{
		"out_trade_no":   refund.TradeNo,
		"refund_amount":  decimal.NewFromFloat(refund.RefundAmount).StringFixed(2),
		"refund_reason":  normalizeAlipayRefundReason(req.Reason),
		"out_request_no": refund.OutRequestNo,
		"query_options":  []string{"deposit_back_info"},
	})
	rawResponse := common.GetJsonString(response)
	if err != nil {
		rollbackErr := model.MarkTopUpRefundFailed(refund.OutRequestNo, err.Error())
		if rollbackErr != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方退款失败后回滚本地退款失败 trade_no=%s out_request_no=%s error=%q rollback_error=%q", refund.TradeNo, refund.OutRequestNo, err.Error(), rollbackErr.Error()))
		}
		common.ApiError(c, err)
		return
	}
	if response.FundChange != "Y" {
		queryResponse, queryErr := client.RefundQuery(c.Request.Context(), map[string]any{
			"out_trade_no":   refund.TradeNo,
			"out_request_no": refund.OutRequestNo,
			"query_options":  []string{"deposit_back_info"},
		})
		queryRefundConfirmed := queryResponse != nil && queryResponse.RefundStatus == "REFUND_SUCCESS"
		if queryErr != nil || !queryRefundConfirmed {
			reason := fmt.Sprintf("支付宝退款未确认 fund_change=%s", response.FundChange)
			if queryErr != nil {
				reason = queryErr.Error()
			}
			rollbackErr := model.MarkTopUpRefundFailed(refund.OutRequestNo, reason)
			if rollbackErr != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方退款未确认后回滚本地退款失败 trade_no=%s out_request_no=%s reason=%q rollback_error=%q", refund.TradeNo, refund.OutRequestNo, reason, rollbackErr.Error()))
			}
			common.ApiErrorMsg(c, "支付宝退款结果未确认，请稍后重试或查询退款状态")
			return
		}
		rawResponse = common.GetJsonString(queryResponse)
		if strings.TrimSpace(response.TradeNo) == "" {
			response.TradeNo = queryResponse.TradeNo
		}
	}
	if err := model.MarkTopUpRefundSuccess(refund.OutRequestNo, response.TradeNo, rawResponse); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RecordOfficialPaymentRefundLog(
		refund.UserId,
		fmt.Sprintf("管理员发起支付宝官方退款成功，订单号：%s，退款金额：%.2f，退回额度：%s", refund.TradeNo, refund.RefundAmount, logger.FormatQuota(int(refund.RefundQuota))),
		c.ClientIP(),
		refund.PaymentMethod,
		refund.PaymentProvider,
	)
	common.ApiSuccess(c, gin.H{"out_request_no": refund.OutRequestNo})
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
	response, err := client.TradeQuery(c.Request.Context(), map[string]any{
		"out_trade_no": tradeNo,
	})
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
		if err := model.RechargeOfficialPayment(tradeNo, model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial, c.ClientIP(), paidMoney.InexactFloat64()); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	common.ApiSuccess(c, response)
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
	if err := model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderAlipayOfficial, common.TopUpStatusExpired); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func ExpireAlipayOfficialPendingTopUps(ctx context.Context) error {
	if !isAlipayOfficialTopUpEnabled() {
		return nil
	}
	timeoutMinutes := setting.AlipayOfficialOrderTimeoutMin
	if timeoutMinutes <= 0 {
		timeoutMinutes = 10
	}
	expireBefore := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute).Unix()
	expiredTopUps, err := model.ListPendingTopUpsBefore(model.PaymentProviderAlipayOfficial, expireBefore, 20)
	if err != nil {
		return err
	}
	if len(expiredTopUps) == 0 {
		return nil
	}
	client := newAlipayOfficialClient()
	for _, topUp := range expiredTopUps {
		if topUp == nil {
			continue
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
				logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单交易不存在，保留本地待支付状态以避免旧支付入口后续付款造成资金悬挂 trade_no=%s", topUp.TradeNo))
				UnlockOrder(topUp.TradeNo)
				continue
			}
			reconciled, reconcileErr := reconcileAlipayOfficialTopUpAfterCloseFailure(ctx, client, topUp.TradeNo)
			if reconcileErr != nil {
				logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单关闭失败后查询失败 trade_no=%s close_error=%q query_error=%q", topUp.TradeNo, closeErr.Error(), reconcileErr.Error()))
			} else if !reconciled {
				logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单关闭失败 trade_no=%s error=%q", topUp.TradeNo, closeErr.Error()))
			}
			UnlockOrder(topUp.TradeNo)
			continue
		}
		if err := model.UpdatePendingTopUpStatus(topUp.TradeNo, model.PaymentProviderAlipayOfficial, common.TopUpStatusExpired); err != nil &&
			!errors.Is(err, model.ErrTopUpStatusInvalid) {
			logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单本地状态更新失败 trade_no=%s response=%q error=%q", topUp.TradeNo, common.GetJsonString(response), err.Error()))
		}
		UnlockOrder(topUp.TradeNo)
	}
	return nil
}

func reconcileAlipayOfficialTopUpAfterCloseFailure(ctx context.Context, client *service.AlipayOfficialClient, tradeNo string) (bool, error) {
	response, err := client.TradeQuery(ctx, map[string]any{
		"out_trade_no": tradeNo,
	})
	if err != nil {
		return false, err
	}
	switch response.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		paidMoney, err := decimal.NewFromString(response.TotalAmount)
		if err != nil {
			return false, err
		}
		if err := model.RechargeOfficialPayment(tradeNo, model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial, "", paidMoney.InexactFloat64()); err != nil {
			return false, err
		}
		return true, nil
	case "TRADE_CLOSED":
		if err := model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderAlipayOfficial, common.TopUpStatusExpired); err != nil &&
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
	if err := ExpireAlipayOfficialPendingTopUps(ctx); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝官方超时订单维护失败 error=%q", err.Error()))
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
	if err := completeAlipayOfficialSubscriptionOrderIfPresent(tradeNo, params, paidMoney); err != nil {
		if !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 订阅处理失败 trade_no=%s client_ip=%s error=%q", tradeNo, c.ClientIP(), err.Error()))
			c.String(http.StatusOK, "fail")
			return
		}
	} else {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 订阅成功 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
		c.String(http.StatusOK, "success")
		return
	}
	if err := model.RechargeOfficialPayment(tradeNo, model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial, c.ClientIP(), paidMoney.InexactFloat64()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方支付 充值处理失败 trade_no=%s client_ip=%s error=%q", tradeNo, c.ClientIP(), err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方支付 充值成功 trade_no=%s client_ip=%s", tradeNo, c.ClientIP()))
	c.String(http.StatusOK, "success")
}

func completeAlipayOfficialSubscriptionOrderIfPresent(tradeNo string, params map[string]string, paidMoney decimal.Decimal) error {
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		return model.ErrSubscriptionOrderNotFound
	}
	if order.PaymentProvider != model.PaymentProviderAlipayOfficial {
		return model.ErrPaymentMethodMismatch
	}
	expectedMoney := decimal.NewFromFloat(order.Money).Round(2)
	actualMoney := paidMoney.Round(2)
	if !expectedMoney.Equal(actualMoney) {
		return errors.New("支付金额与订阅订单金额不一致")
	}
	return model.CompleteSubscriptionOrder(tradeNo, common.GetJsonString(params), model.PaymentProviderAlipayOfficial, model.PaymentMethodAlipayOfficial)
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
		return model.MarkTopUpRefundSuccess(payload.OutRequestNo, payload.TradeNo, bizContent)
	}
	if payload.DbackStatus == "F" {
		return model.MarkTopUpRefundFailed(payload.OutRequestNo, bizContent)
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方退款冲退通知忽略未知状态 out_request_no=%s status=%s", payload.OutRequestNo, payload.DbackStatus))
	return nil
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

func formatAlipayOfficialTimeoutExpress(timeoutMinutes int) string {
	if timeoutMinutes <= 0 {
		timeoutMinutes = 10
	}
	return fmt.Sprintf("%dm", timeoutMinutes)
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

func buildOfficialRefundRequestNo(tradeNo string) string {
	return fmt.Sprintf("%s_RF_%d_%s", strings.TrimSpace(tradeNo), time.Now().UnixMilli(), randstr.String(4))
}

func normalizeAlipayRefundReason(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "管理员退款"
	}
	return strings.TrimSpace(reason)
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
