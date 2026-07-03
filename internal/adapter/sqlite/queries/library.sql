-- name: DeleteLibrary :exec
DELETE FROM library WHERE steam_id = ?;

-- name: InsertLibraryGame :exec
INSERT INTO library (steam_id, appid, playtime_min, last_played_at)
VALUES (?, ?, ?, ?);

-- name: CountLibrary :one
SELECT COUNT(*) FROM library WHERE steam_id = ?;

-- name: EnsureGame :exec
INSERT INTO games (appid, name)
VALUES (?, ?)
ON CONFLICT (appid) DO UPDATE SET name = excluded.name
WHERE games.name = '';
