package controller

import (
	"fmt"
	"os"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/model"

	"github.com/gin-gonic/gin"
)

var auditContentTemplates = map[string]string{
	"login":               "Logged in successfully via ${method}",
	"user.create":         "Created user ${username} (role ${role})",
	"user.update":         "Updated user ${username} (ID: ${id})",
	"user.delete":         "Deleted user ${username} (ID: ${id})",
	"user.manage":         "Performed ${action} on user ${username} (ID: ${id})",
	"user.quota_add":      "Increased user quota by ${quota}",
	"user.quota_subtract": "Decreased user quota by ${quota}",
	"user.quota_override": "Overrode user quota from ${from} to ${to}",
	"user.binding_clear":  "Cleared ${bindingType} binding for user ${username}",
	"user.2fa_disable":    "Force-disabled two-factor authentication for the user",

	"user.passkey_register": "Registered a passkey",
	"user.passkey_delete":   "Deleted a passkey",
	"user.reset_passkey":    "Reset the user passkey",

	"option.update": "Updated system setting ${key}",

	"channel.create":             "Created channel ${name} (type ${type}, count ${count})",
	"channel.update":             "Updated channel ${name} (ID: ${id})",
	"channel.delete":             "Deleted channel ${name} (ID: ${id})",
	"channel.delete_batch":       "Batch deleted ${count} channels",
	"channel.delete_disabled":    "Deleted all disabled channels (${count})",
	"channel.key_view":           "Viewed channel key ${name} (ID: ${id})",
	"channel.breaker_reset":      "Reset channel circuit breaker ${name} (ID: ${id}, previous state: ${previous_state})",
	"channel.tag_disable":        "Disabled channels with tag ${tag}",
	"channel.tag_enable":         "Enabled channels with tag ${tag}",
	"channel.tag_edit":           "Edited channels with tag ${tag}",
	"channel.tag_batch_set":      "Batch set tag for ${count} channels",
	"channel.copy":               "Copied channel (source ID: ${sourceId}) to ${name} (new ID: ${id})",
	"channel.multi_key_manage":   "Multi-key management ${action} on channel (ID: ${id})",
	"channel.upstream_apply":     "Applied upstream model changes to channel (ID: ${id})",
	"channel.upstream_apply_all": "Applied upstream model changes to ${count} channels",

	"redemption.create": "Created ${count} redemption codes named ${name} (${quota} each)",
}

func auditContentEN(action string, params map[string]interface{}) string {
	tmpl, ok := auditContentTemplates[action]
	if !ok {
		return action
	}
	return os.Expand(tmpl, func(key string) string {
		if v, ok := params[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	})
}

func auditAuthMethod(c *gin.Context) string {
	if c.GetBool("use_access_token") {
		return "access_token"
	}
	return "session"
}

func auditOperatorInfo(c *gin.Context) map[string]interface{} {
	return map[string]interface{}{
		"admin_id":       c.GetInt("id"),
		"admin_username": c.GetString("username"),
		"admin_role":     c.GetInt("role"),
		"auth_method":    auditAuthMethod(c),
	}
}

func markAuditLogged(c *gin.Context) {
	common.SetContextKey(c, constant.ContextKeyAuditLogged, true)
}

func recordManageAudit(c *gin.Context, action string, params map[string]interface{}) {
	recordManageAuditFor(c, c.GetInt("id"), action, params)
}

func recordManageAuditFor(c *gin.Context, logUserId int, action string, params map[string]interface{}) {
	model.RecordOperationAuditLog(logUserId, auditContentEN(action, params), c.ClientIP(), action, params, auditOperatorInfo(c), nil)
	markAuditLogged(c)
}

func recordUserSecurityAudit(c *gin.Context, userId int, action string, params map[string]interface{}) {
	model.RecordOperationAuditLog(userId, auditContentEN(action, params), c.ClientIP(), action, params, nil, nil)
}
