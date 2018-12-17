// +build bindata

package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"github.com/src-d/gitbase-web/server/assets"
)

const (
	staticDirName = "static"
	indexFileName = "index.html"

	serverValuesPlaceholder = "window.REPLACE_BY_SERVER"
	footerPlaceholder       = `<div class="invisible-footer"></div>`
)

// Static contains handlers to serve static using go-bindata
type Static struct {
	dir        string
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
		dir: dir,
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
	filepath := path.Join(s.dir, r.URL.Path)
	b, err := assets.Asset(filepath)
	if err != nil {
		if strings.HasPrefix(filepath, path.Join(s.dir, staticDirName)) {
			http.NotFound(w, r)
			return
		}

		s.serveIndexHTML(nil)(w, r)
		return
	}

	s.serveAsset(w, r, filepath, b)
}

// serveIndexHTML serves index.html file
func (s *Static) serveIndexHTML(initialState interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filepath := path.Join(s.dir, indexFileName)
		b, err := assets.Asset(filepath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		options := s.options
		bData, err := json.Marshal(options)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
		b = bytes.Replace(b, []byte(serverValuesPlaceholder), bData, 1)
		b = bytes.Replace(b, []byte(footerPlaceholder), s.footerHTML, 1)
		s.serveAsset(w, r, filepath, b)
	}
}

func (s *Static) serveAsset(w http.ResponseWriter, r *http.Request, filepath string, content []byte) {
	info, err := assets.AssetInfo(filepath)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(content))
}
