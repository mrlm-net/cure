package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/mrlm-net/cure/pkg/fs"
)

// CheckREADME verifies that a README file exists in the current directory.
func CheckREADME() CheckResult {
	for _, name := range []string{"README.md", "README"} {
		ok, err := fs.Exists(name)
		if err == nil && ok {
			return CheckResult{
				Name:    "README",
				Status:  CheckPass,
				Message: fmt.Sprintf("%s found", name),
			}
		}
	}
	return CheckResult{
		Name:    "README",
		Status:  CheckFail,
		Message: "README not found (expected README.md or README)",
	}
}

// CheckTests verifies that test files or a tests directory exist.
func CheckTests() CheckResult {
	// Check for a tests/ directory first.
	ok, err := fs.Exists("tests")
	if err == nil && ok {
		return CheckResult{
			Name:    "Tests",
			Status:  CheckPass,
			Message: "Tests found (tests/)",
		}
	}

	// Walk the current directory (non-recursively) for *_test.go files.
	found, detail := findTestFiles(".")
	if found {
		return CheckResult{
			Name:    "Tests",
			Status:  CheckPass,
			Message: fmt.Sprintf("Tests found (%s)", detail),
		}
	}

	return CheckResult{
		Name:    "Tests",
		Status:  CheckFail,
		Message: "No tests found (expected *_test.go files or tests/ directory)",
	}
}

// CheckCI verifies that a CI configuration exists.
func CheckCI() CheckResult {
	candidates := []struct {
		path  string
		label string
	}{
		{".github/workflows", ".github/workflows/"},
		{".gitlab-ci.yml", ".gitlab-ci.yml"},
		{".circleci", ".circleci/"},
	}
	for _, c := range candidates {
		ok, err := fs.Exists(c.path)
		if err == nil && ok {
			return CheckResult{
				Name:    "CI Config",
				Status:  CheckPass,
				Message: fmt.Sprintf("CI configuration found (%s)", c.label),
			}
		}
	}
	return CheckResult{
		Name:    "CI Config",
		Status:  CheckFail,
		Message: "No CI configuration found (expected .github/workflows/, .gitlab-ci.yml, or .circleci/)",
	}
}

// CheckGitignore verifies that a .gitignore file exists.
// Missing .gitignore is a warning, not a failure.
func CheckGitignore() CheckResult {
	ok, err := fs.Exists(".gitignore")
	if err == nil && ok {
		return CheckResult{
			Name:    ".gitignore",
			Status:  CheckPass,
			Message: ".gitignore found",
		}
	}
	return CheckResult{
		Name:    ".gitignore",
		Status:  CheckWarn,
		Message: ".gitignore missing (optional but recommended)",
	}
}

// CheckClaudeMD verifies that a CLAUDE.md file exists.
func CheckClaudeMD() CheckResult {
	ok, err := fs.Exists("CLAUDE.md")
	if err == nil && ok {
		return CheckResult{
			Name:    "CLAUDE.md",
			Status:  CheckPass,
			Message: "CLAUDE.md found",
		}
	}
	return CheckResult{
		Name:    "CLAUDE.md",
		Status:  CheckFail,
		Message: "CLAUDE.md not found",
	}
}

// CheckBuildTool verifies that a recognized build tool configuration exists.
func CheckBuildTool() CheckResult {
	candidates := []struct {
		path  string
		label string
	}{
		{"Makefile", "Makefile"},
		{"package.json", "package.json"},
		{"Cargo.toml", "Cargo.toml"},
		{"build.gradle", "build.gradle"},
	}
	for _, c := range candidates {
		ok, err := fs.Exists(c.path)
		if err == nil && ok {
			return CheckResult{
				Name:    "Build Tool",
				Status:  CheckPass,
				Message: fmt.Sprintf("Build tool found (%s)", c.label),
			}
		}
	}
	return CheckResult{
		Name:    "Build Tool",
		Status:  CheckFail,
		Message: "No build tool found (expected Makefile, package.json, Cargo.toml, or build.gradle)",
	}
}

// CheckDependencyManifest verifies that a dependency manifest exists.
func CheckDependencyManifest() CheckResult {
	candidates := []struct {
		path  string
		label string
	}{
		{"go.mod", "go.mod"},
		{"package.json", "package.json"},
		{"requirements.txt", "requirements.txt"},
		{"Cargo.toml", "Cargo.toml"},
	}
	for _, c := range candidates {
		ok, err := fs.Exists(c.path)
		if err == nil && ok {
			return CheckResult{
				Name:    "Dependency Manifest",
				Status:  CheckPass,
				Message: fmt.Sprintf("Dependency manifest found (%s)", c.label),
			}
		}
	}
	return CheckResult{
		Name:    "Dependency Manifest",
		Status:  CheckFail,
		Message: "No dependency manifest found (expected go.mod, package.json, requirements.txt, or Cargo.toml)",
	}
}

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
