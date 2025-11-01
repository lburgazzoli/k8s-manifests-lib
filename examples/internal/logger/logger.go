package logger

import (
	"context"
	"fmt"
	"os"
)

// Logger interface compatible with testing.T.
type Logger interface {
	Log(args ...any)
	Logf(format string, args ...any)
}

type contextKey struct{}

// WithLogger adds logger to context.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves logger from context, returns StdoutLogger if not found.
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(contextKey{}).(Logger); ok {
		return logger
	}

	return &StdoutLogger{}
}

// StdoutLogger writes to os.Stdout.
type StdoutLogger struct{}

// Log prints arguments to stdout.
func (l *StdoutLogger) Log(args ...any) {
	_, _ = fmt.Fprintln(os.Stdout, args...)
}

// Logf prints formatted string to stdout.
func (l *StdoutLogger) Logf(format string, args ...any) {
	_, _ = fmt.Fprintln(os.Stdout, fmt.Sprintf(format, args...))
}
