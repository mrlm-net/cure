// Package mcp provides a stdlib-only Model Context Protocol (MCP) server
// abstraction for Go applications. It implements MCP protocol version 2025-03-26.
//
// # Overview
//
// The package exposes three registerable entity types:
//
//   - [Tool] — a callable function (tools/list, tools/call)
//   - [Resource] — a readable data source (resources/list, resources/read)
//   - [Prompt] — a reusable prompt template (prompts/list, prompts/get)
//
// All JSON-RPC 2.0 dispatch is handled internally. Callers only implement the
// domain interfaces and register them with a [Server].
//
// # Transports
//
// Two transports are supported:
//
//   - Stdio: newline-delimited JSON-RPC over stdin/stdout. Suitable for
//     subprocess-style MCP servers invoked by an AI client.
//   - HTTP Streamable: JSON-RPC over HTTP POST with optional SSE streaming for
//     server-initiated notifications.
//
// [Server.Serve] auto-detects the transport: if os.Stdin is a pipe, stdio is
// used; otherwise HTTP is started on the configured address.
//
// # Schema Building
//
// Use [Schema] and [SchemaBuilder] to describe tool input parameters with a
// fluent API:
//
//	schema := mcp.Schema().
//	    String("query", "Search query", mcp.Required()).
//	    Integer("limit", "Max results", mcp.WithDefault(10)).
//	    Build()
//
// # Example
//
// The following example starts a minimal MCP server with two tools — one using
// a struct and one using [FuncTool] — over stdio:
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"os"
//
//		"github.com/mrlm-net/cure/pkg/mcp"
//	)
//
//	func main() {
//		srv := mcp.New(
//			mcp.WithName("demo"),
//			mcp.WithVersion("1.0.0"),
//		)
//
//		// Tool 1: FuncTool — echo the input message.
//		srv.RegisterTool(mcp.FuncTool(
//			"echo",
//			"Echo the provided message",
//			mcp.Schema().String("message", "Text to echo", mcp.Required()).Build(),
//			func(ctx context.Context, args map[string]any) ([]mcp.Content, error) {
//				msg, _ := args["message"].(string)
//				return mcp.Text(msg), nil
//			},
//		))
//
//		// Tool 2: Struct-based — return a formatted greeting.
//		srv.RegisterTool(&greetTool{})
//
//		if err := srv.ServeStdio(context.Background()); err != nil {
//			fmt.Fprintln(os.Stderr, "mcp server error:", err)
//			os.Exit(1)
//		}
//	}
//
//	type greetTool struct{}
//
//	func (g *greetTool) Name() string        { return "greet" }
//	func (g *greetTool) Description() string { return "Return a greeting for the given name" }
//	func (g *greetTool) Schema() mcp.InputSchema {
//		return mcp.Schema().String("name", "Name to greet", mcp.Required()).Build()
//	}
//	func (g *greetTool) Call(ctx context.Context, args map[string]any) ([]mcp.Content, error) {
//		name, _ := args["name"].(string)
//		return mcp.Textf("Hello, %s!", name), nil
//	}
package mcp
