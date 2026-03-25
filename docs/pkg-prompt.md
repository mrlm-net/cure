---
title: "pkg/prompt"
description: "Interactive terminal input — text prompts, confirmation, single and multi-select menus"
order: 6
section: "libraries"
---

# pkg/prompt

`pkg/prompt` provides interactive user input primitives built on `io.Reader`/`io.Writer`. It never touches `os.Stdin` or `os.Stdout` directly — every method reads from and writes to the streams you inject, making it fully testable without a real terminal.

**Import path:** `github.com/mrlm-net/cure/pkg/prompt`

## Prompter

Construct a `Prompter` by passing your output and input streams:

```go
import "github.com/mrlm-net/cure/pkg/prompt"

p := prompt.NewPrompter(os.Stdout, os.Stdin)
```

In tests, substitute `bytes.Buffer` and `strings.NewReader`:

```go
var out bytes.Buffer
p := prompt.NewPrompter(&out, strings.NewReader("yes\n"))
```

## Text input

```go
// Required — repeats until the user enters something non-empty.
// Returns defaultVal if the user presses Enter and defaultVal is non-empty.
name, err := p.Required("Project name", "my-project")

// Optional — returns defaultVal on Enter; never re-prompts.
desc, err := p.Optional("Description", "")
```

## Confirmation

```go
// Accepts y / yes / n / no (case-insensitive). Re-prompts on invalid input.
ok, err := p.Confirm("Delete this file?")
```

## Menus

```go
options := []prompt.Option{
    {Label: "Go",     Value: "go",     Description: "golang.org"},
    {Label: "Python", Value: "python", Description: "python.org"},
    {Label: "Rust",   Value: "rust",   Description: "rust-lang.org"},
}

// Single selection — numbered 1-based list; re-prompts on invalid choice.
choice, err := p.SingleSelect("Language", options)
fmt.Println(choice.Value) // "go"

// Multi selection — comma-separated numbers, "all", or "none".
// Returns options in original order, deduplicated.
chosen, err := p.MultiSelect("Languages", options)
```

## Terminal detection

```go
// Returns true when stdin is an interactive terminal (os.File + ModeCharDevice).
// Use this to skip prompts in scripts and CI pipelines.
if prompt.IsInteractive(os.Stdin) {
    name, _ = p.Required("Project name", defaultName)
} else {
    name = defaultName
}
```

## Error handling

All methods return `error`. The only error case during normal use is an unexpected EOF (e.g., stdin is closed or redirected and runs out of input before a valid response is received). Invalid menu selections and empty required fields trigger a re-prompt rather than returning an error.
