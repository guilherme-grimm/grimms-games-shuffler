package player

import (
	"context"
	"net/url"
	"time"
)

// Service is the driving contract for login, sessions and Sync.
type Service interface {
	// AuthURL returns the Steam OpenID redirect URL; state is echoed back
	// on the callback for CSRF checking.
	AuthURL(state string) string
	// CompleteLogin verifies the OpenID callback, upserts the Player and
	// opens a Session. The returned Session carries the raw token.
	CompleteLogin(ctx context.Context, callback url.Values) (Session, error)
	// Authenticate resolves a raw session token to its Player.
	Authenticate(ctx context.Context, token string) (Player, error)
	Logout(ctx context.Context, token string) error
	// Sync imports the Player's owned games; returns the Library size.
	Sync(ctx context.Context, steamID string) (int, error)
	LibraryCount(ctx context.Context, steamID string) (int, error)
}

// Storage is the driven contract for persisting players, sessions and
// libraries.
type Storage interface {
	UpsertPlayer(ctx context.Context, p Player) error
	GetPlayer(ctx context.Context, steamID string) (Player, error)
	TouchLastSync(ctx context.Context, steamID string, at time.Time) error

	CreateSession(ctx context.Context, tokenHash string, s Session) error
	// GetSession returns the session for tokenHash if it expires after now.
	GetSession(ctx context.Context, tokenHash string, now time.Time) (Session, error)
	DeleteSession(ctx context.Context, tokenHash string) error

	ReplaceLibrary(ctx context.Context, steamID string, games []OwnedGame) error
	CountLibrary(ctx context.Context, steamID string) (int, error)
}

// SteamClient is the driven contract for the Steam Web API + OpenID.
type SteamClient interface {
	AuthURL(returnTo string) string
	// VerifyCallback replays the OpenID response to Steam and returns the
	// verified SteamID64.
	VerifyCallback(ctx context.Context, callback url.Values) (string, error)
	// Summary fetches persona name and avatar for a SteamID64.
	Summary(ctx context.Context, steamID string) (name, avatarURL string, err error)
	// OwnedGames returns the library; ErrPrivateLibrary when not public.
	OwnedGames(ctx context.Context, steamID string) ([]OwnedGame, error)
}
