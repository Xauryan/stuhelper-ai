package service

import (
	"testing"
	"time"

	"github.com/Xauryan/stuhelper-ai/types"
)

// withBreakerClock installs a controllable clock and a clean breaker registry
// for the duration of a test.
func withBreakerClock(t *testing.T) func(d time.Duration) {
	t.Helper()
	origClock := breakerClock
	origEnabled := breakerEnabled
	breakerEnabled = true

	breakersMu.Lock()
	origBreakers := breakers
	breakers = make(map[int]*channelBreaker)
	breakersMu.Unlock()

	now := time.Unix(1_700_000_000, 0)
	breakerClock = func() time.Time { return now }

	t.Cleanup(func() {
		breakerClock = origClock
		breakerEnabled = origEnabled
		breakersMu.Lock()
		breakers = origBreakers
		breakersMu.Unlock()
	})

	return func(d time.Duration) { now = now.Add(d) }
}

func fatalErr() *types.StuHelperAIError {
	return types.NewErrorWithStatusCode(errString("invalid api key"), types.ErrorCodeBadResponseStatusCode, 401)
}

func transientErr() *types.StuHelperAIError {
	return types.NewErrorWithStatusCode(errString("service unavailable"), types.ErrorCodeBadResponseStatusCode, 503)
}

type errString string

func (e errString) Error() string { return string(e) }

func isOpen(channelID int) bool {
	_, open := BreakerOpenChannelIDs()[channelID]
	return open
}

func TestBreakerConsecutiveFatalTrips(t *testing.T) {
	advance := withBreakerClock(t)
	const ch = 1

	for i := 0; i < breakerConsecFatal-1; i++ {
		ReportRelayResult(ch, fatalErr())
		if isOpen(ch) {
			t.Fatalf("breaker tripped early after %d fatals", i+1)
		}
	}
	ReportRelayResult(ch, fatalErr()) // hits the consecutive-fatal threshold
	if !isOpen(ch) {
		t.Fatalf("breaker should be open after %d consecutive fatals", breakerConsecFatal)
	}

	// Cooldown not elapsed -> still open.
	advance(breakerCooldown - time.Second)
	if !isOpen(ch) {
		t.Fatalf("breaker should remain open during cooldown")
	}
	// Cooldown elapsed -> half-open (no longer in the open set).
	advance(2 * time.Second)
	if isOpen(ch) {
		t.Fatalf("breaker should be half-open after cooldown, not open")
	}
	if got := BreakerStateName(ch); got != "half_open" {
		t.Fatalf("expected half_open, got %s", got)
	}
}

func TestInitChannelBreakerConfigReadsRuntimeEnv(t *testing.T) {
	origEnabled := breakerEnabled
	origWindow := breakerWindow
	origMinSamples := breakerMinSamples
	origTripScorePct := breakerTripScorePct
	origConsecFatal := breakerConsecFatal
	origCooldown := breakerCooldown
	origMaxCooldown := breakerMaxCooldown
	origHalfOpenProbes := breakerHalfOpenProbes
	t.Cleanup(func() {
		breakerEnabled = origEnabled
		breakerWindow = origWindow
		breakerMinSamples = origMinSamples
		breakerTripScorePct = origTripScorePct
		breakerConsecFatal = origConsecFatal
		breakerCooldown = origCooldown
		breakerMaxCooldown = origMaxCooldown
		breakerHalfOpenProbes = origHalfOpenProbes
	})

	t.Setenv("CHANNEL_BREAKER_ENABLED", "false")
	t.Setenv("CHANNEL_BREAKER_WINDOW_SECONDS", "42")
	t.Setenv("CHANNEL_BREAKER_MIN_SAMPLES", "7")
	t.Setenv("CHANNEL_BREAKER_TRIP_SCORE_PCT", "35")
	t.Setenv("CHANNEL_BREAKER_CONSECUTIVE_FATAL", "2")
	t.Setenv("CHANNEL_BREAKER_COOLDOWN_SECONDS", "11")
	t.Setenv("CHANNEL_BREAKER_MAX_COOLDOWN_SECONDS", "12")
	t.Setenv("CHANNEL_BREAKER_HALFOPEN_PROBES", "4")

	InitChannelBreakerConfig()

	if breakerEnabled {
		t.Fatal("breaker enabled should follow CHANNEL_BREAKER_ENABLED=false")
	}
	if breakerWindow != 42*time.Second {
		t.Fatalf("unexpected window: %s", breakerWindow)
	}
	if breakerMinSamples != 7 {
		t.Fatalf("unexpected min samples: %d", breakerMinSamples)
	}
	if breakerTripScorePct != 35 {
		t.Fatalf("unexpected trip score pct: %d", breakerTripScorePct)
	}
	if breakerConsecFatal != 2 {
		t.Fatalf("unexpected consecutive fatal: %d", breakerConsecFatal)
	}
	if breakerCooldown != 11*time.Second {
		t.Fatalf("unexpected cooldown: %s", breakerCooldown)
	}
	if breakerMaxCooldown != 12*time.Second {
		t.Fatalf("unexpected max cooldown: %s", breakerMaxCooldown)
	}
	if breakerHalfOpenProbes != 4 {
		t.Fatalf("unexpected half-open probes: %d", breakerHalfOpenProbes)
	}
}

func TestResetChannelBreakerClearsOpenState(t *testing.T) {
	withBreakerClock(t)
	const ch = 8

	for i := 0; i < breakerConsecFatal; i++ {
		ReportRelayResult(ch, fatalErr())
	}
	if !isOpen(ch) {
		t.Fatal("expected breaker to be open before reset")
	}

	ResetChannelBreaker(ch)

	if isOpen(ch) {
		t.Fatal("expected breaker to be closed after reset")
	}
	if got := BreakerStateName(ch); got != "closed" {
		t.Fatalf("expected closed after reset, got %s", got)
	}
}

func TestBreakerHalfOpenRecovers(t *testing.T) {
	advance := withBreakerClock(t)
	const ch = 2

	for i := 0; i < breakerConsecFatal; i++ {
		ReportRelayResult(ch, fatalErr())
	}
	if !isOpen(ch) {
		t.Fatal("expected open")
	}
	advance(breakerCooldown + time.Second)
	if isOpen(ch) {
		t.Fatal("expected half-open after cooldown")
	}
	// Enough successful probes -> closed.
	for i := 0; i < breakerHalfOpenProbes; i++ {
		ReportRelayResult(ch, nil)
	}
	if got := BreakerStateName(ch); got != "closed" {
		t.Fatalf("expected closed after probes, got %s", got)
	}
}

func TestBreakerHalfOpenFailureBacksOff(t *testing.T) {
	advance := withBreakerClock(t)
	const ch = 3

	for i := 0; i < breakerConsecFatal; i++ {
		ReportRelayResult(ch, fatalErr())
	}
	advance(breakerCooldown + time.Second) // -> half-open
	if isOpen(ch) {
		t.Fatal("expected half-open")
	}
	ReportRelayResult(ch, fatalErr()) // probe fails -> re-open with backoff
	if !isOpen(ch) {
		t.Fatal("expected re-open after failed probe")
	}
	// Backoff should now exceed the base cooldown.
	advance(breakerCooldown + time.Second)
	if !isOpen(ch) {
		t.Fatal("expected still open: backoff should exceed base cooldown")
	}
}

func TestBreakerMinSamplesProtectsFromFalseTrip(t *testing.T) {
	withBreakerClock(t)
	const ch = 4
	// Below MinSamples, even all-transient-failures must not trip (avoids killing
	// a channel on a tiny number of blips).
	n := breakerMinSamples - 1
	for i := 0; i < n; i++ {
		ReportRelayResult(ch, transientErr())
	}
	if isOpen(ch) {
		t.Fatalf("breaker tripped on %d samples, below MinSamples=%d", n, breakerMinSamples)
	}
}

func TestBreakerIgnoresClientErrors(t *testing.T) {
	withBreakerClock(t)
	const ch = 5
	// 400-class errors are the client's fault and must not count against the
	// channel, no matter how many.
	clientErr := types.NewErrorWithStatusCode(errString("bad request"), types.ErrorCodeInvalidRequest, 400)
	for i := 0; i < breakerMinSamples*3; i++ {
		ReportRelayResult(ch, clientErr)
	}
	if isOpen(ch) {
		t.Fatal("breaker must not trip on client (400) errors")
	}
}

func TestBreakerTransientRateTrips(t *testing.T) {
	withBreakerClock(t)
	const ch = 6
	// All-transient failures above MinSamples: score = 0.3 >= ... only trips if
	// TripScorePct <= 30. With the default 50 it should NOT trip on pure
	// transient, guarding against over-eager disabling of briefly-degraded
	// channels.
	for i := 0; i < breakerMinSamples*2; i++ {
		ReportRelayResult(ch, transientErr())
	}
	if breakerTripScorePct > 30 && isOpen(ch) {
		t.Fatalf("pure-transient should not trip at TripScorePct=%d", breakerTripScorePct)
	}
}

func TestBreakerDisabledIsNoop(t *testing.T) {
	origEnabled := breakerEnabled
	breakerEnabled = false
	t.Cleanup(func() { breakerEnabled = origEnabled })

	const ch = 7
	for i := 0; i < 100; i++ {
		ReportRelayResult(ch, fatalErr())
	}
	if BreakerOpenChannelIDs() != nil {
		t.Fatal("disabled breaker must return no open channels")
	}
	if got := BreakerStateName(ch); got != "disabled" {
		t.Fatalf("expected disabled, got %s", got)
	}
}
