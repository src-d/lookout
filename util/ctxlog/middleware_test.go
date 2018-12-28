package ctxlog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	log "gopkg.in/src-d/go-log.v1"
)

func TestMiddleware(t *testing.T) {
	require := require.New(t)

	testLogger := &TestLogger{}
	log.DefaultLogger = testLogger

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Get(r.Context()).Infof("inside handler")

		w.Write([]byte("test"))
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("User-Agent", "test-agent")

	RequestLogger(handler).ServeHTTP(w, r)

	require.Len(testLogger.messages, 3)

	// incoming message
	msg := testLogger.messages[0]
	require.Equal("info", msg.Level)
	require.Equal("request started", msg.Text)

	requestFields := log.Fields{
		"http_method": "GET",
		"http_proto":  "HTTP/1.1",
		"http_scheme": "http",
		"uri":         "http://example.com/",
		"user_agent":  "test-agent",
		"remote_addr": "192.0.2.1:1234",
	}
	require.Equal(requestFields, msg.Fields)

	// message inside handler should have request fields
	msg = testLogger.messages[1]
	require.Equal(msg.Level, "info")
	require.Equal(msg.Text, "inside handler")

	for k, v := range requestFields {
		require.Equal(v, msg.Fields[k])
	}

	// finished message
	msg = testLogger.messages[2]
	require.Equal(msg.Level, "info")
	require.Equal(msg.Text, "request complete")

	for k, v := range requestFields {
		require.Equal(v, msg.Fields[k])
	}
	require.Equal(http.StatusOK, msg.Fields["resp_status"])
	require.Equal(4, msg.Fields["resp_bytes_length"])
	require.True(msg.Fields["resp_elapsed_ms"].(float64) > 0)
}

type logMessage struct {
	Level  string
	Text   string
	Fields log.Fields
}

// TestLogger implement go-log.Logger for tests
// works correctly only if With/Levelf methods were called sequentially
type TestLogger struct {
	fields   log.Fields
	messages []logMessage
}

func (l *TestLogger) New(f log.Fields) log.Logger {
	l.fields = f
	return l
}

func (l *TestLogger) With(f log.Fields) log.Logger {
	if l.fields == nil {
		l.fields = f
		return l
	}

	for k, v := range f {
		l.fields[k] = v
	}

	return l
}

func (l *TestLogger) Debugf(format string, args ...interface{}) {
	l.messages = append(l.messages, logMessage{
		Level:  "debug",
		Text:   fmt.Sprintf(format, args...),
		Fields: cloneFields(l.fields),
	})
}

func (l *TestLogger) Infof(format string, args ...interface{}) {
	l.messages = append(l.messages, logMessage{
		Level:  "info",
		Text:   fmt.Sprintf(format, args...),
		Fields: cloneFields(l.fields),
	})
}

func (l *TestLogger) Warningf(format string, args ...interface{}) {
	l.messages = append(l.messages, logMessage{
		Level:  "warning",
		Text:   fmt.Sprintf(format, args...),
		Fields: cloneFields(l.fields),
	})
}

func (l *TestLogger) Errorf(err error, format string, args ...interface{}) {
	l.messages = append(l.messages, logMessage{
		Level:  "error",
		Text:   fmt.Sprintf(format, args...),
		Fields: cloneFields(l.fields),
	})
}

func cloneFields(f log.Fields) log.Fields {
	n := log.Fields{}
	for k, v := range f {
		n[k] = v
	}
	return n
}
