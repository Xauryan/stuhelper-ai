package service

import (
	"strconv"
	"sync"
	"time"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/types"
)

const availabilityBucketCount = 12

var (
	channelAvailabilityWindow = 10 * time.Minute
	availabilityClock         = time.Now
)

type availabilityBucket struct {
	epoch     int64
	success   int64
	channel   int64
	transient int64
	ignored   int64
}

type channelAvailability struct {
	mu            sync.Mutex
	buckets       [availabilityBucketCount]availabilityBucket
	lastSuccessAt int64
	lastFailureAt int64
	lastError     string
	lastClass     RetryClass
}

var (
	channelAvailabilityByID = make(map[int]*channelAvailability)
	channelAvailabilityMu   sync.RWMutex
)

// InitChannelAvailabilityConfig loads the telemetry window after .env and the
// process environment have been initialized.
func InitChannelAvailabilityConfig() {
	channelAvailabilityWindow = positiveSecondsEnv("CHANNEL_AVAILABILITY_WINDOW_SECONDS", 600)
}

func getChannelAvailability(channelID int) *channelAvailability {
	channelAvailabilityMu.RLock()
	a := channelAvailabilityByID[channelID]
	channelAvailabilityMu.RUnlock()
	if a != nil {
		return a
	}
	channelAvailabilityMu.Lock()
	defer channelAvailabilityMu.Unlock()
	if a = channelAvailabilityByID[channelID]; a == nil {
		a = &channelAvailability{}
		channelAvailabilityByID[channelID] = a
	}
	return a
}

func availabilityBucketDur() time.Duration {
	dur := channelAvailabilityWindow / availabilityBucketCount
	if dur <= 0 {
		return time.Second
	}
	return dur
}

func availabilityBucketEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(availabilityBucketDur())
}

func (a *channelAvailability) record(now time.Time, classification RetryClassification, err *types.StuHelperAIError) {
	a.mu.Lock()
	defer a.mu.Unlock()

	epoch := availabilityBucketEpoch(now)
	slot := epoch % availabilityBucketCount
	bucket := &a.buckets[slot]
	if bucket.epoch != epoch {
		*bucket = availabilityBucket{epoch: epoch}
	}

	switch {
	case err == nil:
		bucket.success++
		a.lastSuccessAt = now.Unix()
	case classification.ChannelSide:
		bucket.channel++
		a.lastFailureAt = now.Unix()
		a.lastClass = classification.Class
		a.lastError = common.LocalLogPreview(err.InternalError())
	case classification.Transient:
		bucket.transient++
		a.lastFailureAt = now.Unix()
		a.lastClass = classification.Class
		a.lastError = common.LocalLogPreview(err.InternalError())
	default:
		bucket.ignored++
	}
}

func (a *channelAvailability) snapshot(now time.Time) *types.ChannelAvailabilitySnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	minEpoch := availabilityBucketEpoch(now) - (availabilityBucketCount - 1)
	snapshot := &types.ChannelAvailabilitySnapshot{
		WindowSeconds: int64(channelAvailabilityWindow.Seconds()),
		LastSuccessAt: a.lastSuccessAt,
		LastFailureAt: a.lastFailureAt,
		LastError:     a.lastError,
		LastClass:     string(a.lastClass),
	}
	for i := range a.buckets {
		bucket := a.buckets[i]
		if bucket.epoch < minEpoch {
			continue
		}
		snapshot.Success += bucket.success
		snapshot.ChannelFailures += bucket.channel
		snapshot.TransientFailures += bucket.transient
		snapshot.Ignored += bucket.ignored
	}
	snapshot.Total = snapshot.Success + snapshot.ChannelFailures + snapshot.TransientFailures
	if snapshot.Total > 0 {
		snapshot.SuccessRate = float64(snapshot.Success) / float64(snapshot.Total)
	}
	return snapshot
}

// ReportChannelAvailability records relay outcome telemetry without affecting
// channel selection. Client-side and skip-retry errors are counted as ignored
// samples because they do not describe upstream channel health.
func ReportChannelAvailability(channelID int, apiErr *types.StuHelperAIError) {
	if channelID <= 0 {
		return
	}
	classification := RetryClassification{Class: RetryClassNone}
	if apiErr != nil {
		classification = ClassifyRelayError(apiErr)
	}
	getChannelAvailability(channelID).record(availabilityClock(), classification, apiErr)
}

func ChannelAvailabilitySnapshot(channelID int) *types.ChannelAvailabilitySnapshot {
	if channelID <= 0 {
		return nil
	}
	channelAvailabilityMu.RLock()
	a := channelAvailabilityByID[channelID]
	channelAvailabilityMu.RUnlock()
	if a == nil {
		return &types.ChannelAvailabilitySnapshot{
			WindowSeconds: int64(channelAvailabilityWindow.Seconds()),
		}
	}
	return a.snapshot(availabilityClock())
}

func ResetChannelAvailability(channelID int) {
	channelAvailabilityMu.Lock()
	delete(channelAvailabilityByID, channelID)
	channelAvailabilityMu.Unlock()
	common.SysLog("channel availability reset for channel " + strconv.Itoa(channelID))
}
