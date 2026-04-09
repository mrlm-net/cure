// Package managed generates and maintains AI config files that are
// managed by cure. Each managed file includes a marker comment with a
// SHA-256 hash for drift detection.
package managed

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/mrlm-net/cure/pkg/fs"
)

// MarkerPrefix is the prefix for the managed-file marker line.
const MarkerPrefix = "<!-- managed by cure: sha256:"

// MarkerSuffix closes the marker comment.
const MarkerSuffix = " -->"

// GenerateMarker creates a marker line for the given content.
func GenerateMarker(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%s%x%s", MarkerPrefix, hash, MarkerSuffix)
}

// WriteManaged writes content to path with a managed-file marker at the top.
// Uses atomic write to prevent partial writes.
func WriteManaged(path, content string) error {
	marker := GenerateMarker(content)
	full := marker + "\n" + content

	if err := fs.EnsureDir(pathDir(path), 0755); err != nil {
		return fmt.Errorf("managed write %s: %w", path, err)
	}
	return fs.AtomicWrite(path, []byte(full), 0644)
}

// ReadMarkerHash extracts the SHA-256 hash from a managed file's marker.
// Returns empty string if the file has no marker.
func ReadMarkerHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.SplitN(string(data), "\n", 2)
	if len(lines) == 0 {
		return "", nil
	}
	first := lines[0]
	if !strings.HasPrefix(first, MarkerPrefix) || !strings.HasSuffix(first, MarkerSuffix) {
		return "", nil // not managed
	}
	hash := strings.TrimPrefix(first, MarkerPrefix)
	hash = strings.TrimSuffix(hash, MarkerSuffix)
	return hash, nil
}

// ContentHash computes the SHA-256 hash of content (without marker).
func ContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// IsManaged reports whether a file has a cure managed-file marker.
func IsManaged(path string) bool {
	hash, err := ReadMarkerHash(path)
	return err == nil && hash != ""
}

// HasDrifted reports whether a managed file's content has changed since
// cure last wrote it. Returns false for unmanaged files.
func HasDrifted(path string) (bool, error) {
	markerHash, err := ReadMarkerHash(path)
	if err != nil {
		return false, err
	}
	if markerHash == "" {
		return false, nil // not managed
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Extract content after marker line
	full := string(data)
	idx := strings.Index(full, "\n")
	if idx < 0 {
		return true, nil // marker only, no content
	}
	content := full[idx+1:]

	currentHash := ContentHash(content)
	return currentHash != markerHash, nil
}

func pathDir(path string) string {
	dir := path
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' {
			return dir[:i]
		}
	}
	return "."
}
