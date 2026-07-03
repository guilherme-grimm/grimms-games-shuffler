// Command seedgen builds the embedded Seed Catalog (ADR 0003): it walks
// SteamSpy's owners-ranked listing and fetches tags/genres for the top N
// games. Slow by design — SteamSpy allows 1 "all" page per minute and 1
// appdetails per second. Run occasionally and commit the output:
//
//	go run ./cmd/seedgen -count 2000 -out internal/seeddata/seed.jsonl.gz
package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guilherme-grimm/ggs/internal/adapter/steamspy"
	"github.com/guilherme-grimm/ggs/internal/dto/catalog"
	"github.com/guilherme-grimm/ggs/internal/seeddata"
)

func main() {
	count := flag.Int("count", 2000, "how many top-owned games to seed")
	out := flag.String("out", "internal/seeddata/seed.jsonl.gz", "output file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *count, *out); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, count int, out string) error {
	client := steamspy.NewClient()

	// Phase 1: owners-ranked appids, 1000 per page, 1 page per minute.
	var refs []steamspy.AppRef
	for page := 0; len(refs) < count; page++ {
		if page > 0 {
			log.Printf("rate limit: waiting 60s before page %d", page)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(60 * time.Second):
			}
		}
		batch, err := client.AllPage(ctx, page)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		if len(batch) == 0 {
			break
		}
		refs = append(refs, batch...)
		log.Printf("page %d: %d apps (total %d)", page, len(batch), len(refs))
	}
	if len(refs) > count {
		refs = refs[:count]
	}

	// Phase 2: tags/genres per app, 1 per second.
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("create %s: %w", out, err)
	}
	defer func() { _ = f.Close() }()
	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)

	written := 0
	for i, ref := range refs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
		e, err := client.Fetch(ctx, ref.AppID)
		if errors.Is(err, catalog.ErrNotFound) {
			continue
		}
		if err != nil {
			log.Printf("skip %d (%s): %v", ref.AppID, ref.Name, err)
			continue
		}
		if err := enc.Encode(seeddata.Game{
			AppID: e.AppID, Name: ref.Name, Tags: e.Tags, Genres: e.Genres,
		}); err != nil {
			return fmt.Errorf("encode %d: %w", ref.AppID, err)
		}
		written++
		if (i+1)%100 == 0 {
			log.Printf("progress: %d/%d fetched, %d written", i+1, len(refs), written)
		}
	}

	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", out, err)
	}
	log.Printf("done: %d games seeded to %s", written, out)
	return nil
}
