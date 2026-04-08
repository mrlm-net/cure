package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/doctor"
)

// testDeps returns a Deps suitable for testing with deterministic checks.
func testDeps() Deps {
	return Deps{
		Config: config.ConfigObject{
			"version": "0.11.1",
			"server":  map[string]interface{}{"host": "localhost"},
		},
		Checks: []doctor.CheckFunc{
			func() doctor.CheckResult {
				return doctor.CheckResult{Name: "Go", Status: doctor.CheckPass, Message: "go found"}
			},
			func() doctor.CheckResult {
				return doctor.CheckResult{Name: "Lint", Status: doctor.CheckWarn, Message: "linter outdated"}
			},
			func() doctor.CheckResult {
				return doctor.CheckResult{Name: "CI", Status: doctor.CheckFail, Message: "CI config missing"}
			},
		},
		Port: 9090,
	}
}

func TestHealthEndpoint(t *testing.T) {
	handler := NewAPIRouter(testDeps())

	t.Run("GET returns 200 with status ok and port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("status = %v, want %q", body["status"], "ok")
		}
		// JSON numbers are float64
		if port, ok := body["port"].(float64); !ok || int(port) != 9090 {
			t.Errorf("port = %v, want 9090", body["port"])
		}
	})

	t.Run("POST returns 405 method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestConfigEndpoint(t *testing.T) {
	handler := NewAPIRouter(testDeps())

	t.Run("GET returns 200 with config data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body["version"] != "0.11.1" {
			t.Errorf("version = %v, want %q", body["version"], "0.11.1")
		}
		server, ok := body["server"].(map[string]any)
		if !ok {
			t.Fatalf("server key is not a map: %T", body["server"])
		}
		if server["host"] != "localhost" {
			t.Errorf("server.host = %v, want %q", server["host"], "localhost")
		}
	})

	t.Run("nil config returns empty object", func(t *testing.T) {
		deps := testDeps()
		deps.Config = nil
		h := NewAPIRouter(deps)

		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("body = %v, want empty object", body)
		}
	})
}

func TestDoctorEndpoint(t *testing.T) {
	handler := NewAPIRouter(testDeps())

	t.Run("GET returns 200 with check results array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/doctor", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var results []CheckResultResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("len(results) = %d, want 3", len(results))
		}

		tests := []struct {
			name    string
			status  string
			message string
		}{
			{"Go", "pass", "go found"},
			{"Lint", "warn", "linter outdated"},
			{"CI", "fail", "CI config missing"},
		}
		for i, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := results[i]
				if r.Name != tt.name {
					t.Errorf("name = %q, want %q", r.Name, tt.name)
				}
				if r.Status != tt.status {
					t.Errorf("status = %q, want %q", r.Status, tt.status)
				}
				if r.Message != tt.message {
					t.Errorf("message = %q, want %q", r.Message, tt.message)
				}
			})
		}
	})

	t.Run("empty checks returns empty array", func(t *testing.T) {
		deps := testDeps()
		deps.Checks = nil
		h := NewAPIRouter(deps)

		req := httptest.NewRequest(http.MethodGet, "/api/doctor", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var results []CheckResultResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("len(results) = %d, want 0", len(results))
		}
	})
}

func TestUnknownRoute(t *testing.T) {
	handler := NewAPIRouter(testDeps())

	req := httptest.NewRequest(http.MethodGet, "/api/foo", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCheckStatusString(t *testing.T) {
	tests := []struct {
		status doctor.CheckStatus
		want   string
	}{
		{doctor.CheckPass, "pass"},
		{doctor.CheckWarn, "warn"},
		{doctor.CheckFail, "fail"},
		{doctor.CheckStatus(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := checkStatusString(tt.status); got != tt.want {
				t.Errorf("checkStatusString(%d) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
