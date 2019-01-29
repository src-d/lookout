package ctxlog

import (
	"context"

	log "gopkg.in/src-d/go-log.v1"
)

// NewLogger function to create new log.Logger, needed for mocking in tests
var NewLogger = log.New

type ctxKey int

// logFieldsKey is the key that holds log Fields in a context.
const logFieldsKey ctxKey = 0

// Get returns a logger configured with the context log Fields or the default fields
func Get(ctx context.Context) log.Logger {
	var fields log.Fields

	if v := ctx.Value(logFieldsKey); v != nil {
		fields = v.(log.Fields)
	}

	return NewLogger(fields)
}

// Fields returns a copy of the context log fields. It can be nil
func Fields(ctx context.Context) log.Fields {
	if v := ctx.Value(logFieldsKey); v != nil {
		f := v.(log.Fields)
		copy := make(map[string]interface{}, len(f))

		for key, val := range f {
			copy[key] = val
		}

		return copy
	}

	return nil
}

// WithLogFields returns a context with new logger Fields added to the current
// ones, and a logger
func WithLogFields(ctx context.Context, fields log.Fields) (context.Context, log.Logger) {
	if fields == nil {
		return ctx, Get(ctx)
	}

	newFields := make(map[string]interface{}, len(fields))

	if v := ctx.Value(logFieldsKey); v != nil {
		for key, val := range v.(log.Fields) {
			newFields[key] = val
		}
	}

	for key, val := range fields {
		newFields[key] = val
	}

	ctx = set(ctx, newFields)
	return ctx, Get(ctx)
}

// set returns a copy of ctx with the log Fields saves as context Value
func set(ctx context.Context, fields log.Fields) context.Context {
	return context.WithValue(ctx, logFieldsKey, fields)
}
