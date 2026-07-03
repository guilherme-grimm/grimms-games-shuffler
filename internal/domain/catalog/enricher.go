// Package catalog implements Catalog enrichment: a single background worker
// that fills games rows with tags/genres from the metadata source, and the
// progress view the UI polls.
package catalog

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/catalog"
)

// Service implements catalog.Service and owns the enrichment loop.
type Service struct {
	store  catalog.Storage
	source catalog.MetadataSource
	log    *slog.Logger
	tick   time.Duration
	now    func() time.Time
}

var _ catalog.Service = (*Service)(nil)

// NewService wires the enricher. tick controls the source request rate
// (SteamSpy tolerates ~1/s; a slower tick is polite).
func NewService(store catalog.Storage, source catalog.MetadataSource, log *slog.Logger, tick time.Duration) *Service {
	return &Service{store: store, source: source, log: log, tick: tick, now: time.Now}
}

// Progress implements catalog.Service.
func (s *Service) Progress(ctx context.Context, steamID string) (catalog.Progress, error) {
	return s.store.Progress(ctx, steamID)
}

// Run enriches until ctx is cancelled. One appid per tick keeps the source
// rate-limited by construction; the loop idles when nothing is pending.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.enrichOne(ctx)
		}
	}
}

func (s *Service) enrichOne(ctx context.Context) {
	ids, err := s.store.NextUnenriched(ctx, 1)
	if err != nil {
		s.log.Error("next unenriched", "error", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	appID := ids[0]

	e, err := s.source.Fetch(ctx, appID)
	switch {
	case errors.Is(err, catalog.ErrNotFound):
		if err := s.store.MarkEnrichmentFailed(ctx, appID, s.now().UTC()); err != nil {
			s.log.Error("mark enrichment failed", "appid", appID, "error", err)
		}
		return
	case err != nil:
		// Transient (network, 5xx): leave the row pending; next tick moves
		// on to the same appid, so persistent errors just slow the loop
		// instead of crashing it.
		s.log.Warn("enrichment fetch failed", "appid", appID, "error", err)
		return
	}
	if err := s.store.SaveEnrichment(ctx, e, s.now().UTC()); err != nil {
		s.log.Error("save enrichment", "appid", appID, "error", err)
	}
}
