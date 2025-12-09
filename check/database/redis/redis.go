package redis

import (
	"context"
	"fmt"

	"github.com/alarmistdev/status/check"
	"github.com/redis/go-redis/v9"
)

// Check creates a health check for Redis.
func Check(addr string, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		client := redis.NewClient(&redis.Options{
			Addr:         addr,
			DialTimeout:  config.Timeout,
			ReadTimeout:  config.Timeout,
			WriteTimeout: config.Timeout,
		})
		defer client.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("failed to ping redis: %w", err)
		}

		return nil
	})
}

// CheckWithAuth creates a health check for Redis with authentication.
func CheckWithAuth(addr, username, password string, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		client := redis.NewClient(&redis.Options{
			Addr:         addr,
			Username:     username,
			Password:     password,
			DialTimeout:  config.Timeout,
			ReadTimeout:  config.Timeout,
			WriteTimeout: config.Timeout,
		})
		defer client.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("failed to ping redis: %w", err)
		}

		return nil
	})
}
