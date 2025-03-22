package less

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Implementation is responsible for internal (per call) retries
type Storage interface {
	Renew(ctx context.Context, key string, deadline time.Time) error
	Get(ctx context.Context, key string) (string, error)
	SetNX(ctx context.Context, key, val string, deadline time.Time) (bool, error)
}

type Candidate struct {
	id       string
	isLeader atomic.Bool

	storage Storage
	key     string

	ttl       time.Duration
	applyRate time.Duration
	renewRate time.Duration
}

func New(ctx context.Context, storage Storage, opts ...Option) *Candidate {
	cand := &Candidate{
		id:       uuid.New().String(),
		isLeader: atomic.Bool{},

		storage: storage,
		key:     "leader",

		ttl:       10 * time.Second,
		applyRate: 2 * time.Second,
		renewRate: 2 * time.Second,
	}

	for _, opt := range opts {
		opt(cand)
	}

	go follow(ctx, cand)

	return cand
}

func follow(ctx context.Context, cand *Candidate) {
	for {
		// TODO: move to bottom
		select {
		case <-ctx.Done():
			return
		case <-time.After(cand.applyRate):
		}

		ok, err := cand.storage.SetNX(ctx, cand.key, cand.id, time.Now().Add(cand.ttl))
		if err != nil {
			slog.Error("can't setnx", "err", err)
			continue
		}

		if ok {
			slog.Info("we acquired leadership")
			cand.isLeader.Store(true)
			hold(ctx, cand)
		}
	}
}

func hold(ctx context.Context, cand *Candidate) {
	errCount := 0

	for {
		if errCount >= 3 {
			slog.Warn("we lost leadership")
			cand.isLeader.Store(false)
			return
		}

		err := cand.storage.Renew(ctx, cand.key, time.Now().Add(cand.ttl))
		if err != nil {
			// TODO: if we always get error - node will think it has
			// leadership
			slog.Error("can't renew", "err", err)
			errCount++
			continue
		}

		current, err := cand.storage.Get(ctx, cand.key)
		if err != nil {
			slog.Error("can't get", "err", err)
			errCount++
			continue
		}

		if current != cand.id {
			slog.Warn("we lost leadership")
			cand.isLeader.Store(false)
			return
		}
	}
}

func (c *Candidate) IsLeader() bool {
	return c.isLeader.Load()
}
