package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/setting"

	"github.com/gin-gonic/gin"
)

type SelfServeTopUpPreviewRequest struct {
	DeclaredMoney float64 `json:"declared_money"`
}

type SelfServeTopUpRequest struct {
	PaymentMethod string  `json:"payment_method"`
	DeclaredMoney float64 `json:"declared_money"`
	TransactionNo string  `json:"transaction_no"`
}

type AdminSelfServeTopUpTradeRequest struct {
	TradeNo string `json:"trade_no"`
	Reason  string `json:"reason"`
}

type AdminSelfServeTopUpUpdateRequest struct {
	TradeNo       string  `json:"trade_no"`
	DeclaredMoney float64 `json:"declared_money"`
	TransactionNo string  `json:"transaction_no"`
	Reason        string  `json:"reason"`
}

type AdminSelfServeTopUpRejectRequest struct {
	TradeNo string `json:"trade_no"`
	Reason  string `json:"reason"`
	BanUser bool   `json:"ban_user"`
}

func isSelfServePaymentMethodEnabled(paymentMethod string) bool {
	switch model.NormalizeSelfServePaymentMethod(paymentMethod) {
	case model.PaymentMethodAlipaySelfServe:
		return isSelfServeAlipayTopUpEnabled()
	case model.PaymentMethodWechatSelfServe:
		return isSelfServeWechatPayTopUpEnabled()
	default:
		return false
	}
}

func RequestSelfServeTopUpPreview(c *gin.Context) {
	if !isSelfServeTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "自助充值未启用"})
		return
	}

	var req SelfServeTopUpPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	preview, err := model.PreviewSelfServeTopUp(c.GetInt("id"), req.DeclaredMoney)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": preview})
}

func RequestSelfServeTopUp(c *gin.Context) {
	var req SelfServeTopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	req.PaymentMethod = model.NormalizeSelfServePaymentMethod(req.PaymentMethod)
	if req.PaymentMethod == "" || !isSelfServePaymentMethodEnabled(req.PaymentMethod) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "自助充值支付方式未启用"})
		return
	}
	if model.NormalizeSelfServePaymentMethod(req.PaymentMethod) == model.PaymentMethodWechatSelfServe &&
		!setting.SelfServeWechatPayModeRequiresTransactionNo(setting.SelfServeWechatPayMode) {
		req.TransactionNo = ""
	}

	result, err := model.CreateSelfServeTopUp(model.SelfServeTopUpCreateParams{
		UserId:        c.GetInt("id"),
		PaymentMethod: req.PaymentMethod,
		DeclaredMoney: req.DeclaredMoney,
		TransactionNo: req.TransactionNo,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}

	model.RecordTopupLog(
		result.TopUp.UserId,
		fmt.Sprintf("%s提交成功，充值额度: %v，申报金额: %.2f 元，等待管理员审核", model.SelfServeTopUpPaymentMethodName(result.TopUp.PaymentMethod), logger.FormatQuota(int(result.QuotaDelta)), result.TopUp.Money),
		c.ClientIP(),
		result.TopUp.PaymentMethod,
		model.PaymentProviderSelfServe,
	)
	common.ApiSuccess(c, gin.H{
		"trade_no":         result.TopUp.TradeNo,
		"audit_status":     result.Audit.Status,
		"credited_quota":   result.Audit.CreditedQuota,
		"declared_money":   result.Audit.DeclaredMoney,
		"transaction_no":   result.Audit.TransactionNo,
		"single_max_money": setting.SelfServeTopUpSingleMaxAmount,
		"daily_max_money":  setting.SelfServeTopUpDailyMaxAmount,
	})
}

func AdminApproveSelfServeTopUp(c *gin.Context) {
	var req AdminSelfServeTopUpTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	result, err := model.ApproveSelfServeTopUp(req.TradeNo, c.GetInt("id"), req.Reason)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RecordLogWithAdminInfo(result.TopUp.UserId, model.LogTypeManage, fmt.Sprintf("管理员通过自助充值审核，订单号：%s", result.TopUp.TradeNo), map[string]interface{}{
		"admin_id":       c.GetInt("id"),
		"admin_username": c.GetString("username"),
		"trade_no":       result.TopUp.TradeNo,
		"transaction_no": result.Audit.TransactionNo,
		"payment_method": result.TopUp.PaymentMethod,
		"reason":         strings.TrimSpace(req.Reason),
	})
	model.RecordReferralCommissionLog(result.ReferralResult)
	common.ApiSuccess(c, result)
}

func AdminUpdateSelfServeTopUp(c *gin.Context) {
	var req AdminSelfServeTopUpUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	result, err := model.UpdateSelfServeTopUp(model.SelfServeTopUpEditParams{
		TradeNo:       req.TradeNo,
		DeclaredMoney: req.DeclaredMoney,
		TransactionNo: req.TransactionNo,
		AdminReason:   req.Reason,
		AuditorId:     c.GetInt("id"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RecordLogWithAdminInfo(result.TopUp.UserId, model.LogTypeManage, fmt.Sprintf("管理员编辑自助充值订单，订单号：%s，额度调整：%s", result.TopUp.TradeNo, logger.FormatQuota(int(result.QuotaDelta))), map[string]interface{}{
		"admin_id":       c.GetInt("id"),
		"admin_username": c.GetString("username"),
		"trade_no":       result.TopUp.TradeNo,
		"transaction_no": result.Audit.TransactionNo,
		"payment_method": result.TopUp.PaymentMethod,
		"declared_money": result.Audit.DeclaredMoney,
		"quota_delta":    result.QuotaDelta,
		"reason":         strings.TrimSpace(req.Reason),
	})
	common.ApiSuccess(c, result)
}

func AdminRejectSelfServeTopUp(c *gin.Context) {
	var req AdminSelfServeTopUpRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	LockOrder(req.TradeNo)
	defer UnlockOrder(req.TradeNo)

	result, err := model.RejectSelfServeTopUp(req.TradeNo, c.GetInt("id"), req.Reason, req.BanUser)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	content := fmt.Sprintf("管理员拒绝自助充值审核并扣回额度，订单号：%s，扣回额度：%s", result.TopUp.TradeNo, logger.FormatQuota(int(-result.QuotaDelta)))
	if result.Banned {
		content += "，用户已封禁"
	}
	model.RecordTopupRefundLog(
		result.TopUp.UserId,
		content,
		c.ClientIP(),
		result.TopUp.PaymentMethod,
		model.PaymentProviderSelfServe,
	)
	model.RecordLogWithAdminInfo(result.TopUp.UserId, model.LogTypeManage, content, map[string]interface{}{
		"admin_id":       c.GetInt("id"),
		"admin_username": c.GetString("username"),
		"trade_no":       result.TopUp.TradeNo,
		"transaction_no": result.Audit.TransactionNo,
		"payment_method": result.TopUp.PaymentMethod,
		"quota_delta":    result.QuotaDelta,
		"ban_user":       result.Banned,
		"reason":         strings.TrimSpace(req.Reason),
	})
	common.ApiSuccess(c, result)
}
