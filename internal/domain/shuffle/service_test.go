package shuffle_test

import (
	"context"
	"errors"
	"testing"

	domain "github.com/guilherme-grimm/ggs/internal/domain/shuffle"
	"github.com/guilherme-grimm/ggs/internal/dto/shuffle"
)

type fakeStore struct {
	candidates []shuffle.Candidate
	records    []shuffle.Record
}

func (f *fakeStore) CountToday(_ context.Context, steamID, dateUTC string) (int, error) {
	n := 0
	for _, r := range f.records {
		if r.SteamID == steamID && r.DateUTC == dateUTC {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) TodaysAppIDs(_ context.Context, steamID, dateUTC string) ([]int64, error) {
	var ids []int64
	for _, r := range f.records {
		if r.SteamID == steamID && r.DateUTC == dateUTC {
			ids = append(ids, r.AppID)
		}
	}
	return ids, nil
}

func (f *fakeStore) Insert(_ context.Context, r shuffle.Record) error {
	f.records = append(f.records, r)
	return nil
}

func (f *fakeStore) Candidates(context.Context, string) ([]shuffle.Candidate, error) {
	return f.candidates, nil
}

func lib() []shuffle.Candidate {
	return []shuffle.Candidate{
		{AppID: 1, Name: "Cozy Farm", Tags: []string{"Relaxing", "Farming Sim", "Casual"}, Enriched: true, PlaytimeMin: 0},
		{AppID: 2, Name: "Bullet Storm", Tags: []string{"Action", "Bullet Hell", "Fast-Paced"}, Enriched: true, PlaytimeMin: 30},
		{AppID: 3, Name: "Epic Saga", Tags: []string{"RPG", "Story Rich", "Open World"}, Enriched: true, PlaytimeMin: 3000},
		{AppID: 4, Name: "Mystery Box", Tags: nil, Enriched: false, PlaytimeMin: 0},
	}
}

func mood(energy, t, fam, brain string) shuffle.Mood {
	return shuffle.Mood{Energy: energy, Time: t, Familiarity: fam, Brain: brain}
}

func TestShuffleFilters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		mood        shuffle.Mood
		wantAppID   int64
		wantRelaxed int
	}{
		{
			name:      "chill surprise finds the cozy unplayed game",
			mood:      mood(shuffle.EnergyChill, shuffle.TimeMedium, shuffle.FamiliaritySurprise, ""),
			wantAppID: 1,
		},
		{
			name:      "adrenaline backlog finds the barely-played shooter",
			mood:      mood(shuffle.EnergyAdrenaline, shuffle.TimeMedium, shuffle.FamiliarityBacklog, ""),
			wantAppID: 2,
		},
		{
			name:      "story favorite finds the played RPG",
			mood:      mood(shuffle.EnergyBalanced, shuffle.TimeLong, shuffle.FamiliarityFavorite, shuffle.BrainStory),
			wantAppID: 3,
		},
		{
			name: "impossible combo relaxes until something matches",
			// Adrenaline + favorite: only favorite is Epic Saga (RPG, not
			// action) → energy (and any earlier dims) get relaxed.
			mood:        mood(shuffle.EnergyAdrenaline, shuffle.TimeMedium, shuffle.FamiliarityFavorite, ""),
			wantAppID:   3,
			wantRelaxed: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := domain.NewService(&fakeStore{candidates: lib()})
			res, err := svc.Shuffle(t.Context(), "p1", tt.mood)
			if err != nil {
				t.Fatalf("shuffle: %v", err)
			}
			if res.AppID != tt.wantAppID {
				t.Fatalf("picked %d (%s), want %d", res.AppID, res.Name, tt.wantAppID)
			}
			if len(res.Relaxed) != tt.wantRelaxed {
				t.Fatalf("relaxed = %v, want %d dims", res.Relaxed, tt.wantRelaxed)
			}
			if res.Why == "" {
				t.Fatal("why is empty")
			}
			if res.UsedAI {
				t.Fatal("non-AI shuffle marked usedAI")
			}
		})
	}
}

func TestShuffleBudgetAndNoRepeat(t *testing.T) {
	t.Parallel()
	store := &fakeStore{candidates: lib()}
	svc := domain.NewService(store)
	m := mood(shuffle.EnergyBalanced, shuffle.TimeMedium, shuffle.FamiliaritySurprise, "")

	picked := map[int64]bool{}
	var lefts []int
	for i := 0; i < shuffle.DailyBudget; i++ {
		res, err := svc.Shuffle(t.Context(), "p1", m)
		if err != nil {
			t.Fatalf("shuffle %d: %v", i+1, err)
		}
		if picked[res.AppID] {
			t.Fatalf("appid %d repeated within the same day", res.AppID)
		}
		picked[res.AppID] = true
		lefts = append(lefts, res.ShufflesLeft)
	}
	if lefts[0] != 2 || lefts[1] != 1 || lefts[2] != 0 {
		t.Fatalf("shufflesLeft sequence = %v, want [2 1 0]", lefts)
	}

	if _, err := svc.Shuffle(t.Context(), "p1", m); !errors.Is(err, shuffle.ErrBudgetSpent) {
		t.Fatalf("4th shuffle err = %v, want ErrBudgetSpent", err)
	}

	// A different Player has a fresh budget.
	if _, err := svc.Shuffle(t.Context(), "p2", m); err != nil {
		t.Fatalf("other player blocked: %v", err)
	}
}

func TestShuffleEmptyLibrary(t *testing.T) {
	t.Parallel()
	svc := domain.NewService(&fakeStore{})
	m := mood(shuffle.EnergyChill, shuffle.TimeQuick, shuffle.FamiliaritySurprise, "")
	_, err := svc.Shuffle(t.Context(), "p1", m)
	if !errors.Is(err, shuffle.ErrNoCandidates) {
		t.Fatalf("err = %v, want ErrNoCandidates", err)
	}
}

func TestShuffleInvalidMood(t *testing.T) {
	t.Parallel()
	svc := domain.NewService(&fakeStore{candidates: lib()})
	_, err := svc.Shuffle(t.Context(), "p1", shuffle.Mood{Energy: "hyped"})
	if !errors.Is(err, shuffle.ErrInvalidMood) {
		t.Fatalf("err = %v, want ErrInvalidMood", err)
	}
}

func TestUnenrichedNeverMatchesTagFilters(t *testing.T) {
	t.Parallel()
	store := &fakeStore{candidates: []shuffle.Candidate{
		{AppID: 4, Name: "Mystery Box", Enriched: false, PlaytimeMin: 0},
	}}
	svc := domain.NewService(store)
	// Chill requires tags; the only game is unenriched → energy relaxes,
	// surprise (playtime) still matches.
	res, err := svc.Shuffle(t.Context(), "p1",
		mood(shuffle.EnergyChill, shuffle.TimeMedium, shuffle.FamiliaritySurprise, ""))
	if err != nil {
		t.Fatalf("shuffle: %v", err)
	}
	if res.AppID != 4 || len(res.Relaxed) == 0 {
		t.Fatalf("got appid=%d relaxed=%v, want 4 with energy relaxed", res.AppID, res.Relaxed)
	}
}
