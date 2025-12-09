package dns

import (
	"context"
	"fmt"
	"net"

	"github.com/alarmistdev/status/check"
)

// Check creates a health check for DNS resolution.
func Check(host string) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		_, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", host, err)
		}

		return nil
	})
}
