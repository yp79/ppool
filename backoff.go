package ppool

import "time"

// Backoff is an interface with a single method to return duration for
// the delay before retrying.
type Backoff interface {
	Duration() (time.Duration, bool)
}

// BackoffSimple implements simple backoff strategy as a slice of durations.
// -1 duration marks termination. If no -1 value is in the slice it will be
// returning last duration indefinitely.
type BackoffSimple []time.Duration

// Duration returns time to wait before retrying and a stop flag.
func (b *BackoffSimple) Duration() (time.Duration, bool) {
	if len(*b) == 0 {
		return 0, true
	}

	d := (*b)[0]
	if d == -1 {
		return 0, true
	}

	if len(*b) > 1 {
		*b = (*b)[1:]
	}

	return d, false
}
