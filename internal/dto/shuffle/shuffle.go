// Package shuffle is the contract layer for the Shuffle: Mood in, one
// recommendation with a Why out, 3 per Player per UTC day.
package shuffle

import (
	"context"
	"errors"
	"time"
)

// Answer values for each Mood question. Empty means "not asked / any".
const (
	EnergyChill      = "chill"
	EnergyBalanced   = "balanced"
	EnergyAdrenaline = "adrenaline"

	TimeQuick  = "quick"
	TimeMedium = "medium"
	TimeLong   = "long"

	FamiliarityFavorite = "favorite"
	FamiliarityBacklog  = "backlog"
	FamiliaritySurprise = "surprise"

	BrainStory  = "story"
	BrainPuzzle = "puzzle"
	BrainReflex = "reflex"
)

// Mood is a Player's answers to the fixed questionnaire. Brain is optional.
// Note is free text read only when AI mode is on.
type Mood struct {
	Energy      string `json:"energy"`
	Time        string `json:"time"`
	Familiarity string `json:"familiarity"`
	Brain       string `json:"brain,omitempty"`
	Note        string `json:"note,omitempty"`
}

// Valid reports whether every answered dimension uses a known value.
func (m Mood) Valid() bool {
	ok := func(v string, allowed ...string) bool {
		for _, a := range allowed {
			if v == a {
				return true
			}
		}
		return false
	}
	return ok(m.Energy, EnergyChill, EnergyBalanced, EnergyAdrenaline) &&
		ok(m.Time, TimeQuick, TimeMedium, TimeLong) &&
		ok(m.Familiarity, FamiliarityFavorite, FamiliarityBacklog, FamiliaritySurprise) &&
		(m.Brain == "" || ok(m.Brain, BrainStory, BrainPuzzle, BrainReflex))
}

// Candidate is one Library entry joined with its Catalog metadata.
type Candidate struct {
	AppID       int64
	Name        string
	PlaytimeMin int64
	Tags        []string
	Enriched    bool
}

// Result is one Shuffle outcome.
type Result struct {
	AppID        int64    `json:"appId"`
	Name         string   `json:"name"`
	Why          string   `json:"why"`
	UsedAI       bool     `json:"usedAi"`
	ShufflesLeft int      `json:"shufflesLeft"`
	Relaxed      []string `json:"relaxed,omitempty"`
}

// Record is a persisted Shuffle, used for budget and no-repeat rules.
type Record struct {
	SteamID   string
	DateUTC   string // YYYY-MM-DD
	AppID     int64
	Mood      Mood
	UsedAI    bool
	Why       string
	CreatedAt time.Time
}

// DailyBudget is how many Shuffles a Player gets per UTC day.
const DailyBudget = 3

var (
	// ErrBudgetSpent means the Player used all Shuffles for the UTC day.
	ErrBudgetSpent = errors.New("shuffle: daily budget spent")
	// ErrNoCandidates means even fully relaxed filters matched nothing —
	// an unusable Library. Never burns budget.
	ErrNoCandidates = errors.New("shuffle: no candidates in library")
	// ErrInvalidMood means an answer value is unknown.
	ErrInvalidMood = errors.New("shuffle: invalid mood")
)

// Service is the driving contract. useAI is honored only when the Instance
// has a Picker configured; otherwise the Shuffle silently stays non-AI.
type Service interface {
	Shuffle(ctx context.Context, steamID string, mood Mood, useAI bool) (Result, error)
	// Left returns remaining Shuffles and when the budget resets.
	Left(ctx context.Context, steamID string) (int, time.Time, error)
	// AIAvailable reports whether this Instance can garnish with AI.
	AIAvailable() bool
}

// Picker is the driven contract for the AI garnish: given the deterministic
// Candidates it returns the chosen appid and a Why. It must never pick
// outside the given list (ADR 0001).
type Picker interface {
	Pick(ctx context.Context, mood Mood, candidates []Candidate) (appID int64, why string, err error)
}

// Storage is the driven contract for Shuffle persistence and candidates.
type Storage interface {
	CountToday(ctx context.Context, steamID, dateUTC string) (int, error)
	TodaysAppIDs(ctx context.Context, steamID, dateUTC string) ([]int64, error)
	Insert(ctx context.Context, r Record) error
	// Candidates returns the Player's whole Library with metadata joined.
	Candidates(ctx context.Context, steamID string) ([]Candidate, error)
}
