package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/setting/access_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func withAccessControlSetting(t *testing.T, update func(setting *access_setting.AccessControlSetting)) {
	t.Helper()
	setting := access_setting.GetAccessControlSetting()
	original := *setting
	update(setting)
	t.Cleanup(func() {
		*setting = original
	})
}

func performAccessControlRequest(scope AccessPolicyScope, header map[string]string, routeTag string) *httptest.ResponseRecorder {
	return performAccessControlRequestWithRoleAtPath(scope, header, routeTag, nil, "/test")
}

func performAccessControlRequestWithRole(scope AccessPolicyScope, header map[string]string, routeTag string, role *int) *httptest.ResponseRecorder {
	return performAccessControlRequestWithRoleAtPath(scope, header, routeTag, role, "/test")
}

func performAccessControlRequestAtPath(scope AccessPolicyScope, header map[string]string, routeTag string, path string) *httptest.ResponseRecorder {
	return performAccessControlRequestWithRoleAtPath(scope, header, routeTag, nil, path)
}

func performAccessControlRequestWithRoleAtPath(scope AccessPolicyScope, header map[string]string, routeTag string, role *int, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if routeTag != "" {
		router.Use(RouteTag(routeTag))
	}
	if role != nil {
		router.Use(func(c *gin.Context) {
			c.Set("role", *role)
			c.Next()
		})
	}
	router.Use(AccessControl(scope))
	router.GET("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest(http.MethodGet, path, nil)
	for key, value := range header {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func boolPtr(value bool) *bool {
	return &value
}

func requireWebAccessDeniedPage(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	body := recorder.Body.String()
	require.Contains(t, body, "本站不对您所在的地区开放")
	require.Contains(t, body, "您当前 IP")
	require.Contains(t, body, "IP 归属地")
	require.Contains(t, body, `replaceState(null, "", "/forbidden?access_limited=1")`)
	require.NotContains(t, body, `"success":false`)
}

func TestAccessControlAllowsWhenPolicyDisabled(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = false
		setting.BlockChinaMainland = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"CF-IPCountry": "CN",
	}, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestAccessControlBlocksChinaMainlandHeader(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockChinaMainland = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"CF-IPCountry": "CN",
	}, "")

	requireWebAccessDeniedPage(t, recorder)
	require.Contains(t, recorder.Body.String(), "中国大陆")
}

func TestAccessControlRecognizesTencentEOClientIPCountryHeader(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockChinaMainland = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"EO-Client-IPCountry": "CN",
	}, "")

	requireWebAccessDeniedPage(t, recorder)
}

func TestAccessControlBlocksEuropeanUnionHeader(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockEuropeanUnion = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"CloudFront-Viewer-Country": "DE",
	}, "")

	requireWebAccessDeniedPage(t, recorder)
	require.Contains(t, recorder.Body.String(), "欧盟地区（DE）")
}

func TestAccessControlRoleGeoRuleBlocksChinaMainlandUserOnly(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.RoleGeoRules = map[string]access_setting.RoleGeoAccessRule{
			access_setting.RoleGeoSourceChinaMainland: {
				User: boolPtr(true),
			},
		}
	})

	header := map[string]string{"EO-Client-IPCountry": "CN"}
	userRole := common.RoleCommonUser
	userRecorder := performAccessControlRequestWithRole(AccessPolicyScopeWeb, header, "web", &userRole)
	require.Equal(t, http.StatusForbidden, userRecorder.Code)

	adminRole := common.RoleAdminUser
	adminRecorder := performAccessControlRequestWithRole(AccessPolicyScopeWeb, header, "web", &adminRole)
	require.Equal(t, http.StatusOK, adminRecorder.Code)

	usUserRecorder := performAccessControlRequestWithRole(AccessPolicyScopeWeb, map[string]string{
		"EO-Client-IPCountry": "US",
	}, "web", &userRole)
	require.Equal(t, http.StatusOK, usUserRecorder.Code)
}

func TestAccessControlRoleGeoRuleBlocksEuropeanUnionGuest(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.RoleGeoRules = map[string]access_setting.RoleGeoAccessRule{
			access_setting.RoleGeoSourceEuropeanUnion: {
				Guest: boolPtr(true),
			},
		}
	})

	guestRecorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"CloudFront-Viewer-Country": "DE",
	}, "web")
	require.Equal(t, http.StatusForbidden, guestRecorder.Code)

	userRole := common.RoleCommonUser
	userRecorder := performAccessControlRequestWithRole(AccessPolicyScopeWeb, map[string]string{
		"CloudFront-Viewer-Country": "DE",
	}, "web", &userRole)
	require.Equal(t, http.StatusOK, userRecorder.Code)
}

func TestAccessControlRoleGeoRuleBlocksAllSourceAuditAdminOnly(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.RoleGeoRules = map[string]access_setting.RoleGeoAccessRule{
			access_setting.RoleGeoSourceAll: {
				AuditAdmin: boolPtr(true),
			},
		}
	})

	auditRole := common.RoleAuditAdminUser
	auditRecorder := performAccessControlRequestWithRole(AccessPolicyScopeAPI, nil, "api", &auditRole)
	require.Equal(t, http.StatusForbidden, auditRecorder.Code)

	adminRole := common.RoleAdminUser
	adminRecorder := performAccessControlRequestWithRole(AccessPolicyScopeAPI, nil, "api", &adminRole)
	require.Equal(t, http.StatusOK, adminRecorder.Code)
}

func TestAccessControlRoleGeoRuleBlocksUnknownCountryGuest(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.RoleGeoRules = map[string]access_setting.RoleGeoAccessRule{
			access_setting.RoleGeoSourceUnknown: {
				Guest: boolPtr(true),
			},
		}
	})

	unknownRecorder := performAccessControlRequest(AccessPolicyScopeWeb, nil, "web")
	require.Equal(t, http.StatusForbidden, unknownRecorder.Code)

	knownRecorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"EO-Client-IPCountry": "US",
	}, "web")
	require.Equal(t, http.StatusOK, knownRecorder.Code)
}

func TestAccessControlRoleGeoRuleDefersCredentialedAPIUntilAuth(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.RoleGeoRules = map[string]access_setting.RoleGeoAccessRule{
			access_setting.RoleGeoSourceChinaMainland: {
				Guest: boolPtr(true),
			},
		}
	})

	recorder := performAccessControlRequest(AccessPolicyScopeAPI, map[string]string{
		"EO-Client-IPCountry": "CN",
		"Authorization":       "Bearer sk-test",
	}, "api")

	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAccessControlSourceResourceRuleBlocksChinaMainlandGuestHome(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.SourceResourceRules = map[string]map[string]access_setting.SourceResourceAccessRule{
			access_setting.RoleGeoSourceChinaMainland: {
				AccessResourceHome: {
					Guest: boolPtr(true),
				},
			},
		}
	})

	header := map[string]string{"EO-Client-IPCountry": "CN"}
	homeRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, header, "web", "/")
	require.Equal(t, http.StatusForbidden, homeRecorder.Code)

	fallbackRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, header, "web", "/1")
	require.Equal(t, http.StatusForbidden, fallbackRecorder.Code)

	pricingRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, header, "web", "/pricing")
	require.Equal(t, http.StatusOK, pricingRecorder.Code)
}

func TestAccessControlSourceResourceRuleDefersCredentialedAPIUntilAuth(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.SourceResourceRules = map[string]map[string]access_setting.SourceResourceAccessRule{
			access_setting.RoleGeoSourceChinaMainland: {
				AccessResourceModelAPI: {
					Guest: boolPtr(true),
				},
			},
		}
	})

	recorder := performAccessControlRequestAtPath(AccessPolicyScopeAPI, map[string]string{
		"EO-Client-IPCountry": "CN",
		"Authorization":       "Bearer sk-test",
	}, "relay", "/v1/chat/completions")
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAccessControlBlocksAPIGuestWithoutCredential(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.BlockGuests = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeAPI, nil, "api")

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
}

func TestAccessControlDefersCredentialedAPIIdentityUntilAuth(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.BlockGuests = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeAPI, map[string]string{
		"Authorization": "Bearer sk-test",
	}, "api")

	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAccessControlDoesNotDeferWebGuestWithAPICredentialHeader(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockGuests = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"Authorization": "Bearer sk-test",
	}, "web")

	requireWebAccessDeniedPage(t, recorder)
}

func TestAccessControlRelayUsesOpenAIErrorShape(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.BlockEuropeanUnion = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeAPI, map[string]string{
		"CF-IPCountry": "FR",
	}, "relay")

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"error":`)
	require.Contains(t, recorder.Body.String(), `"type":"new_api_error"`)
}

func TestAccessControlBlocksAuditAdminAndHigherWhenAdminsBlocked(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.BlockAdmins = true
	})

	for _, role := range []int{
		common.RoleAuditAdminUser,
		common.RoleAdminUser,
		common.RoleRootUser,
	} {
		recorder := performAccessControlRequestWithRole(AccessPolicyScopeAPI, nil, "api", &role)

		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"success":false`)
	}
}

func TestAccessControlDoesNotBlockAdminsWhenOnlyUsersBlocked(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.BlockUsers = true
	})

	adminRole := common.RoleAuditAdminUser
	adminRecorder := performAccessControlRequestWithRole(AccessPolicyScopeAPI, nil, "api", &adminRole)
	require.Equal(t, http.StatusOK, adminRecorder.Code)

	userRole := common.RoleCommonUser
	userRecorder := performAccessControlRequestWithRole(AccessPolicyScopeAPI, nil, "api", &userRole)
	require.Equal(t, http.StatusForbidden, userRecorder.Code)
}

func TestAccessControlBlocksChinaMainlandHomepageOnly(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockChinaMainlandHomepage = true
	})

	header := map[string]string{"EO-Client-IPCountry": "CN"}
	homepageRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, header, "web", "/")
	require.Equal(t, http.StatusForbidden, homepageRecorder.Code)

	logRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, header, "web", "/console/log")
	require.Equal(t, http.StatusOK, logRecorder.Code)
}

func TestAccessControlBlocksChinaMainlandHomepageFallbackPath(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockChinaMainlandHomepage = true
	})

	header := map[string]string{"EO-Client-IPCountry": "CN"}
	fallbackRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, header, "web", "/1")
	require.Equal(t, http.StatusForbidden, fallbackRecorder.Code)

	loginRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, header, "web", "/login")
	require.Equal(t, http.StatusOK, loginRecorder.Code)
}

func TestAccessControlChinaMainlandHomepageAllowsAdmins(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockChinaMainlandHomepage = true
	})

	adminRole := common.RoleAdminUser
	recorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeWeb, map[string]string{
		"EO-Client-IPCountry": "CN",
	}, "web", &adminRole, "/")

	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAccessControlBlocksChinaMainlandSensitiveWebPagesForUsersOnly(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockChinaMainlandUserSensitivePages = true
	})

	header := map[string]string{"EO-Client-IPCountry": "CN"}
	userRole := common.RoleCommonUser
	for _, path := range []string{"/console/token", "/console/topup", "/console/billing"} {
		recorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeWeb, header, "web", &userRole, path)
		require.Equal(t, http.StatusForbidden, recorder.Code, path)
	}

	logRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeWeb, header, "web", &userRole, "/console/log")
	require.Equal(t, http.StatusOK, logRecorder.Code)

	adminRole := common.RoleAuditAdminUser
	adminRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeWeb, header, "web", &adminRole, "/console/token")
	require.Equal(t, http.StatusOK, adminRecorder.Code)
}

func TestAccessControlDefersChinaMainlandSensitiveAPIWithCredentialUntilAuth(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.BlockChinaMainlandUserSensitivePages = true
	})

	recorder := performAccessControlRequestAtPath(AccessPolicyScopeAPI, map[string]string{
		"EO-Client-IPCountry": "CN",
		"Authorization":       "Bearer sk-test",
	}, "api", "/api/token")

	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestAccessControlBlocksChinaMainlandSensitiveAPIAfterAuth(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.BlockChinaMainlandUserSensitivePages = true
	})

	header := map[string]string{"EO-Client-IPCountry": "CN"}
	userRole := common.RoleCommonUser
	for _, path := range []string{
		"/api/token",
		"/api/user/topup/info",
		"/api/user/topup/self",
		"/api/user/waffo/amount",
		"/api/user/stripe/pay",
		"/api/user/self-serve/preview",
		"/api/user/wechat-pay/official/status",
		"/api/subscription/stripe/pay",
		"/api/subscription/balance/pay",
		"/api/subscription/self/preference",
	} {
		recorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeAPI, header, "api", &userRole, path)
		require.Equal(t, http.StatusForbidden, recorder.Code, path)
	}

	logRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeAPI, header, "api", &userRole, "/api/log/self")
	require.Equal(t, http.StatusOK, logRecorder.Code)

	callbackRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeAPI, header, "api", &userRole, "/api/subscription/epay/notify")
	require.Equal(t, http.StatusOK, callbackRecorder.Code)

	adminRole := common.RoleAdminUser
	adminRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeAPI, header, "api", &adminRole, "/api/token")
	require.Equal(t, http.StatusOK, adminRecorder.Code)
}

func TestAccessControlResourceRuleBlocksGuestHomepage(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.ResourceRules = map[string]access_setting.ResourceAccessRule{
			AccessResourceHome: {
				Guest: boolPtr(false),
			},
		}
	})

	homepageRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, nil, "web", "/")
	require.Equal(t, http.StatusForbidden, homepageRecorder.Code)

	loginRecorder := performAccessControlRequestAtPath(AccessPolicyScopeWeb, nil, "web", "/login")
	require.Equal(t, http.StatusOK, loginRecorder.Code)
}

func TestAccessControlResourceRuleBlocksTokenForUserAndAdminButAllowsRoot(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.APIPolicyEnabled = true
		setting.ResourceRules = map[string]access_setting.ResourceAccessRule{
			AccessResourceToken: {
				User:  boolPtr(false),
				Admin: boolPtr(false),
				Root:  boolPtr(true),
			},
		}
	})

	userRole := common.RoleCommonUser
	userWebRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeWeb, nil, "web", &userRole, "/console/token")
	require.Equal(t, http.StatusForbidden, userWebRecorder.Code)

	adminRole := common.RoleAdminUser
	adminAPIRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeAPI, nil, "api", &adminRole, "/api/token")
	require.Equal(t, http.StatusForbidden, adminAPIRecorder.Code)

	rootRole := common.RoleRootUser
	rootAPIRecorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeAPI, nil, "api", &rootRole, "/api/token")
	require.Equal(t, http.StatusOK, rootAPIRecorder.Code)
}

func TestAccessControlModelAPIRuleDoesNotBlockOrdinaryManagementAPI(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.ResourceRules = map[string]access_setting.ResourceAccessRule{
			AccessResourceModelAPI: {
				Guest: boolPtr(false),
				User:  boolPtr(false),
				Admin: boolPtr(false),
				Root:  boolPtr(false),
			},
		}
	})

	statusRecorder := performAccessControlRequestAtPath(AccessPolicyScopeAPI, nil, "api", "/api/status")
	require.Equal(t, http.StatusOK, statusRecorder.Code)

	relayRecorder := performAccessControlRequestAtPath(AccessPolicyScopeAPI, nil, "relay", "/v1/chat/completions")
	require.Equal(t, http.StatusForbidden, relayRecorder.Code)
	require.Contains(t, relayRecorder.Body.String(), `"error":`)
}

func TestAccessControlModelAPIRuleDoesNotBlockOldDashboardBillingAPI(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.APIPolicyEnabled = true
		setting.ResourceRules = map[string]access_setting.ResourceAccessRule{
			AccessResourceModelAPI: {
				User: boolPtr(false),
			},
		}
	})

	userRole := common.RoleCommonUser
	recorder := performAccessControlRequestWithRoleAtPath(AccessPolicyScopeAPI, nil, "old_api", &userRole, "/v1/dashboard/billing/usage")
	require.Equal(t, http.StatusOK, recorder.Code)
}
