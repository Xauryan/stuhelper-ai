package service

import (
	"strconv"
	"sync"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/types"
)

// Channel circuit breaker.
//
// A sliding-window, self-recovering breaker that complements the one-shot
// DisableChannel path: instead of writing the channel's DB status (which needs
// manual/auto-test recovery), it shields an unhealthy channel *in the selection
// layer* by feeding its id into the same exclude set used for failover, and
// automatically probes for recovery (closed -> open -> half-open -> closed).
//
// Failure weighting uses ClassifyRelayError: fatal channel-side errors
// (401/403, revoked key, quota) weigh 1.0 and trip fast on a short consecutive
// streak; transient errors (429/5xx/timeout) weigh 0.3 and only trip on a
// sustained failure rate. Client errors (400/invalid/sensitive) are not counted
// against the channel.
//
// State is in-memory per process. It is intentionally NOT persisted: a breaker
// trip is a fast, local, self-healing reaction; durable disabling remains the
// job of DisableChannel.

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerOpen
	breakerHalfOpen
)

const breakerBucketCount = 12 // sliding-window ring resolution

// Configuration (read from env during InitChannelBreakerConfig, after .env has
// been loaded). The breaker only excludes channels from selection and
// auto-recovers, so it is safe to default on; set CHANNEL_BREAKER_ENABLED=false
// to disable entirely.
var (
	breakerEnabled        = true
	breakerWindow         = 120 * time.Second
	breakerMinSamples     = 10
	breakerTripScorePct   = 50
	breakerConsecFatal    = 3
	breakerCooldown       = 60 * time.Second
	breakerMaxCooldown    = 1800 * time.Second
	breakerHalfOpenProbes = 3
)

// breakerClock is injectable so tests can drive time deterministically.
var breakerClock = time.Now

type breakerBucket struct {
	epoch     int64 // window-bucket index this slot currently represents
	total     int64
	fatal     int64
	transient int64
}

type channelBreaker struct {
	mu              sync.Mutex
	state           breakerState
	buckets         [breakerBucketCount]breakerBucket
	openUntil       time.Time
	cooldown        time.Duration // current (backed-off) cooldown
	consecFatal     int
	halfOpenSuccess int
}

var (
	breakers   = make(map[int]*channelBreaker)
	breakersMu sync.RWMutex
)

// InitChannelBreakerConfig loads breaker configuration after process env and
// .env files have been initialized. Keep validation defensive: invalid zero or
// negative values fall back to safe defaults instead of panicking in bucket math
// or leaving channels permanently open.
func InitChannelBreakerConfig() {
	breakerEnabled = common.GetEnvOrDefaultBool("CHANNEL_BREAKER_ENABLED", true)
	breakerWindow = positiveSecondsEnv("CHANNEL_BREAKER_WINDOW_SECONDS", 120)
	breakerMinSamples = positiveIntEnv("CHANNEL_BREAKER_MIN_SAMPLES", 10)
	breakerTripScorePct = boundedIntEnv("CHANNEL_BREAKER_TRIP_SCORE_PCT", 50, 1, 100)
	breakerConsecFatal = positiveIntEnv("CHANNEL_BREAKER_CONSECUTIVE_FATAL", 3)
	breakerCooldown = positiveSecondsEnv("CHANNEL_BREAKER_COOLDOWN_SECONDS", 60)
	breakerMaxCooldown = positiveSecondsEnv("CHANNEL_BREAKER_MAX_COOLDOWN_SECONDS", 1800)
	if breakerMaxCooldown < breakerCooldown {
		breakerMaxCooldown = breakerCooldown
	}
	breakerHalfOpenProbes = positiveIntEnv("CHANNEL_BREAKER_HALFOPEN_PROBES", 3)
}

func positiveIntEnv(name string, fallback int) int {
	value := common.GetEnvOrDefault(name, fallback)
	if value <= 0 {
		common.SysError("invalid " + name + ", using default value: " + strconv.Itoa(fallback))
		return fallback
	}
	return value
}

func boundedIntEnv(name string, fallback, minValue, maxValue int) int {
	value := common.GetEnvOrDefault(name, fallback)
	if value < minValue || value > maxValue {
		common.SysError("invalid " + name + ", using default value: " + strconv.Itoa(fallback))
		return fallback
	}
	return value
}

func positiveSecondsEnv(name string, fallback int) time.Duration {
	return time.Duration(positiveIntEnv(name, fallback)) * time.Second
}

func getBreaker(channelID int) *channelBreaker {
	breakersMu.RLock()
	b := breakers[channelID]
	breakersMu.RUnlock()
	if b != nil {
		return b
	}
	breakersMu.Lock()
	defer breakersMu.Unlock()
	if b = breakers[channelID]; b == nil {
		b = &channelBreaker{state: breakerClosed, cooldown: breakerCooldown}
		breakers[channelID] = b
	}
	return b
}

func breakerBucketDur() time.Duration {
	return breakerWindow / breakerBucketCount
}

// bucketEpoch returns the window-bucket index for t.
func bucketEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(breakerBucketDur())
}

// recordLocked adds one sample to the current ring bucket (caller holds mu).
func (b *channelBreaker) recordLocked(now time.Time, success, fatal bool) {
	epoch := bucketEpoch(now)
	slot := epoch % breakerBucketCount
	bk := &b.buckets[slot]
	if bk.epoch != epoch {
		*bk = breakerBucket{epoch: epoch}
	}
	bk.total++
	if !success {
		if fatal {
			bk.fatal++
		} else {
			bk.transient++
		}
	}
}

// windowLocked sums the ring buckets that fall inside the sliding window.
func (b *channelBreaker) windowLocked(now time.Time) (total, fatal, transient int64) {
	minEpoch := bucketEpoch(now) - (breakerBucketCount - 1)
	for i := range b.buckets {
		bk := b.buckets[i]
		if bk.epoch >= minEpoch {
			total += bk.total
			fatal += bk.fatal
			transient += bk.transient
		}
	}
	return
}

func (b *channelBreaker) clearWindowLocked() {
	for i := range b.buckets {
		b.buckets[i] = breakerBucket{}
	}
}

// tripLocked moves the channel to Open with the current (then backed-off) cooldown.
func (b *channelBreaker) tripLocked(now time.Time) {
	b.state = breakerOpen
	b.openUntil = now.Add(b.cooldown)
	// exponential backoff for the next trip, capped.
	next := b.cooldown * 2
	if next > breakerMaxCooldown {
		next = breakerMaxCooldown
	}
	b.cooldown = next
	b.halfOpenSuccess = 0
}

func (b *channelBreaker) report(success, fatal bool) {
	now := breakerClock()
	b.mu.Lock()
	defer b.mu.Unlock()

	b.recordLocked(now, success, fatal)
	if success {
		b.consecFatal = 0
	} else if fatal {
		b.consecFatal++
	}

	switch b.state {
	case breakerHalfOpen:
		if success {
			b.halfOpenSuccess++
			if b.halfOpenSuccess >= breakerHalfOpenProbes {
				// recovered
				b.state = breakerClosed
				b.cooldown = breakerCooldown
				b.consecFatal = 0
				b.clearWindowLocked()
			}
		} else {
			// probe failed: re-open with backoff
			b.tripLocked(now)
		}
	case breakerClosed:
		if b.shouldTripLocked(now) {
			b.cooldown = breakerCooldown
			b.tripLocked(now)
		}
	case breakerOpen:
		// requests should be excluded while open; a stray report does not change
		// state until cooldown elapses (handled lazily in evalLocked).
	}
}

func (b *channelBreaker) shouldTripLocked(now time.Time) bool {
	if b.consecFatal >= breakerConsecFatal {
		return true
	}
	total, fatal, transient := b.windowLocked(now)
	if total < int64(breakerMinSamples) {
		return false
	}
	score := (float64(fatal)*1.0 + float64(transient)*0.3) / float64(total)
	return score*100 >= float64(breakerTripScorePct)
}

// evalLocked applies the lazy time-based Open -> HalfOpen transition and returns
// the effective state (caller holds mu).
func (b *channelBreaker) evalLocked(now time.Time) breakerState {
	if b.state == breakerOpen && !now.Before(b.openUntil) {
		b.state = breakerHalfOpen
		b.halfOpenSuccess = 0
	}
	return b.state
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// ReportRelayResult records the outcome of a relay attempt against a channel and
// drives the breaker state machine. A nil error is a success; channel-side
// failures are weighted as fatal, retryable transient errors are weighted low,
// and client errors (400/invalid/sensitive) are ignored so they do not penalize
// the channel.
func ReportRelayResult(channelID int, apiErr *types.StuHelperAIError) {
	if !breakerEnabled || channelID <= 0 {
		return
	}
	if apiErr == nil {
		getBreaker(channelID).report(true, false)
		return
	}
	classification := ClassifyRelayError(apiErr)
	if classification.ChannelSide {
		getBreaker(channelID).report(false, true)
		return
	}
	if classification.Transient {
		getBreaker(channelID).report(false, false)
		return
	}
	// request-invalid / non-retryable client error: not the channel's fault.
}

// BreakerOpenChannelIDs returns the set of channels currently in the Open state
// (cooling down). Half-open channels are intentionally excluded from this set so
// a probe request can flow through and test recovery.
func BreakerOpenChannelIDs() map[int]struct{} {
	if !breakerEnabled {
		return nil
	}
	now := breakerClock()
	breakersMu.RLock()
	defer breakersMu.RUnlock()
	var open map[int]struct{}
	for id, b := range breakers {
		b.mu.Lock()
		state := b.evalLocked(now)
		b.mu.Unlock()
		if state == breakerOpen {
			if open == nil {
				open = make(map[int]struct{})
			}
			open[id] = struct{}{}
		}
	}
	return open
}

// BreakerStateName returns a human-readable breaker state for a channel, for
// admin display ("closed"/"open"/"half_open").
func BreakerStateName(channelID int) string {
	if !breakerEnabled {
		return "disabled"
	}
	now := breakerClock()
	breakersMu.RLock()
	b := breakers[channelID]
	breakersMu.RUnlock()
	if b == nil {
		return "closed"
	}
	b.mu.Lock()
	state := b.evalLocked(now)
	b.mu.Unlock()
	switch state {
	case breakerOpen:
		return "open"
	case breakerHalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}

// ResetChannelBreaker clears any breaker state for a channel (e.g. after a manual
// enable), returning it to closed.
func ResetChannelBreaker(channelID int) {
	breakersMu.Lock()
	delete(breakers, channelID)
	breakersMu.Unlock()
	if breakerEnabled {
		common.SysLog("channel breaker reset for channel " + strconv.Itoa(channelID))
	}
}
