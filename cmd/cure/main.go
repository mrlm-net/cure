package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrlm-net/cure/internal/commands"
	"github.com/mrlm-net/cure/internal/commands/completion"
	"github.com/mrlm-net/cure/internal/commands/generate"
	"github.com/mrlm-net/cure/internal/commands/trace"
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

	router := terminal.New(terminal.WithConfig(cfg))
	router.Register(&commands.VersionCommand{})
	router.Register(terminal.NewHelpCommand(router))
	router.Register(trace.NewTraceCommand())
	router.Register(generate.NewGenerateCommand())
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
