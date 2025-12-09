package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alarmistdev/status/check"
)

type handlerTestCase struct {
	targets        []HealthTarget
	queryParams    string
	expectedStatus int
	expectedBody   []map[string]interface{}
}

type checkTestCase struct {
	targets        []HealthTarget
	expectedStatus []HealthTargetStatus
	expectedErrors []string
	setupContext   func(context.Context) (context.Context, context.CancelFunc)
}

func TestHealthChecker_Handler_NoDeps(t *testing.T) {
	t.Parallel()

	runHandlerTestCase(t, handlerTestCase{
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
	})
}

func TestHealthChecker_Handler_AllTargetsHealthy(t *testing.T) {
	t.Parallel()

	runHandlerTestCase(t, handlerTestCase{
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
	})
}

func TestHealthChecker_Handler_LowImportanceFailure(t *testing.T) {
	t.Parallel()

	runHandlerTestCase(t, handlerTestCase{
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
	})
}

func TestHealthChecker_Handler_HighImportanceFailure(t *testing.T) {
	t.Parallel()

	runHandlerTestCase(t, handlerTestCase{
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
	})
}

func TestHealthChecker_Check_NoTargets(t *testing.T) {
	t.Parallel()

	runCheckTestCase(t, checkTestCase{
		expectedStatus: []HealthTargetStatus{},
		expectedErrors: []string{},
	})
}

func TestHealthChecker_Check_AllTargetsHealthy(t *testing.T) {
	t.Parallel()

	runCheckTestCase(t, checkTestCase{
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
	})
}

func TestHealthChecker_Check_SomeTargetsUnhealthy(t *testing.T) {
	t.Parallel()

	runCheckTestCase(t, checkTestCase{
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
	})
}

func TestHealthChecker_Check_ContextCancellation(t *testing.T) {
	t.Parallel()

	runCheckTestCase(t, checkTestCase{
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
	})
}

func runHandlerTestCase(t *testing.T, tt handlerTestCase) {
	t.Helper()

	checker := newHealthChecker(tt.targets)
	recorder := executeHandlerRequest(t, checker, tt.queryParams)

	assertStatusCode(t, tt.expectedStatus, recorder.Code)
	assertHandlerBody(t, recorder.Body, tt.expectedBody)
}

func runCheckTestCase(t *testing.T, tt checkTestCase) {
	t.Helper()

	ctx, cancel := buildContext(tt.setupContext)
	defer cancel()

	checker := newHealthChecker(tt.targets)
	results := runChecks(t, checker, ctx)

	assertCheckResults(t, results, tt.expectedStatus, tt.expectedErrors)
}

func newHealthChecker(targets []HealthTarget) *HealthChecker {
	checker := NewHealthChecker()
	for _, target := range targets {
		opts := []TargetOption{}
		if target.Importance != TargetImportanceHigh {
			opts = append(opts, WithImportance(target.Importance))
		}
		if target.Icon != "" {
			opts = append(opts, WithIcon(target.Icon))
		}
		checker.WithTarget(target.Name, target.check, opts...)
	}

	return checker
}

func executeHandlerRequest(t *testing.T, checker *HealthChecker, queryParams string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/health"+queryParams, nil)
	w := httptest.NewRecorder()

	checker.Handler().ServeHTTP(w, req)

	return w
}

func assertStatusCode(t *testing.T, expected, actual int) {
	t.Helper()

	if actual != expected {
		t.Fatalf("expected status %d, got %d", expected, actual)
	}
}

func assertHandlerBody(t *testing.T, body *bytes.Buffer, expected []map[string]interface{}) {
	t.Helper()

	if expected == nil {
		return
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(response))
	}

	for i, result := range response {
		expectedResult := expected[i]

		assertTargets(t, i, expectedResult["target"].(map[string]interface{}), result["target"].(map[string]interface{}))
		assertResultFields(t, i, expectedResult, result)
	}
}

func assertTargets(t *testing.T, index int, expected, actual map[string]interface{}) {
	t.Helper()

	if actual["name"] != expected["name"] {
		t.Fatalf(
			"result[%d]: expected target name %s, got %s",
			index,
			expected["name"],
			actual["name"],
		)
	}
	if actual["importance"] != expected["importance"] {
		t.Fatalf(
			"result[%d]: expected target importance %s, got %s",
			index,
			expected["importance"],
			actual["importance"],
		)
	}
}

func assertResultFields(t *testing.T, index int, expected, actual map[string]interface{}) {
	t.Helper()

	if actual["status"] != expected["status"] {
		t.Fatalf("result[%d]: expected status %s, got %s", index, expected["status"], actual["status"])
	}

	if expected["error"] == nil {
		if actual["error"] != nil {
			t.Fatalf("result[%d]: expected no error, got %s", index, actual["error"])
		}

		return
	}

	if actual["error"] != expected["error"] {
		t.Fatalf("result[%d]: expected error %s, got %s", index, expected["error"], actual["error"])
	}
}

func buildContext(
	setup func(context.Context) (context.Context, context.CancelFunc),
) (context.Context, context.CancelFunc) {
	if setup == nil {
		return context.Background(), func() {}
	}

	return setup(context.Background())
}

func runChecks(t *testing.T, checker *HealthChecker, ctx context.Context) []HealthCheckResult {
	t.Helper()

	results, err := checker.Check(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return results
}

func assertCheckResults(
	t *testing.T,
	results []HealthCheckResult,
	expectedStatus []HealthTargetStatus,
	expectedErrors []string,
) {
	t.Helper()

	if len(results) != len(expectedStatus) {
		t.Fatalf("expected %d results, got %d", len(expectedStatus), len(results))
	}

	for i, result := range results {
		if result.Status != expectedStatus[i] {
			t.Fatalf("result[%d]: expected status %s, got %s", i, expectedStatus[i], result.Status)
		}

		if expectedErrors[i] == "" {
			if result.ErrorMessage != "" {
				t.Fatalf("result[%d]: expected no error, got %s", i, result.ErrorMessage)
			}

			continue
		}

		if expectedErrors[i] == "context canceled or deadline exceeded" {
			if result.ErrorMessage != "context canceled" && result.ErrorMessage != "context deadline exceeded" {
				t.Fatalf(
					"result[%d]: expected error context canceled or context deadline exceeded, got %s",
					i,
					result.ErrorMessage,
				)
			}

			continue
		}

		if result.ErrorMessage != expectedErrors[i] {
			t.Fatalf("result[%d]: expected error %s, got %s", i, expectedErrors[i], result.ErrorMessage)
		}
	}
}
