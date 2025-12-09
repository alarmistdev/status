package status

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alarmistdev/status/check"
)

func TestHealthChecker_Handler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		targets        []HealthTarget
		queryParams    string
		expectedStatus int
		expectedBody   []map[string]interface{}
	}{
		{
			name: "no_deps query param returns 200",
			targets: []HealthTarget{
				{
					Name:       "failing_target",
					Importance: TargetImportanceHigh,
					check: check.CheckFunc(func(ctx context.Context) error {
						return errors.New("this error should not be seen due to no_deps")
					}),
				},
			},
			queryParams:    "?no_deps=true",
			expectedStatus: http.StatusOK,
			expectedBody:   nil,
		},
		{
			name: "all targets healthy returns 200",
			targets: []HealthTarget{
				{
					Name:       "test1",
					Importance: TargetImportanceLow,
					check: check.CheckFunc(func(ctx context.Context) error {
						return nil
					}),
				},
				{
					Name:       "test2",
					Importance: TargetImportanceHigh,
					check: check.CheckFunc(func(ctx context.Context) error {
						return nil
					}),
				},
			},
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody: []map[string]interface{}{
				{
					"target": map[string]interface{}{
						"name":       "test1",
						"importance": "low",
					},
					"status":   "ok",
					"duration": float64(0),
				},
				{
					"target": map[string]interface{}{
						"name":       "test2",
						"importance": "high",
					},
					"status":   "ok",
					"duration": float64(0),
				},
			},
		},
		{
			name: "low importance target failure returns 200",
			targets: []HealthTarget{
				{
					Name:       "test1",
					Importance: TargetImportanceLow,
					check: check.CheckFunc(func(ctx context.Context) error {
						return errors.New("low importance error")
					}),
				},
				{
					Name:       "test2",
					Importance: TargetImportanceHigh,
					check: check.CheckFunc(func(ctx context.Context) error {
						return nil
					}),
				},
			},
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody: []map[string]interface{}{
				{
					"target": map[string]interface{}{
						"name":       "test1",
						"importance": "low",
					},
					"status":   "fail",
					"error":    "low importance error",
					"duration": float64(0),
				},
				{
					"target": map[string]interface{}{
						"name":       "test2",
						"importance": "high",
					},
					"status":   "ok",
					"duration": float64(0),
				},
			},
		},
		{
			name: "high importance target failure returns 500",
			targets: []HealthTarget{
				{
					Name:       "test1",
					Importance: TargetImportanceLow,
					check: check.CheckFunc(func(ctx context.Context) error {
						return errors.New("low importance error")
					}),
				},
				{
					Name:       "test2",
					Importance: TargetImportanceHigh,
					check: check.CheckFunc(func(ctx context.Context) error {
						return errors.New("high importance error")
					}),
				},
			},
			queryParams:    "",
			expectedStatus: http.StatusInternalServerError,
			expectedBody: []map[string]interface{}{
				{
					"target": map[string]interface{}{
						"name":       "test1",
						"importance": "low",
					},
					"status":   "fail",
					"error":    "low importance error",
					"duration": float64(0),
				},
				{
					"target": map[string]interface{}{
						"name":       "test2",
						"importance": "high",
					},
					"status":   "fail",
					"error":    "high importance error",
					"duration": float64(0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewHealthChecker()
			for _, target := range tt.targets {
				checker.WithTarget(target.Name, target.Importance, target.check)
			}

			req := httptest.NewRequest(http.MethodGet, "/health"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			checker.Handler().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody == nil {
				return
			}

			var response []map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(response) != len(tt.expectedBody) {
				t.Errorf("expected %d results, got %d", len(tt.expectedBody), len(response))
				return
			}

			// Compare each result
			for i, result := range response {
				expectedResult := tt.expectedBody[i]

				// Compare target
				target := result["target"].(map[string]interface{})
				expectedTarget := expectedResult["target"].(map[string]interface{})
				if target["name"] != expectedTarget["name"] {
					t.Errorf("result[%d]: expected target name %s, got %s", i, expectedTarget["name"], target["name"])
				}
				if target["importance"] != expectedTarget["importance"] {
					t.Errorf("result[%d]: expected target importance %s, got %s", i, expectedTarget["importance"], target["importance"])
				}

				// Compare status
				if result["status"] != expectedResult["status"] {
					t.Errorf("result[%d]: expected status %s, got %s", i, expectedResult["status"], result["status"])
				}

				// Compare error if present
				if expectedResult["error"] != nil {
					if result["error"] != expectedResult["error"] {
						t.Errorf("result[%d]: expected error %s, got %s", i, expectedResult["error"], result["error"])
					}
				}
			}
		})
	}
}

func TestHealthChecker_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		targets        []HealthTarget
		expectedStatus []HealthTargetStatus
		expectedErrors []string
		setupContext   func(context.Context) (context.Context, context.CancelFunc)
	}{
		{
			name:           "no targets",
			targets:        []HealthTarget{},
			expectedStatus: []HealthTargetStatus{},
			expectedErrors: []string{},
		},
		{
			name: "all targets healthy",
			targets: []HealthTarget{
				{
					Name:       "test1",
					Importance: TargetImportanceLow,
					check: check.CheckFunc(func(ctx context.Context) error {
						return nil
					}),
				},
				{
					Name:       "test2",
					Importance: TargetImportanceHigh,
					check: check.CheckFunc(func(ctx context.Context) error {
						return nil
					}),
				},
			},
			expectedStatus: []HealthTargetStatus{
				HealthTargetStatusOk,
				HealthTargetStatusOk,
			},
			expectedErrors: []string{"", ""},
		},
		{
			name: "some targets unhealthy",
			targets: []HealthTarget{
				{
					Name:       "test1",
					Importance: TargetImportanceLow,
					check: check.CheckFunc(func(ctx context.Context) error {
						return errors.New("test1 error")
					}),
				},
				{
					Name:       "test2",
					Importance: TargetImportanceHigh,
					check: check.CheckFunc(func(ctx context.Context) error {
						return nil
					}),
				},
			},
			expectedStatus: []HealthTargetStatus{
				HealthTargetStatusFail,
				HealthTargetStatusOk,
			},
			expectedErrors: []string{"test1 error", ""},
		},
		{
			name: "context cancellation",
			targets: []HealthTarget{
				{
					Name:       "test1",
					Importance: TargetImportanceLow,
					check: check.CheckFunc(func(ctx context.Context) error {
						<-ctx.Done()
						return ctx.Err()
					}),
				},
			},
			expectedStatus: []HealthTargetStatus{
				HealthTargetStatusFail,
			},
			expectedErrors: []string{"context canceled or deadline exceeded"},
			setupContext: func(ctx context.Context) (context.Context, context.CancelFunc) {
				return context.WithTimeout(ctx, 100*time.Millisecond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewHealthChecker()
			for _, target := range tt.targets {
				checker.WithTarget(target.Name, target.Importance, target.check)
			}

			ctx := context.Background()
			if tt.setupContext != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.setupContext(ctx)
				defer cancel()
			}

			results, err := checker.Check(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != len(tt.expectedStatus) {
				t.Errorf("expected %d results, got %d", len(tt.expectedStatus), len(results))
				return
			}

			for i, result := range results {
				if result.Status != tt.expectedStatus[i] {
					t.Errorf("result[%d]: expected status %s, got %s", i, tt.expectedStatus[i], result.Status)
				}

				if tt.expectedErrors[i] != "" {
					if tt.expectedErrors[i] == "context canceled or deadline exceeded" {
						if result.ErrorMessage != "context canceled" && result.ErrorMessage != "context deadline exceeded" {
							t.Errorf("result[%d]: expected error context canceled or context deadline exceeded, got %s", i, result.ErrorMessage)
						}
					} else if result.ErrorMessage != tt.expectedErrors[i] {
						t.Errorf("result[%d]: expected error %s, got %s", i, tt.expectedErrors[i], result.ErrorMessage)
					}
				} else if result.ErrorMessage != "" {
					t.Errorf("result[%d]: expected no error, got %s", i, result.ErrorMessage)
				}
			}
		})
	}
}
