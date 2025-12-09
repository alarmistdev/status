# status

[![Run Tests](https://github.com/alarmistdev/status/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/alarmistdev/status/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/alarmistdev/status)](https://goreportcard.com/report/github.com/alarmistdev/status)
[![GoDoc](https://godoc.org/github.com/alarmistdev/status?status.svg)](https://godoc.org/github.com/alarmistdev/status)

A Go package for health checking and status page generation. It provides a simple way to monitor the health of various dependencies and services in your application, with a status page dashboard.

## Installation

```bash
go get github.com/alarmistdev/status
```

## Usage

```go
package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net/http"

	"github.com/alarmistdev/status"
)

func main() {
	// Create a health checker
	healthChecker := status.NewHealthChecker().
		WithTarget("database", status.TargetImportanceHigh, func(_ context.Context) error {
			// Implement your database health check here
			return generateRandomError()
		}).
		WithTarget("network", status.TargetImportanceLow, func(_ context.Context) error {
			// Implement your network health check here
			return generateRandomError()
		})

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
```
