package service

import (
	"strings"

	"github.com/Xauryan/stuhelper-ai/setting/operation_setting"
	"github.com/Xauryan/stuhelper-ai/types"
)

type RetryClass string

const (
	RetryClassNone      RetryClass = "none"
	RetryClassSkip      RetryClass = "skip"
	RetryClassClient    RetryClass = "client"
	RetryClassChannel   RetryClass = "channel"
	RetryClassTransient RetryClass = "transient"
)

type RetryClassification struct {
	Class       RetryClass
	Retryable   bool
	ForceRetry  bool
	ChannelSide bool
	Transient   bool
}

// ClassifyRelayError centralizes relay error policy so retry, channel-affinity,
// auto-disable and breaker accounting do not drift into separate rule sets.
func ClassifyRelayError(err *types.StuHelperAIError) RetryClassification {
	if err == nil {
		return RetryClassification{Class: RetryClassNone}
	}

	if types.IsChannelError(err) {
		return RetryClassification{
			Class:       RetryClassChannel,
			Retryable:   true,
			ForceRetry:  true,
			ChannelSide: true,
		}
	}

	if types.IsSkipRetryError(err) || operation_setting.IsAlwaysSkipRetryCode(err.GetErrorCode()) ||
		operation_setting.IsAlwaysSkipRetryStatusCode(err.StatusCode) {
		return RetryClassification{Class: RetryClassSkip}
	}

	if isAutomaticDisableError(err) {
		return RetryClassification{
			Class:       RetryClassChannel,
			Retryable:   isRetryableStatus(err.StatusCode),
			ChannelSide: true,
		}
	}

	code := err.StatusCode
	if code >= 200 && code < 300 {
		return RetryClassification{Class: RetryClassSkip}
	}
	if isRetryableStatus(code) {
		return RetryClassification{
			Class:     RetryClassTransient,
			Retryable: true,
			Transient: true,
		}
	}
	return RetryClassification{Class: RetryClassClient}
}

func isAutomaticDisableError(err *types.StuHelperAIError) bool {
	if err == nil {
		return false
	}
	if operation_setting.ShouldDisableByStatusCode(err.StatusCode) {
		return true
	}
	lowerMessage := strings.ToLower(err.InternalError())
	search, _ := AcSearch(lowerMessage, operation_setting.AutomaticDisableKeywords, true)
	return search
}

func isRetryableStatus(code int) bool {
	if code < 100 || code > 599 {
		return true
	}
	return operation_setting.ShouldRetryByStatusCode(code)
}
