package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alarmistdev/status/check"
	_ "github.com/go-sql-driver/mysql" // register mysql driver
)

// Check creates a health check for MySQL.
func Check(dsn string, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to mysql: %w", err)
		}
		defer db.Close()

		ctx, cancel := context.WithTimeout(ctx, config.Timeout)
		defer cancel()

		return db.PingContext(ctx)
	})
}
