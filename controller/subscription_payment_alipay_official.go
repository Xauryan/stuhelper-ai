package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type SubscriptionAlipayOfficialPayRequest struct {
	PlanId int    `json:"plan_id"`
	Scene  string `json:"scene"`
}

type SubscriptionWechatPayOfficialPayRequest struct {
	PlanId int    `json:"plan_id"`
	Scene  string `json:"scene"`
}

var wechatPayOfficialPrepayHTTPClient *http.Client

func SubscriptionRequestAlipayOfficialPay(c *gin.Context) {
	if !isAlipayOfficialTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝官方支付未启用或配置不完整"})
		return
	}

	var req SubscriptionAlipayOfficialPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	scene := normalizeOfficialPaymentScene(req.Scene)
	tradeNo := buildOfficialTradeNo("ALIPAYSUB", userId)
	payMoney := getAlipayOfficialSubscriptionPayMoney(plan.PriceAmount)
	if payMoney < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodAlipayOfficial,
		PaymentProvider: model.PaymentProviderAlipayOfficial,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方订阅支付 创建订单失败 user_id=%d plan_id=%d trade_no=%s error=%q", userId, plan.Id, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
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
		Subject:          fmt.Sprintf("StuHelper AI 订阅 %s", plan.Title),
		TimeoutExpress:   formatAlipayOfficialTimeoutExpress(setting.AlipayOfficialOrderTimeoutMin),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝官方订阅支付 生成表单失败 user_id=%d plan_id=%d trade_no=%s error=%q", userId, plan.Id, tradeNo, err.Error()))
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderAlipayOfficial)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝官方订阅支付 订单创建成功 user_id=%d plan_id=%d trade_no=%s money=%.2f scene=%s", userId, plan.Id, tradeNo, payMoney, scene))
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

func getAlipayOfficialSubscriptionPayMoney(priceAmount float64) float64 {
	return decimal.NewFromFloat(priceAmount).
		Mul(decimal.NewFromFloat(setting.AlipayOfficialUnitPrice)).
		RoundCeil(2).
		InexactFloat64()
}

func SubscriptionRequestWechatPayOfficialPay(c *gin.Context) {
	if !isWechatPayOfficialTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "微信支付官方支付未启用或配置不完整"})
		return
	}

	var req SubscriptionWechatPayOfficialPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.PriceAmount < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	scene := normalizeOfficialPaymentScene(req.Scene)
	tradeNo := buildWechatPayOfficialTradeNo("WXSUB", userId)
	payMoney := getWechatPayOfficialSubscriptionPayMoney(plan.PriceAmount)
	if payMoney < 0.01 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodWechatPayOfficial,
		PaymentProvider: model.PaymentProviderWechatPayOfficial,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方订阅支付 创建订单失败 user_id=%d plan_id=%d trade_no=%s error=%q", userId, plan.Id, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
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
		HTTPClient:        wechatPayOfficialPrepayHTTPClient,
	}
	result, err := client.Prepay(c.Request.Context(), service.WechatPayOfficialPrepayParams{
		Description: fmt.Sprintf("StuHelper AI 订阅 %s", plan.Title),
		OutTradeNo:  tradeNo,
		NotifyURL:   notifyURL,
		AmountTotal: yuanToFen(payMoney),
		ClientIP:    c.ClientIP(),
		WapURL:      wapURL,
		WapName:     "StuHelper AI",
		TradeType:   scene,
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付官方订阅支付 创建预支付订单失败 user_id=%d plan_id=%d trade_no=%s scene=%s error=%q", userId, plan.Id, tradeNo, scene, err.Error()))
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderWechatPayOfficial)
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付官方订阅支付 订单创建成功 user_id=%d plan_id=%d trade_no=%s money=%.2f scene=%s", userId, plan.Id, tradeNo, payMoney, scene))
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

func getWechatPayOfficialSubscriptionPayMoney(priceAmount float64) float64 {
	return decimal.NewFromFloat(priceAmount).
		Mul(decimal.NewFromFloat(setting.WechatPayOfficialUnitPrice)).
		RoundCeil(2).
		InexactFloat64()
}
