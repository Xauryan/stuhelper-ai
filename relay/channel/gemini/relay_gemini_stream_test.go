package gemini

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/dto"
	relaycommon "github.com/Xauryan/stuhelper-ai/relay/common"
	"github.com/Xauryan/stuhelper-ai/types"

	"github.com/gin-gonic/gin"
)

type errorAfterChunkReadCloser struct {
	chunk []byte
	read  bool
}

func (r *errorAfterChunkReadCloser) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		return copy(p, r.chunk), nil
	}
	return 0, errors.New("upstream read failed")
}

func (r *errorAfterChunkReadCloser) Close() error {
	return nil
}

func TestGeminiStreamHandlerReturnsInterruptionErrorOnScannerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/test:streamGenerateContent", nil)

	resp := &http.Response{
		Body: &errorAfterChunkReadCloser{
			chunk: []byte(`data: {"candidates":[{"content":{"parts":[{"text":"partial"}]}}]}` + "\n"),
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-test",
		},
	}

	usage, apiErr := geminiStreamHandler(c, info, resp, func(data string, geminiResponse *dto.GeminiChatResponse) bool {
		return true
	})

	if apiErr == nil {
		t.Fatal("expected stream interruption error")
	}
	if apiErr.GetErrorCode() != types.ErrorCodeStreamInterrupted {
		t.Fatalf("unexpected error code: %s", apiErr.GetErrorCode())
	}
	if usage == nil {
		t.Fatal("expected usage placeholder to be returned")
	}
}
