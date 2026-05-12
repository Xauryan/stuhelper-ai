package relay

import (
	relaycommon "github.com/Xauryan/stuhelper-ai/relay/common"
	"github.com/Xauryan/stuhelper-ai/types"
)

func newAPIErrorFromParamOverride(err error) *types.StuHelperAIError {
	if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
		return relaycommon.StuHelperAIErrorFromParamOverride(fixedErr)
	}
	return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
}
