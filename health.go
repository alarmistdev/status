// Package status provides functionality for health checking and status page generation
// in Go applications. It allows monitoring of various dependencies and services
// with different importance levels.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alarmistdev/status/check"
	"golang.org/x/sync/errgroup"
)

// HealthTarget represents a single health check target.
type HealthTarget struct {
	Name       string           `json:"name"`
	Importance TargetImportance `json:"importance"`
	Icon       string           `json:"icon,omitempty"`
	Group      string           `json:"group,omitempty"`
	check      check.Check
}

// TargetImportance defines the importance level of a health check target.
type TargetImportance string

const (
	// TargetImportanceLow indicates that the target is not critical for the application.
	TargetImportanceLow = TargetImportance("low")
	// TargetImportanceHigh indicates that the target is critical for the application.
	TargetImportanceHigh = TargetImportance("high")
)

// HealthChecker manages a collection of health check targets and provides
// functionality to check their health status.
type HealthChecker struct {
	targets []HealthTarget
}

// NewHealthChecker creates a new HealthChecker instance.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// TargetOption is a function that configures a HealthTarget.
type TargetOption func(*HealthTarget)

// WithImportance sets the importance level of a health check target.
func WithImportance(importance TargetImportance) TargetOption {
	return func(t *HealthTarget) {
		t.Importance = importance
	}
}

// WithIcon sets the icon CSS class name for a health check target.
func WithIcon(icon string) TargetOption {
	return func(t *HealthTarget) {
		t.Icon = icon
	}
}

// WithGroup sets the group name for a health check target.
// Targets with the same group name will be displayed together.
func WithGroup(group string) TargetOption {
	return func(t *HealthTarget) {
		t.Group = group
	}
}

// WithTarget adds a new health check target to the checker.
func (c *HealthChecker) WithTarget(name string, check check.Check, opts ...TargetOption) *HealthChecker {
	target := HealthTarget{
		Name:       name,
		Importance: TargetImportanceHigh,
		check:      check,
	}

	for _, opt := range opts {
		opt(&target)
	}

	c.targets = append(c.targets, target)

	return c
}

func (c *HealthChecker) Handler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, noDeps := r.URL.Query()["no_deps"]; noDeps {
			w.WriteHeader(http.StatusOK)

			return
		}

		ctx := r.Context()

		results, err := c.Check(ctx)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, err)

			return
		}

		status := http.StatusOK

		for _, result := range results {
			if result.Target.Importance == TargetImportanceHigh &&
				(result.Status != HealthTargetStatusOk || result.err != nil) {
				status = http.StatusInternalServerError

				break
			}
		}

		respondJSON(w, status, results)
	})
}

// HealthTargetStatus represents the status of a health check target.
type HealthTargetStatus string

const (
	// HealthTargetStatusOk indicates that the target is healthy.
	HealthTargetStatusOk = HealthTargetStatus("ok")
	// HealthTargetStatusFail indicates that the target is unhealthy.
	HealthTargetStatusFail = HealthTargetStatus("fail")
)

// HealthCheckResult contains the result of a health check for a target.
type HealthCheckResult struct {
	Target       HealthTarget       `json:"target"`
	Status       HealthTargetStatus `json:"status"`
	ErrorMessage string             `json:"error,omitempty"`
	Duration     time.Duration      `json:"duration,omitempty"`
	err          error
}

// Check performs health checks for all registered targets concurrently.
func (c *HealthChecker) Check(ctx context.Context) ([]HealthCheckResult, error) {
	results := make([]HealthCheckResult, len(c.targets))

	g, ctx := errgroup.WithContext(ctx)

	for i := range c.targets {
		index := i
		target := c.targets[i]
		g.Go(func() error {
			start := time.Now()
			err := target.check.Check(ctx)
			duration := time.Since(start)

			if err != nil {
				results[index] = HealthCheckResult{
					Target:       target,
					Status:       HealthTargetStatusFail,
					err:          err,
					ErrorMessage: err.Error(),
					Duration:     duration,
				}
			} else {
				results[index] = HealthCheckResult{
					Target:   target,
					Status:   HealthTargetStatusOk,
					Duration: duration,
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("waiting errgroup: %w", err)
	}

	return results, nil
}

// respondJSON responds JSON body with a given code. It sets
// Content-Type header.
func respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(&data); err != nil {
		log.Printf("encoding data to respond with json: %v", err)
	}
}
