package web

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/cors"
)

type HTTPServer struct {
	mux http.Handler
}

func NewHTTPServer(auth *Auth, gh *GitHub, static *Static) *HTTPServer {
	corsOptions := cors.Options{
		// TODO: make it customizable
		// we can't pass "*" because it's incompatible with "credentials: include" request
		// http://127.0.0.1:3000 is the default url of create-react-app dev server
		AllowedOrigins:   []string{"http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"Location", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}

	r := chi.NewRouter()
	s := &HTTPServer{
		mux: r,
	}

	r.Use(cors.New(corsOptions).Handler)
	r.Use(RequestLogger)
	r.Use(middleware.Recoverer)

	r.Get("/login", auth.Login)
	r.Get("/api/callback", auth.Callback)
	r.With(auth.Middleware).Route("/api", func(r chi.Router) {
		r.Get("/me", auth.Me)
		r.Get("/orgs", gh.Orgs)

		r.Route("/org/{orgName}", func(r chi.Router) {
			r.Get("/", gh.Org)
			r.Put("/", gh.UpdateOrg)
		})
	})
	r.Get("/static/*", static.ServeHTTP)
	r.Get("/*", static.ServeHTTP)

	return s
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
