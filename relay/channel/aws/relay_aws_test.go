package aws

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	relaycommon "github.com/Xauryan/stuhelper-ai/relay/common"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDoAwsClientRequest_AppliesRuntimeHeaderOverrideToAnthropicBeta(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName:           "claude-3-5-sonnet-20240620",
		IsStream:                  false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"anthropic-beta": "computer-use-2025-01-24",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "access-key|secret-key|us-east-1",
			UpstreamModelName: "claude-3-5-sonnet-20240620",
		},
	}

	requestBody := bytes.NewBufferString(`{"messages":[{"role":"user","content":"hello"}],"max_tokens":128}`)
	adaptor := &Adaptor{}

	_, err := doAwsClientRequest(ctx, info, adaptor, requestBody)
	require.NoError(t, err)

	awsReq, ok := adaptor.AwsReq.(*bedrockruntime.InvokeModelInput)
	require.True(t, ok)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(awsReq.Body, &payload))

	anthropicBeta, exists := payload["anthropic_beta"]
	require.True(t, exists)

	values, ok := anthropicBeta.([]any)
	require.True(t, ok)
	require.Equal(t, []any{"computer-use-2025-01-24"}, values)
}

func TestFormatRequestPreservesExplicitZeroAndContextManagement(t *testing.T) {
	t.Parallel()

	body := bytes.NewBufferString(`{
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":0,
		"top_p":0,
		"top_k":0,
		"context_management":{"edits":[{"type":"clear_tool_uses_20250919"}]}
	}`)

	req, err := formatRequest(body, http.Header{})
	require.NoError(t, err)
	require.NotNil(t, req.MaxTokens)
	require.Equal(t, uint(0), *req.MaxTokens)
	require.NotNil(t, req.TopP)
	require.Equal(t, float64(0), *req.TopP)
	require.NotNil(t, req.TopK)
	require.Equal(t, 0, *req.TopK)
	require.JSONEq(t, `{"edits":[{"type":"clear_tool_uses_20250919"}]}`, string(req.ContextManagement))
}
