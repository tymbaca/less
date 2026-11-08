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
	List(ctx context.Context, keys []string) (keysToNodes map[string]string, err error)
	ListNodes(ctx context.Context) (activeNodes []string, err error)
	RenewLiveness(ctx context.Context, node string, deadline time.Time) error
}

type Logger = logger.Logger

type Balancer struct {
	ctx    context.Context
	cancel context.CancelFunc

	id              string
	storage         Storage
	checkRate       time.Duration
	checkJitter     time.Duration
	livenessRate    time.Duration
	livenessTimeout time.Duration

	mu      sync.Mutex
	jobKeys []string

	dropCh chan struct{}
	logger Logger
}

func New(ctx context.Context, storage Storage, opts ...Option) *Balancer {
	ctx, cancel := context.WithCancel(ctx)

	bal := &Balancer{
		ctx:    ctx,
		cancel: cancel,

		id:              uuid.NewString(),
		storage:         storage,
		checkRate:       1 * time.Second,
		checkJitter:     500 * time.Millisecond,
		livenessRate:    1 * time.Second,
		livenessTimeout: 5 * time.Second,

		dropCh: make(chan struct{}),
		logger: logger.NoopLogger{},
	}

	for _, opt := range opts {
		opt(bal)
	}

	go renewLiveness(bal)
	go listen(bal)

	return bal
}

// Close closes internal context. You can cancel the original context
// passed to [New] instead of calling this.
func (b *Balancer) Close() error {
	b.cancel()

	return nil
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

func renewLiveness(b *Balancer) {
	ctx := b.ctx

	for run := true; run; run = tick(ctx, b.livenessTimeout) {
		err := b.storage.RenewLiveness(ctx, b.id, time.Now().Add(b.livenessTimeout))
		if err != nil {
			b.logger.Error("can't renew liveness", "err", err)
		}
	}
}

func listen(b *Balancer) {
	defer close(b.dropCh)
	ctx := b.ctx

	err := b.storage.RenewLiveness(ctx, b.id, time.Now().Add(b.livenessTimeout))
	if err != nil {
		b.logger.Error("can't renew liveness", "err", err)
	}

	for run := true; run; run = tick(ctx, b.checkRate+time.Duration(rand.Int64N(int64(b.checkJitter)))) {
		b.mu.Lock()
		jobKeys := b.jobKeys
		b.mu.Unlock()

		jobsToNodes, err := b.storage.List(ctx, jobKeys)
		if err != nil {
			b.logger.Error("can't get keys", "keys", jobKeys, "err", err)
			continue
		}

		activeNodes, err := b.storage.ListNodes(ctx)
		if err != nil {
			b.logger.Error("can't get active nodes", "keys", jobKeys, "err", err)
			continue
		}

		myJobCount := 0
		for _, node := range jobsToNodes {
			if node == b.id {
				myJobCount++
			}
		}

		toDrop := needDrop(len(jobKeys), myJobCount, len(activeNodes))
		if toDrop > 0 {
			b.logger.Info("must drop leaders", "leadersToDrop", toDrop)
		}

		for range toDrop {
			select {
			case b.dropCh <- struct{}{}:
				b.logger.Debug("sent drop signal")
			case <-time.After(5 * time.Second):
				b.logger.Error("timeout exceeded when sending drop message", "totalJobCount", len(jobKeys), "myJobCount", myJobCount, "nodeCount", len(activeNodes))
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
