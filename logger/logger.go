package logger

// Logger accepts even amount of kvargs:
// logger.Info("something", "key1", "val1", "key2", "val2", "key3", "val3")
type Logger interface {
	Debug(msg string, kvargs ...any)
	Info(msg string, kvargs ...any)
	Warn(msg string, kvargs ...any)
	Error(msg string, kvargs ...any)
}

type NoopLogger struct{}

func (no NoopLogger) Debug(msg string, kvargs ...any) {}
func (no NoopLogger) Info(msg string, kvargs ...any)  {}
func (no NoopLogger) Warn(msg string, kvargs ...any)  {}
func (no NoopLogger) Error(msg string, kvargs ...any) {}
