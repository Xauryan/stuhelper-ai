package controller

import (
	"fmt"
	"net/http"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/setting"
	"github.com/gin-gonic/gin"
)

type SubscriptionSelfServePayRequest struct {
	PlanId        int     `json:"plan_id"`
	PaymentMethod string  `json:"payment_method"`
	DeclaredMoney float64 `json:"declared_money"`
	TransactionNo string  `json:"transaction_no"`
}

func SubscriptionRequestSelfServePay(c *gin.Context) {
	if !isSelfServeTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "自助充值未启用"})
		return
	}

	var req SubscriptionSelfServePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	req.PaymentMethod = model.NormalizeSelfServePaymentMethod(req.PaymentMethod)
	if req.PaymentMethod == "" || !isSelfServePaymentMethodEnabled(req.PaymentMethod) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "自助充值支付方式未启用"})
		return
	}

	result, err := model.PurchaseSubscriptionWithSelfServe(model.SelfServeSubscriptionPurchaseParams{
		UserId:        c.GetInt("id"),
		PlanId:        req.PlanId,
		PaymentMethod: req.PaymentMethod,
		DeclaredMoney: req.DeclaredMoney,
		TransactionNo: req.TransactionNo,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	model.RecordTopupLog(
		result.Order.UserId,
		fmt.Sprintf("%s提交订阅购买成功，订单号：%s，申报金额：%.2f 元，等待管理员审核", model.SelfServeTopUpPaymentMethodName(result.Order.PaymentMethod), result.Order.TradeNo, result.Order.Money),
		c.ClientIP(),
		result.Order.PaymentMethod,
		model.PaymentProviderSelfServe,
	)
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("自助订阅购买提交成功 user_id=%d plan_id=%d trade_no=%s payment_method=%s money=%.2f", result.Order.UserId, result.Order.PlanId, result.Order.TradeNo, result.Order.PaymentMethod, result.Order.Money))
	common.ApiSuccess(c, gin.H{
		"trade_no":         result.Order.TradeNo,
		"audit_status":     result.Audit.Status,
		"declared_money":   result.Audit.DeclaredMoney,
		"expected_money":   result.ExpectedMoney,
		"transaction_no":   result.Audit.TransactionNo,
		"subscription_id":  result.Subscription.Id,
		"single_max_money": setting.SelfServeTopUpSingleMaxAmount,
		"daily_max_money":  setting.SelfServeTopUpDailyMaxAmount,
	})
}
