package less

import "github.com/google/uuid"

type Balancer interface {
	ID() string
	Register(key string)
	// Drop returns the channel then caller must check for messages. If
	// caller got message from channel, then he must drop the leadership
	Drop() chan struct{}
}

type noopBalancer struct{}

func (no noopBalancer) ID() string {
	return uuid.NewString()
}

func (no noopBalancer) Register(key string) {}

func (no noopBalancer) CanBeLeader() bool {
	return true
}

func (no noopBalancer) Drop() chan struct{} {
	return nil
}
