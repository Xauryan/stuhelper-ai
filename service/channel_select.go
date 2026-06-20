package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/constant"
	"github.com/Xauryan/stuhelper-ai/logger"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/types"
	"github.com/gin-gonic/gin"
)

var ErrRelayLoopNoAvailableChannel = errors.New("relay loop guard excluded all available channels")
var ErrNoAvailableChannelAfterExclusions = errors.New("excluded all available channels")

type RetryParam struct {
	Ctx               *gin.Context
	TokenGroup        string
	ModelName         string
	Retry             *int
	ExcludeChannelIDs map[int]struct{}
	resetNextTry      bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// ExcludeChannel marks a channel as failed for the remainder of this request so
// later retries (including auto-group cross-group retries) do not reselect it.
func (p *RetryParam) ExcludeChannel(channelID int) {
	if channelID <= 0 {
		return
	}
	if p.ExcludeChannelIDs == nil {
		p.ExcludeChannelIDs = make(map[int]struct{})
	}
	p.ExcludeChannelIDs[channelID] = struct{}{}
}

// mergeExcludeChannelIDs returns the union of two exclude sets without mutating
// either input. Used so per-request failed channels and the cross-instance
// relay-loop guard set are both honored during selection.
func mergeExcludeChannelIDs(a, b map[int]struct{}) map[int]struct{} {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	merged := make(map[int]struct{}, len(a)+len(b))
	for id := range a {
		merged[id] = struct{}{}
	}
	for id := range b {
		merged[id] = struct{}{}
	}
	return merged
}

func isAutoGroupRequest(c *gin.Context) bool {
	if c == nil {
		return false
	}
	tokenGroup := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
	if tokenGroup != "" {
		return tokenGroup == "auto"
	}
	return common.GetContextKeyString(c, constant.ContextKeyUsingGroup) == "auto"
}

func nextAutoGroupRetryIndex(c *gin.Context) (int, []string, bool) {
	if c == nil {
		return 0, nil, false
	}
	if !isAutoGroupRequest(c) {
		return 0, nil, false
	}
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenCrossGroupRetry) {
		return 0, nil, false
	}

	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	autoGroups := GetContextAutoGroups(c, userGroup)
	if len(autoGroups) < 2 {
		return 0, autoGroups, false
	}

	nextGroupIndex := -1
	currentGroup := common.GetContextKeyString(c, constant.ContextKeyAutoGroup)
	if currentGroup != "" {
		for i, group := range autoGroups {
			if group == currentGroup {
				nextGroupIndex = i + 1
				break
			}
		}
	}
	if nextGroupIndex < 0 {
		if lastGroupIndex, exists := common.GetContextKey(c, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				nextGroupIndex = idx
			}
		}
	}
	if nextGroupIndex < 0 || nextGroupIndex >= len(autoGroups) {
		return nextGroupIndex, autoGroups, false
	}
	return nextGroupIndex, autoGroups, true
}

func HasNextAutoGroupRetry(c *gin.Context) bool {
	_, _, ok := nextAutoGroupRetryIndex(c)
	return ok
}

// PrepareAutoGroupRetry advances an auto token request to the next concrete
// group after the current selected group failed.
func PrepareAutoGroupRetry(c *gin.Context) bool {
	nextGroupIndex, autoGroups, ok := nextAutoGroupRetryIndex(c)
	if !ok {
		return false
	}

	common.SetContextKey(c, constant.ContextKeyAutoGroupIndex, nextGroupIndex)
	common.SetContextKey(c, constant.ContextKeyAutoGroupRetryIndex, 0)
	logger.LogDebug(c, "Auto group retry advancing to group: %s", autoGroups[nextGroupIndex])
	return true
}

func setSelectedAutoGroup(c *gin.Context, group string) {
	if c == nil || group == "" {
		return
	}
	common.SetContextKey(c, constant.ContextKeyAutoGroup, group)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, group)
}

func SetSelectedAutoGroup(c *gin.Context, group string) {
	setSelectedAutoGroup(c, group)
}

func ShouldRetryChannelSetupError(err *types.StuHelperAIError) bool {
	if types.IsSkipRetryError(err) {
		return false
	}
	return ClassifyRelayError(err).Retryable
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
//
//   - priorityRetry = Retry - startRetryIndex, represents the priority level within current group.
//     priorityRetry = Retry - startRetryIndex，表示当前分组内的优先级级别。
//
//   - When GetRandomSatisfiedChannel returns nil (priorities exhausted), moves to next group.
//     当 GetRandomSatisfiedChannel 返回 nil（优先级用完）时，切换到下一个分组。
//
// Example flow (2 groups, each with 2 priorities, RetryTimes=3):
// 示例流程（2个分组，每个有2个优先级，RetryTimes=3）：
//
//	Retry=0: GroupA, priority0 (startRetryIndex=0, priorityRetry=0)
//	         分组A, 优先级0
//
//	Retry=1: GroupA, priority1 (startRetryIndex=0, priorityRetry=1)
//	         分组A, 优先级1
//
//	Retry=2: GroupA exhausted → GroupB, priority0 (startRetryIndex=2, priorityRetry=0)
//	         分组A用完 → 分组B, 优先级0
//
//	Retry=3: GroupB, priority1 (startRetryIndex=2, priorityRetry=1)
//	         分组B, 优先级1
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := param.TokenGroup
	if param.Ctx != nil && param.TokenGroup != "" && common.GetContextKeyString(param.Ctx, constant.ContextKeyTokenGroup) == "" {
		common.SetContextKey(param.Ctx, constant.ContextKeyTokenGroup, param.TokenGroup)
	}
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
	requestExcludeChannelIDs := param.ExcludeChannelIDs
	relayLoopExcludeChannelIDs := RelayLoopExcludeChannelIDs(param.Ctx)
	breakerExcludeChannelIDs := BreakerOpenChannelIDs()
	excludeChannelIDs := mergeExcludeChannelIDs(requestExcludeChannelIDs, relayLoopExcludeChannelIDs)
	// Shield channels whose breaker is Open (cooling down) by folding them into
	// the same exclude set used for per-request failover. Half-open channels are
	// not in this set, so a probe request can flow through and test recovery.
	excludeChannelIDs = mergeExcludeChannelIDs(excludeChannelIDs, breakerExcludeChannelIDs)

	if param.TokenGroup == "auto" {
		autoGroups := GetContextAutoGroups(param.Ctx, userGroup)
		if len(autoGroups) == 0 {
			return nil, selectGroup, fmt.Errorf("auto groups has no usable groups for user group %s", userGroup)
		}

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			selectGroup = autoGroup
			// Calculate priorityRetry for current group
			// 计算当前分组的 priorityRetry
			priorityRetry := param.GetRetry()
			// If moved to a new group, reset priorityRetry and update startRetryIndex
			// 如果切换到新分组，重置 priorityRetry 并更新 startRetryIndex
			if i > startGroupIndex {
				priorityRetry = 0
			}
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, priorityRetry: %d", autoGroup, priorityRetry)

			channel, err = model.GetRandomSatisfiedChannelExcluding(autoGroup, param.ModelName, priorityRetry, excludeChannelIDs)
			if err != nil {
				return nil, selectGroup, err
			}
			if channel == nil {
				// Current group has no available channel for this model, try next group
				// 当前分组没有该模型的可用渠道，尝试下一个分组
				logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityRetry %d, trying next group", autoGroup, param.ModelName, priorityRetry)
				// 重置状态以尝试下一个分组
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				continue
			}
			setSelectedAutoGroup(param.Ctx, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)

			// Prepare state for next retry
			// 为下一次重试准备状态
			if crossGroupRetry && priorityRetry >= common.RetryTimes {
				// Current group has exhausted all retries, prepare to switch to next group
				// This request still uses current group, but next retry will use next group
				// 当前分组已用完所有重试次数，准备切换到下一个分组
				// 本次请求仍使用当前分组，但下次重试将使用下一个分组
				logger.LogDebug(param.Ctx, "Current group %s retries exhausted (priorityRetry=%d >= RetryTimes=%d), preparing switch to next group for next retry", autoGroup, priorityRetry, common.RetryTimes)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				param.ResetRetryNextTry()
			} else {
				// Stay in current group, save current state
				// 保持在当前分组，保存当前状态
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			}
			break
		}
		if channel == nil && len(excludeChannelIDs) == 0 {
			return nil, selectGroup, fmt.Errorf("auto groups have no available channel for model %s: %s", param.ModelName, strings.Join(autoGroups, ","))
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannelExcluding(param.TokenGroup, param.ModelName, param.GetRetry(), excludeChannelIDs)
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}
	if channel == nil && len(excludeChannelIDs) > 0 {
		if len(requestExcludeChannelIDs) == 0 && len(breakerExcludeChannelIDs) == 0 && len(relayLoopExcludeChannelIDs) > 0 {
			return nil, selectGroup, fmt.Errorf("%w: group=%s, model=%s", ErrRelayLoopNoAvailableChannel, selectGroup, param.ModelName)
		}
		return nil, selectGroup, fmt.Errorf("%w: group=%s, model=%s", ErrNoAvailableChannelAfterExclusions, selectGroup, param.ModelName)
	}
	return channel, selectGroup, nil
}
