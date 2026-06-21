package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
)

const (
	ChannelMonitorSourceAll   = "all"
	ChannelMonitorSourceLog   = "log"
	ChannelMonitorSourceProbe = "probe"

	DefaultChannelMonitorWindowSeconds = 7 * 24 * 60 * 60
	DefaultChannelMonitorErrorLimit    = 20
	MaxChannelMonitorErrorLimit        = 100
)

type ChannelMonitorStatsParams struct {
	WindowSeconds int64
	Source        string
	ChannelID     int
	ModelName     string
	Group         string
	ErrorLimit    int
	IncludeNames  bool
}

type ChannelMonitorStatsBucket struct {
	Total             int64   `json:"total"`
	Success           int64   `json:"success"`
	Failures          int64   `json:"failures"`
	ChannelFailures   int64   `json:"channel_failures"`
	TransientFailures int64   `json:"transient_failures"`
	Ignored           int64   `json:"ignored"`
	SuccessRate       float64 `json:"success_rate"`
	SLA               float64 `json:"sla"`
	AvgUseTimeSeconds float64 `json:"avg_use_time_seconds"`
	LastSuccessAt     int64   `json:"last_success_at"`
	LastFailureAt     int64   `json:"last_failure_at"`
	LastError         string  `json:"last_error,omitempty"`
}

type ChannelMonitorErrorItem struct {
	ID                int    `json:"id"`
	CreatedAt         int64  `json:"created_at"`
	Source            string `json:"source"`
	ProbeStatus       string `json:"probe_status,omitempty"`
	ChannelID         int    `json:"channel_id"`
	ChannelName       string `json:"channel_name,omitempty"`
	ModelName         string `json:"model_name,omitempty"`
	Group             string `json:"group,omitempty"`
	StatusCode        int    `json:"status_code,omitempty"`
	ErrorCode         string `json:"error_code,omitempty"`
	ErrorType         string `json:"error_type,omitempty"`
	Message           string `json:"message"`
	RequestPath       string `json:"request_path,omitempty"`
	RequestID         string `json:"request_id,omitempty"`
	UpstreamRequestID string `json:"upstream_request_id,omitempty"`
	IsStream          bool   `json:"is_stream"`
	UseTimeSeconds    int    `json:"use_time_seconds"`
	Ignored           bool   `json:"ignored"`
}

type ChannelMonitorStatsResponse struct {
	WindowSeconds int64                     `json:"window_seconds"`
	GeneratedAt   int64                     `json:"generated_at"`
	Log           ChannelMonitorStatsBucket `json:"log"`
	Probe         ChannelMonitorStatsBucket `json:"probe"`
	Combined      ChannelMonitorStatsBucket `json:"combined"`
	Errors        []ChannelMonitorErrorItem `json:"errors"`
}

type channelMonitorPrivacy struct {
	IncludeNames bool
	ChannelNames map[int]string
}

type channelMonitorClass int

const (
	channelMonitorClassSuccess channelMonitorClass = iota
	channelMonitorClassChannelFailure
	channelMonitorClassTransientFailure
	channelMonitorClassIgnored
)

func normalizeChannelMonitorStatsParams(params ChannelMonitorStatsParams) ChannelMonitorStatsParams {
	if params.WindowSeconds <= 0 {
		params.WindowSeconds = DefaultChannelMonitorWindowSeconds
	}
	params.Source = strings.ToLower(strings.TrimSpace(params.Source))
	switch params.Source {
	case ChannelMonitorSourceLog, ChannelMonitorSourceProbe:
	default:
		params.Source = ChannelMonitorSourceAll
	}
	params.ModelName = strings.TrimSpace(params.ModelName)
	params.Group = strings.TrimSpace(params.Group)
	if params.ErrorLimit <= 0 {
		params.ErrorLimit = DefaultChannelMonitorErrorLimit
	}
	if params.ErrorLimit > MaxChannelMonitorErrorLimit {
		params.ErrorLimit = MaxChannelMonitorErrorLimit
	}
	return params
}

func GetChannelMonitorStats(params ChannelMonitorStatsParams) (*ChannelMonitorStatsResponse, error) {
	params = normalizeChannelMonitorStatsParams(params)
	if LOG_DB == nil {
		return nil, fmt.Errorf("log database is not initialized")
	}

	since := common.GetTimestamp() - params.WindowSeconds
	tx := LOG_DB.Model(&Log{}).
		Where("created_at >= ?", since).
		Where("type IN ?", []int{LogTypeConsume, LogTypeError})
	if params.ChannelID > 0 {
		tx = tx.Where("channel_id = ?", params.ChannelID)
	}
	if params.ModelName != "" {
		var err error
		tx, err = applyExplicitLogTextFilter(tx, "model_name", params.ModelName)
		if err != nil {
			return nil, err
		}
	}
	if params.Group != "" {
		tx = tx.Where(logGroupCol+" = ?", params.Group)
	}

	var logs []*Log
	if err := tx.
		Select("id, created_at, type, content, token_name, model_name, use_time, is_stream, channel_id, " + logGroupCol + ", request_id, upstream_request_id, other").
		Order("created_at desc, id desc").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	channelNames := loadChannelNamesForLogs(logs)
	privacy := channelMonitorPrivacy{
		IncludeNames: params.IncludeNames,
		ChannelNames: channelNames,
	}
	result := &ChannelMonitorStatsResponse{
		WindowSeconds: params.WindowSeconds,
		GeneratedAt:   common.GetTimestamp(),
		Errors:        make([]ChannelMonitorErrorItem, 0, params.ErrorLimit),
	}

	for _, log := range logs {
		if log == nil {
			continue
		}
		other := parseLogOtherMap(log.Other)
		source := channelMonitorSourceForLog(log, other)
		class := classifyChannelMonitorLog(log, other)
		addChannelMonitorSample(&result.Combined, log, other, class, privacy)
		if source == ChannelMonitorSourceProbe {
			addChannelMonitorSample(&result.Probe, log, other, class, privacy)
		} else {
			addChannelMonitorSample(&result.Log, log, other, class, privacy)
		}

		if log.Type == LogTypeError && shouldIncludeChannelMonitorError(params.Source, source) && len(result.Errors) < params.ErrorLimit {
			item := buildChannelMonitorErrorItem(log, other, source, privacy)
			item.Ignored = class == channelMonitorClassIgnored
			result.Errors = append(result.Errors, item)
		}
	}

	finalizeChannelMonitorBucket(&result.Log)
	finalizeChannelMonitorBucket(&result.Probe)
	finalizeChannelMonitorBucket(&result.Combined)
	return result, nil
}

func parseLogOtherMap(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	other, err := common.StrToMap(raw)
	if err != nil {
		return nil
	}
	return other
}

func channelMonitorSourceForLog(log *Log, other map[string]interface{}) string {
	if isChannelMonitorProbeLog(log, other) {
		return ChannelMonitorSourceProbe
	}
	return ChannelMonitorSourceLog
}

func isChannelMonitorProbeLog(log *Log, other map[string]interface{}) bool {
	if log == nil {
		return false
	}
	if monitorSource, ok := otherString(other, "monitor_source"); ok && strings.EqualFold(monitorSource, ChannelMonitorSourceProbe) {
		return true
	}
	if probe, ok := otherBool(other, "probe"); ok && probe {
		return true
	}
	return log.TokenName == "模型测试" && log.Content == "模型测试"
}

func classifyChannelMonitorLog(log *Log, other map[string]interface{}) channelMonitorClass {
	if log == nil {
		return channelMonitorClassIgnored
	}
	if log.Type == LogTypeConsume {
		return channelMonitorClassSuccess
	}
	if log.Type != LogTypeError {
		return channelMonitorClassIgnored
	}

	probeStatus, _ := otherString(other, "probe_status")
	if strings.EqualFold(probeStatus, "local_error") {
		return channelMonitorClassIgnored
	}

	statusCode := otherInt(other, "status_code")
	errorCode, _ := otherString(other, "error_code")
	errorType, _ := otherString(other, "error_type")
	lowerCode := strings.ToLower(errorCode)
	lowerType := strings.ToLower(errorType)
	lowerMessage := strings.ToLower(log.Content)

	if isIgnoredClientMonitorFailure(statusCode, lowerCode, lowerType) {
		return channelMonitorClassIgnored
	}
	if isChannelSideMonitorFailure(statusCode, lowerCode, lowerMessage) {
		return channelMonitorClassChannelFailure
	}
	if isTransientMonitorFailure(statusCode, lowerCode, lowerMessage) {
		return channelMonitorClassTransientFailure
	}
	if statusCode >= 400 {
		return channelMonitorClassChannelFailure
	}
	return channelMonitorClassIgnored
}

func isIgnoredClientMonitorFailure(statusCode int, lowerCode string, lowerType string) bool {
	if statusCode == 400 || statusCode == 422 {
		return true
	}
	if strings.Contains(lowerType, "invalid_request") {
		return true
	}
	switch lowerCode {
	case string("invalid_request"), string("bad_request_body"), string("sensitive_words_detected"), string("prompt_blocked"):
		return true
	}
	return false
}

func isChannelSideMonitorFailure(statusCode int, lowerCode string, lowerMessage string) bool {
	if statusCode == 401 || statusCode == 403 {
		return true
	}
	if strings.HasPrefix(lowerCode, "channel:") {
		return true
	}
	for _, marker := range []string{"invalid_key", "no_available_key", "quota", "permission", "unauthorized", "forbidden", "revoked"} {
		if strings.Contains(lowerCode, marker) || strings.Contains(lowerMessage, marker) {
			return true
		}
	}
	return false
}

func isTransientMonitorFailure(statusCode int, lowerCode string, lowerMessage string) bool {
	if statusCode == 408 || statusCode == 409 || statusCode == 425 || statusCode == 429 || statusCode >= 500 {
		return true
	}
	for _, marker := range []string{"timeout", "temporar", "rate_limit", "too_many", "connection refused", "connection reset", "eof", "do_request_failed"} {
		if strings.Contains(lowerCode, marker) || strings.Contains(lowerMessage, marker) {
			return true
		}
	}
	return false
}

func addChannelMonitorSample(bucket *ChannelMonitorStatsBucket, log *Log, other map[string]interface{}, class channelMonitorClass, privacy channelMonitorPrivacy) {
	if bucket == nil || log == nil {
		return
	}
	switch class {
	case channelMonitorClassSuccess:
		bucket.Success++
		if log.CreatedAt > bucket.LastSuccessAt {
			bucket.LastSuccessAt = log.CreatedAt
		}
		if log.UseTime > 0 {
			bucket.AvgUseTimeSeconds += float64(log.UseTime)
		}
	case channelMonitorClassChannelFailure:
		bucket.ChannelFailures++
		if log.CreatedAt > bucket.LastFailureAt {
			bucket.LastFailureAt = log.CreatedAt
			bucket.LastError = common.LocalLogPreview(channelMonitorMessageForRole(log, other, privacy))
		}
	case channelMonitorClassTransientFailure:
		bucket.TransientFailures++
		if log.CreatedAt > bucket.LastFailureAt {
			bucket.LastFailureAt = log.CreatedAt
			bucket.LastError = common.LocalLogPreview(channelMonitorMessageForRole(log, other, privacy))
		}
	default:
		bucket.Ignored++
	}
}

func finalizeChannelMonitorBucket(bucket *ChannelMonitorStatsBucket) {
	if bucket == nil {
		return
	}
	bucket.Failures = bucket.ChannelFailures + bucket.TransientFailures
	bucket.Total = bucket.Success + bucket.Failures
	if bucket.Total > 0 {
		bucket.SuccessRate = float64(bucket.Success) / float64(bucket.Total)
		bucket.SLA = bucket.SuccessRate
	}
	if bucket.Success > 0 && bucket.AvgUseTimeSeconds > 0 {
		bucket.AvgUseTimeSeconds = bucket.AvgUseTimeSeconds / float64(bucket.Success)
	}
}

func shouldIncludeChannelMonitorError(sourceFilter string, source string) bool {
	switch sourceFilter {
	case ChannelMonitorSourceLog:
		return source == ChannelMonitorSourceLog
	case ChannelMonitorSourceProbe:
		return source == ChannelMonitorSourceProbe
	default:
		return true
	}
}

func buildChannelMonitorErrorItem(log *Log, other map[string]interface{}, source string, privacy channelMonitorPrivacy) ChannelMonitorErrorItem {
	item := ChannelMonitorErrorItem{
		ID:                log.Id,
		CreatedAt:         log.CreatedAt,
		Source:            source,
		ProbeStatus:       otherStringDefault(other, "probe_status"),
		ChannelID:         log.ChannelId,
		ModelName:         log.ModelName,
		Group:             log.Group,
		StatusCode:        otherInt(other, "status_code"),
		ErrorCode:         otherStringDefault(other, "error_code"),
		ErrorType:         otherStringDefault(other, "error_type"),
		Message:           channelMonitorMessageForRole(log, other, privacy),
		RequestPath:       otherStringDefault(other, "request_path"),
		RequestID:         log.RequestId,
		UpstreamRequestID: log.UpstreamRequestId,
		IsStream:          log.IsStream,
		UseTimeSeconds:    log.UseTime,
	}
	if privacy.IncludeNames && privacy.ChannelNames != nil {
		item.ChannelName = otherStringDefault(other, "channel_name")
		if item.ChannelName == "" {
			item.ChannelName = privacy.ChannelNames[log.ChannelId]
		}
	}
	if item.Message == "" {
		item.Message = otherStringDefault(other, "error_message")
		if !privacy.IncludeNames {
			item.Message = replaceChannelIdentifierHints(item.Message, log.ChannelId, channelMonitorChannelNameHints(other)...)
		}
	}
	return item
}

func channelMonitorMessageForRole(log *Log, other map[string]interface{}, privacy channelMonitorPrivacy) string {
	if log == nil {
		return ""
	}
	message := log.Content
	if message == "" {
		message = otherStringDefault(other, "error_message")
	}
	if privacy.IncludeNames {
		return message
	}
	hints := channelMonitorChannelNameHints(other)
	if privacy.ChannelNames != nil {
		hints = append(hints, privacy.ChannelNames[log.ChannelId])
	}
	return replaceChannelIdentifierHints(message, log.ChannelId, hints...)
}

func channelMonitorChannelNameHints(other map[string]interface{}) []string {
	if other == nil {
		return nil
	}
	hints := make([]string, 0, 1)
	if channelName, ok := otherString(other, "channel_name"); ok {
		hints = append(hints, channelName)
	}
	return hints
}

func loadChannelNamesForLogs(logs []*Log) map[int]string {
	ids := make(map[int]struct{})
	for _, log := range logs {
		if log != nil && log.ChannelId > 0 {
			ids[log.ChannelId] = struct{}{}
		}
	}
	if len(ids) == 0 || DB == nil {
		return nil
	}
	idList := make([]int, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	sort.Ints(idList)

	var channels []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := DB.Table("channels").Select("id, name").Where("id IN ?", idList).Find(&channels).Error; err != nil {
		return nil
	}
	result := make(map[int]string, len(channels))
	for _, channel := range channels {
		result[channel.Id] = channel.Name
	}
	return result
}

func otherStringDefault(other map[string]interface{}, key string) string {
	value, _ := otherString(other, key)
	return value
}

func otherString(other map[string]interface{}, key string) (string, bool) {
	if other == nil {
		return "", false
	}
	raw, ok := other[key]
	if !ok || raw == nil {
		return "", false
	}
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value), true
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value)), true
	}
}

func otherBool(other map[string]interface{}, key string) (bool, bool) {
	if other == nil {
		return false, false
	}
	raw, ok := other[key]
	if !ok {
		return false, false
	}
	value, ok := raw.(bool)
	return value, ok
}

func otherInt(other map[string]interface{}, key string) int {
	if other == nil {
		return 0
	}
	raw, ok := other[key]
	if !ok || raw == nil {
		return 0
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
	return 0
}
