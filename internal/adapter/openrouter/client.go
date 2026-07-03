// Package openrouter is the driven adapter for the AI garnish, talking to
// OpenRouter's OpenAI-compatible chat completions API.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/shuffle"
)

const _apiURL = "https://openrouter.ai/api/v1/chat/completions"

// Client implements shuffle.Picker.
type Client struct {
	http  *http.Client
	key   string
	model string
}

var _ shuffle.Picker = (*Client)(nil)

// NewClient returns an OpenRouter picker using the given model slug
// (e.g. "meta-llama/llama-3.3-70b-instruct:free").
func NewClient(key, model string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 30 * time.Second},
		key:   key,
		model: model,
	}
}

const _system = `You are GGS, a retro arcade machine that picks ONE game from a player's Steam library based on their mood. You will get a mood and a numbered list of candidate games (appid, name, tags, hours played).

Rules:
- Pick EXACTLY ONE game. Its appid MUST come from the list.
- "why": 1-2 sentences, second person, playful arcade-announcer voice, grounded in the game's tags/playtime and the mood (mention the mood note if there is one).
- Reply with ONLY this JSON, nothing else: {"appId": <number>, "why": "<string>"}`

// Pick implements shuffle.Picker.
func (c *Client) Pick(ctx context.Context, mood shuffle.Mood, candidates []shuffle.Candidate) (int64, string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Mood: energy=%s time=%s familiarity=%s", mood.Energy, mood.Time, mood.Familiarity)
	if mood.Brain != "" {
		fmt.Fprintf(&b, " brain=%s", mood.Brain)
	}
	if mood.Note != "" {
		fmt.Fprintf(&b, "\nMood note: %q", mood.Note)
	}
	b.WriteString("\n\nCandidates:\n")
	for _, cand := range candidates {
		fmt.Fprintf(&b, "- appid=%d %q tags=%s hours=%.1f\n",
			cand.AppID, cand.Name, strings.Join(cand.Tags, ","), float64(cand.PlaytimeMin)/60)
	}

	payload, err := json.Marshal(map[string]any{
		"model":       c.model,
		"temperature": 0.8,
		"messages": []map[string]string{
			{"role": "system", "content": _system},
			{"role": "user", "content": b.String()},
		},
	})
	if err != nil {
		return 0, "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, _apiURL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	// OpenRouter attribution headers (optional but polite).
	req.Header.Set("HTTP-Referer", "https://games.grimm0.dev")
	req.Header.Set("X-Title", "GGS - Grimm's Games Shuffler")

	res, err := c.http.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("openrouter: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("openrouter: status %d", res.StatusCode)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return 0, "", fmt.Errorf("decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return 0, "", fmt.Errorf("openrouter: empty choices")
	}
	return parsePick(out.Choices[0].Message.Content)
}

// parsePick extracts {"appId":..,"why":..} from the completion, tolerating
// markdown fences and chatter around the JSON — free models are sloppy.
func parsePick(content string) (int64, string, error) {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return 0, "", fmt.Errorf("no JSON object in completion %q", truncate(content))
	}
	var pick struct {
		AppID int64  `json:"appId"`
		Why   string `json:"why"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &pick); err != nil {
		return 0, "", fmt.Errorf("parse completion %q: %w", truncate(content), err)
	}
	if pick.AppID == 0 || pick.Why == "" {
		return 0, "", fmt.Errorf("incomplete pick in completion %q", truncate(content))
	}
	return pick.AppID, pick.Why, nil
}

func truncate(s string) string {
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
