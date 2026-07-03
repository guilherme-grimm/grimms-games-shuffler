package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite"
	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite/gen"
)

func TestOpenMigratesAndIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "ggs.db")

	db, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}

	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version < 1 {
		t.Fatalf("user_version = %d, want >= 1", version)
	}

	// Generated queries run against the migrated schema.
	q := gen.New(db)
	if err := q.UpsertPlayer(ctx, gen.UpsertPlayerParams{
		SteamID:     "76561198000000000",
		PersonaName: "grimm",
		AvatarUrl:   "https://example.invalid/a.jpg",
	}); err != nil {
		t.Fatalf("upsert player: %v", err)
	}
	player, err := q.GetPlayer(ctx, "76561198000000000")
	if err != nil {
		t.Fatalf("get player: %v", err)
	}
	if player.PersonaName != "grimm" {
		t.Fatalf("persona = %q, want grimm", player.PersonaName)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Re-open: migrations must be skipped, data preserved.
	db2, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer func() { _ = db2.Close() }()
	if _, err := gen.New(db2).GetPlayer(ctx, "76561198000000000"); err != nil {
		t.Fatalf("get player after reopen: %v", err)
	}
}
