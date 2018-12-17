package web

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pressly/lg"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

type HTTPServer struct {
	mux http.Handler
}

func NewHTTPServer(static *Static) *HTTPServer {
	corsOptions := cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"Location", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}

	r := chi.NewRouter()
	s := &HTTPServer{
		mux: r,
	}

	r.Use(cors.New(corsOptions).Handler)
	r.Use(lg.RequestLogger(logrus.StandardLogger()))
	r.Use(middleware.Recoverer)

	r.Get("/static/*", static.ServeHTTP)
	r.Get("/*", static.ServeHTTP)

	return s
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
