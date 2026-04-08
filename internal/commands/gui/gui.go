// Package guicmd implements the "cure gui" command, which starts a local
// HTTP server with an embedded browser-based GUI. The server binds to
// 127.0.0.1 only and auto-discovers a free port unless --port is specified.
package guicmd

import (
	"context"
	"flag"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/mrlm-net/cure/internal/gui"
	"github.com/mrlm-net/cure/internal/gui/api"
	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/doctor"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// GUICommand starts the browser-based GUI server.
type GUICommand struct {
	port      int
	noBrowser bool

	cfgData config.ConfigObject
	checks  []doctor.CheckFunc
	store   agent.SessionStore
}

// NewGUICommand creates a GUICommand with the given configuration data,
// doctor checks, and optional session store (nil disables session endpoints).
func NewGUICommand(cfgData config.ConfigObject, checks []doctor.CheckFunc, store agent.SessionStore) terminal.Command {
	return &GUICommand{
		cfgData: cfgData,
		checks:  checks,
		store:   store,
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

// Run starts the GUI server and blocks until the context is cancelled or
// SIGINT/SIGTERM is received.
func (c *GUICommand) Run(ctx context.Context, tc *terminal.Context) error {
	deps := api.Deps{
		Config: c.cfgData,
		Checks: c.checks,
		Port:   c.port,
		Store:  c.store,
	}

	apiRouter := api.NewAPIRouter(deps)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)

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
