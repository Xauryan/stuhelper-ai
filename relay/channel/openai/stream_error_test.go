package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/constant"
	relaycommon "github.com/Xauryan/stuhelper-ai/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newOpenAIStreamTestContext(t *testing.T, body string) (*gin.Context, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
		IsStream:    true,
		RelayFormat: "openai",
	}
	return c, resp, info
}

func TestOaiStreamHandlerReturnsErrorOnSSEErrorPayload(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := strings.Join([]string{
		`data: {"error":{"message":"upstream quota exhausted","type":"insufficient_quota","code":"insufficient_quota"}}`,
		``,
	}, "\n")

	c, resp, info := newOpenAIStreamTestContext(t, body)

	usage, err := OaiStreamHandler(c, info, resp)
	require.NotNil(t, usage)
	require.NotNil(t, err)
	require.Equal(t, http.StatusInternalServerError, err.StatusCode)
	oaiErr := err.ToOpenAIError()
	require.Equal(t, "upstream quota exhausted", oaiErr.Message)
	require.Equal(t, "insufficient_quota", oaiErr.Type)
	require.Equal(t, "insufficient_quota", oaiErr.Code)
}
