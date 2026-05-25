package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func GetTopUpInfo(c *gin.Context) {
	// 获取支付方式
	payMethods := make([]map[string]string, 0, len(operation_setting.PayMethods)+4)
	if isEpayTopUpEnabled() {
		payMethods = append(payMethods, operation_setting.PayMethods...)
	}

	// 如果启用了 Stripe 支付，添加到支付方法列表
	if isStripeTopUpEnabled() {
		// 检查是否已经包含 Stripe
		hasStripe := false
		for _, method := range payMethods {
			if method["type"] == "stripe" {
				hasStripe = true
				break
			}
		}

		if !hasStripe {
			stripeMethod := map[string]string{
				"name":      "Stripe",
				"type":      "stripe",
				"color":     "rgba(var(--semi-purple-5), 1)",
				"min_topup": strconv.Itoa(setting.StripeMinTopUp),
			}
			payMethods = append(payMethods, stripeMethod)
		}
	}

	// 如果启用了 Waffo 支付，添加到支付方法列表
	enableWaffo := isWaffoTopUpEnabled()
	if enableWaffo {
		hasWaffo := false
		for _, method := range payMethods {
			if method["type"] == model.PaymentMethodWaffo {
				hasWaffo = true
				break
			}
		}

		if !hasWaffo {
			waffoMethod := map[string]string{
				"name":      "Waffo (Global Payment)",
				"type":      model.PaymentMethodWaffo,
				"color":     "rgba(var(--semi-blue-5), 1)",
				"min_topup": strconv.Itoa(setting.WaffoMinTopUp),
			}
			payMethods = append(payMethods, waffoMethod)
		}
	}

	enableWaffoPancake := isWaffoPancakeTopUpEnabled()
	if enableWaffoPancake {
		hasWaffoPancake := false
		for _, method := range payMethods {
			if method["type"] == model.PaymentMethodWaffoPancake {
				hasWaffoPancake = true
				break
			}
		}

		if !hasWaffoPancake {
			payMethods = append(payMethods, map[string]string{
				"name":                "Waffo Pancake",
				"type":                model.PaymentMethodWaffoPancake,
				"color":               "rgba(var(--semi-orange-5), 1)",
				"min_topup":           strconv.Itoa(setting.WaffoPancakeMinTopUp),
				"service_fee_percent": strconv.FormatFloat(setting.WaffoPancakeServiceFeePercent, 'f', -1, 64),
			})
		}
	}

	enableAlipayOfficial := isAlipayOfficialTopUpEnabled()
	if enableAlipayOfficial {
		hasAlipayOfficial := false
		for _, method := range payMethods {
			if method["type"] == model.PaymentMethodAlipayOfficial {
				hasAlipayOfficial = true
				break
			}
		}
		if !hasAlipayOfficial {
			payMethods = append(payMethods, map[string]string{
				"name":                  "支付宝",
				"type":                  model.PaymentMethodAlipayOfficial,
				"color":                 "rgba(var(--semi-blue-5), 1)",
				"min_topup":             strconv.Itoa(setting.AlipayOfficialMinTopUp),
				"unit_price":            strconv.FormatFloat(setting.AlipayOfficialUnitPrice, 'f', -1, 64),
				"service_fee_percent":   strconv.FormatFloat(setting.AlipayOfficialServiceFeePercent, 'f', -1, 64),
				"order_timeout_seconds": strconv.Itoa(getAlipayOfficialOrderTimeoutSeconds()),
			})
		}
	}

	enableWechatPayOfficial := isWechatPayOfficialTopUpEnabled()
	if enableWechatPayOfficial {
		hasWechatPayOfficial := false
		for _, method := range payMethods {
			if method["type"] == model.PaymentMethodWechatPayOfficial {
				hasWechatPayOfficial = true
				break
			}
		}
		if !hasWechatPayOfficial {
			payMethods = append(payMethods, map[string]string{
				"name":                  "微信",
				"type":                  model.PaymentMethodWechatPayOfficial,
				"color":                 "rgba(var(--semi-green-5), 1)",
				"min_topup":             strconv.Itoa(setting.WechatPayOfficialMinTopUp),
				"unit_price":            strconv.FormatFloat(setting.WechatPayOfficialUnitPrice, 'f', -1, 64),
				"service_fee_percent":   strconv.FormatFloat(setting.WechatPayOfficialServiceFeePercent, 'f', -1, 64),
				"order_timeout_seconds": strconv.Itoa(getWechatPayOfficialOrderTimeoutSeconds()),
			})
		}
	}

	data := gin.H{
		"enable_online_topup":              isEpayTopUpEnabled(),
		"enable_stripe_topup":              isStripeTopUpEnabled(),
		"enable_creem_topup":               isCreemTopUpEnabled(),
		"enable_waffo_topup":               enableWaffo,
		"enable_waffo_pancake_topup":       enableWaffoPancake,
		"enable_alipay_official_topup":     enableAlipayOfficial,
		"enable_wechat_pay_official_topup": enableWechatPayOfficial,
		"waffo_pay_methods": func() interface{} {
			if enableWaffo {
				return setting.GetWaffoPayMethods()
			}
			return nil
		}(),
		"creem_products":                    setting.CreemProducts,
		"pay_methods":                       payMethods,
		"min_topup":                         operation_setting.MinTopUp,
		"stripe_min_topup":                  setting.StripeMinTopUp,
		"waffo_min_topup":                   setting.WaffoMinTopUp,
		"waffo_pancake_min_topup":           setting.WaffoPancakeMinTopUp,
		"alipay_official_min_topup":         setting.AlipayOfficialMinTopUp,
		"alipay_official_order_timeout":     getAlipayOfficialOrderTimeoutSeconds(),
		"wechat_pay_official_min_topup":     setting.WechatPayOfficialMinTopUp,
		"wechat_pay_official_order_timeout": getWechatPayOfficialOrderTimeoutSeconds(),
		"amount_options":                    operation_setting.GetPaymentSetting().AmountOptions,
		"discount":                          operation_setting.GetPaymentSetting().AmountDiscount,
		"topup_link":                        common.TopUpLink,
	}
	common.ApiSuccess(c, data)
}

type EpayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

type AmountRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

func GetEpayClient() *epay.Client {
	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		return nil
	}
	withUrl, err := epay.NewClient(&epay.Config{
		PartnerID: operation_setting.EpayId,
		Key:       operation_setting.EpayKey,
	}, operation_setting.PayAddress)
	if err != nil {
		return nil
	}
	return withUrl
}

func getPayMoney(amount int64, group string, paymentMethod string) float64 {
	return getPayMoneyBreakdown(amount, group, paymentMethod).TotalMoney
}

func getPayMoneyBreakdown(amount int64, group string, paymentMethod string) payMoneyBreakdown {
	dAmount := decimal.NewFromInt(amount)
	// 充值金额以“展示类型”为准：
	// - USD/CNY: 前端传 amount 为金额单位；TOKENS: 前端传 tokens，需要换成 USD 金额
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		dAmount = dAmount.Div(dQuotaPerUnit)
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	dTopupGroupRatio := decimal.NewFromFloat(topupGroupRatio)
	dPrice := decimal.NewFromFloat(operation_setting.Price)
	// apply optional preset discount by the original request amount (if configured), default 1.0
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	dDiscount := decimal.NewFromFloat(discount)

	payMoney := dAmount.Mul(dPrice).Mul(dTopupGroupRatio).Mul(dDiscount)

	return buildPayMoneyBreakdown(payMoney, getEpayServiceFeePercent(paymentMethod))
}

func getEpayServiceFeePercent(paymentMethod string) float64 {
	for _, method := range operation_setting.PayMethods {
		if method["type"] == paymentMethod {
			return parseServiceFeePercent(method["service_fee_percent"])
		}
	}
	return 0
}

func getMinTopup() int64 {
	minTopup := operation_setting.MinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dMinTopup := decimal.NewFromInt(int64(minTopup))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		minTopup = int(dMinTopup.Mul(dQuotaPerUnit).IntPart())
	}
	return int64(minTopup)
}

func RequestEpay(c *gin.Context) {
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoneyBreakdown(req.Amount, group, req.PaymentMethod)
	if payMoney.TotalMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付方式不存在"})
		return
	}

	callBackAddress := service.GetCallbackAddress()
	returnUrl, _ := url.Parse(paymentReturnPath("/console/log"))
	notifyUrl, _ := url.Parse(callBackAddress + "/api/user/epay/notify")
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("USR%dNO%s", id, tradeNo)
	client := GetEpayClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}
	uri, params, err := client.Purchase(&epay.PurchaseArgs{
		Type:           req.PaymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("TUC%d", req.Amount),
		Money:          formatPayMoneyToCents(payMoney.TotalMoney),
		Device:         epay.PC,
		NotifyUrl:      notifyUrl,
		ReturnUrl:      returnUrl,
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 拉起支付失败 user_id=%d trade_no=%s payment_method=%s amount=%d error=%q", id, tradeNo, req.PaymentMethod, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(int64(amount))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney.EffectiveMoney,
		Fee:             payMoney.Fee,
		TradeNo:         tradeNo,
		PaymentMethod:   req.PaymentMethod,
		PaymentProvider: model.PaymentProviderEpay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	err = topUp.Insert()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 创建充值订单失败 user_id=%d trade_no=%s payment_method=%s amount=%d error=%q", id, tradeNo, req.PaymentMethod, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 充值订单创建成功 user_id=%d trade_no=%s payment_method=%s amount=%d money=%.2f fee=%.2f total_money=%.2f uri=%q params=%q", id, tradeNo, req.PaymentMethod, req.Amount, payMoney.EffectiveMoney, payMoney.Fee, payMoney.TotalMoney, uri, common.GetJsonString(params)))
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": params, "url": uri})
}

// tradeNo lock
var orderLocks sync.Map
var createLock sync.Mutex

// refCountedMutex 带引用计数的互斥锁，确保最后一个使用者才从 map 中删除
type refCountedMutex struct {
	mu       sync.Mutex
	refCount int
}

// LockOrder 尝试对给定订单号加锁
func LockOrder(tradeNo string) {
	createLock.Lock()
	var rcm *refCountedMutex
	if v, ok := orderLocks.Load(tradeNo); ok {
		rcm = v.(*refCountedMutex)
	} else {
		rcm = &refCountedMutex{}
		orderLocks.Store(tradeNo, rcm)
	}
	rcm.refCount++
	createLock.Unlock()
	rcm.mu.Lock()
}

// UnlockOrder 释放给定订单号的锁
func UnlockOrder(tradeNo string) {
	v, ok := orderLocks.Load(tradeNo)
	if !ok {
		return
	}
	rcm := v.(*refCountedMutex)
	rcm.mu.Unlock()

	createLock.Lock()
	rcm.refCount--
	if rcm.refCount == 0 {
		orderLocks.Delete(tradeNo)
	}
	createLock.Unlock()
}

func EpayNotify(c *gin.Context) {
	if !isEpayWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	var params map[string]string

	if c.Request.Method == "POST" {
		// POST 请求：从 POST body 解析参数
		if err := c.Request.ParseForm(); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 webhook POST 表单解析失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
			_, _ = c.Writer.Write([]byte("fail"))
			return
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		// GET 请求：从 URL Query 解析参数
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 webhook 收到请求 path=%q client_ip=%s method=%s params=%q", c.Request.RequestURI, c.ClientIP(), c.Request.Method, common.GetJsonString(params)))

	if len(params) == 0 {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 参数为空 path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	client := GetEpayClient()
	if client == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 client 未初始化 path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 webhook 响应写入失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		}
		return
	}
	verifyInfo, err := client.Verify(params)
	if err == nil && verifyInfo.VerifyStatus {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 webhook 验签成功 trade_no=%s callback_type=%s trade_status=%s client_ip=%s verify_info=%q", verifyInfo.ServiceTradeNo, verifyInfo.Type, verifyInfo.TradeStatus, c.ClientIP(), common.GetJsonString(verifyInfo)))
		_, err := c.Writer.Write([]byte("success"))
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 webhook 响应写入失败 trade_no=%s client_ip=%s error=%q", verifyInfo.ServiceTradeNo, c.ClientIP(), err.Error()))
		}
	} else {
		_, err := c.Writer.Write([]byte("fail"))
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 webhook 响应写入失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		}
		if err != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 验签失败 path=%q client_ip=%s verify_error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		} else {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 webhook 验签失败 path=%q client_ip=%s verify_status=false", c.Request.RequestURI, c.ClientIP()))
		}
		return
	}

	if verifyInfo.TradeStatus == epay.StatusTradeSuccess {
		LockOrder(verifyInfo.ServiceTradeNo)
		defer UnlockOrder(verifyInfo.ServiceTradeNo)
		topUp, quotaToAdd, referralResult, completed, err := model.CompleteEpayTopUp(verifyInfo.ServiceTradeNo, verifyInfo.Type)
		if errors.Is(err, model.ErrTopUpNotFound) {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 回调订单不存在 trade_no=%s callback_type=%s client_ip=%s verify_info=%q", verifyInfo.ServiceTradeNo, verifyInfo.Type, c.ClientIP(), common.GetJsonString(verifyInfo)))
			return
		}
		if errors.Is(err, model.ErrPaymentMethodMismatch) {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("易支付 订单支付网关不匹配 trade_no=%s order_provider=%s callback_type=%s client_ip=%s", verifyInfo.ServiceTradeNo, topUp.PaymentProvider, verifyInfo.Type, c.ClientIP()))
			return
		}
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("易支付 完成充值订单失败 trade_no=%s client_ip=%s error=%q", verifyInfo.ServiceTradeNo, c.ClientIP(), err.Error()))
			return
		}
		if completed {
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 充值成功 trade_no=%s user_id=%d client_ip=%s quota_to_add=%d money=%.2f topup=%q", topUp.TradeNo, topUp.UserId, c.ClientIP(), quotaToAdd, topUp.Money, common.GetJsonString(topUp)))
			model.RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money), c.ClientIP(), topUp.PaymentMethod, "epay")
			model.RecordReferralCommissionLog(referralResult)
		}
	} else {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("易支付 webhook 忽略事件 trade_no=%s callback_type=%s trade_status=%s client_ip=%s verify_info=%q", verifyInfo.ServiceTradeNo, verifyInfo.Type, verifyInfo.TradeStatus, c.ClientIP(), common.GetJsonString(verifyInfo)))
	}
}

func RequestAmount(c *gin.Context) {
	var req AmountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group, req.PaymentMethod)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": formatPayMoneyToCents(payMoney)})
}

func GetUserTopUps(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	refreshOfficialPaymentTimeoutsForTopUpList(c.Request.Context(), userId)
	result, err := model.GetUserTopUpsResultWithOptions(userId, getTopUpQueryOptions(c), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, topUpQueryResponse(pageInfo, result))
}

// GetAllTopUps 管理员获取全平台充值记录
func GetAllTopUps(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	// 异步触发支付宝官方超时订单清理：
	// 非 master 节点的 StartAlipayOfficialOrderExpireTask 会直接返回，
	// 这里用 TryLock 保护的 goroutine 兜底，避免无 master 部署下本地订单永远停留在 pending；
	// 不阻塞列表请求。
	go runAlipayOfficialOrderExpireTaskOnce(context.Background())
	refreshOfficialPaymentTimeoutsForTopUpList(c.Request.Context(), 0)

	result, err := model.GetAllTopUpsResultWithOptions(getTopUpQueryOptions(c), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, topUpQueryResponse(pageInfo, result))
}

func refreshOfficialPaymentTimeoutsForTopUpList(ctx context.Context, userId int) {
	syncCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := ExpireOfficialPaymentPendingTopUpsByTimeout(syncCtx, userId); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		logger.LogWarn(syncCtx, fmt.Sprintf("官方支付账单超时状态同步失败 error=%q", err.Error()))
	}
}

func getTopUpQueryOptions(c *gin.Context) model.TopUpQueryOptions {
	startTime, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	userId, _ := strconv.Atoi(c.Query("user_id"))
	return model.TopUpQueryOptions{
		Keyword:       c.Query("keyword"),
		UserID:        userId,
		Username:      c.Query("username"),
		PaymentMethod: c.Query("payment_method"),
		TradeNo:       c.Query("trade_no"),
		StartTime:     startTime,
		EndTime:       endTime,
		PendingRefund: c.Query("pending_refund") == "true",
	}
}

func topUpQueryResponse(pageInfo *common.PageInfo, result model.TopUpQueryResult) gin.H {
	pageInfo.SetTotal(int(result.Total))
	pageInfo.SetItems(result.Items)
	return gin.H{
		"page":        pageInfo.Page,
		"page_size":   pageInfo.PageSize,
		"total":       pageInfo.Total,
		"items":       pageInfo.Items,
		"total_money": result.TotalMoney,
	}
}

type AdminCompleteTopupRequest struct {
	TradeNo string `json:"trade_no"`
}

type AdminTopUpTradeRequest struct {
	TradeNo string `json:"trade_no"`
}

type AdminRefundTopUpRequest struct {
	TradeNo      string  `json:"trade_no"`
	RefundAmount float64 `json:"refund_amount"`
	RefundQuota  int64   `json:"refund_quota"`
	Reason       string  `json:"reason"`
	FullRefund   bool    `json:"full_refund"`
}

// AdminCompleteTopUp 管理员补单接口
func AdminCompleteTopUp(c *gin.Context) {
	var req AdminCompleteTopupRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	// 订单级互斥，防止并发补单
	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	topUp := model.GetTopUpByTradeNo(req.TradeNo)
	if model.IsSubscriptionTopUpRecord(topUp) && model.IsOfficialPaymentProvider(topUp.PaymentProvider) {
		if err := model.ManualCompleteOfficialSubscriptionTopUp(req.TradeNo); err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, nil)
		return
	}

	if err := model.ManualCompleteTopUp(req.TradeNo, c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminRefundAdminTopUp records an offline/admin top-up refund and deducts quota.
func AdminRefundAdminTopUp(c *gin.Context) {
	var req AdminRefundTopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TradeNo == "" || req.RefundQuota <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	refund, err := model.RefundAdminBalanceTopUp(req.TradeNo, req.RefundQuota, req.Reason)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	model.RecordTopupRefundLog(
		refund.UserId,
		fmt.Sprintf("管理员退款充值额度 %s，订单号：%s", logger.LogQuota(int(refund.RefundQuota)), refund.TradeNo),
		c.ClientIP(),
		model.PaymentMethodAdminAdd,
		model.PaymentProviderAdmin,
	)
	common.ApiSuccess(c, gin.H{
		"trade_no":     refund.TradeNo,
		"refund_quota": refund.RefundQuota,
	})
}
