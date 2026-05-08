package balancer

import (
	"math/rand/v2"
	"time"
)

type Option func(b *Balancer)

// WithLogger sets logger for the candidate. In most cases leader election
// process recommended to remain silent. Can be used for debug purposes.
//
// Default: noop logger
func WithLogger(logger Logger) Option {
	return func(b *Balancer) {
		b.logger = logger
	}
}

// WithCheckRate sets keys check interval, with additional randomness.
//
// Default: 1 second, 500 millisecond
func WithCheckRate(interval time.Duration, addRand time.Duration) Option {
	return func(b *Balancer) {
		b.checkRate = interval + time.Duration(rand.Int64N(int64(addRand)))
	}
}
