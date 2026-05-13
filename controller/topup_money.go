package controller

import "github.com/shopspring/decimal"

func ceilPayMoneyToCents(payMoney decimal.Decimal) float64 {
	return payMoney.RoundCeil(2).InexactFloat64()
}

func formatPayMoneyToCents(payMoney float64) string {
	return decimal.NewFromFloat(payMoney).RoundCeil(2).StringFixed(2)
}

func payMoneyYuanToFen(payMoney float64) int64 {
	return decimal.NewFromFloat(payMoney).Mul(decimal.NewFromInt(100)).RoundCeil(0).IntPart()
}
