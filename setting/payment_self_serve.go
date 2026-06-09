package setting

var (
	SelfServeTopUpEnabled         bool
	SelfServeAlipayEnabled        bool
	SelfServeWechatPayEnabled     bool
	SelfServeAlipayQRCode         string
	SelfServeWechatPayQRCode      string
	SelfServeTopUpSingleMaxAmount float64
	SelfServeTopUpDailyMaxAmount  float64
	SelfServeRejectAutoBan        = true
)

func SelfServeTopUpLimitsConfigured() bool {
	return SelfServeTopUpSingleMaxAmount > 0 &&
		SelfServeTopUpDailyMaxAmount > 0 &&
		SelfServeTopUpDailyMaxAmount >= SelfServeTopUpSingleMaxAmount
}
