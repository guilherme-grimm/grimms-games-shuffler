-- name: UpsertPlayer :exec
INSERT INTO players (steam_id, persona_name, avatar_url)
VALUES (?, ?, ?)
ON CONFLICT (steam_id) DO UPDATE SET
    persona_name = excluded.persona_name,
    avatar_url   = excluded.avatar_url;

-- name: GetPlayer :one
SELECT * FROM players WHERE steam_id = ?;

-- name: TouchLastSync :exec
UPDATE players SET last_sync_at = ? WHERE steam_id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (token, steam_id, created_at, expires_at)
VALUES (?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions WHERE token = ? AND expires_at > ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= ?;
