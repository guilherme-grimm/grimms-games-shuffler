// Package shuffle implements the Shuffle: deterministic Mood filtering with
// fixed-order relaxation, a weighted random pick and a templated Why. AI, if
// enabled, only replaces the final pick + prose (ADR 0001).
package shuffle

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/shuffle"
)

// Service implements shuffle.Service.
type Service struct {
	store shuffle.Storage
	now   func() time.Time
	pick  func(weights []int) int
}

var _ shuffle.Service = (*Service)(nil)

// NewService wires the shuffle service.
func NewService(store shuffle.Storage) *Service {
	return &Service{store: store, now: time.Now, pick: weightedIndex}
}

// Shuffle implements shuffle.Service: budget check → filter → relax →
// weighted pick → templated Why → persist.
func (s *Service) Shuffle(ctx context.Context, steamID string, mood shuffle.Mood) (shuffle.Result, error) {
	if !mood.Valid() {
		return shuffle.Result{}, shuffle.ErrInvalidMood
	}
	now := s.now().UTC()
	today := now.Format(time.DateOnly)

	used, err := s.store.CountToday(ctx, steamID, today)
	if err != nil {
		return shuffle.Result{}, err
	}
	if used >= shuffle.DailyBudget {
		return shuffle.Result{}, shuffle.ErrBudgetSpent
	}

	all, err := s.store.Candidates(ctx, steamID)
	if err != nil {
		return shuffle.Result{}, err
	}
	seen, err := s.store.TodaysAppIDs(ctx, steamID, today)
	if err != nil {
		return shuffle.Result{}, err
	}
	pool := excludeSeen(all, seen)
	if len(pool) == 0 {
		return shuffle.Result{}, shuffle.ErrNoCandidates
	}

	candidates, relaxed := filterWithRelax(pool, dimensions(mood))
	if len(candidates) == 0 {
		// Every dimension relaxed and still nothing means pool itself was
		// non-empty, so this cannot happen; guard anyway.
		return shuffle.Result{}, shuffle.ErrNoCandidates
	}

	chosen := candidates[s.pick(weights(candidates))]
	why := templateWhy(chosen, relaxed)

	if err := s.store.Insert(ctx, shuffle.Record{
		SteamID: steamID, DateUTC: today, AppID: chosen.AppID,
		Mood: mood, UsedAI: false, Why: why, CreatedAt: now,
	}); err != nil {
		return shuffle.Result{}, err
	}
	return shuffle.Result{
		AppID: chosen.AppID, Name: chosen.Name, Why: why,
		UsedAI: false, ShufflesLeft: shuffle.DailyBudget - used - 1,
		Relaxed: relaxed,
	}, nil
}

// Left implements shuffle.Service.
func (s *Service) Left(ctx context.Context, steamID string) (int, time.Time, error) {
	now := s.now().UTC()
	used, err := s.store.CountToday(ctx, steamID, now.Format(time.DateOnly))
	if err != nil {
		return 0, time.Time{}, err
	}
	left := shuffle.DailyBudget - used
	if left < 0 {
		left = 0
	}
	reset := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	return left, reset, nil
}

func excludeSeen(all []shuffle.Candidate, seen []int64) []shuffle.Candidate {
	seenSet := make(map[int64]struct{}, len(seen))
	for _, id := range seen {
		seenSet[id] = struct{}{}
	}
	out := make([]shuffle.Candidate, 0, len(all))
	for _, c := range all {
		if _, ok := seenSet[c.AppID]; !ok {
			out = append(out, c)
		}
	}
	return out
}

// filterWithRelax applies all dimensions, dropping them front-first (Brain →
// Energy → Time → Familiarity) until candidates exist. Returns the surviving
// candidates and the names of relaxed dimensions.
func filterWithRelax(pool []shuffle.Candidate, dims []dimension) ([]shuffle.Candidate, []string) {
	for drop := 0; drop <= len(dims); drop++ {
		active := dims[drop:]
		var out []shuffle.Candidate
		for _, c := range pool {
			ok := true
			for _, d := range active {
				if !d.match(c) {
					ok = false
					break
				}
			}
			if ok {
				out = append(out, c)
			}
		}
		if len(out) > 0 {
			relaxed := make([]string, 0, drop)
			for _, d := range dims[:drop] {
				relaxed = append(relaxed, d.name)
			}
			return out, relaxed
		}
	}
	return nil, nil
}

// weights favor barely-played games so the Shuffle surfaces the backlog
// instead of the comfort picks.
func weights(cs []shuffle.Candidate) []int {
	w := make([]int, len(cs))
	for i, c := range cs {
		switch {
		case c.PlaytimeMin == 0:
			w[i] = 4
		case c.PlaytimeMin < _backlogMaxPlaytime:
			w[i] = 3
		case c.PlaytimeMin < _favoriteMinPlaytime:
			w[i] = 2
		default:
			w[i] = 1
		}
	}
	return w
}

func weightedIndex(w []int) int {
	total := 0
	for _, x := range w {
		total += x
	}
	n := rand.IntN(total)
	for i, x := range w {
		if n < x {
			return i
		}
		n -= x
	}
	return len(w) - 1
}

func templateWhy(c shuffle.Candidate, relaxed []string) string {
	var parts []string

	if len(c.Tags) > 0 {
		n := min(3, len(c.Tags))
		parts = append(parts, fmt.Sprintf("Tagged %s", strings.Join(c.Tags[:n], ", ")))
	} else {
		parts = append(parts, "Not much is known about this one yet")
	}

	switch {
	case c.PlaytimeMin == 0:
		parts = append(parts, "you've never launched it — a true surprise")
	case c.PlaytimeMin < _backlogMaxPlaytime:
		parts = append(parts, fmt.Sprintf("only %dh%02dm played — backlog material",
			c.PlaytimeMin/60, c.PlaytimeMin%60))
	case c.PlaytimeMin >= _favoriteMinPlaytime:
		parts = append(parts, fmt.Sprintf("%dh on the clock — an old favorite",
			c.PlaytimeMin/60))
	default:
		parts = append(parts, fmt.Sprintf("%dh%02dm played so far",
			c.PlaytimeMin/60, c.PlaytimeMin%60))
	}

	why := strings.Join(parts, "; ") + "."
	if len(relaxed) > 0 {
		why += fmt.Sprintf(" (Nothing matched your full mood — ignored: %s.)",
			strings.Join(relaxed, ", "))
	}
	return why
}
