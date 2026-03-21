package mcp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Server is the core MCP server. It maintains a registry of Tools, Resources,
// and Prompts, and dispatches JSON-RPC 2.0 requests over stdio or HTTP
// Streamable transport.
//
// Create a Server with [New] and register handlers before calling [Server.Serve],
// [Server.ServeStdio], or [Server.ServeHTTP].
type Server struct {
	name    string
	version string

	mu        sync.RWMutex
	tools     map[string]Tool
	toolOrder []string
	resources map[string]Resource
	resOrder  []string
	prompts   map[string]Prompt
	prmOrder  []string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	addr           string
	allowedOrigins []string
	sessionTimeout time.Duration

	logger *slog.Logger
}

// Option is a functional option for configuring a [Server].
type Option func(*Server)

// WithName sets the server name reported in the MCP initialize response.
//
// Default: "mcp-server"
func WithName(name string) Option {
	return func(s *Server) {
		s.name = name
	}
}

// WithVersion sets the server version reported in the MCP initialize response.
//
// Default: "0.0.0"
func WithVersion(v string) Option {
	return func(s *Server) {
		s.version = v
	}
}

// WithStdin overrides the reader used for stdio transport input.
//
// Default: os.Stdin
func WithStdin(r io.Reader) Option {
	return func(s *Server) {
		s.stdin = r
	}
}

// WithStdout overrides the writer used for stdio transport output.
//
// Default: os.Stdout
func WithStdout(w io.Writer) Option {
	return func(s *Server) {
		s.stdout = w
	}
}

// WithStderr overrides the writer used for diagnostic output.
//
// Default: os.Stderr
func WithStderr(w io.Writer) Option {
	return func(s *Server) {
		s.stderr = w
	}
}

// WithAddr sets the TCP address for HTTP Streamable transport.
// The default ("127.0.0.1:8080") binds the loopback interface only, which is
// the safe default for local development tools. To expose the server on all
// interfaces (e.g. in a container), pass ":8080" or "0.0.0.0:8080" explicitly.
//
// Default: "127.0.0.1:8080"
func WithAddr(addr string) Option {
	return func(s *Server) {
		s.addr = addr
	}
}

// WithAllowedOrigins restricts CORS Origin header values for the HTTP transport.
// An empty slice (the default) allows all origins — safe for local development
// but not for production deployments accessible over a network.
// Provide explicit allowed origins in production to prevent DNS rebinding attacks.
//
// When a non-empty list is provided, the literal "null" origin (sent by browsers
// for file:// and sandboxed iframe requests) is always rejected. Requests without
// an Origin header (non-browser / server-to-server) are always allowed.
//
// Default: nil (allow all origins)
func WithAllowedOrigins(origins ...string) Option {
	return func(s *Server) {
		s.allowedOrigins = origins
	}
}

// WithSessionTimeout sets the idle timeout for HTTP sessions.
//
// Default: 30 minutes
func WithSessionTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.sessionTimeout = d
	}
}

// WithLogger sets a structured logger for the server. The server logs request
// dispatch and errors at Debug and Error levels.
//
// Default: nil (no logging)
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

// New creates a new Server with sensible defaults. Pass [Option] values to
// override defaults.
//
// Defaults:
//   - name: "mcp-server"
//   - version: "0.0.0"
//   - stdin/stdout/stderr: os.Stdin/os.Stdout/os.Stderr
//   - addr: "127.0.0.1:8080" (loopback-only — safe default for local tools)
//   - allowedOrigins: nil (all origins allowed — suitable for development/local use;
//     provide explicit origins in production to prevent DNS rebinding attacks)
//   - sessionTimeout: 30 minutes
func New(opts ...Option) *Server {
	s := &Server{
		name:           "mcp-server",
		version:        "0.0.0",
		tools:          make(map[string]Tool),
		resources:      make(map[string]Resource),
		prompts:        make(map[string]Prompt),
		stdin:          os.Stdin,
		stdout:         os.Stdout,
		stderr:         os.Stderr,
		addr:           "127.0.0.1:8080",
		sessionTimeout: 30 * time.Minute,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RegisterTool registers a [Tool] with the server. Tools are listed and callable
// by MCP clients. Returns the server for method chaining.
//
// Panics if tool.Name() is empty or if a tool with the same name is already
// registered.
func (s *Server) RegisterTool(tool Tool) *Server {
	name := tool.Name()
	if name == "" {
		panic("mcp: tool name cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tools[name]; exists {
		panic(fmt.Sprintf("mcp: tool %q already registered", name))
	}
	s.tools[name] = tool
	s.toolOrder = append(s.toolOrder, name)
	return s
}

// RegisterResource registers a [Resource] with the server. Resources are listed
// and readable by MCP clients. Returns the server for method chaining.
//
// Panics if resource.URI() is empty or if a resource with the same URI is
// already registered.
func (s *Server) RegisterResource(r Resource) *Server {
	uri := r.URI()
	if uri == "" {
		panic("mcp: resource URI cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.resources[uri]; exists {
		panic(fmt.Sprintf("mcp: resource %q already registered", uri))
	}
	s.resources[uri] = r
	s.resOrder = append(s.resOrder, uri)
	return s
}

// RegisterPrompt registers a [Prompt] with the server. Prompts are listed and
// retrievable by MCP clients. Returns the server for method chaining.
//
// Panics if prompt.Name() is empty or if a prompt with the same name is already
// registered.
func (s *Server) RegisterPrompt(p Prompt) *Server {
	name := p.Name()
	if name == "" {
		panic("mcp: prompt name cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.prompts[name]; exists {
		panic(fmt.Sprintf("mcp: prompt %q already registered", name))
	}
	s.prompts[name] = p
	s.prmOrder = append(s.prmOrder, name)
	return s
}

// Serve auto-detects the appropriate transport and starts serving MCP requests.
// If os.Stdin is a pipe (non-interactive), stdio transport is used; otherwise
// HTTP Streamable transport is used on the configured address.
//
// Serve blocks until ctx is cancelled or a fatal error occurs.
func (s *Server) Serve(ctx context.Context) error {
	if IsStdinPipe() {
		return s.ServeStdio(ctx)
	}
	return s.ServeHTTP(ctx, "")
}

// IsStdinPipe reports whether os.Stdin is connected to a pipe or other
// non-character-device (i.e., not an interactive terminal). This is used by
// [Server.Serve] to auto-detect the appropriate transport.
func IsStdinPipe() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}
