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

func applyRelayRequestFilterWorker(jsonData []byte, info *relaycommon.RelayInfo) ([]byte, *types.StuHelperAIError) {
	filtered, err := relaycommon.ApplyRelayFilterWorkerRequest(jsonData, info)
	if err != nil {
		return nil, newAPIErrorFromParamOverride(err)
	}
	return filtered, nil
}

func newAPIErrorFromRelayFilterWorker(err error) *types.StuHelperAIError {
	return relaycommon.StuHelperAIErrorFromRelayFilterWorker(err)
}
