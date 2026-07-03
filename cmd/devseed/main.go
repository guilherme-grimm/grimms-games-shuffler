// Command devseed seeds a scratch GGS database with a fake player, session and library so
// the HTTP flow can be driven without a real Steam login.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite"
)

func main() {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, os.Args[1]+"/ggs.db")
	if err != nil {
		panic(err)
	}
	defer func() { _ = db.Close() }()

	token := "testtoken"
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	now := time.Now().UTC().Format(time.RFC3339)
	exp := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	mustExec := func(q string, args ...any) {
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			panic(fmt.Sprintf("%s: %v", q, err))
		}
	}
	mustExec(`INSERT INTO players (steam_id, persona_name) VALUES ('76561198000000001','tester')`)
	mustExec(`INSERT INTO sessions (token, steam_id, created_at, expires_at) VALUES (?,?,?,?)`,
		hash, "76561198000000001", now, exp)

	// Real appids so Steam CDN artwork resolves in the result screen.
	games := []struct {
		appid    int64
		name     string
		tags     string
		playtime int64
	}{
		{413150, "Stardew Valley", `["Relaxing","Farming Sim","Pixel Graphics"]`, 0},
		{1145360, "Hades", `["Action","Roguelike","Fast-Paced"]`, 30},
		{292030, "The Witcher 3: Wild Hunt", `["RPG","Story Rich","Open World"]`, 3000},
		{736260, "Baba Is You", `["Puzzle","Logic","2D"]`, 90},
		{753640, "Outer Wilds", `[]`, 0},
	}
	for _, g := range games {
		enriched := "2026-01-01T00:00:00Z"
		src := "steamspy"
		if g.appid == 753640 {
			mustExec(`INSERT INTO games (appid, name) VALUES (?,?)`, g.appid, g.name)
		} else {
			mustExec(`INSERT INTO games (appid, name, tags, source, enriched_at) VALUES (?,?,?,?,?)`,
				g.appid, g.name, g.tags, src, enriched)
		}
		mustExec(`INSERT INTO library (steam_id, appid, playtime_min) VALUES (?,?,?)`,
			"76561198000000001", g.appid, g.playtime)
	}
	fmt.Println("seeded; cookie: ggs_session=" + token)
}
