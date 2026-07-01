package setting

import "strings"

const (
	SelfServeWechatPayModePersonalQRCode      = "personal_qr"
	SelfServeWechatPayModeEnterpriseRedPacket = "enterprise_red_packet"
)

var (
	SelfServeTopUpEnabled         bool
	SelfServeAlipayEnabled        bool
	SelfServeWechatPayEnabled     bool
	SelfServeWechatPayMode        = SelfServeWechatPayModePersonalQRCode
	SelfServeAlipayQRCode         string
	SelfServeWechatPayQRCode      string
	SelfServeWechatPayEnterpriseQRCode string
	SelfServeTopUpUnitPrice       = 1.0
	SelfServeTopUpSingleMaxAmount float64
	SelfServeTopUpDailyMaxAmount  float64
)

func NormalizeSelfServeWechatPayMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", SelfServeWechatPayModePersonalQRCode, "personal", "qrcode", "qr":
		return SelfServeWechatPayModePersonalQRCode
	case SelfServeWechatPayModeEnterpriseRedPacket, "enterprise", "red_packet", "redpacket", "red-packet":
		return SelfServeWechatPayModeEnterpriseRedPacket
	default:
		return SelfServeWechatPayModePersonalQRCode
	}
}

func IsValidSelfServeWechatPayMode(mode string) bool {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", SelfServeWechatPayModePersonalQRCode, "personal", "qrcode", "qr", SelfServeWechatPayModeEnterpriseRedPacket, "enterprise", "red_packet", "redpacket", "red-packet":
		return true
	default:
		return false
	}
}

func SelfServeWechatPayModeRequiresTransactionNo(mode string) bool {
	return NormalizeSelfServeWechatPayMode(mode) != SelfServeWechatPayModeEnterpriseRedPacket
}

func SelfServeWechatPayQRCodeForMode(mode string) string {
	switch NormalizeSelfServeWechatPayMode(mode) {
	case SelfServeWechatPayModeEnterpriseRedPacket:
		return SelfServeWechatPayEnterpriseQRCode
	default:
		return SelfServeWechatPayQRCode
	}
}

func SelfServeWechatPayQRCodeContent() string {
	return SelfServeWechatPayQRCodeForMode(SelfServeWechatPayMode)
}

func SelfServeTopUpLimitsConfigured() bool {
	return SelfServeTopUpSingleMaxAmount > 0 &&
		SelfServeTopUpDailyMaxAmount > 0 &&
		SelfServeTopUpDailyMaxAmount >= SelfServeTopUpSingleMaxAmount
}

func SelfServeTopUpPricingConfigured() bool {
	return SelfServeTopUpUnitPrice > 0
}
