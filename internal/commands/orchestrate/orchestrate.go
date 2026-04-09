// Package orchcmd implements the "cure orchestrate" command group.
package orchcmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mrlm-net/cure/internal/orchestrator"
	"github.com/mrlm-net/cure/pkg/project"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewOrchestrateCommand returns the "orchestrate" command group.
func NewOrchestrateCommand(store *project.Store) terminal.Command {
	router := terminal.New(
		terminal.WithName("orchestrate"),
		terminal.WithDescription("Manage orchestrated agent containers"),
	)
	router.Register(&initCmd{store: store})
	router.Register(&upCmd{store: store})
	router.Register(&downCmd{store: store})
	router.Register(&statusCmd{store: store})
	return router
}

func detectProject(store *project.Store) (*project.Project, string, error) {
	det := project.NewDetector(store)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	p, err := det.Detect(cwd)
	if err != nil || p == nil {
		return nil, "", fmt.Errorf("no project detected in %s", cwd)
	}
	return p, cwd, nil
}

type initCmd struct{ store *project.Store }
func (c *initCmd) Name() string        { return "init" }
func (c *initCmd) Description() string { return "Generate docker-compose.cure.yml" }
func (c *initCmd) Usage() string       { return "Usage: cure orchestrate init" }
func (c *initCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("init", flag.ContinueOnError) }
func (c *initCmd) Run(_ context.Context, tc *terminal.Context) error {
	p, cwd, err := detectProject(c.store)
	if err != nil {
		return err
	}
	orch := orchestrator.New(p, cwd)
	if err := orch.Init(); err != nil {
		return err
	}
	fmt.Fprintf(tc.Stdout, "Generated docker-compose.cure.yml in %s\n", cwd)
	return nil
}

type upCmd struct{ store *project.Store }
func (c *upCmd) Name() string        { return "up" }
func (c *upCmd) Description() string { return "Start agent containers" }
func (c *upCmd) Usage() string       { return "Usage: cure orchestrate up" }
func (c *upCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("up", flag.ContinueOnError) }
func (c *upCmd) Run(ctx context.Context, tc *terminal.Context) error {
	p, cwd, err := detectProject(c.store)
	if err != nil {
		return err
	}
	orch := orchestrator.New(p, cwd)
	fmt.Fprintln(tc.Stdout, "Starting agent containers...")
	return orch.Up(ctx)
}

type downCmd struct{ store *project.Store }
func (c *downCmd) Name() string        { return "down" }
func (c *downCmd) Description() string { return "Stop agent containers" }
func (c *downCmd) Usage() string       { return "Usage: cure orchestrate down" }
func (c *downCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("down", flag.ContinueOnError) }
func (c *downCmd) Run(ctx context.Context, tc *terminal.Context) error {
	p, cwd, err := detectProject(c.store)
	if err != nil {
		return err
	}
	orch := orchestrator.New(p, cwd)
	fmt.Fprintln(tc.Stdout, "Stopping agent containers...")
	return orch.Down(ctx)
}

type statusCmd struct{ store *project.Store }
func (c *statusCmd) Name() string        { return "status" }
func (c *statusCmd) Description() string { return "Show container status" }
func (c *statusCmd) Usage() string       { return "Usage: cure orchestrate status" }
func (c *statusCmd) Flags() *flag.FlagSet { return flag.NewFlagSet("status", flag.ContinueOnError) }
func (c *statusCmd) Run(ctx context.Context, tc *terminal.Context) error {
	p, cwd, err := detectProject(c.store)
	if err != nil {
		return err
	}
	orch := orchestrator.New(p, cwd)
	statuses, err := orch.Status(ctx)
	if err != nil {
		return err
	}
	if len(statuses) == 0 {
		fmt.Fprintln(tc.Stdout, "No containers running.")
		return nil
	}
	for _, s := range statuses {
		fmt.Fprintf(tc.Stdout, "  %s: %s\n", s.Name, s.State)
	}
	return nil
}
