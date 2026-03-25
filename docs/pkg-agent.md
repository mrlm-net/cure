---
title: "pkg/agent"
description: "Provider-agnostic AI agent abstraction with session management and registry"
order: 2
section: "libraries"
---

# pkg/agent

> The [`cure context`](/docs/context) command group is the ready-to-use CLI front-end for this package. Use `pkg/agent` directly when you want to embed session management into your own Go programs.

`pkg/agent` provides a provider-agnostic abstraction for AI agent context management. It defines the core interfaces, session lifecycle, a global provider registry, and a persistence interface — without coupling to any specific AI provider.

**Import path:** `github.com/mrlm-net/cure/pkg/agent`

## Registering a provider

Provider adapters live in `internal/agent/<provider>/` and self-register via `init()` using the blank-import driver pattern — the same convention as `database/sql` drivers.

```go
import (
    "github.com/mrlm-net/cure/pkg/agent"

    // Register the "claude" provider by importing its adapter package.
    // The adapter calls agent.Register("claude", factory) in its init() function.
    _ "github.com/mrlm-net/cure/internal/agent/claude"
)
```

After the blank import, `agent.New("claude", cfg)` is available. `agent.Registered()` returns a sorted list of all registered provider names.

## internal/agent/claude — Anthropic Claude adapter

The Claude adapter lives in `internal/agent/claude` and wires the Anthropic Go SDK into `pkg/agent`. It uses the blank-import driver pattern so your application code stays decoupled from the adapter package.

**Prerequisite** — set the API key in the environment before creating the agent:

```sh
export ANTHROPIC_API_KEY=sk-ant-...
```

**Full example** — blank-import the adapter, create a session, stream a response, and persist it with `JSONStore`:

```go
import (
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/mrlm-net/cure/pkg/agent"
    "github.com/mrlm-net/cure/pkg/agent/store"

    // Registers the "claude" provider via init().
    _ "github.com/mrlm-net/cure/internal/agent/claude"
)

func main() {
    ctx := context.Background()

    // Create a file-backed session store.
    s, err := store.NewJSONStore("~/.local/share/cure/sessions")
    if err != nil {
        log.Fatal(err)
    }

    // Instantiate the Claude agent.
    a, err := agent.New("claude", map[string]any{
        "model":      "claude-opus-4-6",
        "max_tokens": 8192,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Build a session.
    session := agent.NewSession("claude", "claude-opus-4-6")
    session.SystemPrompt = "You are a concise technical assistant."
    session.AppendUserMessage("What is the difference between os.Rename and os.Link in Go?")

    // Stream the response using Go 1.23 range-over-function syntax.
    var response strings.Builder
    for ev, err := range a.Run(ctx, session) {
        if err != nil {
            log.Fatal(err)
        }
        switch ev.Kind {
        case agent.EventKindToken:
            response.WriteString(ev.Text)
            fmt.Print(ev.Text) // stream to terminal
        case agent.EventKindDone:
            fmt.Printf("\n\ntokens: in=%d out=%d stop=%s\n",
                ev.InputTokens, ev.OutputTokens, ev.StopReason)
        case agent.EventKindError:
            log.Fatalf("provider error: %s", ev.Err)
        }
    }

    // Append the assistant reply and persist the session.
    session.AppendAssistantMessage(response.String())
    if err := s.Save(ctx, session); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("session saved: %s\n", session.ID)
}
```

**Configuration keys accepted by `NewClaudeAgent`:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api_key_env` | `string` | `"ANTHROPIC_API_KEY"` | Name of the environment variable holding the API key |
| `model` | `string` | `"claude-opus-4-6"` | Anthropic model ID |
| `max_tokens` | `int` / `int64` / `float64` | `8192` | Maximum tokens in the completion |

**Note** — `internal/agent/claude` is an internal package and cannot be imported by external modules. This is intentional: cure ships the adapter as part of the application layer, while `pkg/agent` remains dependency-free. If you need to use the Claude adapter outside of cure, copy the adapter source into your own `internal/` package.

## Creating a session

`Session` holds the full conversation state. `NewSession` generates a 128-bit cryptographically random ID.

```go
session := agent.NewSession("claude", "claude-opus-4-6")
session.SystemPrompt = "You are a helpful assistant."
session.AppendUserMessage("Summarise the Go 1.23 release notes.")
```

Fork a session to branch a conversation without mutating the original:

```go
branch := session.Fork()
// branch.ForkOf == session.ID
// branch.History is a deep copy — appending to branch does not affect session
```

## Streaming a response

`Agent.Run` returns `iter.Seq2[Event, error]`, iterated with Go 1.23's range-over-function syntax.

```go
var response strings.Builder

for ev, err := range a.Run(ctx, session) {
    if err != nil {
        log.Fatal(err)
    }
    switch ev.Kind {
    case agent.EventKindToken:
        response.WriteString(ev.Text)
    case agent.EventKindDone:
        fmt.Printf("tokens: in=%d out=%d stop=%s\n",
            ev.InputTokens, ev.OutputTokens, ev.StopReason)
    case agent.EventKindError:
        log.Fatalf("provider error: %s", ev.Err)
    }
}

session.AppendAssistantMessage(response.String())
```

Cancelling the context terminates the stream early.

## Event kinds

| Constant | When emitted |
|----------|-------------|
| `EventKindToken` | Each text token streamed from the provider |
| `EventKindStart` | Stream started |
| `EventKindDone` | Stream completed with token usage stats |
| `EventKindError` | Provider returned an error |

## Persisting sessions

`SessionStore` is the interface for concrete persistence implementations:

```go
type SessionStore interface {
    Save(ctx context.Context, s *Session) error
    Load(ctx context.Context, id string) (*Session, error)
    List(ctx context.Context) ([]*Session, error)
    Delete(ctx context.Context, id string) error
    Fork(ctx context.Context, id string) (*Session, error)
}
```

`Load`, `Delete`, and `Fork` return `ErrSessionNotFound` (or a wrapped form) when the session ID does not exist. Check with `errors.Is`:

```go
_, err := store.Load(ctx, id)
if errors.Is(err, agent.ErrSessionNotFound) {
    // session does not exist
}
```

## Token estimation

For quick context window budget calculations before calling a provider:

```go
n := agent.EstimateTokens(text) // len(text) / 4
```

For precise counts, use `Agent.CountTokens`. If the provider does not support it, `ErrCountNotSupported` is returned:

```go
n, err := a.CountTokens(ctx, session)
if errors.Is(err, agent.ErrCountNotSupported) {
    n = agent.EstimateTokens(session.SystemPrompt) // fallback heuristic
}
```

## Sentinel errors

| Error | When returned |
|-------|---------------|
| `ErrProviderNotFound` | `agent.New` called with an unregistered provider name |
| `ErrSessionNotFound` | `Load`, `Delete`, or `Fork` for a non-existent session ID |
| `ErrCountNotSupported` | Provider does not implement token counting |

## Testing custom stores

Use the shared test suite to verify any `SessionStore` implementation:

```go
func TestMyStore(t *testing.T) {
    store := NewMyStore(t.TempDir())
    agent.RunSessionStoreTests(t, store)
}
```

`RunSessionStoreTests` covers save/load round-trips, not-found errors, list, delete, and fork semantics.
