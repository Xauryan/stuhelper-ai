package model

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPaymentTradeNoUsesUnifiedProviderFormat(t *testing.T) {
	patterns := map[string]*regexp.Regexp{
		"ALIPAY":    regexp.MustCompile(`^ALIPAY_42_\d+_[A-Za-z0-9]+$`),
		"ALIPAYSUB": regexp.MustCompile(`^ALIPAYSUB_42_\d+_[A-Za-z0-9]+$`),
	}

	for prefix, pattern := range patterns {
		tradeNo := BuildPaymentTradeNo(prefix, 42)
		assert.True(t, pattern.MatchString(tradeNo), tradeNo)
	}
}

func TestBuildWechatPayPaymentTradeNoKeepsWechatLengthLimit(t *testing.T) {
	tradeNo := BuildWechatPayPaymentTradeNo("wxsub", 42)

	assert.LessOrEqual(t, len(tradeNo), 32)
	assert.Regexp(t, regexp.MustCompile(`^WXSUB_42_[A-Za-z0-9]+$`), tradeNo)
}

func TestBuildSelfServePaymentTradeNoFollowsProviderFormat(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		subscription bool
		pattern      *regexp.Regexp
		maxLength    int
	}{
		{
			name:      "wechat topup",
			method:    PaymentMethodWechatSelfServe,
			pattern:   regexp.MustCompile(`^WX_SS_42_[A-Za-z0-9]+$`),
			maxLength: 32,
		},
		{
			name:      "alipay topup",
			method:    PaymentMethodAlipaySelfServe,
			pattern:   regexp.MustCompile(`^ALIPAY_SS_42_\d+_[A-Za-z0-9]+$`),
			maxLength: 255,
		},
		{
			name:         "wechat subscription",
			method:       PaymentMethodWechatSelfServe,
			subscription: true,
			pattern:      regexp.MustCompile(`^WXSUB_SS_42_[A-Za-z0-9]+$`),
			maxLength:    32,
		},
		{
			name:         "alipay subscription",
			method:       PaymentMethodAlipaySelfServe,
			subscription: true,
			pattern:      regexp.MustCompile(`^ALIPAYSUB_SS_42_\d+_[A-Za-z0-9]+$`),
			maxLength:    255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tradeNo := BuildSelfServePaymentTradeNo(tt.method, tt.subscription, 42)
			assert.LessOrEqual(t, len(tradeNo), tt.maxLength)
			assert.Regexp(t, tt.pattern, tradeNo)
		})
	}
}

func TestBuildBalancePaymentTradeNoUsesBalancePrefix(t *testing.T) {
	tradeNo := BuildBalancePaymentTradeNo(42)

	assert.Regexp(t, regexp.MustCompile(`^BALANCE__42_[A-Za-z0-9]+$`), tradeNo)
}
