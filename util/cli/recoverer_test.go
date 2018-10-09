package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"
	"github.com/stretchr/testify/require"
	log "gopkg.in/src-d/go-log.v1"
)

func TestWatcherError(t *testing.T) {
	require := require.New(t)

	logMock := &MockLogger{}

	watcher := &ErrorWatcherMock{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx = ctxlog.Set(ctx, logMock)
	defer cancel()

	err := RunWatcher(ctx, watcher, nil)

	require.EqualError(err, "context deadline exceeded")
	require.True(len(logMock.errors) > 0)
	require.EqualError(logMock.errors[0], "some error")
}

type ErrorWatcherMock struct{}

func (w *ErrorWatcherMock) Watch(ctx context.Context, e lookout.EventHandler) error {
	return errors.New("some error")
}

type MockLogger struct {
	errors []error
}

func (l *MockLogger) New(log.Fields) log.Logger {
	return l
}
func (l *MockLogger) With(log.Fields) log.Logger {
	return l
}
func (l *MockLogger) Debugf(format string, args ...interface{})   {}
func (l *MockLogger) Infof(format string, args ...interface{})    {}
func (l *MockLogger) Warningf(format string, args ...interface{}) {}
func (l *MockLogger) Errorf(err error, format string, args ...interface{}) {
	l.errors = append(l.errors, err)
}
