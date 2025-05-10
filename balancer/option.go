package balancer

type Option func(b *Balancer)

// WithLogger sets logger for the candidate. In most cases leader election
// process recommended to remain silent. Can be used for debug purposes.
//
// Default: noop logger
func WithLogger(logger Logger) Option {
	return func(b *Balancer) {
		b.logger = logger
	}
}
