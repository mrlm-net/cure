package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileEntry represents a file or directory in a listing.
type FileEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// filesListHandler returns directory listings.
func filesListHandler(projectRoots []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			path = "."
		}

		// Resolve to absolute and validate
		absPath, err := filepath.Abs(path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid path")
			return
		}

		if !isWithinRoots(absPath, projectRoots) {
			writeError(w, http.StatusForbidden, "path outside project boundaries")
			return
		}

		entries, err := os.ReadDir(absPath)
		if err != nil {
			writeError(w, http.StatusNotFound, "directory not found")
			return
		}

		files := make([]FileEntry, 0, len(entries))
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				continue // skip hidden files
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, FileEntry{
				Name:  e.Name(),
				IsDir: e.IsDir(),
				Size:  info.Size(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}
}

// fileReadHandler reads a file's content.
func fileReadHandler(projectRoots []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		absPath, err := filepath.Abs(path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid path")
			return
		}

		if !isWithinRoots(absPath, projectRoots) {
			writeError(w, http.StatusForbidden, "path outside project boundaries")
			return
		}

		info, err := os.Stat(absPath)
		if err != nil {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		if info.IsDir() {
			writeError(w, http.StatusBadRequest, "path is a directory")
			return
		}
		if info.Size() > 5*1024*1024 {
			writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 5 MB)")
			return
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read file")
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	}
}

// fileWriteHandler writes content to a file.
func fileWriteHandler(projectRoots []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		absPath, err := filepath.Abs(path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid path")
			return
		}

		if !isWithinRoots(absPath, projectRoots) {
			writeError(w, http.StatusForbidden, "path outside project boundaries")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 5*1024*1024))
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read body")
			return
		}

		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create directory")
			return
		}

		if err := os.WriteFile(absPath, body, 0644); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write file")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// isWithinRoots checks that path is within at least one allowed root.
func isWithinRoots(path string, roots []string) bool {
	if len(roots) == 0 {
		return true // no restrictions if no roots configured
	}
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if path == absRoot || strings.HasPrefix(path, absRoot+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
