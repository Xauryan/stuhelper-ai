package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/types"

	"github.com/gin-gonic/gin"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type Log struct {
	Id                int    `json:"id" gorm:"index:idx_created_at_id,priority:2;index:idx_user_id_id,priority:2"`
	UserId            int    `json:"user_id" gorm:"index;index:idx_user_id_id,priority:1"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;index:idx_created_at_id,priority:1;index:idx_created_at_type"`
	Type              int    `json:"type" gorm:"index:idx_created_at_type"`
	Content           string `json:"content"`
	Username          string `json:"username" gorm:"index;index:index_username_model_name,priority:2;default:''"`
	TokenName         string `json:"token_name" gorm:"index;default:''"`
	ModelName         string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota             int    `json:"quota" gorm:"default:0"`
	PromptTokens      int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens  int    `json:"completion_tokens" gorm:"default:0"`
	UseTime           int    `json:"use_time" gorm:"default:0"`
	IsStream          bool   `json:"is_stream"`
	ChannelId         int    `json:"channel" gorm:"index"`
	ChannelName       string `json:"channel_name" gorm:"->"`
	TokenId           int    `json:"token_id" gorm:"default:0;index"`
	Group             string `json:"group" gorm:"index"`
	Ip                string `json:"ip" gorm:"index;default:''"`
	RequestId         string `json:"request_id,omitempty" gorm:"type:varchar(64);index:idx_logs_request_id;default:''"`
	UpstreamRequestId string `json:"upstream_request_id,omitempty" gorm:"type:varchar(128);index:idx_logs_upstream_request_id;default:''"`
	Other             string `json:"other"`
}

// don't use iota, avoid change log type value
const (
	LogTypeUnknown = 0
	LogTypeTopup   = 1
	LogTypeConsume = 2
	LogTypeManage  = 3
	LogTypeSystem  = 4
	LogTypeError   = 5
	LogTypeRefund  = 6
	LogTypeLogin   = 7
)

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		var otherMap map[string]interface{}
		otherMap, _ = common.StrToMap(logs[i].Other)
		if otherMap != nil {
			// Remove admin-only debug fields.
			delete(otherMap, "admin_info")
			delete(otherMap, "audit_info")
			// delete(otherMap, "reject_reason")
			delete(otherMap, "stream_status")
		}
		logs[i].Other = common.MapToJsonStr(otherMap)
		logs[i].Id = startIdx + i + 1
	}
}

func formatAuditAdminLogs(logs []*Log, startIdx int, channelNames map[int]string) {
	for i := range logs {
		logs[i].ChannelName = ""
		sanitizeAuditAdminChannelText(logs[i], channelNames)
		sanitizeLogOtherForAuditAdmin(logs[i])
	}
}

func sanitizeAuditAdminChannelText(log *Log, channelNames map[int]string) {
	if log == nil {
		return
	}
	if log.ChannelId > 0 {
		hints := auditAdminLogChannelNameHints(log)
		if channelNames != nil {
			hints = append(hints, channelNames[log.ChannelId])
		}
		log.Content = replaceChannelIdentifierHints(log.Content, log.ChannelId, hints...)
	}
	otherMap, err := common.StrToMap(log.Other)
	if err != nil || otherMap == nil {
		return
	}
	action, _ := opActionFromOther(otherMap).(string)
	if strings.HasPrefix(action, "channel.") {
		log.Content = auditAdminChannelContent(action, otherMap, log.ChannelId)
	}
}

func auditAdminLogChannelNameHints(log *Log) []string {
	if log == nil || strings.TrimSpace(log.Other) == "" {
		return nil
	}
	otherMap, err := common.StrToMap(log.Other)
	if err != nil || otherMap == nil {
		return nil
	}
	hints := make([]string, 0, 2)
	if channelName, ok := otherMap["channel_name"].(string); ok {
		hints = append(hints, channelName)
	}
	if adminInfo, ok := otherMap["admin_info"].(map[string]interface{}); ok {
		if channelName, ok := adminInfo["channel_name"].(string); ok {
			hints = append(hints, channelName)
		}
	}
	if op, ok := otherMap["op"].(map[string]interface{}); ok {
		if params, ok := op["params"].(map[string]interface{}); ok {
			if name, ok := params["name"].(string); ok {
				hints = append(hints, name)
			}
		}
	}
	return hints
}

func auditAdminChannelContent(action string, otherMap map[string]interface{}, fallbackChannelID int) string {
	params := map[string]interface{}{}
	if op, ok := otherMap["op"].(map[string]interface{}); ok {
		if rawParams, ok := op["params"].(map[string]interface{}); ok {
			params = rawParams
		}
	}
	channelID := auditAdminIntParam(params, "id", fallbackChannelID)
	sourceID := auditAdminIntParam(params, "sourceId", 0)
	count := auditAdminIntParam(params, "count", 0)
	switch action {
	case "channel.create":
		if count > 0 {
			return fmt.Sprintf("Created %d channel(s)", count)
		}
	case "channel.update":
		return fmt.Sprintf("Updated channel %s", auditAdminChannelLabel(channelID))
	case "channel.delete":
		return fmt.Sprintf("Deleted channel %s", auditAdminChannelLabel(channelID))
	case "channel.key_view":
		return fmt.Sprintf("Viewed channel key %s", auditAdminChannelLabel(channelID))
	case "channel.copy":
		return fmt.Sprintf("Copied channel %s to %s", auditAdminChannelLabel(sourceID), auditAdminChannelLabel(channelID))
	case "channel.delete_batch":
		if count > 0 {
			return fmt.Sprintf("Batch deleted %d channel(s)", count)
		}
	case "channel.delete_disabled":
		if count > 0 {
			return fmt.Sprintf("Deleted %d disabled channel(s)", count)
		}
	case "channel.tag_disable":
		return "Disabled channels by tag"
	case "channel.tag_enable":
		return "Enabled channels by tag"
	case "channel.tag_edit":
		return "Edited channels by tag"
	case "channel.tag_batch_set":
		if count > 0 {
			return fmt.Sprintf("Batch set tag for %d channel(s)", count)
		}
		return "Batch set channel tag"
	case "channel.multi_key_manage":
		return fmt.Sprintf("Managed multi-key channel %s", auditAdminChannelLabel(channelID))
	case "channel.upstream_apply":
		return fmt.Sprintf("Applied upstream model changes to channel %s", auditAdminChannelLabel(channelID))
	case "channel.upstream_apply_all":
		if count > 0 {
			return fmt.Sprintf("Applied upstream model changes to %d channel(s)", count)
		}
	}
	if channelID > 0 {
		return fmt.Sprintf("%s %s", action, auditAdminChannelLabel(channelID))
	}
	return action
}

func auditAdminChannelLabel(channelID int) string {
	if channelID <= 0 {
		return "#-"
	}
	return fmt.Sprintf("#%d", channelID)
}

func auditAdminIntParam(params map[string]interface{}, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	raw, ok := params[key]
	if !ok || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func replaceChannelIdentifierHints(value string, channelID int, hints ...string) string {
	if value == "" || channelID <= 0 {
		return value
	}
	replacement := auditAdminChannelLabel(channelID)
	for _, hint := range hints {
		hint = strings.TrimSpace(hint)
		if hint == "" || hint == replacement {
			continue
		}
		value = strings.ReplaceAll(value, hint, replacement)
	}
	return value
}

func sanitizeLogOtherForAuditAdmin(log *Log) {
	if log == nil || strings.TrimSpace(log.Other) == "" {
		return
	}
	otherMap, err := common.StrToMap(log.Other)
	if err != nil || otherMap == nil {
		return
	}
	removeAuditAdminChannelFields(otherMap)
	if adminInfo, ok := otherMap["admin_info"].(map[string]interface{}); ok {
		removeAuditAdminChannelFields(adminInfo)
	}
	if op, ok := otherMap["op"].(map[string]interface{}); ok {
		if params, ok := op["params"].(map[string]interface{}); ok {
			sanitizeAuditAdminChannelParams(op["action"], params)
			if len(params) == 0 {
				delete(op, "params")
			}
		}
	}
	if auditInfo, ok := otherMap["audit_info"].(map[string]interface{}); ok {
		if params, ok := auditInfo["params"].(map[string]interface{}); ok {
			sanitizeAuditAdminChannelParams(opActionFromOther(otherMap), params)
			if len(params) == 0 {
				delete(auditInfo, "params")
			}
		}
	}
	log.Other = common.MapToJsonStr(otherMap)
}

func opActionFromOther(otherMap map[string]interface{}) interface{} {
	if op, ok := otherMap["op"].(map[string]interface{}); ok {
		return op["action"]
	}
	return nil
}

func removeAuditAdminChannelFields(values map[string]interface{}) {
	for _, key := range []string{
		"channel_name",
		"channel_affinity",
	} {
		delete(values, key)
	}
}

func sanitizeAuditAdminChannelParams(actionValue interface{}, params map[string]interface{}) {
	action, _ := actionValue.(string)
	if !strings.HasPrefix(action, "channel.") {
		return
	}
	allowed := map[string]struct{}{
		"id":             {},
		"sourceId":       {},
		"count":          {},
		"changed_fields": {},
		"action":         {},
		"status":         {},
		"success":        {},
	}
	for key := range params {
		if _, ok := allowed[key]; !ok {
			delete(params, key)
		}
	}
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order("id desc").Limit(common.MaxRecentItems).Find(&logs).Error
	formatUserLogs(logs, 0)
	return logs, err
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

// RecordLogWithAdminInfo 记录操作日志，并将管理员相关信息存入 Other.admin_info，
func RecordLogWithAdminInfo(userId int, logType int, content string, adminInfo map[string]interface{}, quota ...int) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	if len(quota) > 0 {
		log.Quota = quota[0]
	}
	if len(adminInfo) > 0 {
		other := map[string]interface{}{
			"admin_info": adminInfo,
		}
		log.Other = common.MapToJsonStr(other)
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

func buildOpField(action string, params map[string]interface{}) map[string]interface{} {
	op := map[string]interface{}{
		"action": action,
	}
	if len(params) > 0 {
		op["params"] = params
	}
	return op
}

func RecordLoginLog(userId int, username string, content string, ip string, action string, params map[string]interface{}, extra map[string]interface{}) {
	other := map[string]interface{}{}
	for k, v := range extra {
		other[k] = v
	}
	other["op"] = buildOpField(action, params)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeLogin,
		Content:   content,
		Ip:        ip,
		Other:     common.MapToJsonStr(other),
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record login log: " + err.Error())
	}
}

func RecordOperationAuditLog(logUserId int, content string, ip string, action string, params map[string]interface{}, adminInfo map[string]interface{}, auditInfo map[string]interface{}) {
	username, _ := GetUsernameById(logUserId, false)
	other := map[string]interface{}{
		"op": buildOpField(action, params),
	}
	if len(adminInfo) > 0 {
		other["admin_info"] = adminInfo
	}
	if len(auditInfo) > 0 {
		other["audit_info"] = auditInfo
	}
	log := &Log{
		UserId:    logUserId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeManage,
		Content:   content,
		Ip:        ip,
		Other:     common.MapToJsonStr(other),
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record operation audit log: " + err.Error())
	}
}

func RecordTopupLog(userId int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	recordPaymentAuditLog(userId, LogTypeTopup, content, callerIp, paymentMethod, callbackPaymentMethod)
}

func RecordTopupRefundLog(userId int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	recordPaymentAuditLog(userId, LogTypeRefund, content, callerIp, paymentMethod, callbackPaymentMethod)
}

func RecordOfficialPaymentRefundLog(userId int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	RecordTopupRefundLog(userId, content, callerIp, paymentMethod, callbackPaymentMethod)
}

func recordPaymentAuditLog(userId int, logType int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	username, _ := GetUsernameById(userId, false)
	other := map[string]interface{}{
		"admin_info": map[string]interface{}{
			"server_ip":               common.GetIp(),
			"node_name":               common.NodeName,
			"caller_ip":               callerIp,
			"payment_method":          paymentMethod,
			"callback_payment_method": callbackPaymentMethod,
			"version":                 common.Version,
		},
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
		Ip:        callerIp,
		Other:     common.MapToJsonStr(other),
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record payment audit log: " + err.Error())
	}
}

func RecordErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int,
	isStream bool, group string, other map[string]interface{}) {
	logger.LogInfo(c, fmt.Sprintf("record error log: userId=%d, channelId=%d, modelName=%s, tokenName=%s, content=%s", userId, channelId, modelName, tokenName, common.LocalLogPreview(content)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	upstreamRequestId := c.GetString(common.UpstreamRequestIdKey)
	otherStr := common.MapToJsonStr(other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeError,
		Content:          content,
		PromptTokens:     0,
		CompletionTokens: 0,
		TokenName:        tokenName,
		ModelName:        modelName,
		Quota:            0,
		ChannelId:        channelId,
		TokenId:          tokenId,
		UseTime:          useTimeSeconds,
		IsStream:         isStream,
		Group:            group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId:         requestId,
		UpstreamRequestId: upstreamRequestId,
		Other:             otherStr,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
}

type RecordConsumeLogParams struct {
	ChannelId        int                    `json:"channel_id"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	ModelName        string                 `json:"model_name"`
	TokenName        string                 `json:"token_name"`
	Quota            int                    `json:"quota"`
	Content          string                 `json:"content"`
	TokenId          int                    `json:"token_id"`
	UseTimeSeconds   int                    `json:"use_time_seconds"`
	IsStream         bool                   `json:"is_stream"`
	Group            string                 `json:"group"`
	Other            map[string]interface{} `json:"other"`
}

func RecordConsumeLog(c *gin.Context, userId int, params RecordConsumeLogParams) {
	if !common.LogConsumeEnabled {
		return
	}
	logger.LogInfo(c, fmt.Sprintf("record consume log: userId=%d, params=%s", userId, common.GetJsonString(params)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	upstreamRequestId := c.GetString(common.UpstreamRequestIdKey)
	otherStr := common.MapToJsonStr(params.Other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeConsume,
		Content:          params.Content,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		TokenName:        params.TokenName,
		ModelName:        params.ModelName,
		Quota:            params.Quota,
		ChannelId:        params.ChannelId,
		TokenId:          params.TokenId,
		UseTime:          params.UseTimeSeconds,
		IsStream:         params.IsStream,
		Group:            params.Group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId:         requestId,
		UpstreamRequestId: upstreamRequestId,
		Other:             otherStr,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
	if common.DataExportEnabled {
		gopool.Go(func() {
			LogQuotaData(QuotaDataLogParams{
				UserID:    userId,
				Username:  username,
				ModelName: params.ModelName,
				Quota:     params.Quota,
				CreatedAt: common.GetTimestamp(),
				TokenUsed: params.PromptTokens + params.CompletionTokens,
				UseGroup:  params.Group,
				TokenID:   params.TokenId,
				ChannelID: params.ChannelId,
				NodeName:  common.NodeName,
			})
		})
	}
}

type RecordTaskBillingLogParams struct {
	UserId    int
	LogType   int
	Content   string
	ChannelId int
	ModelName string
	Quota     int
	TokenId   int
	Group     string
	Other     map[string]interface{}
}

func RecordTaskBillingLog(params RecordTaskBillingLogParams) {
	if params.LogType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(params.UserId, false)
	tokenName := ""
	if params.TokenId > 0 {
		if token, err := GetTokenById(params.TokenId); err == nil {
			tokenName = token.Name
		}
	}
	log := &Log{
		UserId:    params.UserId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      params.LogType,
		Content:   params.Content,
		TokenName: tokenName,
		ModelName: params.ModelName,
		Quota:     params.Quota,
		ChannelId: params.ChannelId,
		TokenId:   params.TokenId,
		Group:     params.Group,
		Other:     common.MapToJsonStr(params.Other),
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record task billing log: " + err.Error())
	}
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string, upstreamRequestId string, includeChannelNames bool) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("logs.type = ?", logType)
	}

	if tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", modelName); err != nil {
		return nil, 0, err
	}
	if tx, err = applyExplicitLogTextFilter(tx, "logs.username", username); err != nil {
		return nil, 0, err
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if upstreamRequestId != "" {
		tx = tx.Where("logs.upstream_request_id = ?", upstreamRequestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("logs.channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = tx.Order("logs.created_at desc, logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	if includeChannelNames {
		channelIds := types.NewSet[int]()
		for _, log := range logs {
			if log.ChannelId != 0 {
				channelIds.Add(log.ChannelId)
			}
		}
		if channelIds.Len() > 0 {
			var channels []struct {
				Id   int    `gorm:"column:id"`
				Name string `gorm:"column:name"`
			}
			if common.MemoryCacheEnabled {
				// Cache get channel
				for _, channelId := range channelIds.Items() {
					if cacheChannel, err := CacheGetChannel(channelId); err == nil {
						channels = append(channels, struct {
							Id   int    `gorm:"column:id"`
							Name string `gorm:"column:name"`
						}{
							Id:   channelId,
							Name: cacheChannel.Name,
						})
					}
				}
			} else {
				// Bulk query channels from DB
				if err = DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err != nil {
					return logs, total, err
				}
			}
			channelMap := make(map[int]string, len(channels))
			for _, channel := range channels {
				channelMap[channel.Id] = channel.Name
			}
			for i := range logs {
				logs[i].ChannelName = channelMap[logs[i].ChannelId]
			}
		}
	} else {
		formatAuditAdminLogs(logs, startIdx, loadChannelNamesForLogs(logs))
	}

	return logs, total, err
}

const logSearchCountLimit = 10000

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string, upstreamRequestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("logs.user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("logs.user_id = ? and logs.type = ?", userId, logType)
	}

	if tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", modelName); err != nil {
		return nil, 0, err
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if upstreamRequestId != "" {
		tx = tx.Where("logs.upstream_request_id = ?", upstreamRequestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		common.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		common.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}

	formatUserLogs(logs, startIdx)
	return logs, total, err
}

type Stat struct {
	Quota int `json:"quota"`
	Rpm   int `json:"rpm"`
	Tpm   int `json:"tpm"`
}

func applyExplicitLogTextFilter(tx *gorm.DB, column string, value string) (*gorm.DB, error) {
	if value == "" {
		return tx, nil
	}
	if strings.Contains(value, "%") {
		pattern, err := sanitizeLikePattern(value)
		if err != nil {
			return nil, err
		}
		return tx.Where(column+" LIKE ? ESCAPE '!'", pattern), nil
	}
	return tx.Where(column+" = ?", value), nil
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	tx := LOG_DB.Table("logs").Select("sum(quota) quota")

	// 为rpm和tpm创建单独的查询
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, sum(prompt_tokens) + sum(completion_tokens) tpm")

	if tx, err = applyExplicitLogTextFilter(tx, "username", username); err != nil {
		return stat, err
	}
	if rpmTpmQuery, err = applyExplicitLogTextFilter(rpmTpmQuery, "username", username); err != nil {
		return stat, err
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
		rpmTpmQuery = rpmTpmQuery.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if tx, err = applyExplicitLogTextFilter(tx, "model_name", modelName); err != nil {
		return stat, err
	}
	if rpmTpmQuery, err = applyExplicitLogTextFilter(rpmTpmQuery, "model_name", modelName); err != nil {
		return stat, err
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
		rpmTpmQuery = rpmTpmQuery.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
		rpmTpmQuery = rpmTpmQuery.Where(logGroupCol+" = ?", group)
	}

	tx = tx.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)

	// 只统计最近60秒的rpm和tpm
	rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	// 执行查询
	if err := tx.Scan(&stat).Error; err != nil {
		common.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := rpmTpmQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}

	return stat, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	tx := LOG_DB.Table("logs").Select("ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0)")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

func DeleteOldLog(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0

	for {
		if nil != ctx.Err() {
			return total, ctx.Err()
		}

		result := LOG_DB.Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&Log{})
		if nil != result.Error {
			return total, result.Error
		}

		total += result.RowsAffected

		if result.RowsAffected < int64(limit) {
			break
		}
	}

	return total, nil
}
