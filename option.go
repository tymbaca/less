package less

import "time"

type Option func(c *Candidate)

// WithID sets the current candidate's ID, wich will be stored in the leader
// record when he acquires the leadership. If you use this, make sure that all
// candidates will have diffent IDs (unless you know what you are doing).
func WithID(id string) Option {
	return func(c *Candidate) {
		c.id = id
	}
}

// WithKey sets the key name in which the leader record will be stored.
// Can be used when you want split the leadership for different
// asynchronous job (e.g. "job1", "job2", etc).
func WithKey(key string) Option {
	return func(c *Candidate) {
		c.key = key
	}
}

// WithTTL sets the time-to-live time for both [Storage.SetNX] (when candidate
// acquires the leadership) and [Storage.Renew] (when leader holds his leadership) calls.
func WithTTL(ttl time.Duration) Option {
	return func(c *Candidate) {
		c.ttl = ttl
	}
}

// WithFollowRate sets the interval at which candidate will call [Storage.SetNX] to set
// his id and acquire the leadership.
func WithFollowRate(rate time.Duration) Option {
	return func(c *Candidate) {
		c.followRate = rate
	}
}

// WithHoldRate sets the interval at which leader will call [Storage.Renew] to
// prolong his leadership.
func WithHoldRate(rate time.Duration) Option {
	return func(c *Candidate) {
		c.holdRate = rate
	}
}
