package ctxlog

import (
	"context"

	log "gopkg.in/src-d/go-log.v1"
)

type ctxKey int

// logFieldsKey is the key that holds log Fields in a context.
const logFieldsKey ctxKey = 0

// Get returns a logger configured with the context log Fields or the default fields
func Get(ctx context.Context) log.Logger {
	var fields log.Fields

	if v := ctx.Value(logFieldsKey); v != nil {
		fields = v.(log.Fields)
	}

	return log.New(fields)
}

// WithLogFields returns a context with new logger Fields added to the current
// ones, and a logger
func WithLogFields(ctx context.Context, fields log.Fields) (context.Context, log.Logger) {
	var newFields log.Fields

	if v := ctx.Value(logFieldsKey); v != nil {
		newFields = v.(log.Fields)

		for k, v := range fields {
			newFields[k] = v
		}
	} else {
		newFields = fields
	}

	ctx = set(ctx, newFields)
	return ctx, Get(ctx)
}

// set returns a copy of ctx with the log Fields saves as context Value
func set(ctx context.Context, fields log.Fields) context.Context {
	return context.WithValue(ctx, logFieldsKey, fields)
}
