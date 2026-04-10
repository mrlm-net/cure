// Package project defines the Project entity — the top-level abstraction
// grouping one or more repositories with shared configuration, workflow
// rules, notification channels, and AI config distribution settings.
//
// A project is stored as a JSON file at ~/.cure/projects/<name>/project.json.
// The project entity sits above individual repositories in the cure hierarchy
// and provides a 6th configuration layer between global and repo-level config.
package project

import "time"

// Project is the top-level entity grouping repositories, configuration,
// and operational settings for a multi-repo development effort.
type Project struct {
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	Repos         []Repo           `json:"repos"`
	Defaults      Defaults         `json:"defaults"`
	Devcontainer  *DevcontainerCfg `json:"devcontainer,omitempty"`
	Notifications NotificationsCfg `json:"notifications,omitempty"`
	Workflow      *WorkflowCfg     `json:"workflow,omitempty"`
	Registry      *RegistryCfg     `json:"registry,omitempty"`
	AIConfig      *AIConfigCfg     `json:"ai_config,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// Repo represents a repository within a project.
type Repo struct {
	Path          string `json:"path"`
	Remote        string `json:"remote,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
	LocalPath     string `json:"local_path,omitempty"` // cure-managed clone in workdir
}

// GlobalConfig holds cure-level user configuration stored at ~/.cure/config.json.
type GlobalConfig struct {
	WorkDir string `json:"workdir,omitempty"` // cure-managed working directory (default ~/.cure/workdir)
}

// Defaults contains default configuration values for the project.
type Defaults struct {
	Provider     string      `json:"provider,omitempty"`
	Model        string      `json:"model,omitempty"`
	SystemPrompt string      `json:"system_prompt,omitempty"`
	Tracker      *TrackerCfg `json:"tracker,omitempty"`
	MaxAgents    int         `json:"max_agents,omitempty"`
	MaxTurns     int         `json:"max_turns,omitempty"`
	MaxBudgetUSD float64     `json:"max_budget_usd,omitempty"`
}

// TrackerCfg configures the work item tracker for the project.
type TrackerCfg struct {
	Type          string `json:"type"`
	Owner         string `json:"owner,omitempty"`
	Repo          string `json:"repo,omitempty"`
	ProjectNumber int    `json:"project_number,omitempty"`
	ProjectID     string `json:"project_id,omitempty"`
}

// DevcontainerCfg configures the devcontainer for orchestrated agents.
type DevcontainerCfg struct {
	Image      string   `json:"image,omitempty"`
	Dockerfile string   `json:"dockerfile,omitempty"`
	Features   []string `json:"features,omitempty"`
}

// NotificationsCfg configures notification channels for the project.
type NotificationsCfg struct {
	Teams *TeamsCfg `json:"teams,omitempty"`
	Local *LocalCfg `json:"local,omitempty"`
}

// TeamsCfg configures Microsoft Teams notification integration.
type TeamsCfg struct {
	WebhookURL    string `json:"webhook_url,omitempty"`
	BotAppID      string `json:"bot_app_id,omitempty"`
	BotAppSecret  string `json:"bot_app_secret,omitempty"`
	Bidirectional bool   `json:"bidirectional,omitempty"`
}

// LocalCfg configures OS-level desktop notifications.
type LocalCfg struct {
	Enabled    bool     `json:"enabled,omitempty"`
	EventTypes []string `json:"event_types,omitempty"`
}

// WorkflowCfg defines development workflow rules enforced by cure.
type WorkflowCfg struct {
	BranchPattern     string   `json:"branch_pattern,omitempty"`
	CommitPattern     string   `json:"commit_pattern,omitempty"`
	RequireReview     bool     `json:"require_review,omitempty"`
	ProtectedBranches []string `json:"protected_branches,omitempty"`
}

// RegistryCfg configures AI config registry sources for the project.
type RegistryCfg struct {
	Sources []string `json:"sources,omitempty"`
}

// AIConfigCfg configures which AI config files are managed by cure.
type AIConfigCfg struct {
	ManagedFiles []string `json:"managed_files,omitempty"`
	SyncTriggers []string `json:"sync_triggers,omitempty"`
	Watch        bool     `json:"watch,omitempty"`
}
