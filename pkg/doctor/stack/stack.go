// Package stack provides multi-stack detection and per-stack health checks.
// It extends pkg/doctor with technology-specific check suites.
package stack

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mrlm-net/cure/pkg/doctor"
)

// Stack represents a detected technology stack with its health checks.
type Stack struct {
	Name   string
	Detect func(dir string) bool
	Checks func() []doctor.CheckFunc
}

// AllStacks returns all known stack detectors.
func AllStacks() []Stack {
	return []Stack{
		goStack(),
		nodeStack(),
		pythonStack(),
		rustStack(),
		javaStack(),
	}
}

// DetectStacks scans a directory and returns all detected stacks.
func DetectStacks(dir string) []Stack {
	var found []Stack
	for _, s := range AllStacks() {
		if s.Detect(dir) {
			found = append(found, s)
		}
	}
	return found
}

// ChecksForDir returns all health checks applicable to the given directory.
func ChecksForDir(dir string) []doctor.CheckFunc {
	var checks []doctor.CheckFunc
	for _, s := range DetectStacks(dir) {
		checks = append(checks, s.Checks()...)
	}
	return checks
}

func fileExists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

func cmdExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// --- Go ---

func goStack() Stack {
	return Stack{
		Name:   "go",
		Detect: func(dir string) bool { return fileExists(dir, "go.mod") },
		Checks: func() []doctor.CheckFunc {
			return []doctor.CheckFunc{
				func() doctor.CheckResult {
					if !cmdExists("go") {
						return doctor.CheckResult{Name: "Go toolchain", Status: doctor.CheckFail, Message: "go binary not found"}
					}
					return doctor.CheckResult{Name: "Go toolchain", Status: doctor.CheckPass, Message: "go found"}
				},
				func() doctor.CheckResult {
					if !cmdExists("gofmt") {
						return doctor.CheckResult{Name: "gofmt", Status: doctor.CheckWarn, Message: "gofmt not found"}
					}
					return doctor.CheckResult{Name: "gofmt", Status: doctor.CheckPass, Message: "gofmt found"}
				},
			}
		},
	}
}

// --- Node ---

func nodeStack() Stack {
	return Stack{
		Name:   "node",
		Detect: func(dir string) bool { return fileExists(dir, "package.json") },
		Checks: func() []doctor.CheckFunc {
			return []doctor.CheckFunc{
				func() doctor.CheckResult {
					if !cmdExists("node") {
						return doctor.CheckResult{Name: "Node.js", Status: doctor.CheckFail, Message: "node binary not found"}
					}
					return doctor.CheckResult{Name: "Node.js", Status: doctor.CheckPass, Message: "node found"}
				},
				func() doctor.CheckResult {
					if !cmdExists("npm") {
						return doctor.CheckResult{Name: "npm", Status: doctor.CheckWarn, Message: "npm not found"}
					}
					return doctor.CheckResult{Name: "npm", Status: doctor.CheckPass, Message: "npm found"}
				},
			}
		},
	}
}

// --- Python ---

func pythonStack() Stack {
	return Stack{
		Name: "python",
		Detect: func(dir string) bool {
			return fileExists(dir, "requirements.txt") || fileExists(dir, "pyproject.toml") || fileExists(dir, "setup.py") || fileExists(dir, "Pipfile")
		},
		Checks: func() []doctor.CheckFunc {
			return []doctor.CheckFunc{
				func() doctor.CheckResult {
					if !cmdExists("python3") && !cmdExists("python") {
						return doctor.CheckResult{Name: "Python", Status: doctor.CheckFail, Message: "python not found"}
					}
					return doctor.CheckResult{Name: "Python", Status: doctor.CheckPass, Message: "python found"}
				},
				func() doctor.CheckResult {
					if !cmdExists("pip3") && !cmdExists("pip") {
						return doctor.CheckResult{Name: "pip", Status: doctor.CheckWarn, Message: "pip not found"}
					}
					return doctor.CheckResult{Name: "pip", Status: doctor.CheckPass, Message: "pip found"}
				},
			}
		},
	}
}

// --- Rust ---

func rustStack() Stack {
	return Stack{
		Name:   "rust",
		Detect: func(dir string) bool { return fileExists(dir, "Cargo.toml") },
		Checks: func() []doctor.CheckFunc {
			return []doctor.CheckFunc{
				func() doctor.CheckResult {
					if !cmdExists("rustc") {
						return doctor.CheckResult{Name: "Rust compiler", Status: doctor.CheckFail, Message: "rustc not found"}
					}
					return doctor.CheckResult{Name: "Rust compiler", Status: doctor.CheckPass, Message: "rustc found"}
				},
				func() doctor.CheckResult {
					if !cmdExists("cargo") {
						return doctor.CheckResult{Name: "Cargo", Status: doctor.CheckWarn, Message: "cargo not found"}
					}
					return doctor.CheckResult{Name: "Cargo", Status: doctor.CheckPass, Message: "cargo found"}
				},
			}
		},
	}
}

// --- Java ---

func javaStack() Stack {
	return Stack{
		Name: "java",
		Detect: func(dir string) bool {
			return fileExists(dir, "pom.xml") || fileExists(dir, "build.gradle") || fileExists(dir, "build.gradle.kts")
		},
		Checks: func() []doctor.CheckFunc {
			return []doctor.CheckFunc{
				func() doctor.CheckResult {
					if !cmdExists("java") {
						return doctor.CheckResult{Name: "Java", Status: doctor.CheckFail, Message: "java not found"}
					}
					return doctor.CheckResult{Name: "Java", Status: doctor.CheckPass, Message: "java found"}
				},
				func() doctor.CheckResult {
					if !cmdExists("mvn") && !cmdExists("gradle") {
						return doctor.CheckResult{Name: "Build tool", Status: doctor.CheckWarn, Message: "neither mvn nor gradle found"}
					}
					return doctor.CheckResult{Name: "Build tool", Status: doctor.CheckPass, Message: "build tool found"}
				},
			}
		},
	}
}
