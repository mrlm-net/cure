package gui

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// testFS builds an in-memory filesystem that mimics the embed "dist/..." layout.
func testFS(files map[string]string) fs.FS {
	m := fstest.MapFS{}
	for name, content := range files {
		m["dist/"+name] = &fstest.MapFile{Data: []byte(content)}
	}
	return m
}

func TestNewSPAHandler(t *testing.T) {
	const port = 4321

	t.Run("serves known file", func(t *testing.T) {
		fsys := testFS(map[string]string{
			"index.html":    "<html>port=__CURE_PORT_PLACEHOLDER__</html>",
			"assets/app.js": "console.log('hello');",
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "console.log") {
			t.Errorf("body = %q, want JS content", body)
		}
	})

	t.Run("SPA fallback for unknown path", func(t *testing.T) {
		fsys := testFS(map[string]string{
			"index.html": "<html>SPA port=__CURE_PORT_PLACEHOLDER__</html>",
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/some/deep/route", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "SPA") {
			t.Errorf("body = %q, want SPA index content", body)
		}
	})

	t.Run("port injection in index.html", func(t *testing.T) {
		fsys := testFS(map[string]string{
			"index.html": `<script>window.__CURE_PORT__ = __CURE_PORT_PLACEHOLDER__;</script>`,
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		expected := "window.__CURE_PORT__ = 4321;"
		if !strings.Contains(body, expected) {
			t.Errorf("body = %q, want to contain %q", body, expected)
		}
		if strings.Contains(body, "__CURE_PORT_PLACEHOLDER__") {
			t.Error("placeholder was not replaced")
		}
	})

	t.Run("port injection on SPA fallback", func(t *testing.T) {
		fsys := testFS(map[string]string{
			"index.html": `<div>__CURE_PORT_PLACEHOLDER__</div>`,
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		body := rec.Body.String()
		if strings.Contains(body, "__CURE_PORT_PLACEHOLDER__") {
			t.Error("placeholder was not replaced on fallback")
		}
		if !strings.Contains(body, "4321") {
			t.Errorf("body = %q, want port injection", body)
		}
	})

	t.Run("content-type for index.html", func(t *testing.T) {
		fsys := testFS(map[string]string{
			"index.html": "<html></html>",
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		ct := rec.Header().Get("Content-Type")
		if ct != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
		}
	})

	t.Run("nil FS returns 503", func(t *testing.T) {
		// emptyFS has no "dist" subdirectory, so fs.Sub fails.
		handler := NewSPAHandler(emptyFS{}, port)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want 503", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "frontend not built") {
			t.Errorf("body = %q, want 'frontend not built' message", body)
		}
	})

	t.Run("FS without index.html returns 503 on fallback", func(t *testing.T) {
		// Has dist/ but no index.html.
		fsys := testFS(map[string]string{
			"other.txt": "not html",
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want 503", rec.Code)
		}
	})

	t.Run("root path serves index.html", func(t *testing.T) {
		fsys := testFS(map[string]string{
			"index.html": "<html>root</html>",
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "root") {
			t.Error("root path did not serve index.html")
		}
	})

	t.Run("directory path falls back to index.html", func(t *testing.T) {
		fsys := testFS(map[string]string{
			"index.html": "<html>fallback</html>",
			"assets/":    "",
		})

		handler := NewSPAHandler(fsys, port)
		req := httptest.NewRequest(http.MethodGet, "/assets", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Directory should fallback to index.html.
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})
}

func TestServeIndexWithPort(t *testing.T) {
	t.Run("replaces multiple occurrences", func(t *testing.T) {
		content := "port=__CURE_PORT_PLACEHOLDER__ and port=__CURE_PORT_PLACEHOLDER__"
		f := &memFile{data: []byte(content)}
		rec := httptest.NewRecorder()

		serveIndexWithPort(rec, f, "9999")

		body := rec.Body.String()
		if strings.Contains(body, "__CURE_PORT_PLACEHOLDER__") {
			t.Error("not all placeholders were replaced")
		}
		count := strings.Count(body, "9999")
		if count != 2 {
			t.Errorf("port count = %d, want 2", count)
		}
	})
}

// memFile is a minimal fs.File backed by a byte slice for testing.
type memFile struct {
	data []byte
	pos  int
}

func (f *memFile) Read(b []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *memFile) Stat() (fs.FileInfo, error) { return nil, nil }
func (f *memFile) Close() error               { return nil }
