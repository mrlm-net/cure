---
title: "pkg/terminal"
description: "CLI routing framework with serial, concurrent, and pipeline execution strategies"
order: 1
section: "libraries"
---

# pkg/terminal

`pkg/terminal` is cure's CLI framework. It provides the `Command` interface, a radix-tree `Router` for command dispatch, and three built-in execution strategies (runners). Any Go program can embed it to build structured CLI tooling without external dependencies.

**Import path:** `github.com/mrlm-net/cure/pkg/terminal`

## Command interface

Implement five methods to define a command:

```go
type VersionCommand struct{}

func (c *VersionCommand) Name()        string        { return "version" }
func (c *VersionCommand) Description() string        { return "Print version information" }
func (c *VersionCommand) Usage()       string        { return "Usage: myapp version" }
func (c *VersionCommand) Flags()       *flag.FlagSet { return nil }

func (c *VersionCommand) Run(ctx context.Context, tc *terminal.Context) error {
    fmt.Fprintln(tc.Stdout, "myapp version 1.0.0")
    return nil
}
```

Commands receive a `*terminal.Context` containing:

- `tc.Args` — parsed positional arguments
- `tc.Flags` — parsed `*flag.FlagSet`
- `tc.Stdout`, `tc.Stderr`, `tc.Stdin` — I/O streams
- `tc.Logger` — structured logger (`log/slog`)
- `tc.Config` — merged configuration

Commands must write all output to these streams — never to `os.Stdout` directly.

## Router

`terminal.New` creates a router with functional options. Commands are registered and dispatched by name via a radix tree:

```go
router := terminal.New(
    terminal.WithStdout(os.Stdout),
    terminal.WithStderr(os.Stderr),
    terminal.WithRunner(&terminal.SerialRunner{}), // default
)
router.Register(&VersionCommand{})
router.Register(terminal.NewHelpCommand(router))

if err := router.RunContext(ctx, os.Args[1:]); err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
}
```

**Functional options:**

| Option | Default | Description |
|--------|---------|-------------|
| `WithStdout(w)` | `os.Stdout` | Standard output stream |
| `WithStderr(w)` | `os.Stderr` | Standard error stream |
| `WithRunner(r)` | `&SerialRunner{}` | Execution strategy |
| `WithConfig(cfg)` | `nil` | Merged config passed to commands |
| `WithSignalHandler()` | off | Cancel context on SIGINT/SIGTERM; second signal calls `os.Exit(1)` |
| `WithTimeout(d)` | none | Per-command execution deadline |
| `WithGracePeriod(d)` | `5s` | Time for cleanup after cancellation |

## Nested routers

A `Router` implements `Command`, so sub-routers can be registered as subcommand groups:

```go
parent := terminal.New()
child  := terminal.New(terminal.WithName("context"), terminal.WithDescription("Manage sessions"))
child.Register(&NewCommand{})
child.Register(&ListCommand{})
parent.Register(child) // cure context new / cure context list
```

## Aliases

Register alternative names for a command:

```go
router.RegisterWithAliases(&VersionCommand{}, "v", "ver")
// now "myapp v", "myapp ver", and "myapp version" all work
```

## Execution strategies

The runner controls how matched commands are executed. Pass one via `WithRunner`.

### SerialRunner (default)

Executes commands sequentially. Stops at the first error.

```go
router := terminal.New() // SerialRunner is the default
```

All commands share the same `*Context`. Suitable for single-command dispatch (the standard case) and ordered batch execution.

### ConcurrentRunner

Executes commands concurrently using goroutines, up to a configurable worker limit. Each command receives its own `*Context` copy to prevent data races. All errors are collected and returned together via `errors.Join`.

```go
router := terminal.New(
    terminal.WithRunner(terminal.WithMaxWorkers(4)),
)
```

`WithMaxWorkers(0)` or negative values default to `runtime.NumCPU()`. Context cancellation prevents new commands from starting; in-flight commands receive the cancelled context.

### PipelineRunner

Connects commands in sequence: the stdout of command N is piped directly to the stdin of command N+1 using `io.Pipe`. The first command reads from the router's stdin; the last writes to the router's stdout.

```go
router := terminal.New(
    terminal.WithRunner(&terminal.PipelineRunner{}),
)
router.Register(&FetchCommand{})
router.Register(&TransformCommand{})
router.Register(&FormatCommand{})
// FetchCommand → TransformCommand → FormatCommand, all running concurrently
```

All stages launch concurrently and are throttled naturally by data flow — a stage blocks until the previous stage produces output. A failed stage closes its write pipe with the error, causing downstream stages to receive `io.ErrClosedPipe` on their next read. The first non-nil error across all stages is returned.

## Signal handling and timeouts

```go
router := terminal.New(
    terminal.WithSignalHandler(),           // SIGINT/SIGTERM → cancel context
    terminal.WithTimeout(30*time.Second),   // hard deadline per command
    terminal.WithGracePeriod(5*time.Second),// cleanup window after cancellation
)
```

On the first SIGINT or SIGTERM the command context is cancelled — commands should return promptly. A second signal calls `os.Exit(1)` immediately.

## Error types

`pkg/terminal` defines structured error types for precise error handling:

| Type | When returned |
|------|---------------|
| `CommandError` | A command returned a non-nil error |
| `CommandNotFoundError` | No command matched the given name (includes "did you mean?" suggestion) |
| `NoCommandError` | No arguments were provided |
| `FlagParseError` | Flag parsing failed |
