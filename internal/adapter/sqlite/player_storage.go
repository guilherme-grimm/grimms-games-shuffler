package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite/gen"
	"github.com/guilherme-grimm/ggs/internal/dto/player"
)

// PlayerStorage implements player.Storage over the sqlc-generated queries.
type PlayerStorage struct {
	db *sql.DB
	q  *gen.Queries
}

var _ player.Storage = (*PlayerStorage)(nil)

// NewPlayerStorage wraps db with the player.Storage implementation.
func NewPlayerStorage(db *sql.DB) *PlayerStorage {
	return &PlayerStorage{db: db, q: gen.New(db)}
}

// UpsertPlayer implements player.Storage.
func (s *PlayerStorage) UpsertPlayer(ctx context.Context, p player.Player) error {
	err := s.q.UpsertPlayer(ctx, gen.UpsertPlayerParams{
		SteamID:     p.SteamID,
		PersonaName: p.PersonaName,
		AvatarUrl:   p.AvatarURL,
	})
	if err != nil {
		return fmt.Errorf("upsert player %s: %w", p.SteamID, err)
	}
	return nil
}

// GetPlayer implements player.Storage.
func (s *PlayerStorage) GetPlayer(ctx context.Context, steamID string) (player.Player, error) {
	row, err := s.q.GetPlayer(ctx, steamID)
	if err != nil {
		return player.Player{}, fmt.Errorf("get player %s: %w", steamID, err)
	}
	return player.Player{
		SteamID:     row.SteamID,
		PersonaName: row.PersonaName,
		AvatarURL:   row.AvatarUrl,
		LastSyncAt:  parseTimePtr(row.LastSyncAt),
	}, nil
}

// TouchLastSync implements player.Storage.
func (s *PlayerStorage) TouchLastSync(ctx context.Context, steamID string, at time.Time) error {
	ts := formatTime(at)
	err := s.q.TouchLastSync(ctx, gen.TouchLastSyncParams{LastSyncAt: &ts, SteamID: steamID})
	if err != nil {
		return fmt.Errorf("touch last sync %s: %w", steamID, err)
	}
	return nil
}

// CreateSession implements player.Storage.
func (s *PlayerStorage) CreateSession(ctx context.Context, tokenHash string, sess player.Session) error {
	err := s.q.CreateSession(ctx, gen.CreateSessionParams{
		Token:     tokenHash,
		SteamID:   sess.SteamID,
		CreatedAt: formatTime(sess.CreatedAt),
		ExpiresAt: formatTime(sess.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSession implements player.Storage.
func (s *PlayerStorage) GetSession(ctx context.Context, tokenHash string, now time.Time) (player.Session, error) {
	row, err := s.q.GetSession(ctx, gen.GetSessionParams{Token: tokenHash, ExpiresAt: formatTime(now)})
	if errors.Is(err, sql.ErrNoRows) {
		return player.Session{}, player.ErrInvalidSession
	}
	if err != nil {
		return player.Session{}, fmt.Errorf("get session: %w", err)
	}
	return player.Session{
		SteamID:   row.SteamID,
		CreatedAt: parseTime(row.CreatedAt),
		ExpiresAt: parseTime(row.ExpiresAt),
	}, nil
}

// DeleteSession implements player.Storage.
func (s *PlayerStorage) DeleteSession(ctx context.Context, tokenHash string) error {
	if err := s.q.DeleteSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// ReplaceLibrary swaps the Player's whole library in one transaction and
// makes sure every owned appid exists in games for later enrichment.
func (s *PlayerStorage) ReplaceLibrary(ctx context.Context, steamID string, games []player.OwnedGame) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace library: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)

	if err := q.DeleteLibrary(ctx, steamID); err != nil {
		return fmt.Errorf("clear library %s: %w", steamID, err)
	}
	for _, g := range games {
		var lastPlayed *string
		if g.LastPlayed != nil {
			ts := formatTime(*g.LastPlayed)
			lastPlayed = &ts
		}
		if err := q.InsertLibraryGame(ctx, gen.InsertLibraryGameParams{
			SteamID:      steamID,
			Appid:        g.AppID,
			PlaytimeMin:  g.PlaytimeMin,
			LastPlayedAt: lastPlayed,
		}); err != nil {
			return fmt.Errorf("insert library game %d: %w", g.AppID, err)
		}
		if err := q.EnsureGame(ctx, gen.EnsureGameParams{Appid: g.AppID, Name: g.Name}); err != nil {
			return fmt.Errorf("ensure game %d: %w", g.AppID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace library: %w", err)
	}
	return nil
}

// CountLibrary implements player.Storage.
func (s *PlayerStorage) CountLibrary(ctx context.Context, steamID string) (int, error) {
	n, err := s.q.CountLibrary(ctx, steamID)
	if err != nil {
		return 0, fmt.Errorf("count library %s: %w", steamID, err)
	}
	return int(n), nil
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func parseTimePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t := parseTime(*s)
	return &t
}
