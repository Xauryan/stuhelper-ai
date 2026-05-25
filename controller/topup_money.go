package controller

import (
	"math"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

type payMoneyBreakdown struct {
	EffectiveMoney float64
	Fee            float64
	TotalMoney     float64
}

func buildPayMoneyBreakdown(effectiveMoney decimal.Decimal, serviceFeePercent float64) payMoneyBreakdown {
	effectiveMoney = effectiveMoney.RoundCeil(2)
	if effectiveMoney.IsNegative() {
		effectiveMoney = decimal.Zero
	}

	feePercent := decimal.NewFromFloat(normalizeServiceFeePercent(serviceFeePercent))
	if feePercent.IsNegative() {
		feePercent = decimal.Zero
	}

	totalMoney := effectiveMoney
	if feePercent.IsPositive() {
		feeMultiplier := decimal.NewFromInt(1).Add(feePercent.Div(decimal.NewFromInt(100)))
		totalMoney = effectiveMoney.Mul(feeMultiplier).RoundCeil(2)
	}

	fee := totalMoney.Sub(effectiveMoney).Round(2)
	if fee.IsNegative() {
		fee = decimal.Zero
	}

	return payMoneyBreakdown{
		EffectiveMoney: effectiveMoney.InexactFloat64(),
		Fee:            fee.InexactFloat64(),
		TotalMoney:     totalMoney.InexactFloat64(),
	}
}

func normalizeServiceFeePercent(serviceFeePercent float64) float64 {
	if math.IsNaN(serviceFeePercent) || math.IsInf(serviceFeePercent, 0) {
		return 0
	}
	feePercent := decimal.NewFromFloat(serviceFeePercent)
	if feePercent.IsNegative() {
		return 0
	}
	return feePercent.InexactFloat64()
}

func parseServiceFeePercent(value string) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	percent, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return normalizeServiceFeePercent(percent)
}

func ceilPayMoneyToCents(payMoney decimal.Decimal) float64 {
	return payMoney.RoundCeil(2).InexactFloat64()
}

func formatPayMoneyToCents(payMoney float64) string {
	return decimal.NewFromFloat(payMoney).RoundCeil(2).StringFixed(2)
}

func payMoneyYuanToFen(payMoney float64) int64 {
	return decimal.NewFromFloat(payMoney).Mul(decimal.NewFromInt(100)).RoundCeil(0).IntPart()
}
