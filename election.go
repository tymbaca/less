// Package less provides storage agnostic leader election.
package less

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Storage is a generic way to access the data storage shared between the
// candidates.
type Storage interface {
	// Renew sets the new deadline for the record with provided key.
	// If such record is not created yet, it does nothing and returns nil.
	Renew(ctx context.Context, key string, deadline time.Time) error

	// Get gets the record value with provided key. If record is not created
	// or expired, empty string will be returned without error.
	Get(ctx context.Context, key string) (string, error)

	// SetNX creates/sets record on provided key only if it's not created
	// yet or if it's expired. Returns true if record was successfully
	// created/set, otherwise returns false.
	SetNX(ctx context.Context, key, val string, deadline time.Time) (bool, error)
}

// Candidate constantly tries to acquire the leadership. Once acquire, it tries
// to renew it's leader record to not lose it.
type Candidate struct {
	id       string
	isLeader atomic.Bool

	storage Storage
	key     string

	ttl        time.Duration
	followRate time.Duration
	holdRate   time.Duration

	errsToFallback int
	logger         Logger
}

// New creates and launches the [Candidate] with default settings.
// Settings can be changed with [Option]s.
func New(ctx context.Context, storage Storage, opts ...Option) *Candidate {
	cand := &Candidate{
		id:       uuid.New().String(),
		isLeader: atomic.Bool{},

		storage: storage,
		key:     "default",

		ttl:        10 * time.Second,
		followRate: 2 * time.Second,
		holdRate:   2 * time.Second,

		errsToFallback: 3,
		logger:         noopLogger{},
	}

	for _, opt := range opts {
		opt(cand)
	}

	if cand.errsToFallback <= 0 {
		cand.errsToFallback = 1
	}

	go follow(ctx, cand)

	return cand
}

// IsLeader returns whether this candidate is currently a leader or not.
func (c *Candidate) IsLeader() bool {
	return c.isLeader.Load()
}

func follow(ctx context.Context, cand *Candidate) {
	cand.logger.Debug("following", "id", cand.id)

	for run := true; run; run = tick(ctx, cand.followRate) {
		cand.logger.Debug("try to set", "id", cand.id)

		ok, err := cand.storage.SetNX(ctx, cand.key, cand.id, time.Now().Add(cand.ttl))
		if err != nil {
			cand.logger.Error("can't setnx", "id", cand.id, "err", err)
			continue
		}
		cand.logger.Debug("setnx", "id", cand.id, "ok", ok)

		if ok {
			cand.logger.Info("we acquired leadership", "id", cand.id)
			cand.isLeader.Store(true)
			hold(ctx, cand)
		}
	}
}

func hold(ctx context.Context, cand *Candidate) {
	errCount := 0

	for run := true; run && errCount < cand.errsToFallback; run = tick(ctx, cand.holdRate) {
		err := cand.storage.Renew(ctx, cand.key, time.Now().Add(cand.ttl))
		if err != nil {
			cand.logger.Error("can't renew", "id", cand.id, "err", err)
			errCount++
			continue
		}

		current, err := cand.storage.Get(ctx, cand.key)
		if err != nil {
			cand.logger.Error("can't get", "id", cand.id, "err", err)
			errCount++
			continue
		}

		if current != cand.id {
			break
		}

		// if we got here after some errors, we can forget about them
		errCount = 0
	}

	cand.logger.Warn("we lost leadership", "id", cand.id)
	cand.isLeader.Store(false)
}

func tick(ctx context.Context, interval time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(interval):
		return true
	}
}
