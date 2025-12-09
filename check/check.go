package check

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	defaultTimeout    = 5 * time.Second
	defaultRetries    = 3
	defaultRetryDelay = time.Second
)

// Config holds common configuration for health checks.
type Config struct {
	Timeout    time.Duration
	Retries    int
	RetryDelay time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout:    defaultTimeout,
		Retries:    defaultRetries,
		RetryDelay: defaultRetryDelay,
	}
}

// WithTimeout sets the timeout for the health check.
func (c Config) WithTimeout(timeout time.Duration) Config {
	c.Timeout = timeout

	return c
}

// WithRetries sets the number of retries for the health check.
func (c Config) WithRetries(retries int) Config {
	c.Retries = retries

	return c
}

// WithRetryDelay sets the delay between retries.
func (c Config) WithRetryDelay(delay time.Duration) Config {
	c.RetryDelay = delay

	return c
}

// Check is the interface that all health checks must implement.
type Check interface {
	// Check performs the health check and returns an error if unhealthy
	Check(ctx context.Context) error
}

// CheckFunc is a function type that implements the Check interface.
type CheckFunc func(ctx context.Context) error

// Check implements the Check interface for CheckFunc.
func (f CheckFunc) Check(ctx context.Context) error {
	return f(ctx)
}

// WithTimeout wraps a Check with a timeout.
func WithTimeout(check Check, timeout time.Duration) Check {
	return CheckFunc(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return check.Check(ctx)
	})
}

// WithRetries wraps a Check with retry logic.
func WithRetries(check Check, attempts int, delay time.Duration) Check {
	return CheckFunc(func(ctx context.Context) error {
		var lastErr error
		for i := 0; i < attempts; i++ {
			err := check.Check(ctx)
			if err == nil {
				return nil
			}
			lastErr = err

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		return lastErr
	})
}

// All creates a health check that requires all checks to pass.
func All(checks ...Check) Check {
	return CheckFunc(func(ctx context.Context) error {
		g, ctx := errgroup.WithContext(ctx)

		for _, check := range checks {
			g.Go(func() error {
				return check.Check(ctx)
			})
		}

		if err := g.Wait(); err != nil {
			return fmt.Errorf("waiting errgroup: %w", err)
		}

		return nil
	})
}

// Any creates a health check that requires at least one check to pass.
func Any(checks ...Check) Check {
	return CheckFunc(func(ctx context.Context) error {
		g, ctx := errgroup.WithContext(ctx)
		results := make(chan error, len(checks))

		for _, check := range checks {
			g.Go(func() error {
				err := check.Check(ctx)
				select {
				case results <- err:
				case <-ctx.Done():
					return ctx.Err()
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return fmt.Errorf("waiting errgroup: %w", err)
		}
		close(results)

		var lastErr error
		success := false
		for err := range results {
			if err == nil {
				success = true

				break
			}
			lastErr = err
		}

		if !success {
			return fmt.Errorf("all checks failed: %w", lastErr)
		}

		return nil
	})
}

// WithThreshold creates a health check that requires a minimum number of checks to pass.
func WithThreshold(threshold int, checks ...Check) Check {
	return CheckFunc(func(ctx context.Context) error {
		g, ctx := errgroup.WithContext(ctx)
		results := make(chan error, len(checks))

		for _, check := range checks {
			g.Go(func() error {
				err := check.Check(ctx)
				select {
				case results <- err:
				case <-ctx.Done():
					return ctx.Err()
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return fmt.Errorf("waiting errgroup: %w", err)
		}
		close(results)

		successCount := 0
		var lastErr error
		for err := range results {
			if err == nil {
				successCount++
			} else {
				lastErr = err
			}
		}

		if successCount < threshold {
			return fmt.Errorf("insufficient successful checks: got %d, want %d: %w",
				successCount, threshold, lastErr)
		}

		return nil
	})
}
