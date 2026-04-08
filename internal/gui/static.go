package gui

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// NewSPAHandler returns an http.Handler that serves static files from the
// embedded filesystem. For paths that don't match a file, it serves index.html
// (SPA fallback). The __CURE_PORT_PLACEHOLDER__ token in index.html is replaced
// with the actual port number at serve time.
func NewSPAHandler(fsys fs.FS, port int) http.Handler {
	// The embed directive produces "dist/..." paths. Sub into "dist" to get
	// the root of the SPA files.
	root, err := fs.Sub(fsys, "dist")
	if err != nil {
		// distFS doesn't have dist/ — return 503 handler.
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "frontend not built — run make gui-frontend", http.StatusServiceUnavailable)
		})
	}

	portStr := fmt.Sprintf("%d", port)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the request path.
		p := path.Clean("/" + r.URL.Path)[1:] // strip leading /
		if p == "" {
			p = "index.html"
		}

		// Try to open the exact file.
		f, err := root.Open(p)
		if err == nil {
			defer f.Close()
			stat, statErr := f.Stat()
			if statErr == nil && !stat.IsDir() {
				// For index.html, inject the port placeholder.
				if p == "index.html" {
					serveIndexWithPort(w, f, portStr)
					return
				}
				http.ServeFileFS(w, r, root, p)
				return
			}
		}

		// SPA fallback: serve index.html for unknown paths.
		idx, err := root.Open("index.html")
		if err != nil {
			http.Error(w, "frontend not built — run make gui-frontend", http.StatusServiceUnavailable)
			return
		}
		defer idx.Close()
		serveIndexWithPort(w, idx, portStr)
	})
}

// serveIndexWithPort reads the file, replaces the port placeholder, and writes
// the result as text/html.
func serveIndexWithPort(w http.ResponseWriter, f fs.File, port string) {
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	content := strings.ReplaceAll(string(data), "__CURE_PORT_PLACEHOLDER__", port)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(content))
}
