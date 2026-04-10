package project

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultWorkDir returns the default cure working directory (~/.cure/workdir).
func DefaultWorkDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("project: resolve home: %w", err)
	}
	return filepath.Join(home, ".cure", "workdir"), nil
}

// GlobalConfigPath returns ~/.cure/config.json.
func GlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("project: resolve home: %w", err)
	}
	return filepath.Join(home, ".cure", "config.json"), nil
}

// LoadGlobalConfig loads the global cure config from ~/.cure/config.json.
// Returns defaults if the file doesn't exist.
func LoadGlobalConfig() (*GlobalConfig, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return &GlobalConfig{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			workdir, _ := DefaultWorkDir()
			return &GlobalConfig{WorkDir: workdir}, nil
		}
		return nil, fmt.Errorf("project: read global config: %w", err)
	}

	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("project: parse global config: %w", err)
	}

	if cfg.WorkDir == "" {
		cfg.WorkDir, _ = DefaultWorkDir()
	}
	return &cfg, nil
}

// SaveGlobalConfig writes the global cure config to ~/.cure/config.json.
func SaveGlobalConfig(cfg *GlobalConfig) error {
	path, err := GlobalConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("project: create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("project: marshal global config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// CloneRepo clones a repo's remote into the cure workdir and sets LocalPath.
// If LocalPath is already set and exists, this is a no-op.
func CloneRepo(repo *Repo, workDir, projectName string) error {
	if repo.Remote == "" {
		return fmt.Errorf("project: repo has no remote URL")
	}

	if repo.LocalPath != "" {
		if _, err := os.Stat(repo.LocalPath); err == nil {
			return nil // already cloned
		}
	}

	// Determine clone destination
	repoName := filepath.Base(repo.Remote)
	// Strip .git suffix
	if len(repoName) > 4 && repoName[len(repoName)-4:] == ".git" {
		repoName = repoName[:len(repoName)-4]
	}

	dest := filepath.Join(workDir, projectName, repoName)

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("project: create workdir: %w", err)
	}

	// Check if already exists
	if _, err := os.Stat(dest); err == nil {
		repo.LocalPath = dest
		return nil
	}

	cmd := exec.Command("git", "clone", repo.Remote, dest)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("project: git clone failed: %s", string(out))
	}

	repo.LocalPath = dest
	return nil
}

// EffectivePath returns the path to use for a repo — LocalPath if set, otherwise Path.
func (r *Repo) EffectivePath() string {
	if r.LocalPath != "" {
		return r.LocalPath
	}
	return r.Path
}
