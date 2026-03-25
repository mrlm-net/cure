// Package prompt provides interactive terminal input with validation and menu support.
//
// The package centers on [Prompter], a struct that wraps an io.Reader and io.Writer
// pair to collect user input from any stream — a real TTY, a test buffer, or a pipe.
// This design makes prompt interactions fully testable without a real terminal.
//
// # Prompter
//
// Create a Prompter with [NewPrompter] by providing the output and input streams:
//
//	p := prompt.NewPrompter(os.Stdout, os.Stdin)
//
// # Required Input
//
// [Prompter.Required] displays a prompt, optionally showing a default value in brackets,
// and repeats until the user provides a non-empty value. Pressing Enter with a default
// set returns the default.
//
//	name, err := p.Required("Project name", "my-project")
//	// Displays: Project name [my-project]
//
// # Optional Input
//
// [Prompter.Optional] accepts empty input and returns the default without re-prompting:
//
//	desc, err := p.Optional("Description (optional)", "")
//
// # Confirmation
//
// [Prompter.Confirm] accepts y/yes/n/no (case-insensitive) and re-prompts on invalid input:
//
//	ok, err := p.Confirm("Overwrite existing file?")
//	// Displays: Overwrite existing file? (y/n):
//
// # Single Selection
//
// [Prompter.SingleSelect] displays a numbered list and returns the selected [Option]:
//
//	opts := []prompt.Option{
//	    {Label: "Go", Value: "go"},
//	    {Label: "TypeScript", Value: "ts"},
//	}
//	selected, err := p.SingleSelect("Choose language", opts)
//	fmt.Println(selected.Value) // "go" or "ts"
//
// # Multi Selection
//
// [Prompter.MultiSelect] accepts comma-separated numbers, "all", or "none":
//
//	selected, err := p.MultiSelect("Select features", opts)
//	// User can enter: "1,3", "all", "none", or "2"
//
// # Terminal Detection
//
// [IsInteractive] reports whether a given io.Reader is a terminal (as opposed to a pipe
// or redirect). Use this to skip prompts in non-interactive environments:
//
//	if prompt.IsInteractive(os.Stdin) {
//	    name, err = p.Required("Project name", "my-project")
//	} else {
//	    name = defaultName
//	}
package prompt
