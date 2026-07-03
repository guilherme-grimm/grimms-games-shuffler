// Package http is the driving HTTP adapter: routes, session handling and
// the embedded SPA fileserver.
package http

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/guilherme-grimm/ggs/internal/dto/player"
)

// Pinger reports storage liveness for health checks.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Server holds the wired dependencies for all HTTP routes.
type Server struct {
	log     *slog.Logger
	db      Pinger
	dist    fs.FS
	players player.Service // nil when steam auth is not configured
	baseURL string
}

// NewServer wires the HTTP layer's dependencies; dist is the built SPA and
// players may be nil on an Instance without steam credentials.
func NewServer(log *slog.Logger, db Pinger, dist fs.FS, players player.Service, baseURL string) *Server {
	return &Server{log: log, db: db, dist: dist, players: players, baseURL: baseURL}
}

// Handler returns the root http.Handler with all routes mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	mux.HandleFunc("GET /auth/steam/login", s.handleSteamLogin)
	mux.HandleFunc("GET /auth/steam/callback", s.handleSteamCallback)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)

	mux.HandleFunc("GET /api/me", s.withPlayer(s.handleMe))
	mux.HandleFunc("POST /api/sync", s.withPlayer(s.handleSync))

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
