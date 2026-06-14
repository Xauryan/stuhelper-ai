package types

// ChannelAvailabilitySnapshot is a process-local, read-only telemetry snapshot
// for recent relay attempts against one channel. It is attached to admin
// channel-list responses and is intentionally not persisted.
type ChannelAvailabilitySnapshot struct {
	WindowSeconds     int64   `json:"window_seconds"`
	Total             int64   `json:"total"`
	Success           int64   `json:"success"`
	ChannelFailures   int64   `json:"channel_failures"`
	TransientFailures int64   `json:"transient_failures"`
	Ignored           int64   `json:"ignored"`
	SuccessRate       float64 `json:"success_rate"`
	LastSuccessAt     int64   `json:"last_success_at,omitempty"`
	LastFailureAt     int64   `json:"last_failure_at,omitempty"`
	LastError         string  `json:"last_error,omitempty"`
	LastClass         string  `json:"last_class,omitempty"`
}
