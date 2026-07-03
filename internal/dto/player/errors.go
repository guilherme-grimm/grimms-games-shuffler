package player

import "errors"

var (
	// ErrPrivateLibrary means Steam returned no game data because the
	// profile's game details are not public.
	ErrPrivateLibrary = errors.New("player: steam library is private")

	// ErrInvalidSession means the session token is unknown or expired.
	ErrInvalidSession = errors.New("player: invalid session")

	// ErrOpenIDVerify means Steam rejected the OpenID callback signature.
	ErrOpenIDVerify = errors.New("player: openid verification failed")

	// ErrNotConfigured means the Instance lacks STEAM_API_KEY / BASE_URL.
	ErrNotConfigured = errors.New("player: steam auth not configured")
)
