package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alarmistdev/status/check"
)

// Check creates a health check for HTTP endpoints with custom path and expected status.
func Check(method, url string, expectedStatus int, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		client := &http.Client{Timeout: config.Timeout}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != expectedStatus {
			return fmt.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, expectedStatus)
		}

		return nil
	})
}

// CheckGraphQL creates a health check for GraphQL endpoints
func CheckGraphQL(method, url string, expectedStatus int, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		query := `{ __schema { types { name } } }`
		body := map[string]string{"query": query}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: config.Timeout}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != expectedStatus {
			return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
		}

		return nil
	})
}
