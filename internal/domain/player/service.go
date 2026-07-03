// Package player implements the player.Service contract: Steam login,
// sessions and Library sync.
package player

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/player"
)

const _sessionTTL = 30 * 24 * time.Hour

// Service implements player.Service.
type Service struct {
	store   player.Storage
	steam   player.SteamClient
	baseURL string
	now     func() time.Time
}

var _ player.Service = (*Service)(nil)

// NewService wires the player service with its storage and Steam ports.
func NewService(store player.Storage, steam player.SteamClient, baseURL string) *Service {
	return &Service{store: store, steam: steam, baseURL: baseURL, now: time.Now}
}

// AuthURL builds the Steam redirect with the CSRF state carried in return_to.
func (s *Service) AuthURL(state string) string {
	returnTo := s.baseURL + "/auth/steam/callback?state=" + url.QueryEscape(state)
	return s.steam.AuthURL(returnTo)
}

// CompleteLogin verifies the callback, refreshes the Player profile and
// opens a session. Library sync happens separately so a private library
// does not block login.
func (s *Service) CompleteLogin(ctx context.Context, callback url.Values) (player.Session, error) {
	steamID, err := s.steam.VerifyCallback(ctx, callback)
	if err != nil {
		return player.Session{}, err
	}
	name, avatar, err := s.steam.Summary(ctx, steamID)
	if err != nil {
		return player.Session{}, fmt.Errorf("fetch summary: %w", err)
	}
	p := player.Player{SteamID: steamID, PersonaName: name, AvatarURL: avatar}
	if err := s.store.UpsertPlayer(ctx, p); err != nil {
		return player.Session{}, err
	}

	token, err := newToken()
	if err != nil {
		return player.Session{}, err
	}
	now := s.now().UTC()
	sess := player.Session{
		Token:     token,
		SteamID:   steamID,
		CreatedAt: now,
		ExpiresAt: now.Add(_sessionTTL),
	}
	if err := s.store.CreateSession(ctx, hashToken(token), sess); err != nil {
		return player.Session{}, err
	}
	return sess, nil
}

// Authenticate implements player.Service.
func (s *Service) Authenticate(ctx context.Context, token string) (player.Player, error) {
	sess, err := s.store.GetSession(ctx, hashToken(token), s.now().UTC())
	if err != nil {
		return player.Player{}, err
	}
	return s.store.GetPlayer(ctx, sess.SteamID)
}

// Logout implements player.Service.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, hashToken(token))
}

// Sync imports the Player's owned games and returns the Library size.
func (s *Service) Sync(ctx context.Context, steamID string) (int, error) {
	games, err := s.steam.OwnedGames(ctx, steamID)
	if err != nil {
		return 0, err
	}
	if err := s.store.ReplaceLibrary(ctx, steamID, games); err != nil {
		return 0, err
	}
	if err := s.store.TouchLastSync(ctx, steamID, s.now().UTC()); err != nil {
		return 0, err
	}
	return len(games), nil
}

// LibraryCount implements player.Service.
func (s *Service) LibraryCount(ctx context.Context, steamID string) (int, error) {
	return s.store.CountLibrary(ctx, steamID)
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// hashToken keeps only a digest in storage so a leaked DB cannot be used to
// hijack sessions.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
