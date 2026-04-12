// Package projcmd implements the "cure project" command group for managing
// project entities. Package name is projcmd to avoid shadowing the public
// pkg/project package.
package projcmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mrlm-net/cure/pkg/project"
	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewProjectCommand returns the "project" parent command with init/list/show subcommands.
func NewProjectCommand(store *project.Store) terminal.Command {
	router := terminal.New(
		terminal.WithName("project"),
		terminal.WithDescription("Manage project entities"),
	)
	router.Register(&initCommand{store: store})
	router.Register(&listCommand{store: store})
	router.Register(&showCommand{store: store})
	router.Register(&cloneCommand{store: store})
	return router
}

// --- init subcommand ---

type initCommand struct {
	store *project.Store
}

func (c *initCommand) Name() string        { return "init" }
func (c *initCommand) Description() string { return "Create a new project interactively" }
func (c *initCommand) Usage() string {
	return `Usage: cure project init

Create a new project entity with an interactive wizard.
The wizard prompts for project name, repos, AI provider, tracker, and workflow rules.`
}

func (c *initCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("init", flag.ContinueOnError)
}

func (c *initCommand) Run(_ context.Context, tc *terminal.Context) error {
	p := &project.Project{
		Repos: []project.Repo{},
	}

	prompter := prompt.NewPrompter(tc.Stdout, os.Stdin)

	// Name
	name, err := prompter.Required("Project name", "")
	if err != nil {
		return err
	}
	if err := project.ValidateName(name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}
	p.Name = name

	// Description
	desc, err := prompter.Optional("Description", "")
	if err != nil {
		return err
	}
	p.Description = desc

	// Repos
	cwd, _ := os.Getwd()
	repoPath, err := prompter.Optional("Repository path (comma-separated)", cwd)
	if err != nil {
		return err
	}
	if repoPath != "" {
		for _, rp := range strings.Split(repoPath, ",") {
			rp = strings.TrimSpace(rp)
			if rp != "" {
				p.Repos = append(p.Repos, project.Repo{Path: rp})
			}
		}
	}

	// Provider
	providerOpts := []prompt.Option{
		{Label: "Claude", Value: "claude"},
		{Label: "OpenAI", Value: "openai"},
		{Label: "Gemini", Value: "gemini"},
	}
	providerChoice, err := prompter.SingleSelect("Default AI provider", providerOpts)
	if err != nil {
		return err
	}
	p.Defaults.Provider = providerChoice.Value

	// Tracker
	trackerOpts := []prompt.Option{
		{Label: "GitHub Issues", Value: "github"},
		{Label: "Azure DevOps", Value: "azdo"},
		{Label: "None", Value: ""},
	}
	trackerChoice, err := prompter.SingleSelect("Work item tracker", trackerOpts)
	if err != nil {
		return err
	}
	if trackerChoice.Value != "" {
		p.Defaults.Tracker = &project.TrackerCfg{
			Type: trackerChoice.Value,
		}
		if trackerChoice.Value == "github" {
			owner, _ := prompter.Optional("GitHub owner (org/user)", "")
			repo, _ := prompter.Optional("GitHub repo name", "")
			p.Defaults.Tracker.Owner = owner
			p.Defaults.Tracker.Repo = repo
		}
	}

	// Workflow
	useWorkflow, err := prompter.Confirm("Enable workflow enforcement?")
	if err != nil {
		return err
	}
	if useWorkflow {
		p.Workflow = &project.WorkflowCfg{
			BranchPattern:     `^(feat|fix|docs|refactor|test|chore)/\d+-.*$`,
			CommitPattern:     `^(feat|fix|docs|test|refactor|chore)(\(.+\))?!?: .+`,
			ProtectedBranches: []string{"main"},
		}
	}

	// Save
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	if err := c.store.Save(p); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	fmt.Fprintf(tc.Stdout, "\nProject %q created at ~/.cure/projects/%s/project.json\n", p.Name, p.Name)
	return nil
}

// --- list subcommand ---

type listCommand struct {
	store *project.Store
}

func (c *listCommand) Name() string        { return "list" }
func (c *listCommand) Description() string { return "List all registered projects" }
func (c *listCommand) Usage() string       { return "Usage: cure project list" }
func (c *listCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("list", flag.ContinueOnError)
}

func (c *listCommand) Run(_ context.Context, tc *terminal.Context) error {
	projects, err := c.store.List()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Fprintln(tc.Stdout, "No projects found. Create one with: cure project init")
		return nil
	}

	tw := tabwriter.NewWriter(tc.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tREPOS\tPROVIDER\tUPDATED")
	for _, p := range projects {
		updated := p.UpdatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n",
			p.Name, len(p.Repos), p.Defaults.Provider, updated)
	}
	return tw.Flush()
}

// --- show subcommand ---

type showCommand struct {
	store *project.Store
}

func (c *showCommand) Name() string        { return "show" }
func (c *showCommand) Description() string { return "Show project details" }
func (c *showCommand) Usage() string       { return "Usage: cure project show <name>" }
func (c *showCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("show", flag.ContinueOnError)
}

func (c *showCommand) Run(_ context.Context, tc *terminal.Context) error {
	args := tc.Args
	if len(args) == 0 {
		return fmt.Errorf("project name required. Usage: cure project show <name>")
	}
	name := args[0]

	p, err := c.store.Load(name)
	if err != nil {
		return fmt.Errorf("failed to load project %q: %w", name, err)
	}

	enc := json.NewEncoder(tc.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}

// --- clone subcommand ---

type cloneCommand struct {
	store *project.Store
}

func (c *cloneCommand) Name() string        { return "clone" }
func (c *cloneCommand) Description() string { return "Clone project repos into cure workdir" }
func (c *cloneCommand) Usage() string       { return "Usage: cure project clone <name>" }
func (c *cloneCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("clone", flag.ContinueOnError)
}

func (c *cloneCommand) Run(_ context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("project name required")
	}
	name := tc.Args[0]

	p, err := c.store.Load(name)
	if err != nil {
		return fmt.Errorf("failed to load project %q: %w", name, err)
	}

	cfg, err := project.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	for i := range p.Repos {
		r := &p.Repos[i]
		if r.Remote == "" {
			fmt.Fprintf(tc.Stdout, "  - %s — no remote, skipping\n", r.Path)
			continue
		}
		fmt.Fprintf(tc.Stdout, "  Cloning %s...\n", r.Remote)
		if err := project.CloneRepo(r, cfg.WorkDir, p.Name); err != nil {
			fmt.Fprintf(tc.Stdout, "  ! %s — %v\n", r.Remote, err)
			continue
		}
		fmt.Fprintf(tc.Stdout, "  + %s → %s\n", r.Remote, r.LocalPath)
	}

	// Save updated project with LocalPath values
	if err := c.store.Save(p); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	fmt.Fprintf(tc.Stdout, "\nProject %q updated with local paths.\n", name)
	return nil
}
