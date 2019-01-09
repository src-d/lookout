// +build with_static

package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/src-d/lookout/web/assets"
)

const (
	staticDirName = "static"
	indexFileName = "/index.html"

	serverValuesPlaceholder = "window.REPLACE_BY_SERVER"
	footerPlaceholder       = `<div class="invisible-footer"></div>`
)

// Static contains handlers to serve static using esc
type Static struct {
	fs         http.FileSystem
	options    options
	footerHTML []byte
}

// NewStatic creates new Static
func NewStatic(dir, serverURL string, footerHTML string) *Static {
	var footerBytes []byte
	if footerHTML != "" {
		// skip incorrect base64
		footerBytes, _ = base64.StdEncoding.DecodeString(footerHTML)
	}

	return &Static{
		fs: assets.Dir(false, dir),
		options: options{
			ServerURL: serverURL,
		},
		footerHTML: footerBytes,
	}
}

// struct which will be marshalled and exposed to frontend
type options struct {
	ServerURL string `json:"SERVER_URL"`
}

// ServeHTTP serves any static file from static directory or fallbacks on index.hml
func (s *Static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := s.fs.Open(r.URL.Path)
	if err != nil {
		if strings.HasPrefix(r.URL.Path, staticDirName) {
			http.NotFound(w, r)
			return
		}

		s.serveIndexHTML(nil)(w, r)
		return
	}

	http.FileServer(s.fs).ServeHTTP(w, r)
}

// serveIndexHTML serves index.html file
func (s *Static) serveIndexHTML(initialState interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := s.fs.Open(indexFileName)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		options := s.options
		bData, err := json.Marshal(options)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		b = bytes.Replace(b, []byte(serverValuesPlaceholder), bData, 1)
		b = bytes.Replace(b, []byte(footerPlaceholder), s.footerHTML, 1)

		w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")

		info, err := f.Stat()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(b))
	}
}
