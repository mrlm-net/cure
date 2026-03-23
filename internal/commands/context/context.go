// Package ctxcmd provides the "cure context" command group for managing AI
// conversation sessions. It implements the "new" and "resume" subcommands and
// supports interactive REPL mode as well as single-turn and piped-stdin modes.
package ctxcmd

import (
	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewContextCommand returns the "context" command group, which manages AI
// conversation sessions. It registers the "new" and "resume" subcommands.
func NewContextCommand(st agent.SessionStore) terminal.Command {
	router := terminal.New(
		terminal.WithName("context"),
		terminal.WithDescription("Manage AI conversation sessions"),
	)
	router.Register(&NewCommand{store: st})
	router.Register(&ResumeCommand{store: st})
	router.Register(&ListCommand{store: st})
	router.Register(&ForkCommand{store: st})
	router.Register(&DeleteCommand{store: st})
	return router
}
