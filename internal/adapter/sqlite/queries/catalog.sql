-- name: NextUnenriched :many
SELECT appid FROM games
WHERE enriched_at IS NULL
ORDER BY appid
LIMIT ?;

-- name: SaveEnrichment :exec
UPDATE games
SET tags = ?, genres = ?, source = ?, enriched_at = ?
WHERE appid = ?;

-- name: EnrichmentProgress :one
SELECT
    COUNT(*)                                          AS total,
    COUNT(g.enriched_at)                              AS enriched
FROM library l
JOIN games g ON g.appid = l.appid
WHERE l.steam_id = ?;
