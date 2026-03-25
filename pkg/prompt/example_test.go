package prompt_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mrlm-net/cure/pkg/prompt"
)

// ExamplePrompter demonstrates creating a Prompter backed by an in-memory
// reader. This pattern is identical to what you would use in automated tests
// or non-interactive environments where a real TTY is not available.
func ExamplePrompter() {
	// Simulate a user typing "my-project\n" then "y\n".
	input := strings.NewReader("my-project\ny\n")
	out := &bytes.Buffer{}
	p := prompt.NewPrompter(out, input)

	name, _ := p.Required("Project name", "cure")
	ok, _ := p.Confirm("Initialise git repo?")

	fmt.Println(name)
	fmt.Println(ok)
	// Output:
	// my-project
	// true
}

// ExamplePrompter_multiSelect demonstrates selecting multiple options from a
// list using comma-separated numbers, "all", or "none".
func ExamplePrompter_multiSelect() {
	opts := []prompt.Option{
		{Label: "Logging", Value: "logging"},
		{Label: "Metrics", Value: "metrics"},
		{Label: "Tracing", Value: "tracing"},
	}
	input := strings.NewReader("1,3\n")
	out := &bytes.Buffer{}
	p := prompt.NewPrompter(out, input)

	selected, _ := p.MultiSelect("Select features", opts)
	for _, o := range selected {
		fmt.Println(o.Value)
	}
	// Output:
	// logging
	// tracing
}
