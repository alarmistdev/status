package rabbitmq

import (
	"context"
	"fmt"
	"net"

	"github.com/alarmistdev/status/check"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Check creates a health check for RabbitMQ.
func Check(url string, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		// Create a connection with timeout
		conn, err := amqp.DialConfig(url, amqp.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
				}

				dialer := &net.Dialer{Timeout: config.Timeout}

				return dialer.DialContext(ctx, network, addr)
			},
		})
		if err != nil {
			return fmt.Errorf("failed to connect to rabbitmq: %w", err)
		}
		defer conn.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ch, err := conn.Channel()
		if err != nil {
			return fmt.Errorf("failed to open rabbitmq channel: %w", err)
		}
		defer ch.Close()

		return nil
	})
}
