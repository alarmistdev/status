package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/alarmistdev/status"
	"github.com/alarmistdev/status/check"
	httpcheck "github.com/alarmistdev/status/check/network/http"
	icmpcheck "github.com/alarmistdev/status/check/network/icmp"
	latencycheck "github.com/alarmistdev/status/check/network/latency"
)

func main() {
	// Create a health checker
	healthChecker := status.NewHealthChecker().
		WithTarget("HTTP Google", status.TargetImportanceHigh, httpcheck.Check(http.MethodGet, "https://google.com", 200, check.Config{})).
		WithTarget("ICMP Google", status.TargetImportanceHigh, icmpcheck.Check("google.com")).
		WithTarget("Latency Google", status.TargetImportanceHigh, latencycheck.Check("google.com", 443, 500*time.Millisecond)).
		WithTarget("Database", status.TargetImportanceHigh, check.CheckFunc(func(_ context.Context) error {
			// Implement your database health check here
			return generateRandomError()
		})).
		WithTarget("Network", status.TargetImportanceLow, check.CheckFunc(func(_ context.Context) error {
			// Implement your network health check here
			return generateRandomError()
		}))

	// Create a status page
	statusPage := status.NewPage(
		// Add health checker to status page
		status.WithHealthChecker(healthChecker),
		// Add additional links
		status.WithLink("OpenAPI Documentation", "/swagger"),
		status.WithLink("Metrics", "/metrics"),
	)

	// Set up HTTP handlers
	http.HandleFunc("/health", healthChecker.Handler())
	http.HandleFunc("/status", statusPage.Handler())

	// Start the server
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func generateRandomError() error {
	if rand.Intn(2) == 0 {
		return nil
	}

	return errors.New("dependency is not healthy")
}
