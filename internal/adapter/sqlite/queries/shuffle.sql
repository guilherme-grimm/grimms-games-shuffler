-- name: CountShufflesToday :one
SELECT COUNT(*) FROM shuffles WHERE steam_id = ? AND date_utc = ?;

-- name: TodaysShuffledAppids :many
SELECT appid FROM shuffles WHERE steam_id = ? AND date_utc = ?;

-- name: InsertShuffle :exec
INSERT INTO shuffles (steam_id, date_utc, appid, mood, used_ai, why, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: LibraryCandidates :many
SELECT l.appid, g.name, l.playtime_min, g.tags,
       g.enriched_at IS NOT NULL AND g.source != 'failed' AS enriched
FROM library l
JOIN games g ON g.appid = l.appid
WHERE l.steam_id = ?;
