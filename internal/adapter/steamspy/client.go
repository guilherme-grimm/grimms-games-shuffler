// Package steamspy is the driven adapter for SteamSpy, the Catalog's
// tag/genre metadata source.
package steamspy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/catalog"
)

const _apiURL = "https://steamspy.com/api.php"

// Client implements catalog.MetadataSource against the SteamSpy API.
type Client struct {
	http *http.Client
}

var _ catalog.MetadataSource = (*Client)(nil)

// NewClient returns a SteamSpy client. Callers own rate limiting (SteamSpy
// allows ~1 appdetails request per second).
func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 15 * time.Second}}
}

// Fetch pulls tags and genres for one appid. SteamSpy answers unknown apps
// with name "" — mapped to catalog.ErrNotFound.
func (c *Client) Fetch(ctx context.Context, appID int64) (catalog.Enrichment, error) {
	q := url.Values{
		"request": {"appdetails"},
		"appid":   {strconv.FormatInt(appID, 10)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, _apiURL+"?"+q.Encode(), nil)
	if err != nil {
		return catalog.Enrichment{}, fmt.Errorf("build steamspy request: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return catalog.Enrichment{}, fmt.Errorf("steamspy appdetails %d: %w", appID, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return catalog.Enrichment{}, fmt.Errorf("steamspy appdetails %d: status %d", appID, res.StatusCode)
	}

	var out struct {
		Name  string          `json:"name"`
		Genre string          `json:"genre"`
		Tags  json.RawMessage `json:"tags"` // object appid-known, [] when empty
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return catalog.Enrichment{}, fmt.Errorf("decode steamspy %d: %w", appID, err)
	}
	if out.Name == "" {
		return catalog.Enrichment{}, catalog.ErrNotFound
	}

	tags := map[string]int{}
	// SteamSpy sends {} of tag→votes normally but [] for tagless games.
	_ = json.Unmarshal(out.Tags, &tags)
	byVotes := make([]string, 0, len(tags))
	for t := range tags {
		byVotes = append(byVotes, t)
	}
	sort.Slice(byVotes, func(i, j int) bool {
		if tags[byVotes[i]] != tags[byVotes[j]] {
			return tags[byVotes[i]] > tags[byVotes[j]]
		}
		return byVotes[i] < byVotes[j]
	})
	if len(byVotes) > 20 {
		byVotes = byVotes[:20]
	}

	var genres []string
	for _, g := range strings.Split(out.Genre, ",") {
		if g = strings.TrimSpace(g); g != "" {
			genres = append(genres, g)
		}
	}

	return catalog.Enrichment{
		AppID:  appID,
		Tags:   byVotes,
		Genres: genres,
		Source: "steamspy",
	}, nil
}
