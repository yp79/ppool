package ppool

import "time"

// Backoff is a list of time.Duration values to be returned on consecutive
// calls to Backoff.Duration()
type Backoff []time.Duration

// Duration returns time to wait before retrying and a stop flag.
// -1 duration marks termination. If no -1 value is in the Backoff slice
// it will be returning last duration indefinitely.
func (b *Backoff) Duration() (time.Duration, bool) {
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
