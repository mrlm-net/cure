// Package guicmd implements the "cure gui" command, which starts a local
// HTTP server with an embedded browser-based GUI. The server binds to
// 127.0.0.1 only and auto-discovers a free port unless --port is specified.
package guicmd

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mrlm-net/cure/internal/gui"
	"github.com/mrlm-net/cure/internal/gui/api"
	"github.com/mrlm-net/cure/internal/gui/ws"
	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/doctor"
	"github.com/mrlm-net/cure/pkg/project"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// GUICommand starts the browser-based GUI server.
type GUICommand struct {
	port      int
	noBrowser bool

	cfgData      config.ConfigObject
	checks       []doctor.CheckFunc
	store        agent.SessionStore
	projectStore project.ProjectStore
}

// NewGUICommand creates a GUICommand with the given configuration data,
// doctor checks, optional session store, and optional project store.
func NewGUICommand(cfgData config.ConfigObject, checks []doctor.CheckFunc, store agent.SessionStore, projectStore project.ProjectStore) terminal.Command {
	return &GUICommand{
		cfgData:      cfgData,
		checks:       checks,
		store:        store,
		projectStore: projectStore,
	}
}

// Name returns "gui".
func (c *GUICommand) Name() string { return "gui" }

// Description returns a short description for help output.
func (c *GUICommand) Description() string { return "Start the browser-based GUI" }

// Usage returns detailed usage information.
func (c *GUICommand) Usage() string {
	return `Usage: cure gui [flags]

Start a local HTTP server with an embedded browser-based GUI.

Flags:
  --port int       Port to listen on (0 = auto-discover free port, default 0)
  --no-browser     Do not open the browser automatically`
}

// Flags returns a FlagSet with --port and --no-browser flags.
func (c *GUICommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("gui", flag.ContinueOnError)
	fs.IntVar(&c.port, "port", 0, "port to listen on (0 = auto)")
	fs.BoolVar(&c.noBrowser, "no-browser", false, "do not open browser")
	return fs
}

func makeAgentRun(cfgData config.ConfigObject) api.AgentRunFunc {
	model, _ := cfgData["agent.claude.model"].(string)
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	// Try API direct first, then Claude Code CLI (unified adapter with streaming + tool events)
	a, err := agent.New("claude", map[string]any{"model": model})
	if err != nil {
		a, err = agent.New("claude-code", map[string]any{"model": model})
	}
	if err != nil {
		return nil
	}

	return func(ctx context.Context, sess *agent.Session) <-chan api.AgentResult {
		ch := make(chan api.AgentResult, 16)
		go func() {
			defer close(ch)
			for ev, err := range a.Run(ctx, sess) {
				select {
				case <-ctx.Done():
					return
				case ch <- api.AgentResult{Event: ev, Err: err}:
				}
				if err != nil {
					return
				}
			}
		}()
		return ch
	}
}

// Run starts the GUI server and blocks until the context is cancelled or
// SIGINT/SIGTERM is received.
func (c *GUICommand) Run(ctx context.Context, tc *terminal.Context) error {
	// Detect project from cwd for session association.
	var projectName string
	if c.projectStore != nil {
		det := project.NewDetector(c.projectStore)
		if cwd, err := os.Getwd(); err == nil {
			if p, err := det.Detect(cwd); err == nil && p != nil {
				projectName = p.Name
			}
		}
	}

	// Collect project repo paths for file API scoping.
	var projectRoots []string
	if c.projectStore != nil && projectName != "" {
		if p, err := c.projectStore.Load(projectName); err == nil {
			for _, r := range p.Repos {
				projectRoots = append(projectRoots, r.Path)
			}
		}
	}
	if len(projectRoots) == 0 {
		if cwd, err := os.Getwd(); err == nil {
			projectRoots = []string{cwd}
		}
	}

	deps := api.Deps{
		Config:       c.cfgData,
		Checks:       c.checks,
		Port:         c.port,
		Store:        c.store,
		AgentRun:     makeAgentRun(c.cfgData),
		ProjectStore: c.projectStore,
		ProjectName:  projectName,
		ProjectRoots: projectRoots,
	}

	apiRouter := api.NewAPIRouter(deps)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)

	// WebSocket endpoints for terminal
	termWorkDir := "."
	if len(projectRoots) > 0 {
		termWorkDir = projectRoots[0]
	}
	mux.Handle("/api/terminal/", ws.TerminalHandler(termWorkDir))

	var opts []gui.Option
	if c.port > 0 {
		opts = append(opts, gui.WithPort(c.port))
	}
	if c.noBrowser {
		opts = append(opts, gui.WithNoBrowser())
	}
	opts = append(opts, gui.WithStdout(tc.Stdout), gui.WithStderr(tc.Stderr))

	srv := gui.New(mux, opts...)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx)
}
