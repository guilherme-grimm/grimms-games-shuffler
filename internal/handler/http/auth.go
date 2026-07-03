package http

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/player"
)

const (
	_sessionCookie = "ggs_session"
	_stateCookie   = "ggs_state"
)

func (s *Server) secureCookies() bool { return strings.HasPrefix(s.baseURL, "https://") }

func (s *Server) handleSteamLogin(w http.ResponseWriter, r *http.Request) {
	if s.players == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "steam auth not configured"})
		return
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		s.log.Error("generate state", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	state := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name: _stateCookie, Value: state, Path: "/auth/",
		MaxAge: 600, HttpOnly: true, Secure: s.secureCookies(), SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, s.players.AuthURL(state), http.StatusFound)
}

func (s *Server) handleSteamCallback(w http.ResponseWriter, r *http.Request) {
	if s.players == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "steam auth not configured"})
		return
	}
	stateCookie, err := r.Cookie(_stateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "state mismatch"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: _stateCookie, Path: "/auth/", MaxAge: -1})

	sess, err := s.players.CompleteLogin(r.Context(), r.URL.Query())
	if err != nil {
		s.log.Error("steam login failed", "error", err)
		code := http.StatusBadGateway
		if errors.Is(err, player.ErrOpenIDVerify) {
			code = http.StatusForbidden
		}
		writeJSON(w, code, map[string]string{"error": "login failed"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: _sessionCookie, Value: sess.Token, Path: "/",
		Expires: sess.ExpiresAt, HttpOnly: true, Secure: s.secureCookies(),
		SameSite: http.SameSiteLaxMode,
	})

	// First sync happens right after login; a private library must not
	// block it — the UI surfaces that on /api/sync instead.
	if _, err := s.players.Sync(r.Context(), sess.SteamID); err != nil {
		s.log.Warn("post-login sync failed", "steam_id", sess.SteamID, "error", err)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(_sessionCookie); err == nil && s.players != nil {
		if err := s.players.Logout(r.Context(), c.Value); err != nil {
			s.log.Warn("logout", "error", err)
		}
	}
	http.SetCookie(w, &http.Cookie{Name: _sessionCookie, Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// withPlayer authenticates the session cookie and passes the Player on.
func (s *Server) withPlayer(next func(http.ResponseWriter, *http.Request, player.Player)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.players == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "steam auth not configured"})
			return
		}
		c, err := r.Cookie(_sessionCookie)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not logged in"})
			return
		}
		p, err := s.players.Authenticate(r.Context(), c.Value)
		if err != nil {
			if !errors.Is(err, player.ErrInvalidSession) {
				s.log.Error("authenticate", "error", err)
			}
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not logged in"})
			return
		}
		next(w, r, p)
	}
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, p player.Player) {
	count, err := s.players.LibraryCount(r.Context(), p.SteamID)
	if err != nil {
		s.log.Error("library count", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, meResponse{
		SteamID:      p.SteamID,
		PersonaName:  p.PersonaName,
		AvatarURL:    p.AvatarURL,
		LastSyncAt:   p.LastSyncAt,
		LibraryCount: count,
	})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request, p player.Player) {
	count, err := s.players.Sync(r.Context(), p.SteamID)
	if err != nil {
		if errors.Is(err, player.ErrPrivateLibrary) {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "private_library",
				"message": "Your Steam game details are private. Set Profile > Privacy " +
					"> Game details to Public, then sync again.",
			})
			return
		}
		s.log.Error("sync failed", "steam_id", p.SteamID, "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "sync failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"libraryCount": count})
}

type meResponse struct {
	SteamID      string     `json:"steamId"`
	PersonaName  string     `json:"personaName"`
	AvatarURL    string     `json:"avatarUrl"`
	LastSyncAt   *time.Time `json:"lastSyncAt"`
	LibraryCount int        `json:"libraryCount"`
}
