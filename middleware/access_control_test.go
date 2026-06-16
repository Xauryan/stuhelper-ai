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
	return performAccessControlRequestWithRole(scope, header, routeTag, nil)
}

func performAccessControlRequestWithRole(scope AccessPolicyScope, header map[string]string, routeTag string, role *int) *httptest.ResponseRecorder {
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
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	for key, value := range header {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
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

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), "访问受限")
}

func TestAccessControlBlocksEuropeanUnionHeader(t *testing.T) {
	withAccessControlSetting(t, func(setting *access_setting.AccessControlSetting) {
		setting.WebPolicyEnabled = true
		setting.BlockEuropeanUnion = true
	})

	recorder := performAccessControlRequest(AccessPolicyScopeWeb, map[string]string{
		"CloudFront-Viewer-Country": "DE",
	}, "")

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "访问受限")
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

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
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
