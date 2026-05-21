package service

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/types"

	"github.com/gin-gonic/gin"
)

const (
	RelayLoopPathHeader      = "StuHelper-AI-Relay-Path"
	RelayLoopSignatureHeader = "StuHelper-AI-Relay-Signature"
	relayLoopSignaturePrefix = "relay-loop:"
	maxRelayLoopHops         = 8
)

func relayLoopSignature(path string) string {
	return common.GenerateHMAC(relayLoopSignaturePrefix + strings.TrimSpace(path))
}

func parseRelayLoopPath(path string) ([]int, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}

	parts := strings.Split(path, ",")
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, len(ids) > 0
}

func formatRelayLoopPath(ids []int) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		parts = append(parts, strconv.Itoa(id))
	}
	return strings.Join(parts, ",")
}

func containsRelayLoopChannel(ids []int, channelID int) bool {
	for _, id := range ids {
		if id == channelID {
			return true
		}
	}
	return false
}

func GetRelayLoopChannelIDs(c *gin.Context) []int {
	if c == nil {
		return nil
	}
	if ids, ok := common.GetContextKeyType[[]int](c, constant.ContextKeyRelayLoopChannelIds); ok {
		return ids
	}
	return nil
}

func RelayLoopExcludeChannelIDs(c *gin.Context) map[int]struct{} {
	ids := GetRelayLoopChannelIDs(c)
	if len(ids) == 0 {
		return nil
	}
	exclude := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id > 0 {
			exclude[id] = struct{}{}
		}
	}
	return exclude
}

func CaptureRelayLoopPath(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}

	path := strings.TrimSpace(c.Request.Header.Get(RelayLoopPathHeader))
	signature := strings.TrimSpace(c.Request.Header.Get(RelayLoopSignatureHeader))
	if path == "" || signature == "" {
		return
	}
	expected := relayLoopSignature(path)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return
	}
	if ids, ok := parseRelayLoopPath(path); ok {
		common.SetContextKey(c, constant.ContextKeyRelayLoopChannelIds, ids)
	}
}

func ApplyRelayLoopHeaders(c *gin.Context, req *http.Request, channelID int) {
	if req == nil || channelID <= 0 {
		return
	}
	ApplyRelayLoopHeadersToHeader(c, req.Header, channelID)
}

func ApplyRelayLoopHeadersToHeader(c *gin.Context, header http.Header, channelID int) {
	if header == nil || channelID <= 0 {
		return
	}

	ids := GetRelayLoopChannelIDs(c)
	nextIDs := make([]int, 0, len(ids)+1)
	nextIDs = append(nextIDs, ids...)
	nextIDs = append(nextIDs, channelID)
	path := formatRelayLoopPath(nextIDs)
	if path == "" {
		return
	}
	header.Set(RelayLoopPathHeader, path)
	header.Set(RelayLoopSignatureHeader, relayLoopSignature(path))
}

func CheckRelayLoopForChannel(c *gin.Context, channelID int) *types.StuHelperAIError {
	ids := GetRelayLoopChannelIDs(c)
	if len(ids) == 0 {
		return nil
	}
	if containsRelayLoopChannel(ids, channelID) {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("检测到自引用 relay 循环：渠道 #%d 已在当前请求链路中出现，请不要把本站 auto 令牌作为同站点渠道的上游 key 使用", channelID),
			types.ErrorCodeChannelRelayLoop,
			http.StatusLoopDetected,
			types.ErrOptionWithSkipRetry(),
			types.ErrOptionWithNoRecordErrorLog(),
		)
	}
	if len(ids) >= maxRelayLoopHops {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("relay 链路跳数过多（%d），已中止以避免递归调用", len(ids)),
			types.ErrorCodeChannelRelayLoop,
			http.StatusLoopDetected,
			types.ErrOptionWithSkipRetry(),
			types.ErrOptionWithNoRecordErrorLog(),
		)
	}
	return nil
}
