package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite/gen"
	"github.com/guilherme-grimm/ggs/internal/dto/catalog"
)

// CatalogStorage implements catalog.Storage.
type CatalogStorage struct {
	q *gen.Queries
}

var _ catalog.Storage = (*CatalogStorage)(nil)

// NewCatalogStorage wraps db with the catalog.Storage implementation.
func NewCatalogStorage(db *sql.DB) *CatalogStorage {
	return &CatalogStorage{q: gen.New(db)}
}

// NextUnenriched implements catalog.Storage.
func (s *CatalogStorage) NextUnenriched(ctx context.Context, limit int) ([]int64, error) {
	ids, err := s.q.NextUnenriched(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("next unenriched: %w", err)
	}
	return ids, nil
}

// SaveEnrichment implements catalog.Storage.
func (s *CatalogStorage) SaveEnrichment(ctx context.Context, e catalog.Enrichment, at time.Time) error {
	tags, err := json.Marshal(e.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags %d: %w", e.AppID, err)
	}
	genres, err := json.Marshal(e.Genres)
	if err != nil {
		return fmt.Errorf("marshal genres %d: %w", e.AppID, err)
	}
	ts := formatTime(at)
	err = s.q.SaveEnrichment(ctx, gen.SaveEnrichmentParams{
		Tags: string(tags), Genres: string(genres),
		Source: e.Source, EnrichedAt: &ts, Appid: e.AppID,
	})
	if err != nil {
		return fmt.Errorf("save enrichment %d: %w", e.AppID, err)
	}
	return nil
}

// MarkEnrichmentFailed implements catalog.Storage.
func (s *CatalogStorage) MarkEnrichmentFailed(ctx context.Context, appID int64, at time.Time) error {
	ts := formatTime(at)
	err := s.q.SaveEnrichment(ctx, gen.SaveEnrichmentParams{
		Tags: "[]", Genres: "[]", Source: "failed", EnrichedAt: &ts, Appid: appID,
	})
	if err != nil {
		return fmt.Errorf("mark enrichment failed %d: %w", appID, err)
	}
	return nil
}

// Progress implements catalog.Storage.
func (s *CatalogStorage) Progress(ctx context.Context, steamID string) (catalog.Progress, error) {
	row, err := s.q.EnrichmentProgress(ctx, steamID)
	if err != nil {
		return catalog.Progress{}, fmt.Errorf("enrichment progress %s: %w", steamID, err)
	}
	return catalog.Progress{Enriched: int(row.Enriched), Total: int(row.Total)}, nil
}
