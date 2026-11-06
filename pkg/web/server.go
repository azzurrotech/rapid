package web

import (
	"fmt"
	"log"
	"net/http"
	"time"

	rapid "rapid.ai"
	"rapid.ai/pkg/handlers"
)

// WebServer serves RAPID on top of the real atp platform.
type WebServer struct {
	app  *rapid.RAPIDApplication
	port string
	host string
}

// NewWebServer creates a web server for an application.
func NewWebServer(app *rapid.RAPIDApplication, host, port string) *WebServer {
	return &WebServer{app: app, port: port, host: host}
}

// Handler builds the HTTP handler: RAPID's API and dashboard, wrapped by the
// platform so /pod/, /_song/, the vault and hosted applications stay reachable.
func (s *WebServer) Handler() (http.Handler, error) {
	if err := s.app.PublishDashboard([]byte(dashboardHTML)); err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	handlers.New(s.app).Register(mux)
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/rapid/", s.handleDashboard)

	// Wrap the mux with security headers.
	finalMux := securityHeaders{mux}

	return s.app.Handler(finalMux), nil
}

// securityHeaders adds HTTP security headers to every response.
type securityHeaders struct {
	handler http.Handler
}

func (s securityHeaders) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Content-Security-Policy", "default-src 'self' data:")
	// Hide the server banner
	w.Header().Set("Server", "RAPID/1.0.0")
	s.handler.ServeHTTP(w, r)
}

// Start publishes the dashboard through song and serves the platform.
func (s *WebServer) Start() error {
	handler, err := s.Handler()
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("RAPID listening on http://%s/ (dashboard: /rapid/)", addr)
	return srv.ListenAndServe()
}

// handleRoot sends the browser to the song-hosted dashboard rather than

// handleDashboard serves the dashboard HTML at /rapid/
func (s *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/rapid/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(dashboardHTML))
		return
	}
	http.NotFound(w, r)
}
// duplicating the HTML at "/". One document, one place to edit it.
func (s *WebServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/rapid/", http.StatusFound)
}
