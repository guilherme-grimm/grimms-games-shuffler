// Package player is the contract layer for identity: Steam login, sessions
// and the Player's Library.
package player

import "time"

// Player is a person identified by a verified SteamID64.
type Player struct {
	SteamID     string
	PersonaName string
	AvatarURL   string
	LastSyncAt  *time.Time
}

// Session is a server-side login session. Token is the raw secret handed to
// the client; storage only ever sees its hash.
type Session struct {
	Token     string
	SteamID   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// OwnedGame is one Library entry as reported by Steam.
type OwnedGame struct {
	AppID       int64
	Name        string
	PlaytimeMin int64
	LastPlayed  *time.Time
}
