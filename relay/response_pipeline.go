package relay

import (
	"net/http"
	"strings"

	relaycommon "github.com/Xauryan/stuhelper-ai/relay/common"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/types"

	"github.com/gin-gonic/gin"
)

type responsePipelineOptions struct {
	allowCreated bool
}

func runResponsePipeline(c *gin.Context, info *relaycommon.RelayInfo, resp any, handle func(*http.Response) (any, *types.StuHelperAIError), options responsePipelineOptions) (any, *types.StuHelperAIError) {
	statusCodeMappingStr := c.GetString("status_code_mapping")
	if resp == nil {
		usage, apiErr := handle(nil)
		if apiErr != nil {
			service.ResetStatusCode(apiErr, statusCodeMappingStr)
			return nil, apiErr
		}
		return usage, nil
	}

	httpResp := resp.(*http.Response)
	if info != nil {
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
	}

	if httpResp.StatusCode != http.StatusOK {
		if options.allowCreated && httpResp.StatusCode == http.StatusCreated {
			httpResp.StatusCode = http.StatusOK
		} else {
			apiErr := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			service.ResetStatusCode(apiErr, statusCodeMappingStr)
			return nil, apiErr
		}
	}

	usage, apiErr := handle(httpResp)
	if apiErr != nil {
		service.ResetStatusCode(apiErr, statusCodeMappingStr)
		return nil, apiErr
	}
	return usage, nil
}
