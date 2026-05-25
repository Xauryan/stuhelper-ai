package setting

var (
	AlipayOfficialEnabled           bool
	AlipayOfficialAppID             string
	AlipayOfficialAppAuthToken      string
	AlipayOfficialPrivateKey        string
	AlipayOfficialAlipayPublicKey   string
	AlipayOfficialAppCertSN         string
	AlipayOfficialRootCertSN        string
	AlipayOfficialAlipayCertSN      string
	AlipayOfficialSandbox           bool
	AlipayOfficialNotifyURL         string
	AlipayOfficialReturnURL         string
	AlipayOfficialUnitPrice         float64 = 1.0
	AlipayOfficialServiceFeePercent float64
	AlipayOfficialMinTopUp          int = 1
	AlipayOfficialOrderTimeoutSec   int = 600
	AlipayOfficialOrderTimeoutMin   int = 10

	WechatPayOfficialEnabled           bool
	WechatPayOfficialAppID             string
	WechatPayOfficialMchID             string
	WechatPayOfficialCertificateSerial string
	WechatPayOfficialAPIv3Key          string
	WechatPayOfficialPrivateKey        string
	WechatPayOfficialPlatformPublicKey string
	WechatPayOfficialNotifyURL         string
	WechatPayOfficialReturnURL         string
	WechatPayOfficialUnitPrice         float64 = 1.0
	WechatPayOfficialServiceFeePercent float64
	WechatPayOfficialMinTopUp          int = 1
	WechatPayOfficialOrderTimeoutSec   int = 600
)
