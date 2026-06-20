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
	"EO-Client-IPCountry",
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

	if reason, ok := blockedByScopedChinaMainlandPolicy(c, setting, scope); ok {
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

func blockedByScopedChinaMainlandPolicy(c *gin.Context, setting *access_setting.AccessControlSetting, scope AccessPolicyScope) (string, bool) {
	if !setting.BlockChinaMainlandHomepage && !setting.BlockChinaMainlandUserSensitivePages {
		return "", false
	}
	if !requestFromChinaMainland(c) {
		return "", false
	}

	path := normalizedRequestPath(c)
	if setting.BlockChinaMainlandUserSensitivePages && scope == AccessPolicyScopeAPI && isChinaMainlandSensitiveAPIPath(path) {
		role, ok := currentAuthenticatedRequestRole(c)
		if !ok {
			return "", false
		}
		if role >= common.RoleAuditAdminUser {
			return "", false
		}
		return "sensitive user page access from China Mainland is blocked", true
	}

	role, ok := currentRequestRole(c)
	if !ok {
		role = common.RoleGuestUser
	}
	if role >= common.RoleAuditAdminUser {
		return "", false
	}

	if setting.BlockChinaMainlandHomepage && scope == AccessPolicyScopeWeb && path == "/" {
		return "homepage access from China Mainland is blocked", true
	}
	if setting.BlockChinaMainlandUserSensitivePages && isChinaMainlandSensitivePath(scope, path) {
		return "sensitive user page access from China Mainland is blocked", true
	}
	return "", false
}

func requestFromChinaMainland(c *gin.Context) bool {
	country := requestCountry(c)
	return country.Known && access_setting.IsChinaMainlandCountryCode(country.CountryCode)
}

func normalizedRequestPath(c *gin.Context) string {
	path := c.Request.URL.Path
	if path == "" {
		return "/"
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
		if path == "" {
			return "/"
		}
	}
	return path
}

func isChinaMainlandSensitivePath(scope AccessPolicyScope, path string) bool {
	switch scope {
	case AccessPolicyScopeWeb:
		return isChinaMainlandSensitiveWebPath(path)
	case AccessPolicyScopeAPI:
		return isChinaMainlandSensitiveAPIPath(path)
	default:
		return false
	}
}

func isChinaMainlandSensitiveWebPath(path string) bool {
	switch path {
	case "/console/token", "/console/topup", "/console/billing":
		return true
	default:
		return false
	}
}

func isChinaMainlandSensitiveAPIPath(path string) bool {
	if path == "/api/token" || strings.HasPrefix(path, "/api/token/") {
		return true
	}
	if path == "/api/subscription/self" || strings.HasPrefix(path, "/api/subscription/self/") {
		return true
	}
	if strings.HasPrefix(path, "/api/subscription/") && strings.HasSuffix(path, "/pay") {
		return true
	}
	if path == "/api/user/topup" || strings.HasPrefix(path, "/api/user/topup/") {
		return true
	}
	if strings.HasPrefix(path, "/api/user/stripe/") ||
		strings.HasPrefix(path, "/api/user/creem/") ||
		strings.HasPrefix(path, "/api/user/waffo/") ||
		strings.HasPrefix(path, "/api/user/alipay/official/") ||
		strings.HasPrefix(path, "/api/user/wechat-pay/official/") ||
		strings.HasPrefix(path, "/api/user/self-serve/") {
		return true
	}
	switch path {
	case "/api/user/pay",
		"/api/user/amount",
		"/api/user/stripe/amount",
		"/api/user/aff",
		"/api/user/aff/commissions",
		"/api/user/aff_transfer":
		return true
	default:
		return false
	}
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

func RequestCountry(c *gin.Context) access_setting.CountryLookupResult {
	return requestCountry(c)
}

func IsChinaMainlandRequest(c *gin.Context) bool {
	return requestFromChinaMainland(c)
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

func currentAuthenticatedRequestRole(c *gin.Context) (int, bool) {
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
	return tokenUserRole(c, hasCredential)
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
