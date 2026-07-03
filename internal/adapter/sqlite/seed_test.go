package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite"
	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite/gen"
)

func TestApplySeedGameNeverOverwritesLiveEnrichment(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "ggs.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	q := gen.New(db)
	ts := "2026-01-01T00:00:00Z"
	live := "2026-02-02T00:00:00Z"

	// Row enriched live must survive a seed apply untouched.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO games (appid, name, tags, source, enriched_at) VALUES (1, 'Live', '["Fresh"]', 'steamspy', ?)`,
		live); err != nil {
		t.Fatal(err)
	}
	// Pending row (synced but unenriched) gets seeded.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO games (appid, name) VALUES (2, 'Pending')`); err != nil {
		t.Fatal(err)
	}

	seed := func(appid int64, name string) {
		t.Helper()
		if err := q.ApplySeedGame(ctx, gen.ApplySeedGameParams{
			Appid: appid, Name: name, Tags: `["Seeded"]`, Genres: `[]`, EnrichedAt: &ts,
		}); err != nil {
			t.Fatalf("apply seed %d: %v", appid, err)
		}
	}
	seed(1, "Live Renamed")
	seed(2, "Pending Renamed")
	seed(3, "Brand New")

	type row struct{ name, tags, source string }
	get := func(appid int64) row {
		t.Helper()
		var r row
		if err := db.QueryRowContext(ctx,
			`SELECT name, tags, source FROM games WHERE appid = ?`, appid).
			Scan(&r.name, &r.tags, &r.source); err != nil {
			t.Fatalf("get %d: %v", appid, err)
		}
		return r
	}

	if r := get(1); r.tags != `["Fresh"]` || r.source != "steamspy" || r.name != "Live" {
		t.Fatalf("live row overwritten by seed: %+v", r)
	}
	if r := get(2); r.tags != `["Seeded"]` || r.source != "seed" || r.name != "Pending" {
		t.Fatalf("pending row not seeded correctly (name must keep sync value): %+v", r)
	}
	if r := get(3); r.tags != `["Seeded"]` || r.name != "Brand New" {
		t.Fatalf("new row not inserted: %+v", r)
	}
}
