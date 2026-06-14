package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Xauryan/stuhelper-ai/types"
)

func TestClassifyRelayError(t *testing.T) {
	tests := []struct {
		name        string
		err         *types.StuHelperAIError
		class       RetryClass
		retryable   bool
		forceRetry  bool
		channelSide bool
		transient   bool
	}{
		{
			name:  "nil",
			class: RetryClassNone,
		},
		{
			name:        "channel error",
			err:         types.NewError(errors.New("stream interrupted"), types.ErrorCodeStreamInterrupted),
			class:       RetryClassChannel,
			retryable:   true,
			forceRetry:  true,
			channelSide: true,
		},
		{
			name:  "skip retry",
			err:   types.NewErrorWithStatusCode(errors.New("bad request"), types.ErrorCodeInvalidRequest, http.StatusServiceUnavailable, types.ErrOptionWithSkipRetry()),
			class: RetryClassSkip,
		},
		{
			name:      "transient status",
			err:       types.NewErrorWithStatusCode(errors.New("service unavailable"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable),
			class:     RetryClassTransient,
			retryable: true,
			transient: true,
		},
		{
			name:  "always skip status",
			err:   types.NewErrorWithStatusCode(errors.New("gateway timeout"), types.ErrorCodeBadResponseStatusCode, http.StatusGatewayTimeout),
			class: RetryClassSkip,
		},
		{
			name:        "disable status",
			err:         types.NewErrorWithStatusCode(errors.New("invalid api key"), types.ErrorCodeBadResponseStatusCode, http.StatusUnauthorized),
			class:       RetryClassChannel,
			retryable:   true,
			channelSide: true,
		},
		{
			name:  "client status",
			err:   types.NewErrorWithStatusCode(errors.New("invalid request"), types.ErrorCodeInvalidRequest, http.StatusBadRequest),
			class: RetryClassClient,
		},
		{
			name:      "invalid upstream status",
			err:       types.NewErrorWithStatusCode(errors.New("connection reset"), types.ErrorCodeDoRequestFailed, 0),
			class:     RetryClassTransient,
			retryable: true,
			transient: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRelayError(tt.err)
			if got.Class != tt.class {
				t.Fatalf("class = %s, want %s", got.Class, tt.class)
			}
			if got.Retryable != tt.retryable {
				t.Fatalf("retryable = %v, want %v", got.Retryable, tt.retryable)
			}
			if got.ForceRetry != tt.forceRetry {
				t.Fatalf("forceRetry = %v, want %v", got.ForceRetry, tt.forceRetry)
			}
			if got.ChannelSide != tt.channelSide {
				t.Fatalf("channelSide = %v, want %v", got.ChannelSide, tt.channelSide)
			}
			if got.Transient != tt.transient {
				t.Fatalf("transient = %v, want %v", got.Transient, tt.transient)
			}
		})
	}
}
