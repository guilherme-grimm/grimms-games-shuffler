// Package sqlite is the storage adapter: connection setup, embedded
// migrations, and the sqlc-generated query layer in gen/.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	// Registers the pure-Go "sqlite" database/sql driver.
	_ "modernc.org/sqlite"
)

// Open opens (creating if needed) the GGS database at path and applies
// pending migrations. The returned handle is safe for concurrent use.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	// modernc.org/sqlite serializes writes; a single writer connection avoids
	// SQLITE_BUSY under concurrent request load.
	db.SetMaxOpenConns(1)

	if err := Migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}
