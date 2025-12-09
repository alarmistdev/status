package tcp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/alarmistdev/status/check"
)

// Check creates a health check for a TCP connection.
func Check(host string, port int) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		const defaultDialTimeout = 5 * time.Second

		addr := net.JoinHostPort(host, strconv.Itoa(port))
		dialer := net.Dialer{Timeout: defaultDialTimeout}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", addr, err)
		}
		conn.Close()

		return nil
	})
}
