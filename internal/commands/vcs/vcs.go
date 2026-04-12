// Package vcscmd implements the "cure vcs" command group for git operations.
package vcscmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mrlm-net/cure/pkg/vcs"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewVCSCommand returns the "vcs" parent command.
func NewVCSCommand() terminal.Command {
	router := terminal.New(
		terminal.WithName("vcs"),
		terminal.WithDescription("Version control operations (git)"),
	)
	router.Register(&statusCmd{})
	router.Register(&branchCmd{})
	router.Register(&diffCmd{})
	router.Register(&logCmd{})
	return router
}

type statusCmd struct{}
func (c *statusCmd) Name() string        { return "status" }
func (c *statusCmd) Description() string { return "Show working tree status" }
func (c *statusCmd) Usage() string       { return "Usage: cure vcs status" }
func (c *statusCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("status", flag.ContinueOnError) }
func (c *statusCmd) Run(_ context.Context, tc *terminal.Context) error {
	cwd, _ := os.Getwd()
	s, err := vcs.Status(cwd)
	if err != nil {
		return err
	}
	fmt.Fprintf(tc.Stdout, "Branch: %s\n", s.Branch)
	if s.Clean {
		fmt.Fprintln(tc.Stdout, "Working tree clean")
		return nil
	}
	if len(s.Staged) > 0 {
		fmt.Fprintln(tc.Stdout, "\nStaged:")
		for _, f := range s.Staged {
			fmt.Fprintf(tc.Stdout, "  %s %s\n", f.Status, f.Path)
		}
	}
	if len(s.Unstaged) > 0 {
		fmt.Fprintln(tc.Stdout, "\nUnstaged:")
		for _, f := range s.Unstaged {
			fmt.Fprintf(tc.Stdout, "  %s %s\n", f.Status, f.Path)
		}
	}
	if len(s.Untracked) > 0 {
		fmt.Fprintln(tc.Stdout, "\nUntracked:")
		for _, f := range s.Untracked {
			fmt.Fprintf(tc.Stdout, "  %s\n", f)
		}
	}
	return nil
}

type branchCmd struct{}
func (c *branchCmd) Name() string        { return "branch" }
func (c *branchCmd) Description() string { return "Create and checkout a new branch" }
func (c *branchCmd) Usage() string       { return "Usage: cure vcs branch <name>" }
func (c *branchCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("branch", flag.ContinueOnError) }
func (c *branchCmd) Run(_ context.Context, tc *terminal.Context) error {
	if len(tc.Args) < 1 {
		return fmt.Errorf("usage: cure vcs branch <name>")
	}
	cwd, _ := os.Getwd()
	return vcs.Branch(cwd, tc.Args[0])
}

type diffCmd struct{}
func (c *diffCmd) Name() string        { return "diff" }
func (c *diffCmd) Description() string { return "Show uncommitted changes" }
func (c *diffCmd) Usage() string       { return "Usage: cure vcs diff" }
func (c *diffCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("diff", flag.ContinueOnError) }
func (c *diffCmd) Run(_ context.Context, tc *terminal.Context) error {
	cwd, _ := os.Getwd()
	d, err := vcs.Diff(cwd)
	if err != nil {
		return err
	}
	if d.Patch == "" {
		fmt.Fprintln(tc.Stdout, "No changes")
		return nil
	}
	fmt.Fprint(tc.Stdout, d.Patch)
	return nil
}

type logCmd struct {
	count int
}
func (c *logCmd) Name() string        { return "log" }
func (c *logCmd) Description() string { return "Show commit history" }
func (c *logCmd) Usage() string       { return "Usage: cure vcs log [--count N]" }
func (c *logCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.IntVar(&c.count, "count", 10, "number of commits")
	return fs
}
func (c *logCmd) Run(_ context.Context, tc *terminal.Context) error {
	cwd, _ := os.Getwd()
	entries, err := vcs.Log(cwd, c.count)
	if err != nil {
		return err
	}
	for _, e := range entries {
		fmt.Fprintf(tc.Stdout, "%s %s\n", e.Hash[:7], e.Subject)
	}
	return nil
}
