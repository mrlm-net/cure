---
title: "pkg/mcp"
description: "Stdlib-only MCP server with stdio and HTTP Streamable transports"
order: 4
section: "libraries"
---

# pkg/mcp

`pkg/mcp` is a stdlib-only implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) targeting protocol version `2025-03-26`. It lets you build MCP servers that expose tools, resources, and prompts to AI clients like Claude Code — with zero external dependencies.

**Import path:** `github.com/mrlm-net/cure/pkg/mcp`

## Design

`pkg/mcp` follows the same patterns as `pkg/terminal`:

- `Tool`, `Resource`, and `Prompt` interfaces mirror the `pkg/terminal.Command` pattern
- `Server` uses functional options for configuration
- Registration order is preserved

## Creating a server

```go
srv := mcp.New(
    mcp.WithName("my-server"),
    mcp.WithVersion("1.0.0"),
)
```

## Registering tools

```go
// Implement the Tool interface
type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "Echo the input text" }
func (t *EchoTool) Schema() mcp.Schema  { /* define input schema */ }
func (t *EchoTool) Call(ctx context.Context, params map[string]any) ([]mcp.Content, error) {
    return []mcp.Content{mcp.Text(params["text"].(string))}, nil
}

srv.RegisterTool(&EchoTool{})
```

Or use `FuncTool` for anonymous functions:

```go
srv.RegisterTool(mcp.FuncTool("echo", "Echo the input text", schema,
    func(ctx context.Context, params map[string]any) ([]mcp.Content, error) {
        return []mcp.Content{mcp.Text(params["text"].(string))}, nil
    },
))
```

## Schema builder

The fluent `Schema()` builder defines tool input schemas:

```go
schema := mcp.Schema().
    String("text", "Text to echo").Required().
    Integer("count", "Number of times to repeat").WithDefault(1)
```

Methods: `.String()`, `.Number()`, `.Integer()`, `.Bool()`, `.Required()`, `.WithEnum()`, `.WithDefault()`.

## Transports

### stdio transport

For integration with Claude Code and local MCP clients:

```go
if err := srv.ServeStdio(ctx); err != nil {
    log.Fatal(err)
}
```

### HTTP Streamable transport

For remote MCP clients with SSE streaming:

```go
if err := srv.ServeHTTP(ctx, "127.0.0.1:8080"); err != nil {
    log.Fatal(err)
}
```

The HTTP transport binds to loopback only by default. Per-session management is handled internally. Origin validation is supported via `WithAllowedOrigins` — `"null"` origins are explicitly rejected when an allowlist is set.

HTTP timeouts: ReadHeader 10s, Read 30s, Idle 120s.

### Auto-detect transport

`Serve` auto-detects the appropriate transport based on whether stdin is a pipe:

```go
if err := srv.Serve(ctx); err != nil {
    log.Fatal(err)
}
```

`IsStdinPipe()` is exported for testing.

## Content helpers

```go
mcp.Text("Hello, world!")           // plain text content
mcp.Textf("Hello, %s!", name)       // formatted text content
```

## Security

- Session IDs are generated with 128-bit `crypto/rand`
- HTTP server binds to `127.0.0.1` (loopback) by default
- `"null"` Origin explicitly rejected when an allowlist is configured
- `WithAllowedOrigins` enforces a strict origin allowlist for CORS
