CREATE TABLE players (
    steam_id     TEXT PRIMARY KEY,
    persona_name TEXT NOT NULL DEFAULT '',
    avatar_url   TEXT NOT NULL DEFAULT '',
    last_sync_at TEXT
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    steam_id   TEXT NOT NULL REFERENCES players (steam_id),
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE library (
    steam_id       TEXT NOT NULL REFERENCES players (steam_id),
    appid          INTEGER NOT NULL,
    playtime_min   INTEGER NOT NULL DEFAULT 0,
    last_played_at TEXT,
    PRIMARY KEY (steam_id, appid)
);

CREATE TABLE games (
    appid       INTEGER PRIMARY KEY,
    name        TEXT NOT NULL DEFAULT '',
    tags        TEXT NOT NULL DEFAULT '[]',
    genres      TEXT NOT NULL DEFAULT '[]',
    enriched_at TEXT,
    source      TEXT NOT NULL DEFAULT ''
);

CREATE TABLE shuffles (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    steam_id   TEXT NOT NULL REFERENCES players (steam_id),
    date_utc   TEXT NOT NULL,
    appid      INTEGER NOT NULL,
    mood       TEXT NOT NULL,
    used_ai    INTEGER NOT NULL DEFAULT 0,
    why        TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX idx_shuffles_budget ON shuffles (steam_id, date_utc);
CREATE INDEX idx_sessions_expiry ON sessions (expires_at);
