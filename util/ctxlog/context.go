package ctxlog

import (
	"context"

	log "gopkg.in/src-d/go-log.v1"
)

type ctxLogger int

// LoggerKey is the key that holds logger in a context.
const LoggerKey ctxLogger = 0

// Get returns logger from context or DefaultLogger if there is no logger in context
func Get(ctx context.Context) log.Logger {
	if v := ctx.Value(LoggerKey); v != nil {
		return v.(log.Logger)
	}

	if log.DefaultLogger == nil {
		log.DefaultLogger = log.New(nil)
	}
	return log.DefaultLogger
}

// WithLogFields returns context with new logger and the logger
func WithLogFields(ctx context.Context, f log.Fields) (context.Context, log.Logger) {
	logger := Get(ctx).With(f)
	return Set(ctx, logger), logger
}

// Set puts logger into context and returns new context
func Set(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}
