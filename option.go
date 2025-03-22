package less

import "time"

type Option func(c *Candidate)

func WithTTL(ttl time.Duration) Option {
	return func(c *Candidate) {
		c.ttl = ttl
	}
}

func WithApplyRate(rate time.Duration) Option {
	return func(c *Candidate) {
		c.applyRate = rate
	}
}

func WithRenewRate(rate time.Duration) Option {
	return func(c *Candidate) {
		c.renewRate = rate
	}
}
