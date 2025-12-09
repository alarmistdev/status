package tcp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/alarmistdev/status/check"
)

// Check creates a health check for a TCP connection
func Check(host string, port int) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", addr, err)
		}
		conn.Close()
		return nil
	})
}
