package sqlite_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite"
	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite/gen"
	"github.com/guilherme-grimm/ggs/internal/dto/shuffle"
)

func TestInsertEnforcesDailyBudget(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	db, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "ggs.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	const steamID = "76561198000000001"
	if err := gen.New(db).UpsertPlayer(ctx, gen.UpsertPlayerParams{SteamID: steamID}); err != nil {
		t.Fatalf("upsert player: %v", err)
	}

	store := sqlite.NewShuffleStorage(db)
	rec := func(appID int64) shuffle.Record {
		return shuffle.Record{
			SteamID: steamID, DateUTC: "2026-07-03", AppID: appID,
			Mood: shuffle.Mood{
				Energy: shuffle.EnergyChill, Time: shuffle.TimeQuick,
				Familiarity: shuffle.FamiliaritySurprise,
			},
			Why: fmt.Sprintf("why %d", appID), CreatedAt: time.Now().UTC(),
		}
	}

	for i := int64(1); i <= shuffle.DailyBudget; i++ {
		if err := store.Insert(ctx, rec(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	if err := store.Insert(ctx, rec(99)); !errors.Is(err, shuffle.ErrBudgetSpent) {
		t.Fatalf("insert over budget: err = %v, want ErrBudgetSpent", err)
	}
	if n, err := store.CountToday(ctx, steamID, "2026-07-03"); err != nil || n != shuffle.DailyBudget {
		t.Fatalf("count = %d (err %v), want %d", n, err, shuffle.DailyBudget)
	}

	// A new UTC day starts a fresh budget.
	next := rec(100)
	next.DateUTC = "2026-07-04"
	if err := store.Insert(ctx, next); err != nil {
		t.Fatalf("insert next day: %v", err)
	}
}
