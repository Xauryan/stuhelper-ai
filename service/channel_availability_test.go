package service

import (
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/types"
)

func withAvailabilityClock(t *testing.T) func(d time.Duration) {
	t.Helper()
	origClock := availabilityClock

	channelAvailabilityMu.Lock()
	origAvailability := channelAvailabilityByID
	channelAvailabilityByID = make(map[int]*channelAvailability)
	channelAvailabilityMu.Unlock()

	now := time.Unix(1_700_000_000, 0)
	availabilityClock = func() time.Time { return now }

	t.Cleanup(func() {
		availabilityClock = origClock
		channelAvailabilityMu.Lock()
		channelAvailabilityByID = origAvailability
		channelAvailabilityMu.Unlock()
	})

	return func(d time.Duration) { now = now.Add(d) }
}

func channelChannelErr() *types.StuHelperAIError {
	return types.NewErrorWithStatusCode(errString("channel failure"), types.ErrorCodeBadResponseStatusCode, 401)
}

func channelTransientErr() *types.StuHelperAIError {
	return types.NewErrorWithStatusCode(errString("transient failure"), types.ErrorCodeBadResponseStatusCode, 503)
}

func channelClientErr() *types.StuHelperAIError {
	return types.NewErrorWithStatusCode(errString("bad request"), types.ErrorCodeBadResponseStatusCode, 400)
}

func TestChannelAvailabilityRecordsSamples(t *testing.T) {
	advance := withAvailabilityClock(t)
	const ch = 21

	ReportChannelAvailability(ch, nil)
	ReportChannelAvailability(ch, channelChannelErr())
	ReportChannelAvailability(ch, channelTransientErr())
	ReportChannelAvailability(ch, channelClientErr())
	advance(5 * time.Second)

	snapshot := ChannelAvailabilitySnapshot(ch)
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if snapshot.Success != 1 {
		t.Fatalf("unexpected success: %d", snapshot.Success)
	}
	if snapshot.ChannelFailures != 1 {
		t.Fatalf("unexpected channel failures: %d", snapshot.ChannelFailures)
	}
	if snapshot.TransientFailures != 1 {
		t.Fatalf("unexpected transient failures: %d", snapshot.TransientFailures)
	}
	if snapshot.Ignored != 1 {
		t.Fatalf("unexpected ignored count: %d", snapshot.Ignored)
	}
	if snapshot.Total != 3 {
		t.Fatalf("unexpected total: %d", snapshot.Total)
	}
	if snapshot.SuccessRate != 1.0/3.0 {
		t.Fatalf("unexpected success rate: %v", snapshot.SuccessRate)
	}
	if snapshot.LastSuccessAt == 0 || snapshot.LastFailureAt == 0 {
		t.Fatalf("expected last timestamps to be recorded: %+v", snapshot)
	}
	if snapshot.LastClass != string(RetryClassTransient) {
		t.Fatalf("unexpected last class: %s", snapshot.LastClass)
	}
}

func TestChannelAvailabilityResetClearsState(t *testing.T) {
	withAvailabilityClock(t)
	const ch = 22

	ReportChannelAvailability(ch, nil)
	ResetChannelAvailability(ch)

	snapshot := ChannelAvailabilitySnapshot(ch)
	if snapshot == nil {
		t.Fatal("expected snapshot after reset")
	}
	if snapshot.Total != 0 || snapshot.Success != 0 || snapshot.ChannelFailures != 0 || snapshot.TransientFailures != 0 {
		t.Fatalf("expected cleared snapshot, got %+v", snapshot)
	}
}
