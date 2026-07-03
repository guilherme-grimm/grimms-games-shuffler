package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite/gen"
	"github.com/guilherme-grimm/ggs/internal/dto/shuffle"
)

// ShuffleStorage implements shuffle.Storage.
type ShuffleStorage struct {
	q *gen.Queries
}

var _ shuffle.Storage = (*ShuffleStorage)(nil)

// NewShuffleStorage wraps db with the shuffle.Storage implementation.
func NewShuffleStorage(db *sql.DB) *ShuffleStorage {
	return &ShuffleStorage{q: gen.New(db)}
}

// CountToday implements shuffle.Storage.
func (s *ShuffleStorage) CountToday(ctx context.Context, steamID, dateUTC string) (int, error) {
	n, err := s.q.CountShufflesToday(ctx, gen.CountShufflesTodayParams{SteamID: steamID, DateUtc: dateUTC})
	if err != nil {
		return 0, fmt.Errorf("count shuffles: %w", err)
	}
	return int(n), nil
}

// TodaysAppIDs implements shuffle.Storage.
func (s *ShuffleStorage) TodaysAppIDs(ctx context.Context, steamID, dateUTC string) ([]int64, error) {
	ids, err := s.q.TodaysShuffledAppids(ctx, gen.TodaysShuffledAppidsParams{SteamID: steamID, DateUtc: dateUTC})
	if err != nil {
		return nil, fmt.Errorf("todays appids: %w", err)
	}
	return ids, nil
}

// Insert implements shuffle.Storage.
func (s *ShuffleStorage) Insert(ctx context.Context, r shuffle.Record) error {
	mood, err := json.Marshal(r.Mood)
	if err != nil {
		return fmt.Errorf("marshal mood: %w", err)
	}
	var usedAI int64
	if r.UsedAI {
		usedAI = 1
	}
	err = s.q.InsertShuffle(ctx, gen.InsertShuffleParams{
		SteamID: r.SteamID, DateUtc: r.DateUTC, Appid: r.AppID,
		Mood: string(mood), UsedAi: usedAI, Why: r.Why,
		CreatedAt: formatTime(r.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("insert shuffle: %w", err)
	}
	return nil
}

// Candidates implements shuffle.Storage.
func (s *ShuffleStorage) Candidates(ctx context.Context, steamID string) ([]shuffle.Candidate, error) {
	rows, err := s.q.LibraryCandidates(ctx, steamID)
	if err != nil {
		return nil, fmt.Errorf("library candidates %s: %w", steamID, err)
	}
	out := make([]shuffle.Candidate, 0, len(rows))
	for _, r := range rows {
		var tags []string
		_ = json.Unmarshal([]byte(r.Tags), &tags)
		out = append(out, shuffle.Candidate{
			AppID:       r.Appid,
			Name:        r.Name,
			PlaytimeMin: r.PlaytimeMin,
			Tags:        tags,
			Enriched:    r.Enriched != nil && *r.Enriched,
		})
	}
	return out, nil
}
