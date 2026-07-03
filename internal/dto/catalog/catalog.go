// Package catalog is the contract layer for the Game Catalog: shared
// per-game metadata (tags, genres) that Mood filtering runs against.
package catalog

import (
	"context"
	"errors"
	"time"
)

// Enrichment is the metadata harvested for one game.
type Enrichment struct {
	AppID  int64
	Tags   []string
	Genres []string
	Source string // "steamspy" | "seed"
}

// Progress reports how much of a Player's Library the Catalog covers.
type Progress struct {
	Enriched int `json:"enriched"`
	Total    int `json:"total"`
}

// ErrNotFound means the metadata source knows nothing about the appid.
var ErrNotFound = errors.New("catalog: game not found at source")

// Storage is the driven contract for Catalog persistence.
type Storage interface {
	// NextUnenriched returns up to limit appids that still need metadata.
	NextUnenriched(ctx context.Context, limit int) ([]int64, error)
	SaveEnrichment(ctx context.Context, e Enrichment, at time.Time) error
	// MarkEnrichmentFailed records a permanent miss so the appid is not
	// retried forever (delisted games, DLC, etc.).
	MarkEnrichmentFailed(ctx context.Context, appID int64, at time.Time) error
	Progress(ctx context.Context, steamID string) (Progress, error)
}

// MetadataSource is the driven contract for whatever provides tags/genres.
type MetadataSource interface {
	// Fetch returns metadata for one appid; ErrNotFound for permanent misses.
	Fetch(ctx context.Context, appID int64) (Enrichment, error)
}

// Service is the driving contract exposed to HTTP.
type Service interface {
	Progress(ctx context.Context, steamID string) (Progress, error)
}
