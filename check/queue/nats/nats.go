package nats

import (
	"context"
	"fmt"

	"github.com/alarmistdev/status/check"
	"github.com/nats-io/nats.go"
)

// Check creates a health check for NATS
func Check(url string, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		nc, err := nats.Connect(url,
			nats.Timeout(config.Timeout),
			nats.ReconnectWait(config.RetryDelay),
			nats.MaxReconnects(config.Retries),
		)
		if err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		defer nc.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if we can publish a test message
		err = nc.Publish("health.check", []byte("test"))
		if err != nil {
			return fmt.Errorf("failed to publish test message to nats: %w", err)
		}

		return nil
	})
}
