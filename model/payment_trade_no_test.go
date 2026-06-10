package model

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPaymentTradeNoUsesUnifiedProviderFormat(t *testing.T) {
	patterns := map[string]*regexp.Regexp{
		"SSU":    regexp.MustCompile(`^SSU_42_\d+_[A-Za-z0-9]+$`),
		"SSSUB":  regexp.MustCompile(`^SSSUB_42_\d+_[A-Za-z0-9]+$`),
		"SUBBAL": regexp.MustCompile(`^SUBBAL_42_\d+_[A-Za-z0-9]+$`),
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
