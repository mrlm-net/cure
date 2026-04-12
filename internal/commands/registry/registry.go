// Package regcmd implements the "cure registry" command group for managing
// AI config source registries.
package regcmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/tabwriter"

	"github.com/mrlm-net/cure/pkg/registry"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewRegistryCommand returns the "registry" command group.
func NewRegistryCommand(reg *registry.Registry, baseDir string) terminal.Command {
	router := terminal.New(
		terminal.WithName("registry"),
		terminal.WithDescription("Manage AI config source registries"),
	)
	router.Register(&addCommand{reg: reg, baseDir: baseDir})
	router.Register(&removeCommand{reg: reg, baseDir: baseDir})
	router.Register(&updateCommand{reg: reg, baseDir: baseDir})
	router.Register(&regListCommand{reg: reg})
	return router
}

// --- add ---

type addCommand struct {
	reg     *registry.Registry
	baseDir string
}

func (c *addCommand) Name() string        { return "add" }
func (c *addCommand) Description() string { return "Add a registry source (git clone)" }
func (c *addCommand) Usage() string {
	return "Usage: cure registry add <name> <git-url>"
}
func (c *addCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("add", flag.ContinueOnError)
}

func (c *addCommand) Run(_ context.Context, tc *terminal.Context) error {
	if len(tc.Args) < 2 {
		return fmt.Errorf("usage: cure registry add <name> <git-url>")
	}
	name, url := tc.Args[0], tc.Args[1]

	destDir := filepath.Join(c.baseDir, name)
	if err := os.MkdirAll(filepath.Dir(destDir), 0700); err != nil {
		return err
	}

	fmt.Fprintf(tc.Stdout, "Cloning %s into %s...\n", url, destDir)
	cmd := exec.Command("git", "clone", url, destDir)
	cmd.Stdout = tc.Stdout
	cmd.Stderr = tc.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	if err := c.reg.Add(name, url); err != nil {
		return err
	}
	fmt.Fprintf(tc.Stdout, "Source %q registered.\n", name)
	return nil
}

// --- remove ---

type removeCommand struct {
	reg     *registry.Registry
	baseDir string
}

func (c *removeCommand) Name() string        { return "remove" }
func (c *removeCommand) Description() string { return "Remove a registry source" }
func (c *removeCommand) Usage() string       { return "Usage: cure registry remove <name>" }
func (c *removeCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("remove", flag.ContinueOnError)
}

func (c *removeCommand) Run(_ context.Context, tc *terminal.Context) error {
	if len(tc.Args) < 1 {
		return fmt.Errorf("usage: cure registry remove <name>")
	}
	name := tc.Args[0]

	if err := c.reg.Remove(name); err != nil {
		return err
	}

	destDir := filepath.Join(c.baseDir, name)
	os.RemoveAll(destDir)

	fmt.Fprintf(tc.Stdout, "Source %q removed.\n", name)
	return nil
}

// --- update ---

type updateCommand struct {
	reg     *registry.Registry
	baseDir string
}

func (c *updateCommand) Name() string        { return "update" }
func (c *updateCommand) Description() string { return "Update a registry source (git pull)" }
func (c *updateCommand) Usage() string       { return "Usage: cure registry update <name>" }
func (c *updateCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("update", flag.ContinueOnError)
}

func (c *updateCommand) Run(_ context.Context, tc *terminal.Context) error {
	if len(tc.Args) < 1 {
		return fmt.Errorf("usage: cure registry update <name>")
	}
	name := tc.Args[0]

	src, err := c.reg.Load(name)
	if err != nil {
		return err
	}

	fmt.Fprintf(tc.Stdout, "Updating %s...\n", name)
	cmd := exec.Command("git", "-C", src.Path, "pull")
	cmd.Stdout = tc.Stdout
	cmd.Stderr = tc.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	fmt.Fprintf(tc.Stdout, "Source %q updated.\n", name)
	return nil
}

// --- list ---

type regListCommand struct {
	reg *registry.Registry
}

func (c *regListCommand) Name() string        { return "list" }
func (c *regListCommand) Description() string { return "List registered sources" }
func (c *regListCommand) Usage() string       { return "Usage: cure registry list" }
func (c *regListCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("list", flag.ContinueOnError)
}

func (c *regListCommand) Run(_ context.Context, tc *terminal.Context) error {
	sources, err := c.reg.List()
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		fmt.Fprintln(tc.Stdout, "No sources registered. Add one with: cure registry add <name> <git-url>")
		return nil
	}

	tw := tabwriter.NewWriter(tc.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tURL\tUPDATED")
	for _, s := range sources {
		url := s.URL
		if url == "" {
			url = "(local)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", s.Name, url, s.UpdatedAt.Format("2006-01-02 15:04"))
	}
	return tw.Flush()
}
