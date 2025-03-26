package balancer

import (
	"sync"
)

type Storage interface{}

type Balancer struct {
	ids       []string
	nodeCount int // amount of nodes that application runs on
	storage   Storage

	mu          sync.Mutex
	keys        []string
	canBeLeader bool
}

func New(nodeCount int, storage Storage) *Balancer {
	return &Balancer{
		// id:        uuid.NewString(),
		nodeCount: nodeCount,
		storage:   storage,
	}
}

// func (b *Balancer) ID() string {
// 	return b.id
// }

func (b *Balancer) Register(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.keys = append(b.keys, key)
}

func (b *Balancer) CanBeLeader() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.canBeLeader
}
