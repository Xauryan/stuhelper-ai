package access_setting

import "github.com/Xauryan/stuhelper-ai/setting/config"

type AccessControlSetting struct {
	WebPolicyEnabled bool `json:"web_policy_enabled"`
	APIPolicyEnabled bool `json:"api_policy_enabled"`

	BlockChinaMainland bool `json:"block_china_mainland"`
	BlockEuropeanUnion bool `json:"block_european_union"`

	BlockChinaMainlandHomepage           bool `json:"block_china_mainland_homepage"`
	BlockChinaMainlandUserSensitivePages bool `json:"block_china_mainland_user_sensitive_pages"`

	BlockGuests bool `json:"block_guests"`
	BlockUsers  bool `json:"block_users"`
	BlockAdmins bool `json:"block_admins"`

	GeoIPDatabasePath string `json:"geoip_database_path"`
}

var accessControlSetting = AccessControlSetting{
	WebPolicyEnabled:                     false,
	APIPolicyEnabled:                     false,
	BlockChinaMainland:                   false,
	BlockEuropeanUnion:                   false,
	BlockChinaMainlandHomepage:           false,
	BlockChinaMainlandUserSensitivePages: false,
	BlockGuests:                          false,
	BlockUsers:                           false,
	BlockAdmins:                          false,
	GeoIPDatabasePath:                    "",
}

func init() {
	config.GlobalConfig.Register("access_control", &accessControlSetting)
}

func GetAccessControlSetting() *AccessControlSetting {
	return &accessControlSetting
}
