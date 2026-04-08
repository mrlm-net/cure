package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateListEndpoint(t *testing.T) {
	handler := NewAPIRouter(testDeps())

	t.Run("GET returns 501 with error body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/generate/list", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var body ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Error == "" {
			t.Error("error field is empty, want non-empty message")
		}
	})

	t.Run("POST falls through to run handler and returns 501", func(t *testing.T) {
		// POST /api/generate/list is matched by POST /api/generate/{template}
		// with template="list" — this is expected mux behavior.
		req := httptest.NewRequest(http.MethodPost, "/api/generate/list", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotImplemented {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
		}
	})
}

func TestGenerateRunEndpoint(t *testing.T) {
	handler := NewAPIRouter(testDeps())

	t.Run("POST returns 501 with error body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/generate/sometemplate", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var body ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Error == "" {
			t.Error("error field is empty, want non-empty message")
		}
	})

	t.Run("GET returns 405 method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/generate/sometemplate", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("different template names are routed", func(t *testing.T) {
		templates := []string{"claude-md", "devcontainer", "editorconfig"}
		for _, tmpl := range templates {
			t.Run(tmpl, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/generate/"+tmpl, nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				if rec.Code != http.StatusNotImplemented {
					t.Errorf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
				}
			})
		}
	})
}
