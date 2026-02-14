package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified, use 'help' for usage")
	}

	switch args[0] {
	case "version":
		fmt.Println("cure version dev")
		return nil
	case "help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q, use 'help' for usage", args[0])
	}
}

func printUsage() {
	fmt.Print(`cure - development task automation tool

Usage:
  cure <command> [arguments]

Commands:
  version    Print version information
  help       Show this help message
`)
}
