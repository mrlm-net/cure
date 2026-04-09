package main

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mrlm-net/cure/internal/agent/claude"
	_ "github.com/mrlm-net/cure/internal/agent/claudecode"
	_ "github.com/mrlm-net/cure/internal/agent/gemini"
	_ "github.com/mrlm-net/cure/internal/agent/openai"
	"github.com/mrlm-net/cure/internal/commands"
	"github.com/mrlm-net/cure/internal/commands/completion"
	ctxcmd "github.com/mrlm-net/cure/internal/commands/context"
	"github.com/mrlm-net/cure/internal/commands/doctor"
	"github.com/mrlm-net/cure/internal/commands/generate"
	guicmd "github.com/mrlm-net/cure/internal/commands/gui"
	initcmd "github.com/mrlm-net/cure/internal/commands/init"
	mcmcmd "github.com/mrlm-net/cure/internal/commands/mcp"
	"github.com/mrlm-net/cure/internal/commands/trace"
	agentstore "github.com/mrlm-net/cure/pkg/agent/store"
	"github.com/mrlm-net/cure/pkg/config"
	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
	"github.com/mrlm-net/cure/pkg/project"
	"github.com/mrlm-net/cure/pkg/template"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// Load config with precedence: defaults → global → local → env
	cfg := loadConfig()
	template.SetConfig(cfg) // wire custom template directories

	// Initialise the session store for the context command group.
	storeDir, err := ctxcmd.DefaultStoreDir()
	if err != nil {
		return fmt.Errorf("failed to determine session store directory: %w", err)
	}
	sessionStore, err := agentstore.NewJSONStore(storeDir)
	if err != nil {
		return fmt.Errorf("failed to initialise session store: %w", err)
	}

	router := terminal.New(terminal.WithConfig(cfg))
	router.Register(commands.NewVersionCommand())
	router.Register(terminal.NewHelpCommand(router))
	router.Register(trace.NewTraceCommand())
	router.Register(doctor.NewDoctorCommand())
	router.Register(generate.NewGenerateCommand())
	// Register context command BEFORE completion so it is included in completions.
	router.Register(ctxcmd.NewContextCommand(sessionStore))
	// Register init BEFORE completion so it is visible to completion introspection.
	router.Register(initcmd.NewInitCommand())
	// Register mcp BEFORE completion so it is visible to completion introspection.
	router.Register(mcmcmd.NewMCPCommand())
	// Register gui BEFORE completion so it is visible to completion introspection.
	router.Register(guicmd.NewGUICommand(cfg.Data(), pkgdoctor.BuiltinChecks(), sessionStore))
	router.Register(completion.NewCompletionCommand(router))
	return router.RunArgs(args)
}

func loadConfig() *config.Config {
	// Defaults (lowest precedence)
	defaults := config.ConfigObject{
		"timeout": 30,
		"format":  "json",
		"verbose": false,
		"redact":  true,
		// Claude provider defaults — overridable via config file or env.
		"agent.claude.model":      "claude-opus-4-6",
		"agent.claude.max_tokens": 8192,
	}

	// Global config (~/.cure.json)
	homeDir, _ := os.UserHomeDir()
	var globalCfg config.ConfigObject
	if homeDir != "" {
		globalPath := filepath.Join(homeDir, ".cure.json")
		if cfg, err := config.File(globalPath); err == nil {
			globalCfg = cfg
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v\n", globalPath, err)
		}
	}

	// Project config — auto-detect project from cwd and load its defaults.
	// Slots between global and local in the merge chain.
	var projectCfg config.ConfigObject
	if baseDir, err := project.DefaultBaseDir(); err == nil {
		store := project.NewStore(baseDir)
		detector := project.NewDetector(store)
		if cwd, err := os.Getwd(); err == nil {
			if p, err := detector.Detect(cwd); err == nil && p != nil {
				projectCfg = projectToConfigObject(p)
			}
		}
	}

	// Local config (./.cure.json)
	localPath := ".cure.json"
	localCfg, err := config.File(localPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v\n", localPath, err)
	}

	// Environment variables (highest precedence for file-based config)
	envCfg := config.Environment("CURE_", "_")

	// Merge with precedence: defaults < global < project < local < env
	// Note: CLI flags are applied per-command, not here
	return config.NewConfig(defaults, globalCfg, projectCfg, localCfg, envCfg)
}

// projectToConfigObject converts project defaults into a ConfigObject so they
// can participate in the standard config merge chain.
func projectToConfigObject(p *project.Project) config.ConfigObject {
	obj := config.ConfigObject{}
	if p.Defaults.Provider != "" {
		obj["agent.provider"] = p.Defaults.Provider
	}
	if p.Defaults.Model != "" {
		obj["agent.claude.model"] = p.Defaults.Model
	}
	if p.Defaults.MaxTurns > 0 {
		obj["agent.max_turns"] = p.Defaults.MaxTurns
	}
	if p.Defaults.MaxBudgetUSD > 0 {
		obj["agent.max_budget_usd"] = p.Defaults.MaxBudgetUSD
	}
	if p.Defaults.SystemPrompt != "" {
		obj["agent.system_prompt"] = p.Defaults.SystemPrompt
	}
	if p.Defaults.MaxAgents > 0 {
		obj["agent.max_agents"] = p.Defaults.MaxAgents
	}
	if p.Defaults.Tracker != nil {
		obj["tracker.type"] = p.Defaults.Tracker.Type
		if p.Defaults.Tracker.Owner != "" {
			obj["tracker.owner"] = p.Defaults.Tracker.Owner
		}
		if p.Defaults.Tracker.Repo != "" {
			obj["tracker.repo"] = p.Defaults.Tracker.Repo
		}
	}
	return obj
}
