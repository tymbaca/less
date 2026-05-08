// Package less provides storage agnostic leader election.
package less

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tymbaca/less/logger"
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

type Logger = logger.Logger

// Candidate constantly tries to acquire the leadership. Once acquire, it tries
// to renew it's leader record to not lose it.
type Candidate struct {
	id       string
	isLeader atomic.Bool
	balancer Balancer

	storage Storage
	key     string

	ttl        time.Duration
	followRate time.Duration
	holdRate   time.Duration

	cooldown    time.Time
	cooldownDur time.Duration

	errsToFallback int
	logger         logger.Logger
}

// New creates and launches the [Candidate] with default settings.
// Settings can be changed with [Option]s.
func New(ctx context.Context, storage Storage, opts ...Option) *Candidate {
	cand := &Candidate{
		isLeader: atomic.Bool{},
		balancer: noopBalancer{}, // cannot be nil

		storage: storage,
		key:     "default",

		ttl:        10 * time.Second,
		followRate: 1 * time.Second,
		holdRate:   1 * time.Second, // TODO: add random

		cooldownDur: 1 * time.Second,

		errsToFallback: 3,
		logger:         logger.NoopLogger{},
	}

	for _, opt := range opts {
		opt(cand)
	}

	cand.balancer.Register(cand.key)
	cand.id = cand.balancer.ID()

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

	for run := true; run; run = tickFollow(ctx, cand) {
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

			if cand.cooldown.After(time.Now()) {
				sleep(ctx, time.Until(cand.cooldown))
			}
		}
	}
}

func sleep(ctx context.Context, dur time.Duration) {
	select {
	case <-time.After(dur):
	case <-ctx.Done():
	}
}

func hold(ctx context.Context, cand *Candidate) {
	errCount := 0

	for run := true; run && errCount < cand.errsToFallback; run = tickHold(ctx, cand) {
		// FIX: add timeout, less then ttl
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

	// expire the key, so other candidates will be able to acquire leadership
	err := cand.storage.Renew(ctx, cand.key, time.Now())
	if err != nil {
		cand.logger.Error("can't renew", "id", cand.id, "err", err)
	}
}

func tickFollow(ctx context.Context, cand *Candidate) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(cand.followRate):
		return true
	}
}

func tickHold(ctx context.Context, cand *Candidate) bool {
	select {
	case <-ctx.Done():
		return false
	case <-cand.balancer.Drop():
		cand.cooldown = time.Now().Add(cand.cooldownDur)
		cand.logger.Info("drop signal from balancer")
		return false
	case <-time.After(cand.holdRate):
		return true
	}
}
