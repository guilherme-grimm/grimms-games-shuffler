// Package http is the driving HTTP adapter: routes, session handling and
// the embedded SPA fileserver.
package http

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
)

// Pinger reports storage liveness for health checks.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Server holds the wired dependencies for all HTTP routes.
type Server struct {
	log  *slog.Logger
	db   Pinger
	dist fs.FS
}

// NewServer wires the HTTP layer's dependencies; dist is the built SPA.
func NewServer(log *slog.Logger, db Pinger, dist fs.FS) *Server {
	return &Server{log: log, db: db, dist: dist}
}

// Handler returns the root http.Handler with all routes mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.Handle("/", s.spaHandler())
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.db.PingContext(r.Context()); err != nil {
		s.log.Error("healthz db ping failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
