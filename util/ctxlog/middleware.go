package ctxlog

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	log "gopkg.in/src-d/go-log.v1"
)

// RequestLogger middleware logs all incoming requests and
// adds log fields to the request context
func RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		logFields := logFieldsFromRequest(r)
		logger := Get(r.Context()).With(logFields)
		logger.Infof("request started")

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		t1 := time.Now()
		defer func() {
			t2 := time.Now()

			logger.With(log.Fields{
				"resp_status": ww.Status(), "resp_bytes_length": ww.BytesWritten(),
				"resp_elapsed_ms": float64(t2.Sub(t1).Nanoseconds()) / 1000000.0,
			}).Infof("request complete")
		}()

		r = r.WithContext(Set(r.Context(), logger))
		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

func logFieldsFromRequest(r *http.Request) log.Fields {
	logFields := log.Fields{}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host

	logFields["http_scheme"] = scheme
	logFields["http_proto"] = r.Proto
	logFields["http_method"] = r.Method

	logFields["remote_addr"] = r.RemoteAddr
	logFields["user_agent"] = r.UserAgent()

	if val := r.Header.Get("X-Forwarded-For"); val != "" {
		logFields["X-Forwarded-For"] = val
	}
	if val := r.Header.Get("X-Forwarded-Host"); val != "" {
		logFields["X-Forwarded-Host"] = val
		host = val
	}
	if val := r.Header.Get("X-Forwarded-Scheme"); val != "" {
		logFields["X-Forwarded-Scheme"] = val
		scheme = val
	}

	logFields["uri"] = fmt.Sprintf("%s://%s%s", scheme, host, r.RequestURI)

	return logFields
}
