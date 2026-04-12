package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ControlPlaneChecks returns health checks for the cure platform itself
// (not project-specific). These verify that cure's dependencies and
// infrastructure are available.
func ControlPlaneChecks() []CheckFunc {
	return []CheckFunc{
		CheckGit,
		CheckDocker,
		CheckCureCLI,
		CheckCureConfig,
		CheckCureProjects,
	}
}

// CheckGit verifies that the git binary is available.
func CheckGit() CheckResult {
	if _, err := exec.LookPath("git"); err != nil {
		return CheckResult{Name: "Git", Status: CheckFail, Message: "git binary not found on PATH"}
	}
	return CheckResult{Name: "Git", Status: CheckPass, Message: "git available"}
}

// CheckDocker verifies that Docker is available.
func CheckDocker() CheckResult {
	if _, err := exec.LookPath("docker"); err != nil {
		return CheckResult{Name: "Docker", Status: CheckWarn, Message: "docker not found (needed for orchestration)"}
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return CheckResult{Name: "Docker", Status: CheckWarn, Message: "docker found but not running"}
	}
	return CheckResult{Name: "Docker", Status: CheckPass, Message: "docker available and running"}
}

// CheckCureCLI verifies the cure binary is the current version.
func CheckCureCLI() CheckResult {
	if _, err := exec.LookPath("gh"); err != nil {
		return CheckResult{Name: "GitHub CLI", Status: CheckWarn, Message: "gh not found (needed for backlog)"}
	}
	return CheckResult{Name: "GitHub CLI", Status: CheckPass, Message: "gh available"}
}

// CheckCureConfig verifies that ~/.cure/ directory exists.
func CheckCureConfig() CheckResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{Name: "Cure Config", Status: CheckFail, Message: "cannot determine home directory"}
	}
	cureDir := filepath.Join(home, ".cure")
	if _, err := os.Stat(cureDir); os.IsNotExist(err) {
		return CheckResult{Name: "Cure Config", Status: CheckWarn, Message: "~/.cure/ not found — run cure project init"}
	}
	return CheckResult{Name: "Cure Config", Status: CheckPass, Message: "~/.cure/ exists"}
}

// CheckCureProjects verifies that at least one project is registered.
func CheckCureProjects() CheckResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{Name: "Projects", Status: CheckWarn, Message: "cannot check projects"}
	}
	projDir := filepath.Join(home, ".cure", "projects")
	entries, err := os.ReadDir(projDir)
	if err != nil || len(entries) == 0 {
		return CheckResult{Name: "Projects", Status: CheckWarn, Message: "no projects registered — run cure project init"}
	}
	return CheckResult{Name: "Projects", Status: CheckPass, Message: pluralize(len(entries), "project") + " registered"}
}

func pluralize(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
