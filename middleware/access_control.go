package middleware

import (
	"fmt"
	"html"
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

type AccessDeniedRequestInfo struct {
	IP            string `json:"ip"`
	CountryCode   string `json:"country_code"`
	CountryLabel  string `json:"country_label"`
	CountryKnown  bool   `json:"country_known"`
	CountrySource string `json:"country_source"`
}

const (
	AccessResourceAll               = "all"
	AccessResourceWeb               = "web"
	AccessResourceHome              = "home"
	AccessResourceModelAPI          = "model_api"
	AccessResourceToken             = "token"
	AccessResourceWallet            = "wallet"
	AccessResourceBilling           = "billing"
	AccessResourceUsageLog          = "usage_log"
	AccessResourceDashboard         = "dashboard"
	AccessResourcePlayground        = "playground"
	AccessResourceChat              = "chat"
	AccessResourcePersonal          = "personal"
	AccessResourceDrawingLog        = "drawing_log"
	AccessResourceTaskLog           = "task_log"
	AccessResourceAdminChannel      = "admin_channel"
	AccessResourceAdminSubscription = "admin_subscription"
	AccessResourceAdminModel        = "admin_model"
	AccessResourceAdminRedemption   = "admin_redemption"
	AccessResourceAdminUser         = "admin_user"
	AccessResourceAdminReferral     = "admin_referral"
	AccessResourceAdminSetting      = "admin_setting"
)

var accessResourceKeys = []string{
	AccessResourceWeb,
	AccessResourceHome,
	AccessResourceModelAPI,
	AccessResourceToken,
	AccessResourceWallet,
	AccessResourceBilling,
	AccessResourceUsageLog,
	AccessResourceDashboard,
	AccessResourcePlayground,
	AccessResourceChat,
	AccessResourcePersonal,
	AccessResourceDrawingLog,
	AccessResourceTaskLog,
	AccessResourceAdminChannel,
	AccessResourceAdminSubscription,
	AccessResourceAdminModel,
	AccessResourceAdminRedemption,
	AccessResourceAdminUser,
	AccessResourceAdminReferral,
	AccessResourceAdminSetting,
}

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

	if reason, ok := blockedByRoleGeoRule(c, setting, scope); ok {
		abortAccessDenied(c, scope, reason)
		return false
	}

	if reason, ok := blockedBySourceResourceRule(c, setting, scope); ok {
		abortAccessDenied(c, scope, reason)
		return false
	}

	if reason, ok := blockedByScopedChinaMainlandPolicy(c, setting, scope); ok {
		abortAccessDenied(c, scope, reason)
		return false
	}

	if reason, ok := blockedByResourceRule(c, setting, scope); ok {
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

func blockedByRoleGeoRule(c *gin.Context, setting *access_setting.AccessControlSetting, scope AccessPolicyScope) (string, bool) {
	if setting == nil || len(setting.RoleGeoRules) == 0 {
		return "", false
	}

	role, ok := currentRequestRole(c)
	if !ok {
		if scope == AccessPolicyScopeAPI && hasAPIAuthCredential(c) {
			return "", false
		}
		role = common.RoleGuestUser
	}

	sources := roleGeoSourcesForRequest(c)
	roleLevel := roleAccessLevel(role)
	for _, source := range sources {
		if roleGeoRuleBlocksRole(setting.RoleGeoRules[source], roleLevel) {
			return fmt.Sprintf("access from %s is blocked for %s", source, roleLevel), true
		}
	}
	return "", false
}

func roleGeoSourcesForRequest(c *gin.Context) []string {
	sources := []string{access_setting.RoleGeoSourceAll}
	country := requestCountry(c)
	if !country.Known {
		return append(sources, access_setting.RoleGeoSourceUnknown)
	}

	if access_setting.IsChinaMainlandCountryCode(country.CountryCode) {
		sources = append(sources, access_setting.RoleGeoSourceChinaMainland)
	}
	if access_setting.IsEuropeanUnionCountryCode(country.CountryCode) {
		sources = append(sources, access_setting.RoleGeoSourceEuropeanUnion)
	}
	return sources
}

func roleGeoRuleBlocksRole(rule access_setting.RoleGeoAccessRule, roleLevel string) bool {
	var value *bool
	switch roleLevel {
	case "root":
		value = rule.Root
	case "admin":
		value = rule.Admin
	case "audit_admin":
		value = rule.AuditAdmin
	case "user":
		value = rule.User
	default:
		value = rule.Guest
	}
	return value != nil && *value
}

func blockedBySourceResourceRule(c *gin.Context, setting *access_setting.AccessControlSetting, scope AccessPolicyScope) (string, bool) {
	if setting == nil || len(setting.SourceResourceRules) == 0 {
		return "", false
	}

	keys := resourceKeysForRequest(scope, c.Request.Method, normalizedRequestPath(c), c.GetString(RouteTagKey))
	if len(keys) == 0 {
		return "", false
	}

	role, ok := currentRequestRole(c)
	if !ok {
		if scope == AccessPolicyScopeAPI && hasAPIAuthCredential(c) {
			return "", false
		}
		role = common.RoleGuestUser
	}

	sources := roleGeoSourcesForRequest(c)
	roleLevel := roleAccessLevel(role)
	for _, source := range sources {
		resourceRules := setting.SourceResourceRules[source]
		if len(resourceRules) == 0 {
			continue
		}
		if sourceResourceRuleBlocksRole(resourceRules[AccessResourceAll], roleLevel) {
			return fmt.Sprintf("all resource access from %s is blocked for %s", source, roleLevel), true
		}
		for _, key := range keys {
			if sourceResourceRuleBlocksRole(resourceRules[key], roleLevel) {
				return fmt.Sprintf("resource %s access from %s is blocked for %s", key, source, roleLevel), true
			}
		}
	}
	return "", false
}

func sourceResourceRuleBlocksRole(rule access_setting.SourceResourceAccessRule, roleLevel string) bool {
	var value *bool
	switch roleLevel {
	case "root":
		value = rule.Root
	case "admin":
		value = rule.Admin
	case "audit_admin":
		value = rule.AuditAdmin
	case "user":
		value = rule.User
	default:
		value = rule.Guest
	}
	return value != nil && *value
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

	if setting.BlockChinaMainlandHomepage && scope == AccessPolicyScopeWeb && webPathMatchesHomeAccess(path) {
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

func webPathMatchesHomeAccess(path string) bool {
	return path == "/" || isWebSPAFallbackPath(path)
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

func blockedByResourceRule(c *gin.Context, setting *access_setting.AccessControlSetting, scope AccessPolicyScope) (string, bool) {
	keys := resourceKeysForRequest(scope, c.Request.Method, normalizedRequestPath(c), c.GetString(RouteTagKey))
	if len(keys) == 0 {
		return "", false
	}

	role, ok := currentRequestRole(c)
	if !ok {
		if scope == AccessPolicyScopeAPI && hasAPIAuthCredential(c) {
			return "", false
		}
		role = common.RoleGuestUser
	}

	for _, key := range keys {
		if !resourceAllowedForRole(setting, key, role) {
			return fmt.Sprintf("resource %s access is blocked for %s", key, roleAccessLevel(role)), true
		}
	}
	return "", false
}

func resourceKeysForRequest(scope AccessPolicyScope, method string, path string, routeTag string) []string {
	switch scope {
	case AccessPolicyScopeWeb:
		return webResourceKeys(path)
	case AccessPolicyScopeAPI:
		return apiResourceKeys(method, path, routeTag)
	default:
		return nil
	}
}

func webResourceKeys(path string) []string {
	var keys []string
	switch path {
	case "/":
		keys = appendResourceKey(keys, AccessResourceWeb)
		keys = appendResourceKey(keys, AccessResourceHome)
	case "/pricing", "/rankings", "/about", "/user-agreement", "/privacy-policy":
		keys = appendResourceKey(keys, AccessResourceWeb)
	case "/setup", "/login", "/register", "/reset", "/user/reset", "/favicon.ico":
		return keys
	case "/console":
		keys = appendResourceKey(keys, AccessResourceDashboard)
	case "/console/token":
		keys = appendResourceKey(keys, AccessResourceToken)
	case "/console/topup":
		keys = appendResourceKey(keys, AccessResourceWallet)
	case "/console/billing":
		keys = appendResourceKey(keys, AccessResourceBilling)
	case "/console/log":
		keys = appendResourceKey(keys, AccessResourceUsageLog)
	case "/console/playground":
		keys = appendResourceKey(keys, AccessResourcePlayground)
	case "/console/personal":
		keys = appendResourceKey(keys, AccessResourcePersonal)
	case "/console/midjourney":
		keys = appendResourceKey(keys, AccessResourceDrawingLog)
	case "/console/task":
		keys = appendResourceKey(keys, AccessResourceTaskLog)
	case "/console/channel":
		keys = appendResourceKey(keys, AccessResourceAdminChannel)
	case "/console/subscription":
		keys = appendResourceKey(keys, AccessResourceAdminSubscription)
	case "/console/models":
		keys = appendResourceKey(keys, AccessResourceAdminModel)
	case "/console/redemption":
		keys = appendResourceKey(keys, AccessResourceAdminRedemption)
	case "/console/user":
		keys = appendResourceKey(keys, AccessResourceAdminUser)
	case "/console/referral":
		keys = appendResourceKey(keys, AccessResourceAdminReferral)
	case "/console/setting":
		keys = appendResourceKey(keys, AccessResourceAdminSetting)
	default:
		if strings.HasPrefix(path, "/console/chat") || path == "/chat2link" {
			keys = appendResourceKey(keys, AccessResourceChat)
		}
		if len(keys) == 0 && isWebSPAFallbackPath(path) {
			keys = appendResourceKey(keys, AccessResourceWeb)
			keys = appendResourceKey(keys, AccessResourceHome)
		}
	}
	return keys
}

func isWebSPAFallbackPath(path string) bool {
	if path == "" || path == "/" {
		return true
	}
	switch path {
	case "/setup", "/login", "/register", "/reset", "/user/reset", "/forbidden", "/favicon.ico":
		return false
	}
	if strings.HasPrefix(path, "/api") ||
		strings.HasPrefix(path, "/v1") ||
		strings.HasPrefix(path, "/v1beta") ||
		strings.HasPrefix(path, "/mj") ||
		strings.HasPrefix(path, "/suno") ||
		strings.HasPrefix(path, "/kling") ||
		strings.HasPrefix(path, "/jimeng") ||
		strings.HasPrefix(path, "/pg") ||
		strings.HasPrefix(path, "/assets") ||
		strings.HasPrefix(path, "/static") ||
		strings.HasPrefix(path, "/oauth") ||
		strings.Contains(path, ".") {
		return false
	}
	return !strings.HasPrefix(path, "/console")
}

func apiResourceKeys(method string, path string, routeTag string) []string {
	var keys []string

	if isTokenAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceToken)
	}
	if isWalletAPIPath(method, path) {
		keys = appendResourceKey(keys, AccessResourceWallet)
	}
	if isBillingAPIPath(method, path, routeTag) {
		keys = appendResourceKey(keys, AccessResourceBilling)
	}
	if isUsageLogAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceUsageLog)
	}
	if isPlaygroundAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourcePlayground)
	}
	if isPersonalAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourcePersonal)
	}
	if isDrawingLogAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceDrawingLog)
	}
	if isTaskLogAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceTaskLog)
	}
	if isAdminChannelAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceAdminChannel)
	}
	if isAdminSubscriptionAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceAdminSubscription)
	}
	if isAdminModelAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceAdminModel)
	}
	if isAdminRedemptionAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceAdminRedemption)
	}
	if isAdminReferralAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceAdminReferral)
	}
	if isAdminUserAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceAdminUser)
	}
	if isAdminSettingAPIPath(path) {
		keys = appendResourceKey(keys, AccessResourceAdminSetting)
	}
	if isModelAPIPath(path, routeTag) {
		keys = appendResourceKey(keys, AccessResourceModelAPI)
	}

	return keys
}

func appendResourceKey(keys []string, key string) []string {
	for _, existing := range keys {
		if existing == key {
			return keys
		}
	}
	return append(keys, key)
}

func resourceAllowedForRole(setting *access_setting.AccessControlSetting, resource string, role int) bool {
	if setting == nil || setting.ResourceRules == nil {
		return true
	}
	rule, ok := setting.ResourceRules[resource]
	if !ok {
		return true
	}

	var value *bool
	switch roleAccessLevel(role) {
	case "root":
		value = rule.Root
	case "admin":
		value = rule.Admin
	case "audit_admin":
		value = rule.AuditAdmin
	case "user":
		value = rule.User
	default:
		value = rule.Guest
	}
	if value == nil {
		return true
	}
	return *value
}

func roleAccessLevel(role int) string {
	switch {
	case role >= common.RoleRootUser:
		return "root"
	case role >= common.RoleAdminUser:
		return "admin"
	case role >= common.RoleAuditAdminUser:
		return "audit_admin"
	case role >= common.RoleCommonUser:
		return "user"
	default:
		return "guest"
	}
}

func isExactOrChildPath(path string, base string) bool {
	return path == base || strings.HasPrefix(path, base+"/")
}

func isTokenAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/token")
}

func isWalletAPIPath(method string, path string) bool {
	if path == "/api/user/topup" {
		return method != http.MethodGet
	}
	if path == "/api/user/topup/info" {
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
	if strings.HasPrefix(path, "/api/subscription/") {
		if strings.HasPrefix(path, "/api/subscription/admin") {
			return false
		}
		if path == "/api/subscription/plans" ||
			path == "/api/subscription/self" ||
			strings.HasPrefix(path, "/api/subscription/self/") ||
			strings.HasSuffix(path, "/pay") {
			return true
		}
	}
	switch path {
	case "/api/user/pay",
		"/api/user/amount",
		"/api/user/aff",
		"/api/user/aff/commissions",
		"/api/user/aff_transfer":
		return true
	default:
		return false
	}
}

func isBillingAPIPath(method string, path string, routeTag string) bool {
	if routeTag == "old_api" {
		return true
	}
	if path == "/api/user/topup" {
		return method == http.MethodGet
	}
	if strings.HasPrefix(path, "/api/user/topup/") {
		if path == "/api/user/topup/info" {
			return false
		}
		return true
	}
	if isExactOrChildPath(path, "/dashboard/billing") ||
		isExactOrChildPath(path, "/v1/dashboard/billing") {
		return true
	}
	return false
}

func isUsageLogAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/log") ||
		isExactOrChildPath(path, "/api/data") ||
		isExactOrChildPath(path, "/api/usage")
}

func isPlaygroundAPIPath(path string) bool {
	return isExactOrChildPath(path, "/pg")
}

func isPersonalAPIPath(path string) bool {
	if strings.HasPrefix(path, "/api/user/passkey") ||
		strings.HasPrefix(path, "/api/user/2fa") ||
		strings.HasPrefix(path, "/api/user/oauth/bindings") ||
		strings.HasPrefix(path, "/api/user/checkin") {
		return true
	}
	switch path {
	case "/api/user/setting":
		return true
	default:
		return false
	}
}

func isDrawingLogAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/mj")
}

func isTaskLogAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/task")
}

func isAdminChannelAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/channel") ||
		isExactOrChildPath(path, "/api/group") ||
		isExactOrChildPath(path, "/api/prefill_group") ||
		isExactOrChildPath(path, "/api/vendors")
}

func isAdminSubscriptionAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/subscription/admin")
}

func isAdminModelAPIPath(path string) bool {
	if path == "/api/models" {
		return false
	}
	return isExactOrChildPath(path, "/api/models")
}

func isAdminRedemptionAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/redemption")
}

func isAdminReferralAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/user/referrals")
}

func isAdminUserAPIPath(path string) bool {
	if path == "/api/user" || path == "/api/user/" {
		return true
	}
	if isAdminReferralAPIPath(path) {
		return false
	}
	if strings.HasPrefix(path, "/api/user/topup") ||
		strings.HasPrefix(path, "/api/user/stripe") ||
		strings.HasPrefix(path, "/api/user/creem") ||
		strings.HasPrefix(path, "/api/user/waffo") ||
		strings.HasPrefix(path, "/api/user/alipay") ||
		strings.HasPrefix(path, "/api/user/wechat-pay") ||
		strings.HasPrefix(path, "/api/user/self-serve") ||
		strings.HasPrefix(path, "/api/user/passkey") ||
		strings.HasPrefix(path, "/api/user/2fa") ||
		strings.HasPrefix(path, "/api/user/oauth") ||
		strings.HasPrefix(path, "/api/user/checkin") {
		return false
	}
	if path == "/api/user/search" {
		return true
	}
	trimmed := strings.TrimPrefix(path, "/api/user/")
	if trimmed == path || trimmed == "" {
		return false
	}
	firstSegment := strings.Split(trimmed, "/")[0]
	_, err := strconv.Atoi(firstSegment)
	return err == nil
}

func isAdminSettingAPIPath(path string) bool {
	return isExactOrChildPath(path, "/api/option") ||
		isExactOrChildPath(path, "/api/custom-oauth-provider") ||
		isExactOrChildPath(path, "/api/performance") ||
		isExactOrChildPath(path, "/api/ratio_sync")
}

func isModelAPIPath(path string, routeTag string) bool {
	if routeTag == "relay" {
		return true
	}
	if routeTag == "old_api" {
		return false
	}
	if isExactOrChildPath(path, "/v1") ||
		isExactOrChildPath(path, "/v1beta") ||
		isExactOrChildPath(path, "/mj") ||
		isExactOrChildPath(path, "/suno") ||
		isExactOrChildPath(path, "/kling/v1") ||
		isExactOrChildPath(path, "/jimeng") ||
		isExactOrChildPath(path, "/pg") {
		return true
	}
	segments := strings.Split(strings.Trim(path, "/"), "/")
	return len(segments) >= 2 && segments[1] == "mj"
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

func AccessDeniedInfo(c *gin.Context) AccessDeniedRequestInfo {
	country := requestCountry(c)
	return AccessDeniedRequestInfo{
		IP:            requestIP(c),
		CountryCode:   country.CountryCode,
		CountryLabel:  countryDisplayLabel(country),
		CountryKnown:  country.Known,
		CountrySource: country.Source,
	}
}

func IsChinaMainlandRequest(c *gin.Context) bool {
	return requestFromChinaMainland(c)
}

func CurrentRequestRole(c *gin.Context) (int, bool) {
	return currentRequestRole(c)
}

func ResourceAccessForRole(role int) map[string]bool {
	setting := access_setting.GetAccessControlSetting()
	access := make(map[string]bool, len(accessResourceKeys))
	for _, key := range accessResourceKeys {
		access[key] = resourceAllowedForRole(setting, key, role)
	}
	return access
}

func AccessResourceKeys() []string {
	keys := make([]string, len(accessResourceKeys))
	copy(keys, accessResourceKeys)
	return keys
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

func requestIP(c *gin.Context) string {
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		return "未知"
	}
	return ip
}

func countryDisplayLabel(country access_setting.CountryLookupResult) string {
	if !country.Known {
		return "未知"
	}
	code := access_setting.NormalizeCountryCode(country.CountryCode)
	if code == "" {
		return "未知"
	}
	if access_setting.IsChinaMainlandCountryCode(code) {
		return "中国大陆"
	}
	if access_setting.IsEuropeanUnionCountryCode(code) {
		return "欧盟地区（" + code + "）"
	}
	return code
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

	if scope == AccessPolicyScopeWeb {
		abortWithAccessDeniedPage(c, message)
		return
	}

	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": message,
	})
	c.Abort()
}

func abortWithAccessDeniedPage(c *gin.Context, message string) {
	info := AccessDeniedInfo(c)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Robots-Tag", "noindex, nofollow")
	c.Data(http.StatusForbidden, "text/html; charset=utf-8", []byte(renderAccessDeniedPage(message, info)))
	c.Abort()
}

func renderAccessDeniedPage(message string, info AccessDeniedRequestInfo) string {
	title := html.EscapeString(message)
	ip := html.EscapeString(info.IP)
	countryLabel := html.EscapeString(info.CountryLabel)

	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="robots" content="noindex,nofollow">
  <title>` + title + `</title>
  <style>
    :root {
      color-scheme: light dark;
      --page-bg: #f7f8fa;
      --panel-bg: #ffffff;
      --text: #1f2329;
      --muted: #646a73;
      --border: #dfe3e8;
      --accent: #d92d20;
      --accent-bg: #fff1f0;
      --shadow: 0 18px 60px rgba(31, 35, 41, 0.12);
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --page-bg: #111318;
        --panel-bg: #191c22;
        --text: #f2f3f5;
        --muted: #a7adb8;
        --border: #343841;
        --accent: #ff7875;
        --accent-bg: rgba(255, 120, 117, 0.12);
        --shadow: 0 18px 60px rgba(0, 0, 0, 0.3);
      }
    }
    * {
      box-sizing: border-box;
    }
    body {
      min-height: 100vh;
      margin: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
      color: var(--text);
      background: var(--page-bg);
      font-family: Lato, "Helvetica Neue", Arial, "Microsoft YaHei", sans-serif;
    }
    main {
      width: min(100%, 560px);
      padding: 32px;
      background: var(--panel-bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      box-shadow: var(--shadow);
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      min-height: 32px;
      padding: 4px 12px;
      color: var(--accent);
      background: var(--accent-bg);
      border-radius: 999px;
      font-size: 14px;
      font-weight: 600;
    }
    .badge::before {
      content: "!";
      display: inline-flex;
      width: 18px;
      height: 18px;
      align-items: center;
      justify-content: center;
      color: #ffffff;
      background: var(--accent);
      border-radius: 50%;
      font-size: 12px;
      line-height: 1;
    }
    h1 {
      margin: 22px 0 12px;
      font-size: 28px;
      line-height: 1.25;
      letter-spacing: 0;
    }
    p {
      margin: 0;
      color: var(--muted);
      font-size: 16px;
      line-height: 1.7;
    }
    dl {
      display: grid;
      grid-template-columns: max-content minmax(0, 1fr);
      gap: 12px 18px;
      margin: 28px 0 0;
      padding-top: 22px;
      border-top: 1px solid var(--border);
    }
    dt {
      color: var(--muted);
      font-size: 14px;
    }
    dd {
      min-width: 0;
      margin: 0;
      overflow-wrap: anywhere;
      color: var(--text);
      font-size: 14px;
      font-weight: 600;
    }
    @media (max-width: 520px) {
      body {
        padding: 16px;
      }
      main {
        padding: 24px;
      }
      h1 {
        font-size: 24px;
      }
      dl {
        grid-template-columns: 1fr;
        gap: 6px;
      }
      dd + dt {
        margin-top: 10px;
      }
    }
  </style>
  <script>
    if (window.location.pathname !== "/forbidden" || window.location.search !== "?access_limited=1") {
      window.history.replaceState(null, "", "/forbidden?access_limited=1");
    }
  </script>
</head>
<body>
  <main>
    <div class="badge">` + title + `</div>
    <h1>本站不对您所在的地区开放。</h1>
    <p>访问策略已根据当前网络来源拦截本次请求。</p>
    <dl>
      <dt>您当前 IP：</dt>
      <dd>` + ip + `</dd>
      <dt>IP 归属地：</dt>
      <dd>` + countryLabel + `</dd>
    </dl>
  </main>
</body>
</html>`
}
