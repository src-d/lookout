// +build !with_static

package web

import (
	"net/http"
)

// Static without the 'with_static' tag contains a placeholder handler
// that returns 'not implemented'
type Static struct{}

// NewStatic creates new Static
func NewStatic(dir, serverURL string, footerHTML string) *Static {
	return &Static{}
}

// ServeHTTP serves any static file from static directory or fallbacks on index.hml
func (s *Static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w,
		"Frontend assets are only available when using 'make build' or 'make serve'",
		http.StatusNotImplemented)

	return
}
