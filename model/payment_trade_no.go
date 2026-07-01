package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
)

func BuildPaymentTradeNo(prefix string, userID int) string {
	return fmt.Sprintf("%s_%d_%d_%s", strings.ToUpper(strings.TrimSpace(prefix)), userID, time.Now().UnixMilli(), common.GetRandomString(6))
}

func BuildWechatPayPaymentTradeNo(prefix string, userID int) string {
	base := fmt.Sprintf("%s_%d_", strings.ToUpper(strings.TrimSpace(prefix)), userID)
	randomLength := 32 - len(base)
	if randomLength < 6 {
		randomLength = 6
	}
	tradeNo := base + common.GetRandomString(randomLength)
	if len(tradeNo) > 32 {
		return tradeNo[:32]
	}
	return tradeNo
}

func BuildSelfServePaymentTradeNo(paymentMethod string, subscription bool, userID int) string {
	switch NormalizeSelfServePaymentMethod(paymentMethod) {
	case PaymentMethodWechatSelfServe:
		prefix := "WX_SS"
		if subscription {
			prefix = "WXSUB_SS"
		}
		return BuildWechatPayPaymentTradeNo(prefix, userID)
	case PaymentMethodAlipaySelfServe:
		prefix := "ALIPAY_SS"
		if subscription {
			prefix = "ALIPAYSUB_SS"
		}
		return BuildPaymentTradeNo(prefix, userID)
	default:
		prefix := "SS"
		if subscription {
			prefix = "SUB_SS"
		}
		return BuildPaymentTradeNo(prefix, userID)
	}
}

func BuildSelfServeAuditTransactionNo(paymentMethod string, subscription bool, userID int) string {
	switch NormalizeSelfServePaymentMethod(paymentMethod) {
	case PaymentMethodWechatSelfServe:
		// 微信自助在“企业微信红包”模式下仍需要内部唯一流水号，供账单审核和唯一索引使用。
		prefix := "WX_SS_TX"
		if subscription {
			prefix = "WXSUB_SS_TX"
		}
		return fmt.Sprintf("%s_%d_%s", prefix, userID, common.GetRandomString(12))
	case PaymentMethodAlipaySelfServe:
		prefix := "ALIPAY_SS_TX"
		if subscription {
			prefix = "ALIPAYSUB_SS_TX"
		}
		return fmt.Sprintf("%s_%d_%d_%s", prefix, userID, time.Now().UnixMilli(), common.GetRandomString(8))
	default:
		prefix := "SS_TX"
		if subscription {
			prefix = "SUB_SS_TX"
		}
		return fmt.Sprintf("%s_%d_%d_%s", prefix, userID, time.Now().UnixMilli(), common.GetRandomString(8))
	}
}

func BuildBalancePaymentTradeNo(userID int) string {
	return fmt.Sprintf("BALANCE__%d_%s", userID, common.GetRandomString(20))
}
