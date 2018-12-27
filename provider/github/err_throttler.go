package github

import (
	"time"

	log "gopkg.in/src-d/go-log.v1"
)

// errThrottlerInterval controls how often repeated ErrGitHubAPI (like 404)
// are logged with Error level. The rest of the logs will be logged as Debug
const errThrottlerInterval = 5 * time.Minute

type errThrottlerState struct {
	lastWhen time.Time
	lastErr  string
}

// errThrottlerLogger is a logger that logs repeated error messages as Debug
// level. An Error level is used once every errThrottlerInterval
type errThrottlerLogger struct {
	log.Logger
	*errThrottlerState
}

func newErrThrottlerLogger(log log.Logger, state *errThrottlerState) *errThrottlerLogger {
	return &errThrottlerLogger{log, state}
}

func (l *errThrottlerLogger) With(f log.Fields) log.Logger {
	return &errThrottlerLogger{l.Logger.With(f), l.errThrottlerState}
}

func (l *errThrottlerLogger) Errorf(err error, format string, args ...interface{}) {
	// log on Error level if the last message was different, or if the last time
	// was longer than errThrottlerInterval ago
	asErrLevel := l.lastErr != err.Error() ||
		time.Now().After(l.lastWhen.Add(errThrottlerInterval))

	if asErrLevel {
		l.lastErr = err.Error()
		l.lastWhen = time.Now()

		l.Logger.Errorf(err, format, args...)
		return
	}

	l.Logger.With(log.Fields{"error": err.Error()}).Debugf(format, args...)
}
