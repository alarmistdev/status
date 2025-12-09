package status

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alarmistdev/status/check"
)

func TestPage_Handler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		page           *Page
		expectedStatus int
		expectedBody   []string // List of strings that must be present in the response
	}{
		{
			name: "basic page without health checker",
			page: NewPage(
				WithTitle("Test Status"),
				WithLink("Home", "/"),
			),
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<title>Test Status</title>",
				"<h1>Test Status</h1>",
				`<a href="/">Home</a>`,
			},
		},
		{
			name: "page with successful health check",
			page: NewPage(
				WithTitle("Test Status"),
				WithHealthChecker(NewHealthChecker().
					WithTarget("Database", TargetImportanceHigh, check.CheckFunc(func(ctx context.Context) error {
						return nil
					}))),
			),
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				`<div class="status-item ok">`,
				"<h3>Database</h3>",
				"Status: <strong>ok</strong>",
			},
		},
		{
			name: "page with failed high importance health check",
			page: NewPage(
				WithTitle("Test Status"),
				WithHealthChecker(NewHealthChecker().
					WithTarget("Database", TargetImportanceHigh, check.CheckFunc(func(ctx context.Context) error {
						return errors.New("connection refused")
					}))),
			),
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				`<div class="status-item fail">`,
				"<h3>Database</h3>",
				"Status: <strong>fail</strong>",
				"Error: connection refused",
			},
		},
		{
			name: "page with failed low importance health check",
			page: NewPage(
				WithTitle("Test Status"),
				WithHealthChecker(NewHealthChecker().
					WithTarget("Cache", TargetImportanceLow, check.CheckFunc(func(ctx context.Context) error {
						return errors.New("cache miss")
					}))),
			),
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				`<div class="status-item warning">`,
				"<h3>Cache</h3>",
				"Status: <strong>fail</strong>",
				"Warning: cache miss",
			},
		},
		{
			name: "page with multiple health checks",
			page: NewPage(
				WithTitle("Test Status"),
				WithHealthChecker(NewHealthChecker().
					WithTarget("Database", TargetImportanceHigh, check.CheckFunc(func(ctx context.Context) error {
						return nil
					})).
					WithTarget("Cache", TargetImportanceLow, check.CheckFunc(func(ctx context.Context) error {
						return errors.New("cache miss")
					}))),
			),
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<title>Test Status</title>",
				"<h1>Test Status</h1>",
				`<div class="status-item ok">`,
				"<h3>Database</h3>",
				"Status: <strong>ok</strong>",
				`<div class="status-item warning">`,
				"<h3>Cache</h3>",
				"Status: <strong>fail</strong>",
				"Warning: cache miss",
			},
		},
		{
			name: "page with version info",
			page: NewPage(
				WithTitle("Test Status"),
				WithVersion(true),
			),
			expectedStatus: http.StatusOK,
			expectedBody:   nil, // Don't require build-info div if version is unknown
		},
		{
			name: "template execution error",
			page: NewPage(
				WithTitle("Test Status"),
				WithTemplate(template.Must(template.New("error").Parse("{{.NonExistentField}}"))),
			),
			expectedStatus: http.StatusInternalServerError,
			expectedBody: []string{
				"Error executing template",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			handler := tt.page.Handler()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody == nil {
				return
			}

			body := w.Body.String()
			for _, expected := range tt.expectedBody {
				if !strings.Contains(body, expected) {
					t.Errorf("expected response body to contain %q, got:\n%s", expected, body)
				}
			}
		})
	}
}
