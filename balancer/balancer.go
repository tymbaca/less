package balancer

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tymbaca/less/logger"
)

type Storage interface {
	List(ctx context.Context, keys []string) (map[string]string, error)
}

type Logger = logger.Logger

type Balancer struct {
	id        string
	nodeCount int // amount of nodes that application runs on
	storage   Storage
	checkRate time.Duration

	mu      sync.Mutex
	jobKeys []string

	dropCh chan struct{}
	logger Logger
}

func New(ctx context.Context, nodeCount int, storage Storage, opts ...Option) *Balancer {
	bal := &Balancer{
		id:        uuid.NewString(),
		nodeCount: nodeCount,
		storage:   storage,
		checkRate: 1*time.Second + time.Duration(rand.IntN(500))*time.Millisecond,

		dropCh: make(chan struct{}),
		logger: logger.NoopLogger{},
	}

	for _, opt := range opts {
		opt(bal)
	}

	go listen(ctx, bal)

	return bal
}

func (b *Balancer) ID() string {
	return b.id
}

func (b *Balancer) Register(job string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.jobKeys = append(b.jobKeys, job)
}

func (b *Balancer) Drop() chan struct{} {
	return b.dropCh
}

func listen(ctx context.Context, b *Balancer) {
	defer close(b.dropCh)

	for run := true; run; run = tick(ctx, b.checkRate) {
		b.mu.Lock()
		jobKeys := b.jobKeys
		b.mu.Unlock()

		ids, err := b.storage.List(ctx, jobKeys)
		if err != nil {
			b.logger.Error("can't get keys", "keys", jobKeys, "err", err)
		}

		myJobCount := 0
		for _, id := range ids {
			if id == b.id {
				myJobCount++
			}
		}

		toDrop := needDrop(len(jobKeys), myJobCount, b.nodeCount)
		if toDrop > 0 {
			b.logger.Info("must drop leaders", "leadersToDrop", toDrop)
		}

		for range toDrop {
			select {
			case b.dropCh <- struct{}{}:
				b.logger.Debug("sent drop signal")
			case <-time.After(5 * time.Second):
				b.logger.Error("timeout exceeded when sending drop message", "totalJobCount", len(jobKeys), "myJobCount", myJobCount, "nodeCount", b.nodeCount)
				continue
			}
		}
	}
}

func needDrop(totalJobCount int, myJobCount int, nodeCount int) int {
	// add 1 to guarantee the all jobs will be covered even if job count has a remainder after dividing to node count
	canHave := (totalJobCount / nodeCount) + 1
	mustDrop := myJobCount - canHave

	if mustDrop <= 0 {
		return 0
	}

	return mustDrop
}

func tick(ctx context.Context, interval time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(interval):
		return true
	}
}
