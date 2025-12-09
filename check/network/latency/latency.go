package latency

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/alarmistdev/status/check"
)

// Check creates a health check for network latency
func Check(host string, port int, maxLatency time.Duration) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
		start := time.Now()

		dialer := &net.Dialer{Timeout: maxLatency}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", addr, err)
		}
		defer conn.Close()

		latency := time.Since(start)
		if latency > maxLatency {
			return fmt.Errorf("high latency: %v (maximum: %v)", latency, maxLatency)
		}

		return nil
	})
}
