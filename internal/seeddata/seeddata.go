// Package seeddata carries the Seed Catalog (ADR 0003): pre-fetched
// tags/genres for the most-owned Steam games, embedded in the binary so a
// fresh Instance can Shuffle immediately. Regenerate with cmd/seedgen.
package seeddata

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

//go:embed seed.jsonl.gz
var _seedFS embed.FS

// Game is one Seed Catalog entry.
type Game struct {
	AppID  int64    `json:"appid"`
	Name   string   `json:"name"`
	Tags   []string `json:"tags"`
	Genres []string `json:"genres"`
}

// Each calls fn for every seeded game. An empty seed file yields no calls.
func Each(fn func(Game) error) error {
	f, err := _seedFS.Open("seed.jsonl.gz")
	if err != nil {
		return fmt.Errorf("open seed: %w", err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if errors.Is(err, io.EOF) {
		return nil // zero-byte placeholder — no seed shipped
	}
	if err != nil {
		return fmt.Errorf("gunzip seed: %w", err)
	}
	defer func() { _ = gz.Close() }()

	dec := json.NewDecoder(gz)
	for {
		var g Game
		if err := dec.Decode(&g); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return fmt.Errorf("decode seed entry: %w", err)
		}
		if err := fn(g); err != nil {
			return err
		}
	}
}
