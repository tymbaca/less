package less

type Balancer interface {
	Register(key string)
	CanBeLeader() bool
}

type noopBalancer struct{}

func (no noopBalancer) Register(key string) {}

func (no noopBalancer) CanBeLeader() bool {
	return true
}
