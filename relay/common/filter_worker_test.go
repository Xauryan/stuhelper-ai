package common

import (
	"io"
	"net/http"
	"strings"
	"testing"

	appcommon "github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRelayFilterWorkerRequest(t *testing.T) {
	t.Setenv(relayFilterWorkerEnabledEnv, "true")
	t.Setenv(relayFilterWorkerConfigEnv, `{"enabled":true,"request":[{"name":"trim-model","operations":[{"path":"model","mode":"trim_prefix","value":"openai/"}]}]}`)

	out, err := ApplyRelayFilterWorkerRequest([]byte(`{"model":"openai/gpt-4o","temperature":0.7}`), nil)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"gpt-4o","temperature":0.7}`, string(out))
}

func TestBuildRelayFilterWorkerRequestBody(t *testing.T) {
	t.Setenv(relayFilterWorkerEnabledEnv, "true")
	t.Setenv(relayFilterWorkerConfigEnv, `{"enabled":true,"request":[{"name":"trim-model","operations":[{"path":"model","mode":"trim_prefix","value":"openai/"}]}]}`)

	storage, err := appcommon.CreateBodyStorage([]byte(`{"model":"openai/gpt-4o","temperature":0.7}`))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, storage.Close())
	})

	body, size, closer, err := BuildRelayFilterWorkerRequestBody(storage, nil)
	require.NoError(t, err)
	defer closer.Close()

	assert.Equal(t, int64(len(`{"model":"gpt-4o","temperature":0.7}`)), size)
	filtered, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"gpt-4o","temperature":0.7}`, string(filtered))

	filteredStorage, ok := closer.(appcommon.BodyStorage)
	require.True(t, ok)
	_, err = filteredStorage.Seek(0, io.SeekStart)
	require.NoError(t, err)
	replayed, err := io.ReadAll(filteredStorage)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"gpt-4o","temperature":0.7}`, string(replayed))

	_, err = storage.Seek(0, io.SeekStart)
	require.NoError(t, err)
	original, err := io.ReadAll(storage)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"openai/gpt-4o","temperature":0.7}`, string(original))
}

func TestApplyRelayFilterWorkerResponse(t *testing.T) {
	t.Setenv(relayFilterWorkerEnabledEnv, "true")
	t.Setenv(relayFilterWorkerConfigEnv, `{"enabled":true,"response":[{"name":"mask-data","operations":[{"path":"data.secret","mode":"set","value":"redacted"}]}]}`)

	resp := &http.Response{
		Header: make(http.Header),
		Body:   io.NopCloser(strings.NewReader(`{"data":{"secret":"visible","nested":1}}`)),
	}
	resp.Header.Set("Content-Type", "application/json")

	require.NoError(t, ApplyRelayFilterWorkerResponse(&RelayInfo{}, resp))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"secret":"redacted","nested":1}}`, string(body))
}

func TestApplyRelayFilterWorkerResponseStream(t *testing.T) {
	t.Setenv(relayFilterWorkerEnabledEnv, "true")
	t.Setenv(relayFilterWorkerConfigEnv, `{"enabled":true,"stream_response":[{"name":"mask-stream","operations":[{"path":"delta.secret","mode":"set","value":"redacted"}]}]}`)

	resp := &http.Response{
		Header: make(http.Header),
		Body: io.NopCloser(strings.NewReader(
			"data: {\"delta\":{\"secret\":\"visible\",\"nested\":1}}\n" +
				"data: [DONE]\n",
		)),
	}
	resp.Header.Set("Content-Type", "text/event-stream")

	require.NoError(t, ApplyRelayFilterWorkerResponse(&RelayInfo{}, resp))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `data: {"delta":{"secret":"redacted","nested":1}}`)
	assert.Contains(t, string(body), "data: [DONE]")
}
