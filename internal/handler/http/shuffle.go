package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/player"
	"github.com/guilherme-grimm/ggs/internal/dto/shuffle"
)

func (s *Server) handleShuffle(w http.ResponseWriter, r *http.Request, p player.Player) {
	var mood shuffle.Mood
	if err := json.NewDecoder(r.Body).Decode(&mood); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad body"})
		return
	}
	res, err := s.shuffles.Shuffle(r.Context(), p.SteamID, mood)
	switch {
	case errors.Is(err, shuffle.ErrInvalidMood):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_mood"})
	case errors.Is(err, shuffle.ErrBudgetSpent):
		_, reset, lerr := s.shuffles.Left(r.Context(), p.SteamID)
		if lerr != nil {
			reset = time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		}
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":   "budget_spent",
			"resetAt": reset,
			"message": "All 3 Shuffles used for today. Budget resets at UTC midnight.",
		})
	case errors.Is(err, shuffle.ErrNoCandidates):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":   "no_candidates",
			"message": "Your library has nothing left to shuffle today. Synced yet?",
		})
	case err != nil:
		s.log.Error("shuffle failed", "steam_id", p.SteamID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	default:
		writeJSON(w, http.StatusOK, res)
	}
}

func (s *Server) handleLibraryStatus(w http.ResponseWriter, r *http.Request, p player.Player) {
	prog, err := s.catalog.Progress(r.Context(), p.SteamID)
	if err != nil {
		s.log.Error("catalog progress", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, prog)
}
