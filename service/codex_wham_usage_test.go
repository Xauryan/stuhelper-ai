package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/stretchr/testify/require"
)

func TestFetchCodexWhamUsageSendsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/backend-api/wham/usage", r.URL.Path)
		requireCodexWhamHeaders(t, r)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	statusCode, body, err := FetchCodexWhamUsage(
		context.Background(),
		server.Client(),
		server.URL+"/",
		" access-token ",
		" account-id ",
	)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, statusCode)
	require.JSONEq(t, `{"ok":true}`, string(body))
}

func TestFetchCodexWhamRateLimitResetCreditsSendsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/backend-api/wham/rate-limit-reset-credits", r.URL.Path)
		requireCodexWhamHeaders(t, r)
		_, _ = w.Write([]byte(`{"available_count":2}`))
	}))
	defer server.Close()

	statusCode, body, err := FetchCodexWhamRateLimitResetCredits(
		context.Background(),
		server.Client(),
		server.URL,
		" access-token ",
		" account-id ",
	)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, statusCode)
	require.JSONEq(t, `{"available_count":2}`, string(body))
}

func TestConsumeCodexWhamRateLimitResetCreditSendsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/backend-api/wham/rate-limit-reset-credits/consume", r.URL.Path)
		requireCodexWhamHeaders(t, r)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]string
		require.NoError(t, common.DecodeJson(r.Body, &payload))
		redeemRequestID := payload["redeem_request_id"]
		require.NotEmpty(t, redeemRequestID)
		require.Len(t, redeemRequestID, 36)
		require.Contains(t, strings.ToLower(redeemRequestID), "-")

		_, _ = w.Write([]byte(`{"consumed":true}`))
	}))
	defer server.Close()

	statusCode, body, err := ConsumeCodexWhamRateLimitResetCredit(
		context.Background(),
		server.Client(),
		server.URL,
		" access-token ",
		" account-id ",
	)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, statusCode)
	require.JSONEq(t, `{"consumed":true}`, string(body))
}

func requireCodexWhamHeaders(t *testing.T, r *http.Request) {
	t.Helper()

	require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
	require.Equal(t, "account-id", r.Header.Get("chatgpt-account-id"))
	require.Equal(t, "application/json", r.Header.Get("Accept"))
	require.Equal(t, "codex_cli_rs", r.Header.Get("originator"))
}
