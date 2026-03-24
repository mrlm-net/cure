package main

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mrlm-net/cure/internal/agent/claude"
	"github.com/mrlm-net/cure/internal/commands"
	"github.com/mrlm-net/cure/internal/commands/completion"
	ctxcmd "github.com/mrlm-net/cure/internal/commands/context"
	"github.com/mrlm-net/cure/internal/commands/generate"
	"github.com/mrlm-net/cure/internal/commands/trace"
	agentstore "github.com/mrlm-net/cure/pkg/agent/store"
	"github.com/mrlm-net/cure/pkg/config"
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
	router.Register(generate.NewGenerateCommand())
	// Register context command BEFORE completion so it is included in completions.
	router.Register(ctxcmd.NewContextCommand(sessionStore))
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

	// Local config (./.cure.json)
	localPath := ".cure.json"
	localCfg, err := config.File(localPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v\n", localPath, err)
	}

	// Environment variables (highest precedence for file-based config)
	envCfg := config.Environment("CURE_", "_")

	// Merge with precedence: defaults < global < local < env
	// Note: CLI flags are applied per-command, not here
	return config.NewConfig(defaults, globalCfg, localCfg, envCfg)
}
