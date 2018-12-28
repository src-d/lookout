package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/src-d/lookout"

	"github.com/stretchr/testify/require"
)

func TestWatcherError(t *testing.T) {
	require := require.New(t)

	watcher := &ErrorWatcherMock{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := RunWatcher(ctx, watcher, nil)

	require.EqualError(err, "context deadline exceeded")
	require.True(watcher.nErr > 0)
}

type ErrorWatcherMock struct {
	nErr int
}

func (w *ErrorWatcherMock) Watch(ctx context.Context, e lookout.EventHandler) error {
	w.nErr++
	return errors.New("some error")
}
