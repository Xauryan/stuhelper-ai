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

func BuildBalancePaymentTradeNo(userID int) string {
	return fmt.Sprintf("BALANCE__%d_%s", userID, common.GetRandomString(20))
}
