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
