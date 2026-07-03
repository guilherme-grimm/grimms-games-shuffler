package shuffle

import "github.com/guilherme-grimm/ggs/internal/dto/shuffle"

// Tag lists per Mood answer. A candidate matches a dimension when it has at
// least one of the listed tags (SteamSpy community tags, top-20 by votes).
var _moodTags = map[string]map[string][]string{
	"energy": {
		shuffle.EnergyChill: {
			"Relaxing", "Casual", "Cozy", "Atmospheric", "Cute", "Wholesome",
			"Farming Sim", "Walking Simulator", "Exploration", "Sandbox",
		},
		shuffle.EnergyAdrenaline: {
			"Action", "Fast-Paced", "Shooter", "FPS", "Bullet Hell", "Racing",
			"Fighting", "Hack and Slash", "Survival", "Battle Royale",
		},
	},
	"time": {
		shuffle.TimeQuick: {
			"Arcade", "Short", "Casual", "Roguelike", "Roguelite", "Card Game",
			"Puzzle", "Platformer", "Score Attack",
		},
		shuffle.TimeLong: {
			"RPG", "Open World", "Story Rich", "Strategy", "4X", "Grand Strategy",
			"Simulation", "MMORPG", "JRPG", "Base Building", "City Builder",
		},
	},
	"brain": {
		shuffle.BrainStory: {
			"Story Rich", "Narrative", "Visual Novel", "Adventure", "RPG",
			"Choices Matter", "Interactive Fiction",
		},
		shuffle.BrainPuzzle: {
			"Puzzle", "Logic", "Strategy", "Turn-Based", "Turn-Based Strategy",
			"Card Game", "Programming",
		},
		shuffle.BrainReflex: {
			"Action", "Platformer", "Shooter", "Fighting", "Rhythm", "Bullet Hell",
			"Fast-Paced", "Precision Platformer",
		},
	},
}

// Familiarity thresholds in minutes.
const (
	_favoriteMinPlaytime = 10 * 60 // an old favorite: 10h+
	_backlogMaxPlaytime  = 2 * 60  // backlog shame: touched under 2h
)

// dimension is one relaxable filter over candidates.
type dimension struct {
	name  string
	match func(shuffle.Candidate) bool
}

// dimensions builds the active filters for a Mood, ordered so that relaxing
// walks the fixed order Brain → Energy → Time → Familiarity (least to most
// essential; relax drops from the front).
func dimensions(m shuffle.Mood) []dimension {
	var dims []dimension
	if tags := _moodTags["brain"][m.Brain]; tags != nil {
		dims = append(dims, dimension{"brain", hasAnyTag(tags)})
	}
	if tags := _moodTags["energy"][m.Energy]; tags != nil {
		dims = append(dims, dimension{"energy", hasAnyTag(tags)})
	}
	if tags := _moodTags["time"][m.Time]; tags != nil {
		dims = append(dims, dimension{"time", hasAnyTag(tags)})
	}
	switch m.Familiarity {
	case shuffle.FamiliarityFavorite:
		dims = append(dims, dimension{"familiarity", func(c shuffle.Candidate) bool {
			return c.PlaytimeMin >= _favoriteMinPlaytime
		}})
	case shuffle.FamiliarityBacklog:
		dims = append(dims, dimension{"familiarity", func(c shuffle.Candidate) bool {
			return c.PlaytimeMin > 0 && c.PlaytimeMin < _backlogMaxPlaytime
		}})
	case shuffle.FamiliaritySurprise:
		dims = append(dims, dimension{"familiarity", func(c shuffle.Candidate) bool {
			return c.PlaytimeMin == 0
		}})
	}
	return dims
}

// hasAnyTag requires enrichment: unenriched games never match tag filters
// (they still qualify once every tag dimension is relaxed away).
func hasAnyTag(wanted []string) func(shuffle.Candidate) bool {
	set := make(map[string]struct{}, len(wanted))
	for _, t := range wanted {
		set[t] = struct{}{}
	}
	return func(c shuffle.Candidate) bool {
		if !c.Enriched {
			return false
		}
		for _, t := range c.Tags {
			if _, ok := set[t]; ok {
				return true
			}
		}
		return false
	}
}
