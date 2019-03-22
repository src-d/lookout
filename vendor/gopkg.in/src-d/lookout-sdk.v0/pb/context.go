package pb

import (
	"context"
)

type ctxKey int

// logFieldsKeyContext is the key that holds the log fields in a context.
const logFieldsKeyContext ctxKey = 0

// Fields is a log fields container
type Fields map[string]interface{}

func copyFields(fields Fields) Fields {
	copy := make(Fields, len(fields))
	return addFields(copy, fields)
}

func addFields(fields Fields, newFields Fields) Fields {
	for k, v := range newFields {
		if _, ok := fields[k]; ok {
			continue
		}

		fields[k] = v
	}

	return fields
}

// GetLogFields returns a copy of the log fields of the context. It can be nil.
func GetLogFields(ctx context.Context) Fields {
	if v := ctx.Value(logFieldsKeyContext); v != nil {
		return copyFields(v.(Fields))
	}

	return nil
}

// AddLogFields returns a context by updating the current log fields with those
// provided. Setting a key that is already present has no effect.
func AddLogFields(ctx context.Context, fields Fields) context.Context {
	if fields == nil {
		return ctx
	}

	currentFields := GetLogFields(ctx)
	if currentFields == nil {
		return context.WithValue(ctx, logFieldsKeyContext, fields)
	}

	newFields := addFields(currentFields, fields)
	return context.WithValue(ctx, logFieldsKeyContext, newFields)
}
