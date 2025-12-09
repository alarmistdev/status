package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alarmistdev/status/check"
	_ "github.com/lib/pq" // register postgres driver
)

// Check creates a health check for PostgreSQL.
func Check(dsn string, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to postgres: %w", err)
		}
		defer db.Close()

		ctx, cancel := context.WithTimeout(ctx, config.Timeout)
		defer cancel()

		return db.PingContext(ctx)
	})
}
