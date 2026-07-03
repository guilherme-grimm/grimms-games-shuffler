// Package steam is the driven adapter for Steam: OpenID 2.0 login
// verification and the Steam Web API.
package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/guilherme-grimm/ggs/internal/dto/player"
)

const (
	_openIDEndpoint = "https://steamcommunity.com/openid/login"
	_apiBase        = "https://api.steampowered.com"
)

var _claimedIDRe = regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d{17})$`)

// Client implements player.SteamClient.
type Client struct {
	http   *http.Client
	apiKey string
}

var _ player.SteamClient = (*Client)(nil)

// NewClient returns a Steam client authenticated with the Web API key.
func NewClient(apiKey string) *Client {
	return &Client{
		http:   &http.Client{Timeout: 15 * time.Second},
		apiKey: apiKey,
	}
}

// AuthURL builds the Steam OpenID 2.0 checkid_setup redirect.
func (c *Client) AuthURL(returnTo string) string {
	realm := returnTo
	if u, err := url.Parse(returnTo); err == nil {
		realm = u.Scheme + "://" + u.Host
	}
	q := url.Values{
		"openid.ns":         {"http://specs.openid.net/auth/2.0"},
		"openid.mode":       {"checkid_setup"},
		"openid.return_to":  {returnTo},
		"openid.realm":      {realm},
		"openid.identity":   {"http://specs.openid.net/auth/2.0/identifier_select"},
		"openid.claimed_id": {"http://specs.openid.net/auth/2.0/identifier_select"},
	}
	return _openIDEndpoint + "?" + q.Encode()
}

// VerifyCallback replays the assertion to Steam with check_authentication
// and extracts the SteamID64 from the claimed_id.
func (c *Client) VerifyCallback(ctx context.Context, callback url.Values) (string, error) {
	verify := url.Values{}
	for k, vs := range callback {
		if strings.HasPrefix(k, "openid.") {
			verify[k] = vs
		}
	}
	verify.Set("openid.mode", "check_authentication")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, _openIDEndpoint,
		strings.NewReader(verify.Encode()))
	if err != nil {
		return "", fmt.Errorf("build verify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("openid verify: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("read verify response: %w", err)
	}
	if !strings.Contains(string(body), "is_valid:true") {
		return "", player.ErrOpenIDVerify
	}

	m := _claimedIDRe.FindStringSubmatch(callback.Get("openid.claimed_id"))
	if m == nil {
		return "", fmt.Errorf("%w: unexpected claimed_id", player.ErrOpenIDVerify)
	}
	return m[1], nil
}

// Summary fetches persona name and avatar via GetPlayerSummaries v2.
func (c *Client) Summary(ctx context.Context, steamID string) (string, string, error) {
	var out struct {
		Response struct {
			Players []struct {
				PersonaName string `json:"personaname"`
				AvatarFull  string `json:"avatarfull"`
			} `json:"players"`
		} `json:"response"`
	}
	q := url.Values{"key": {c.apiKey}, "steamids": {steamID}}
	if err := c.getJSON(ctx, "/ISteamUser/GetPlayerSummaries/v2/", q, &out); err != nil {
		return "", "", err
	}
	if len(out.Response.Players) == 0 {
		return "", "", fmt.Errorf("summary: steamid %s not found", steamID)
	}
	p := out.Response.Players[0]
	return p.PersonaName, p.AvatarFull, nil
}

// OwnedGames fetches the library via GetOwnedGames v1. A response without a
// game_count means the profile's game details are private.
func (c *Client) OwnedGames(ctx context.Context, steamID string) ([]player.OwnedGame, error) {
	var out struct {
		Response struct {
			GameCount *int `json:"game_count"`
			Games     []struct {
				AppID           int64  `json:"appid"`
				Name            string `json:"name"`
				PlaytimeForever int64  `json:"playtime_forever"`
				RTimeLastPlayed int64  `json:"rtime_last_played"`
			} `json:"games"`
		} `json:"response"`
	}
	q := url.Values{
		"key":                       {c.apiKey},
		"steamid":                   {steamID},
		"include_appinfo":           {"1"},
		"include_played_free_games": {"1"},
	}
	if err := c.getJSON(ctx, "/IPlayerService/GetOwnedGames/v1/", q, &out); err != nil {
		return nil, err
	}
	if out.Response.GameCount == nil {
		return nil, player.ErrPrivateLibrary
	}
	games := make([]player.OwnedGame, 0, len(out.Response.Games))
	for _, g := range out.Response.Games {
		og := player.OwnedGame{
			AppID:       g.AppID,
			Name:        g.Name,
			PlaytimeMin: g.PlaytimeForever,
		}
		if g.RTimeLastPlayed > 0 {
			t := time.Unix(g.RTimeLastPlayed, 0).UTC()
			og.LastPlayed = &t
		}
		games = append(games, og)
	}
	return games, nil
}

func (c *Client) getJSON(ctx context.Context, path string, q url.Values, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		_apiBase+path+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("steam api %s: %w", path, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("steam api %s: status %d", path, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
