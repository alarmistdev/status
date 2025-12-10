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
	latencycheck "github.com/alarmistdev/status/check/network/latency"
)

func main() {
	const (
		googleTLSPort          = 443
		googleLatencyThreshold = 500 * time.Millisecond
	)

	healthChecker := status.NewHealthChecker().
		WithTarget(
			"HTTP Google",
			httpcheck.Check(
				http.MethodGet,
				"https://google.com",
				http.StatusOK,
				check.Config{},
			),
			status.WithIcon("devicon-google-plain"),
			status.WithGroup("External Services"),
		).
		WithTarget(
			"Latency AWS",
			latencycheck.Check("google.com", googleTLSPort, googleLatencyThreshold),
			status.WithIcon("devicon-amazonwebservices-plain-wordmark"),
			status.WithGroup("External Services"),
		).
		WithTarget(
			"Database",
			check.CheckFunc(func(_ context.Context) error {
				// Implement your database health check here
				return generateRandomError()
			}),
			status.WithIcon("devicon-postgresql-plain"),
			status.WithGroup("Infrastructure"),
		).
		WithTarget(
			"Network",
			check.CheckFunc(func(_ context.Context) error {
				// Implement your network health check here
				return generateRandomError()
			}),
			status.WithImportance(status.TargetImportanceLow),
		)

	statusPage := status.NewPage(
		status.WithHealthChecker(healthChecker),
		status.WithLink("OpenAPI Documentation", "/swagger"),
		status.WithLink("Metrics", "/metrics"),
	)

	http.HandleFunc("/health", healthChecker.Handler())
	http.HandleFunc("/status", statusPage.Handler())

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func generateRandomError() error {
	const randomOutcomeCount = 2

	if rand.Intn(randomOutcomeCount) == 0 {
		return nil
	}

	return errors.New("dependency is not healthy")
}
