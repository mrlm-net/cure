package doctor

import (
	"os"
	"strings"
)

// findTestFiles scans dir (non-recursively) for files matching the *_test.go
// pattern. It returns (true, exampleFilename) on the first match found, or
// (false, "") when no test files exist in dir.
//
// The scan is intentionally shallow — only the top-level entries of dir are
// examined. Recursive tree walks are not performed because the check is meant
// as a quick existence signal, not a deep inventory.
func findTestFiles(dir string) (found bool, example string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, "_test.go") {
			return true, name
		}
	}
	return false, ""
}
