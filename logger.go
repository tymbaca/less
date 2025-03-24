package less

// Logger accepts even amount of kvargs:
// logger.Info("something", "key1", "val1", "key2", "val2", "key3", "val3")
type Logger interface {
	Debug(msg string, kvargs ...any)
	Info(msg string, kvargs ...any)
	Warn(msg string, kvargs ...any)
	Error(msg string, kvargs ...any)
}

type noopLogger struct{}

func (no noopLogger) Debug(msg string, kvargs ...any) {}
func (no noopLogger) Info(msg string, kvargs ...any)  {}
func (no noopLogger) Warn(msg string, kvargs ...any)  {}
func (no noopLogger) Error(msg string, kvargs ...any) {}
