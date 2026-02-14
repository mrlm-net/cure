package main

import (
	"fmt"
	"os"

	"github.com/mrlm-net/cure/internal/commands"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	router := terminal.New()
	router.Register(&commands.VersionCommand{})
	router.Register(terminal.NewHelpCommand(router))
	return router.Run(args)
}
