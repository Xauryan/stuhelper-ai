package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/setting/access_setting"
	"github.com/Xauryan/stuhelper-ai/types"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AccessPolicyScope string

const (
	AccessPolicyScopeWeb AccessPolicyScope = "web"
	AccessPolicyScopeAPI AccessPolicyScope = "api"
)

var countryHeaderCandidates = []string{
	"CF-IPCountry",
	"CloudFront-Viewer-Country",
	"X-Vercel-IP-Country",
	"X-Country-Code",
	"X-Geo-Country",
}

func AccessControl(scope AccessPolicyScope) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enforceAccessPolicy(c, scope) {
			c.Next()
		}
	}
}

func enforceAccessPolicy(c *gin.Context, scope AccessPolicyScope) bool {
	if !isAccessPolicyEnabled(scope) {
		return true
	}

	setting := access_setting.GetAccessControlSetting()
	if reason, ok := blockedByGeo(c, setting); ok {
		abortAccessDenied(c, scope, reason)
		return false
	}

	if reason, ok := blockedByIdentity(c, setting, scope); ok {
		abortAccessDenied(c, scope, reason)
		return false
	}

	return true
}

func isAccessPolicyEnabled(scope AccessPolicyScope) bool {
	setting := access_setting.GetAccessControlSetting()
	switch scope {
	case AccessPolicyScopeWeb:
		return setting.WebPolicyEnabled
	case AccessPolicyScopeAPI:
		return setting.APIPolicyEnabled
	default:
		return false
	}
}

func blockedByGeo(c *gin.Context, setting *access_setting.AccessControlSetting) (string, bool) {
	if !setting.BlockChinaMainland && !setting.BlockEuropeanUnion {
		return "", false
	}

	country := requestCountry(c)
	if !country.Known {
		return "", false
	}

	if setting.BlockChinaMainland && access_setting.IsChinaMainlandCountryCode(country.CountryCode) {
		return fmt.Sprintf("access from China Mainland is blocked (%s)", country.Source), true
	}
	if setting.BlockEuropeanUnion && access_setting.IsEuropeanUnionCountryCode(country.CountryCode) {
		return fmt.Sprintf("access from the European Union is blocked (%s)", country.Source), true
	}
	return "", false
}

func requestCountry(c *gin.Context) access_setting.CountryLookupResult {
	for _, header := range countryHeaderCandidates {
		code := access_setting.NormalizeCountryCode(c.GetHeader(header))
		if validCountryHeaderValue(code) {
			return access_setting.CountryLookupResult{
				CountryCode: code,
				Source:      header,
				Known:       true,
			}
		}
	}

	ip := net.ParseIP(c.ClientIP())
	if ip == nil {
		return access_setting.CountryLookupResult{}
	}
	return access_setting.LookupCountry(ip)
}

func validCountryHeaderValue(code string) bool {
	if len(code) != 2 {
		return false
	}
	if code == "XX" || code == "T1" || code == "A1" || code == "A2" || code == "O1" {
		return false
	}
	return true
}

func blockedByIdentity(c *gin.Context, setting *access_setting.AccessControlSetting, scope AccessPolicyScope) (string, bool) {
	if !setting.BlockGuests && !setting.BlockUsers && !setting.BlockAdmins {
		return "", false
	}

	role, ok := currentRequestRole(c)
	if !ok {
		if scope == AccessPolicyScopeAPI && hasAPIAuthCredential(c) {
			return "", false
		}
		role = common.RoleGuestUser
	}

	switch {
	case role >= common.RoleAuditAdminUser:
		if setting.BlockAdmins {
			return "administrator access is blocked", true
		}
	case role >= common.RoleCommonUser:
		if setting.BlockUsers {
			return "user access is blocked", true
		}
	default:
		if setting.BlockGuests {
			return "guest access is blocked", true
		}
	}
	return "", false
}

func currentRequestRole(c *gin.Context) (int, bool) {
	hasCredential := hasAPIAuthCredential(c)
	if role, exists := c.Get("role"); exists {
		if normalized, ok := normalizeRole(role); ok {
			return roleOrTokenRole(c, normalized, hasCredential)
		}
	}
	if role, ok := common.GetContextKey(c, constant.ContextKeyUserRole); ok {
		if normalized, ok := normalizeRole(role); ok {
			return roleOrTokenRole(c, normalized, hasCredential)
		}
	}

	if role, ok := tokenUserRole(c, hasCredential); ok {
		return role, true
	}

	if _, exists := c.Get(sessions.DefaultKey); exists {
		session := sessions.Default(c)
		if role, ok := normalizeRole(session.Get("role")); ok {
			return role, true
		}
	}

	return common.RoleGuestUser, false
}

func roleOrTokenRole(c *gin.Context, role int, hasCredential bool) (int, bool) {
	if role != common.RoleGuestUser || !hasCredential {
		return role, true
	}
	return tokenUserRole(c, hasCredential)
}

func tokenUserRole(c *gin.Context, hasCredential bool) (int, bool) {
	if tokenUserID := c.GetInt("id"); tokenUserID > 0 && hasCredential {
		if user, err := model.GetUserCache(tokenUserID); err == nil {
			if user.Role > 0 {
				return user.Role, true
			}
			return userRoleFromCacheOrDB(user.Id), true
		}
	}
	return common.RoleGuestUser, false
}

func normalizeRole(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		role, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return role, true
	default:
		return 0, false
	}
}

func hasAPIAuthCredential(c *gin.Context) bool {
	return strings.TrimSpace(c.GetHeader("Authorization")) != "" ||
		strings.TrimSpace(c.GetHeader("api-key")) != "" ||
		strings.TrimSpace(c.GetHeader("mj-api-secret")) != "" ||
		strings.TrimSpace(c.GetHeader("x-api-key")) != "" ||
		strings.TrimSpace(c.GetHeader("x-goog-api-key")) != "" ||
		strings.Contains(c.GetHeader("Sec-WebSocket-Protocol"), "openai-insecure-api-key") ||
		strings.TrimSpace(c.Query("key")) != ""
}

func userRoleFromCacheOrDB(userID int) int {
	user, err := model.GetUserById(userID, false)
	if err != nil {
		return common.RoleCommonUser
	}
	return user.Role
}

func abortAccessDenied(c *gin.Context, scope AccessPolicyScope, reason string) {
	message := "访问受限"
	if common.DebugEnabled && reason != "" {
		message = message + ": " + reason
	}

	if c.GetString(RouteTagKey) == "relay" {
		abortWithOpenAiMessage(c, http.StatusForbidden, message, types.ErrorCodeAccessDenied)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": message,
	})
	c.Abort()
}
