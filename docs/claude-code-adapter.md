# Claude Code CLI Adapter — Guide & Dev-Agent Blueprint

`internal/agent/claudecode` wraps the `claude` CLI (Claude Code) as a cure provider.
Unlike the API-backed providers, this adapter invokes `claude` as a subprocess and
parses its NDJSON event stream. Claude Code's built-in tools (Bash, file editing,
web search, etc.) are available to the model automatically.

## Registration

Blank-import the adapter to register the `"claude-code"` provider:

```go
import _ "github.com/mrlm-net/cure/internal/agent/claudecode"
```

The cure binary already registers all four providers in `cmd/cure/main.go`.

## Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `claude_bin` | string | `"claude"` | Path to the `claude` CLI binary |
| `model` | string | `"claude-opus-4-6"` | Model name passed to `--model` |
| `max_turns` | int | `32` | Max agentic turns passed to `--max-turns` |
| `allowed_tools` | []string | `nil` | Whitelist passed to `--allowedTools` |
| `disallowed_tools` | []string | `nil` | Blacklist passed to `--disabledTools` |

## CLI requirements

- **Claude Code must be installed**: `npm install -g @anthropic-ai/claude-code`
- **`--verbose` is required** — without it `--output-format stream-json` returns an error
- The CLI must be authenticated: `claude` picks up `ANTHROPIC_API_KEY` from the environment

## Running a session with the CLI

```sh
# Start a new context with the claude-code provider
cure context new --provider claude-code --model claude-opus-4-6

# Resume an existing session
cure context resume <session-id>
```

---

## Sample tool: `get_time`

The simplest possible custom tool — returns the current UTC time. Use it to verify
that the adapter wires tools through the MCP bridge correctly once Phase 2 ships.
Until then, register it in an integration test.

```go
package main

import (
    "context"
    "fmt"
    "time"

    _ "github.com/mrlm-net/cure/internal/agent/claudecode"
    "github.com/mrlm-net/cure/pkg/agent"
)

func main() {
    // Build a "get_time" tool using agent.FuncTool.
    getTime := agent.FuncTool(
        "get_time",
        "Returns the current UTC time as an RFC3339 string.",
        map[string]any{
            "type":       "object",
            "properties": map[string]any{},
            "required":   []string{},
        },
        func(_ context.Context, _ map[string]any) (string, error) {
            return time.Now().UTC().Format(time.RFC3339), nil
        },
    )

    // Create the agent via the registry.
    ag, err := agent.New("claude-code", map[string]any{
        "model":     "claude-haiku-4-5-20251001",
        "max_turns": 3,
        // Restrict to only the tools we want; prevents accidental shell exec.
        "allowed_tools": []string{},
    })
    if err != nil {
        panic(err)
    }

    // Wire the session.
    sess := agent.NewSession("claude-code", "claude-haiku-4-5-20251001")
    sess.Tools = []agent.Tool{getTime}
    sess.AppendUserMessage("What time is it right now? Use the get_time tool.")

    // Stream and print the response.
    ctx := context.Background()
    for ev, err := range ag.Run(ctx, sess) {
        if err != nil {
            fmt.Printf("error: %v\n", err)
            return
        }
        switch ev.Kind {
        case agent.EventKindToken:
            fmt.Print(ev.Text)
        case agent.EventKindDone:
            fmt.Printf("\n[done: %d in / %d out tokens]\n", ev.InputTokens, ev.OutputTokens)
        case agent.EventKindError:
            fmt.Printf("\nerror: %s\n", ev.Err)
        }
    }
}
```

> **Note — tool forwarding (Phase 2)**: `sess.Tools` registered above are not yet
> automatically forwarded to Claude Code. Phase 2 will start an in-process MCP stdio
> server from `sess.Tools` and pass it via `--mcpServers`, letting Claude Code invoke
> your custom tools through the MCP protocol. Until then, use `allowed_tools: []string{}`
> to disable all of Claude Code's built-in tools and handle calls in your own loop.

---

## Dev-agent blueprint: autonomous develop-and-publish pipeline

The goal: an agent that receives a GitHub issue number, implements the change,
runs tests, opens a PR, and reports back — all without human intervention.

### Architecture

```
┌─────────────────────────────────────────────────┐
│  cure session (claude-code provider)            │
│                                                 │
│  Tools registered:                             │
│    • run_tests   → go test ./...               │
│    • open_pr     → gh pr create                │
│    • post_status → gh issue comment            │
│                                                 │
│  Claude Code built-in tools (via allowed_tools):│
│    • Bash, Read, Write, Edit, Glob, Grep        │
└─────────────────────────────────────────────────┘
```

### Implementation sketch

```go
package devagent

import (
    "context"
    "fmt"
    "os/exec"
    "strings"

    _ "github.com/mrlm-net/cure/internal/agent/claudecode"
    "github.com/mrlm-net/cure/pkg/agent"
    agentstore "github.com/mrlm-net/cure/pkg/agent/store"
)

// RunDevAgent implements a single GitHub issue → PR pipeline.
func RunDevAgent(ctx context.Context, issueNumber int, store agentstore.Store) error {
    // 1. Fetch issue context via gh CLI.
    issueJSON, err := runCmd("gh", "issue", "view", fmt.Sprint(issueNumber), "--json",
        "title,body,labels,assignees")
    if err != nil {
        return fmt.Errorf("fetch issue: %w", err)
    }

    // 2. Build cure tools.
    runTests := agent.FuncTool(
        "run_tests",
        "Run the full test suite with race detector. Returns stdout+stderr.",
        map[string]any{"type": "object", "properties": map[string]any{}, "required": []string{}},
        func(ctx context.Context, _ map[string]any) (string, error) {
            return runCmd("go", "test", "-race", "./...")
        },
    )

    openPR := agent.FuncTool(
        "open_pr",
        "Open a GitHub pull request. Args: title (string), body (string), base (string, default main).",
        map[string]any{
            "type": "object",
            "properties": map[string]any{
                "title": map[string]any{"type": "string"},
                "body":  map[string]any{"type": "string"},
                "base":  map[string]any{"type": "string"},
            },
            "required": []string{"title", "body"},
        },
        func(ctx context.Context, args map[string]any) (string, error) {
            title, _ := args["title"].(string)
            body, _ := args["body"].(string)
            base, _ := args["base"].(string)
            if base == "" {
                base = "main"
            }
            return runCmd("gh", "pr", "create",
                "--title", title,
                "--body", body,
                "--base", base,
            )
        },
    )

    postStatus := agent.FuncTool(
        "post_status",
        "Post a comment to the GitHub issue. Args: body (string).",
        map[string]any{
            "type": "object",
            "properties": map[string]any{
                "body": map[string]any{"type": "string"},
            },
            "required": []string{"body"},
        },
        func(ctx context.Context, args map[string]any) (string, error) {
            body, _ := args["body"].(string)
            return runCmd("gh", "issue", "comment", fmt.Sprint(issueNumber), "--body", body)
        },
    )

    // 3. Build system prompt.
    systemPrompt := strings.TrimSpace(`
You are an autonomous software engineer working on the cure CLI repository.
Your workflow for every task:
1. Read the issue carefully.
2. Explore the relevant code (use Read, Grep, Glob).
3. Implement the change following the existing code style.
4. Write or update tests — run_tests must pass before opening a PR.
5. Create a feature branch: feat/<issue>-<short-title>.
6. Commit with a Conventional Commits message.
7. Call open_pr with a clear title and body.
8. Call post_status with a brief summary of what was done.
Always run tests before opening a PR. If tests fail, fix the code and retry.
`)

    // 4. Create and wire the session.
    sess := agent.NewSession("claude-code", "claude-opus-4-6")
    sess.SystemPrompt = systemPrompt
    sess.Tools = []agent.Tool{runTests, openPR, postStatus}
    sess.AppendUserMessage(fmt.Sprintf(
        "Implement GitHub issue #%d.\n\nIssue details:\n%s",
        issueNumber, issueJSON,
    ))

    // 5. Persist so the session survives restarts.
    if err := store.Save(sess); err != nil {
        return fmt.Errorf("save session: %w", err)
    }

    // 6. Create the agent (Claude Code with full filesystem tools enabled).
    ag, err := agent.New("claude-code", map[string]any{
        "model":     "claude-opus-4-6",
        "max_turns": 32,
        // Allow Claude Code's built-in read/write tools; disallow network browsing.
        "allowed_tools":    []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep"},
        "disallowed_tools": []string{"WebFetch", "WebSearch"},
    })
    if err != nil {
        return fmt.Errorf("build agent: %w", err)
    }

    // 7. Stream events.
    for ev, err := range ag.Run(ctx, sess) {
        if err != nil {
            return fmt.Errorf("stream: %w", err)
        }
        switch ev.Kind {
        case agent.EventKindToken:
            fmt.Print(ev.Text)
        case agent.EventKindToolCall:
            fmt.Printf("\n[tool] %s(%s)\n", ev.ToolCall.ToolName, ev.ToolCall.InputJSON)
        case agent.EventKindToolResult:
            fmt.Printf("[result] %s\n", truncate(ev.ToolResult.Result, 200))
        case agent.EventKindDone:
            fmt.Printf("\n[done: %d in / %d out]\n", ev.InputTokens, ev.OutputTokens)
        case agent.EventKindError:
            return fmt.Errorf("agent error: %s", ev.Err)
        }
    }

    // 8. Persist final session state.
    return store.Save(sess)
}

func runCmd(name string, args ...string) (string, error) {
    out, err := exec.Command(name, args...).CombinedOutput() //nolint:gosec
    return strings.TrimSpace(string(out)), err
}

func truncate(s string, n int) string {
    if len(s) <= n {
        return s
    }
    return s[:n] + "…"
}
```

### Usage

```sh
# Run the dev agent against issue #200
go run ./cmd/devagent -issue 200

# Or use the cure CLI context with a system prompt preset:
cure context new --provider claude-code --skill cure-dev-agent
cure context resume <id> --message "Implement issue #200"
```

### Publishing to testing (preview environment)

Add a `publish_to_testing` tool that triggers a GitHub Actions workflow dispatch:

```go
publishToTesting := agent.FuncTool(
    "publish_to_testing",
    "Trigger the 'deploy-preview' workflow and return the run URL.",
    map[string]any{
        "type":       "object",
        "properties": map[string]any{},
        "required":   []string{},
    },
    func(ctx context.Context, _ map[string]any) (string, error) {
        return runCmd("gh", "workflow", "run", "deploy-preview.yml", "--ref", "HEAD")
    },
)
```

Add this to `sess.Tools` alongside `runTests`, `openPR`, and `postStatus`, then
update the system prompt to include:

```
After the PR is merged, call publish_to_testing to deploy to the preview environment
and include the workflow URL in your post_status comment.
```

---

## Testing the adapter without a real claude binary

Use the `CLAUDE_CODE_INTEGRATION=1` guard in tests, or build a fake `claude`
binary for CI:

```sh
# Build a stub that emits a minimal NDJSON stream.
cat > /tmp/fake-claude.sh << 'EOF'
#!/bin/sh
echo '{"type":"system","subtype":"init","session_id":"test123"}'
echo '{"type":"assistant","message":{"id":"m1","role":"assistant","content":[{"type":"text","text":"pong"}]}}'
echo '{"type":"result","subtype":"success","session_id":"test123","usage":{"input_tokens":10,"output_tokens":4}}'
EOF
chmod +x /tmp/fake-claude.sh

# Point the adapter at the stub.
export CURE_AGENT_CLAUDE_CODE_BIN=/tmp/fake-claude.sh
go test ./internal/agent/claudecode/ -run Integration -v -count=1 CLAUDE_CODE_INTEGRATION=1
```
