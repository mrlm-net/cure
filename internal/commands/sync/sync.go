// Package synccmd implements the "cure sync" command for syncing managed
// AI config files from registry + project to the current repo.
package synccmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrlm-net/cure/internal/config/managed"
	"github.com/mrlm-net/cure/pkg/registry"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// managedFileMap maps config names to their target paths relative to repo root.
var managedFileMap = map[string]string{
	"claude-md":              "CLAUDE.md",
	"agents-md":              "AGENTS.md",
	"claude-settings":        ".claude/settings.json",
	"mcp-json":               ".mcp.json",
	"cursor-rules":           ".cursor/rules/project.mdc",
	"copilot-instructions":   ".github/copilot-instructions.md",
}

// SyncCommand syncs managed config files from registry to repo.
type SyncCommand struct {
	checkOnly bool
	force     bool
	reg       *registry.Registry
}

// NewSyncCommand creates a new sync command.
func NewSyncCommand(reg *registry.Registry) terminal.Command {
	return &SyncCommand{reg: reg}
}

func (c *SyncCommand) Name() string        { return "sync" }
func (c *SyncCommand) Description() string { return "Sync managed AI config files from registry" }
func (c *SyncCommand) Usage() string {
	return `Usage: cure sync [flags]

Sync managed AI config files from the registry to the current repo.
Files are rendered from registry templates with project context.

Flags:
  --check    Report drift without modifying files
  --force    Overwrite without prompting`
}

func (c *SyncCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.BoolVar(&c.checkOnly, "check", false, "report drift only")
	fs.BoolVar(&c.force, "force", false, "overwrite without prompting")
	return fs
}

func (c *SyncCommand) Run(_ context.Context, tc *terminal.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if c.checkOnly {
		return c.runCheck(cwd, tc)
	}
	return c.runSync(cwd, tc)
}

func (c *SyncCommand) runCheck(cwd string, tc *terminal.Context) error {
	drifted := 0
	for name, relPath := range managedFileMap {
		path := filepath.Join(cwd, relPath)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		if !managed.IsManaged(path) {
			continue
		}

		hasDrifted, err := managed.HasDrifted(path)
		if err != nil {
			fmt.Fprintf(tc.Stdout, "  ? %s — error: %v\n", name, err)
			continue
		}
		if hasDrifted {
			fmt.Fprintf(tc.Stdout, "  ~ %s — drifted\n", name)
			drifted++
		} else {
			fmt.Fprintf(tc.Stdout, "  = %s — in sync\n", name)
		}
	}

	if drifted > 0 {
		fmt.Fprintf(tc.Stdout, "\n%d file(s) have drifted. Run 'cure sync' to update.\n", drifted)
	} else {
		fmt.Fprintln(tc.Stdout, "\nAll managed files are in sync.")
	}
	return nil
}

func (c *SyncCommand) runSync(cwd string, tc *terminal.Context) error {
	synced := 0
	for name, relPath := range managedFileMap {
		// Try to resolve from registry
		tmplPath := c.reg.Resolve(registry.ArtifactTemplate, name+".tmpl")
		if tmplPath == "" {
			continue // no template for this file in any source
		}

		content, err := os.ReadFile(tmplPath)
		if err != nil {
			fmt.Fprintf(tc.Stdout, "  ! %s — read error: %v\n", name, err)
			continue
		}

		targetPath := filepath.Join(cwd, relPath)

		// Check if file exists and isn't force mode
		if !c.force {
			if _, err := os.Stat(targetPath); err == nil {
				if managed.IsManaged(targetPath) {
					hasDrifted, _ := managed.HasDrifted(targetPath)
					if !hasDrifted {
						fmt.Fprintf(tc.Stdout, "  = %s — up to date\n", name)
						continue
					}
				}
			}
		}

		if err := managed.WriteManaged(targetPath, string(content)); err != nil {
			fmt.Fprintf(tc.Stdout, "  ! %s — write error: %v\n", name, err)
			continue
		}
		fmt.Fprintf(tc.Stdout, "  + %s → %s\n", name, relPath)
		synced++
	}

	fmt.Fprintf(tc.Stdout, "\n%d file(s) synced.\n", synced)
	return nil
}
