package player_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	domain "github.com/guilherme-grimm/ggs/internal/domain/player"
	"github.com/guilherme-grimm/ggs/internal/dto/player"
)

type fakeStore struct {
	players   map[string]player.Player
	sessions  map[string]player.Session
	libraries map[string][]player.OwnedGame
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		players:   map[string]player.Player{},
		sessions:  map[string]player.Session{},
		libraries: map[string][]player.OwnedGame{},
	}
}

func (f *fakeStore) UpsertPlayer(_ context.Context, p player.Player) error {
	if existing, ok := f.players[p.SteamID]; ok {
		p.LastSyncAt = existing.LastSyncAt
	}
	f.players[p.SteamID] = p
	return nil
}

func (f *fakeStore) GetPlayer(_ context.Context, id string) (player.Player, error) {
	p, ok := f.players[id]
	if !ok {
		return player.Player{}, errors.New("not found")
	}
	return p, nil
}

func (f *fakeStore) TouchLastSync(_ context.Context, id string, at time.Time) error {
	p := f.players[id]
	p.LastSyncAt = &at
	f.players[id] = p
	return nil
}

func (f *fakeStore) CreateSession(_ context.Context, hash string, s player.Session) error {
	f.sessions[hash] = s
	return nil
}

func (f *fakeStore) GetSession(_ context.Context, hash string, now time.Time) (player.Session, error) {
	s, ok := f.sessions[hash]
	if !ok || !s.ExpiresAt.After(now) {
		return player.Session{}, player.ErrInvalidSession
	}
	return s, nil
}

func (f *fakeStore) DeleteSession(_ context.Context, hash string) error {
	delete(f.sessions, hash)
	return nil
}

func (f *fakeStore) ReplaceLibrary(_ context.Context, id string, games []player.OwnedGame) error {
	f.libraries[id] = games
	return nil
}

func (f *fakeStore) CountLibrary(_ context.Context, id string) (int, error) {
	return len(f.libraries[id]), nil
}

type fakeSteam struct {
	steamID   string
	verifyErr error
	games     []player.OwnedGame
	gamesErr  error
}

func (f *fakeSteam) AuthURL(returnTo string) string { return "https://steam.test/login?rt=" + returnTo }

func (f *fakeSteam) VerifyCallback(context.Context, url.Values) (string, error) {
	return f.steamID, f.verifyErr
}

func (f *fakeSteam) Summary(context.Context, string) (string, string, error) {
	return "grimm", "https://a.test/x.jpg", nil
}

func (f *fakeSteam) OwnedGames(context.Context, string) ([]player.OwnedGame, error) {
	return f.games, f.gamesErr
}

func TestCompleteLoginCreatesPlayerAndSession(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := domain.NewService(store, &fakeSteam{steamID: "765611980000"}, "https://ggs.test")

	sess, err := svc.CompleteLogin(t.Context(), url.Values{})
	if err != nil {
		t.Fatalf("complete login: %v", err)
	}
	if sess.Token == "" || len(sess.Token) < 32 {
		t.Fatalf("token too short: %q", sess.Token)
	}
	if _, ok := store.players["765611980000"]; !ok {
		t.Fatal("player not upserted")
	}
	// Raw token never stored.
	if _, ok := store.sessions[sess.Token]; ok {
		t.Fatal("raw token stored — must be hashed")
	}

	p, err := svc.Authenticate(t.Context(), sess.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if p.SteamID != "765611980000" {
		t.Fatalf("steam id = %s", p.SteamID)
	}
}

func TestAuthenticateRejectsBadToken(t *testing.T) {
	t.Parallel()
	svc := domain.NewService(newFakeStore(), &fakeSteam{}, "https://ggs.test")
	_, err := svc.Authenticate(t.Context(), "bogus")
	if !errors.Is(err, player.ErrInvalidSession) {
		t.Fatalf("err = %v, want ErrInvalidSession", err)
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	svc := domain.NewService(store, &fakeSteam{steamID: "1"}, "https://ggs.test")
	sess, err := svc.CompleteLogin(t.Context(), url.Values{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.Logout(t.Context(), sess.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Authenticate(t.Context(), sess.Token); !errors.Is(err, player.ErrInvalidSession) {
		t.Fatalf("err = %v, want ErrInvalidSession", err)
	}
}

func TestSync(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		steam     *fakeSteam
		wantCount int
		wantErr   error
	}{
		{
			name: "replaces library and touches last sync",
			steam: &fakeSteam{games: []player.OwnedGame{
				{AppID: 1145360, Name: "Hades", PlaytimeMin: 5000},
				{AppID: 413150, Name: "Stardew Valley"},
			}},
			wantCount: 2,
		},
		{
			name:    "private library",
			steam:   &fakeSteam{gamesErr: player.ErrPrivateLibrary},
			wantErr: player.ErrPrivateLibrary,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newFakeStore()
			store.players["7"] = player.Player{SteamID: "7"}
			svc := domain.NewService(store, tt.steam, "https://ggs.test")

			count, err := svc.Sync(t.Context(), "7")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if store.players["7"].LastSyncAt != nil {
					t.Fatal("last sync must not be touched on failure")
				}
				return
			}
			if err != nil {
				t.Fatalf("sync: %v", err)
			}
			if count != tt.wantCount {
				t.Fatalf("count = %d, want %d", count, tt.wantCount)
			}
			if store.players["7"].LastSyncAt == nil {
				t.Fatal("last sync not touched")
			}
		})
	}
}

func TestAuthURLCarriesState(t *testing.T) {
	t.Parallel()
	svc := domain.NewService(newFakeStore(), &fakeSteam{}, "https://ggs.test")
	got := svc.AuthURL("abc123")
	if !strings.Contains(got, "state%3Dabc123") && !strings.Contains(got, "state=abc123") {
		t.Fatalf("auth url missing state: %s", got)
	}
}
