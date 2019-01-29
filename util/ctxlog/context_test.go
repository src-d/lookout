package ctxlog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	log "gopkg.in/src-d/go-log.v1"

	"github.com/src-d/lookout/util/ctxlog"
)

func TestEmptyGet(t *testing.T) {
	ctx := context.Background()

	log := ctxlog.Get(ctx)

	require.NotNil(t, log)

	require.NotPanics(t, func() {
		log.Debugf("a test log msg")
	})
}

func TestNilFields(t *testing.T) {
	ctx := context.Background()

	fields := ctxlog.Fields(ctx)

	require.Nil(t, fields)
}

func TestWithLogFields(t *testing.T) {
	ctxA := context.Background()
	ctxAB, _ := ctxlog.WithLogFields(ctxA, log.Fields{"keyB": "valB"})
	ctxABC, _ := ctxlog.WithLogFields(ctxAB, log.Fields{"keyC": "valC"})
	ctxABD, _ := ctxlog.WithLogFields(ctxAB, log.Fields{"keyD": "valD"})

	fieldsA := ctxlog.Fields(ctxA)
	fieldsB := ctxlog.Fields(ctxAB)
	fieldsC := ctxlog.Fields(ctxABC)
	fieldsD := ctxlog.Fields(ctxABD)

	require.NotEqual(t, fieldsA, fieldsB)
	require.NotEqual(t, fieldsB, fieldsC)

	require.Nil(t, fieldsA["keyB"])
	require.Nil(t, fieldsA["keyC"])
	require.Nil(t, fieldsA["keyD"])

	require.Equal(t, "valB", fieldsB["keyB"])
	require.Nil(t, fieldsB["keyC"])
	require.Nil(t, fieldsB["keyD"])

	require.Equal(t, "valB", fieldsC["keyB"])
	require.Equal(t, "valC", fieldsC["keyC"])
	require.Nil(t, fieldsC["keyD"])

	require.Equal(t, "valB", fieldsD["keyB"])
	require.Nil(t, fieldsD["keyC"])
	require.Equal(t, "valD", fieldsD["keyD"])
}

func TestWithNilLogFields(t *testing.T) {
	ctxA := context.Background()

	ctxB, logger := ctxlog.WithLogFields(ctxA, nil)

	require.NotNil(t, ctxB)
	require.NotNil(t, logger)
}

func TestReadOnlyWithLogFields(t *testing.T) {
	ctxA := context.Background()
	ctxB, _ := ctxlog.WithLogFields(ctxA, log.Fields{"keyB": "valB"})

	fieldsA := ctxlog.Fields(ctxA)
	fieldsB := ctxlog.Fields(ctxB)

	require.Nil(t, fieldsA["keyB"])
	require.Equal(t, "valB", fieldsB["keyB"])

	fieldsB["keyY"] = "valY"

	fieldsA2 := ctxlog.Fields(ctxA)
	fieldsB2 := ctxlog.Fields(ctxB)

	require.Nil(t, fieldsA2["keyB"])
	require.Nil(t, fieldsA2["keyY"])

	require.Equal(t, "valB", fieldsB2["keyB"])
	require.Nil(t, fieldsB2["keyY"])
}
