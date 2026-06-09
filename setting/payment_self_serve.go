package setting

var (
	SelfServeTopUpEnabled         bool
	SelfServeAlipayEnabled        bool
	SelfServeWechatPayEnabled     bool
	SelfServeAlipayQRCode         string
	SelfServeWechatPayQRCode      string
	SelfServeTopUpUnitPrice       = 1.0
	SelfServeTopUpSingleMaxAmount float64
	SelfServeTopUpDailyMaxAmount  float64
)

func SelfServeTopUpLimitsConfigured() bool {
	return SelfServeTopUpSingleMaxAmount > 0 &&
		SelfServeTopUpDailyMaxAmount > 0 &&
		SelfServeTopUpDailyMaxAmount >= SelfServeTopUpSingleMaxAmount
}

func SelfServeTopUpPricingConfigured() bool {
	return SelfServeTopUpUnitPrice > 0
}
