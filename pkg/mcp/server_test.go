package mcp

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// ---- Test helpers ----

// noopTool is a minimal Tool that does nothing.
type noopTool struct{ name string }

func (t *noopTool) Name() string        { return t.name }
func (t *noopTool) Description() string { return "noop tool" }
func (t *noopTool) Schema() InputSchema { return Schema().Build() }
func (t *noopTool) Call(_ context.Context, _ map[string]any) ([]Content, error) {
	return Text("ok"), nil
}

// noopResource is a minimal Resource.
type noopResource struct{ uri string }

func (r *noopResource) URI() string         { return r.uri }
func (r *noopResource) Name() string        { return "noop" }
func (r *noopResource) Description() string { return "noop resource" }
func (r *noopResource) MIMEType() string    { return "text/plain" }
func (r *noopResource) Read(_ context.Context) ([]ResourceContent, error) {
	return []ResourceContent{{Type: "resource", URI: r.uri, Text: "content"}}, nil
}

// noopPrompt is a minimal Prompt.
type noopPrompt struct{ name string }

func (p *noopPrompt) Name() string                { return p.name }
func (p *noopPrompt) Description() string         { return "noop prompt" }
func (p *noopPrompt) Arguments() []PromptArgument { return nil }
func (p *noopPrompt) Get(_ context.Context, _ map[string]string) ([]Message, error) {
	return []Message{{Role: RoleUser, Content: Text("hello")}}, nil
}

// ---- Tests ----

func TestNew_Defaults(t *testing.T) {
	srv := New()
	if srv.name != "mcp-server" {
		t.Errorf("name = %q, want %q", srv.name, "mcp-server")
	}
	if srv.version != "0.0.0" {
		t.Errorf("version = %q, want %q", srv.version, "0.0.0")
	}
	if srv.addr != ":8080" {
		t.Errorf("addr = %q, want %q", srv.addr, ":8080")
	}
	if srv.sessionTimeout != 30*time.Minute {
		t.Errorf("sessionTimeout = %v, want %v", srv.sessionTimeout, 30*time.Minute)
	}
	if srv.stdin == nil {
		t.Error("stdin must not be nil")
	}
	if srv.stdout == nil {
		t.Error("stdout must not be nil")
	}
	if srv.stderr == nil {
		t.Error("stderr must not be nil")
	}
}

func TestNew_WithOptions(t *testing.T) {
	var in strings.Reader
	var out, errOut bytes.Buffer
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	srv := New(
		WithName("test-server"),
		WithVersion("1.2.3"),
		WithStdin(&in),
		WithStdout(&out),
		WithStderr(&errOut),
		WithAddr(":9999"),
		WithAllowedOrigins("https://example.com"),
		WithSessionTimeout(5*time.Minute),
		WithLogger(logger),
	)

	if srv.name != "test-server" {
		t.Errorf("name = %q, want %q", srv.name, "test-server")
	}
	if srv.version != "1.2.3" {
		t.Errorf("version = %q, want %q", srv.version, "1.2.3")
	}
	if srv.addr != ":9999" {
		t.Errorf("addr = %q, want %q", srv.addr, ":9999")
	}
	if srv.sessionTimeout != 5*time.Minute {
		t.Errorf("sessionTimeout = %v, want %v", srv.sessionTimeout, 5*time.Minute)
	}
	if len(srv.allowedOrigins) != 1 || srv.allowedOrigins[0] != "https://example.com" {
		t.Errorf("allowedOrigins = %v, want [https://example.com]", srv.allowedOrigins)
	}
	if srv.logger != logger {
		t.Error("logger was not set")
	}
}

func TestRegisterTool_Success(t *testing.T) {
	srv := New()
	srv.RegisterTool(&noopTool{name: "my-tool"})
	if _, ok := srv.tools["my-tool"]; !ok {
		t.Error("tool 'my-tool' not found after registration")
	}
	if len(srv.toolOrder) != 1 || srv.toolOrder[0] != "my-tool" {
		t.Errorf("toolOrder = %v, want [my-tool]", srv.toolOrder)
	}
}

func TestRegisterTool_Chaining(t *testing.T) {
	srv := New()
	result := srv.RegisterTool(&noopTool{name: "t1"})
	if result != srv {
		t.Error("RegisterTool must return the server for chaining")
	}
}

func TestRegisterTool_PanicsOnEmptyName(t *testing.T) {
	srv := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty tool name")
		}
	}()
	srv.RegisterTool(&noopTool{name: ""})
}

func TestRegisterTool_PanicsOnDuplicate(t *testing.T) {
	srv := New()
	srv.RegisterTool(&noopTool{name: "dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate tool registration")
		}
	}()
	srv.RegisterTool(&noopTool{name: "dup"})
}

func TestRegisterResource_Success(t *testing.T) {
	srv := New()
	srv.RegisterResource(&noopResource{uri: "file:///test"})
	if _, ok := srv.resources["file:///test"]; !ok {
		t.Error("resource not found after registration")
	}
}

func TestRegisterResource_PanicsOnEmptyURI(t *testing.T) {
	srv := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty resource URI")
		}
	}()
	srv.RegisterResource(&noopResource{uri: ""})
}

func TestRegisterResource_PanicsOnDuplicate(t *testing.T) {
	srv := New()
	srv.RegisterResource(&noopResource{uri: "file:///dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate resource registration")
		}
	}()
	srv.RegisterResource(&noopResource{uri: "file:///dup"})
}

func TestRegisterPrompt_Success(t *testing.T) {
	srv := New()
	srv.RegisterPrompt(&noopPrompt{name: "my-prompt"})
	if _, ok := srv.prompts["my-prompt"]; !ok {
		t.Error("prompt not found after registration")
	}
}

func TestRegisterPrompt_PanicsOnEmptyName(t *testing.T) {
	srv := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty prompt name")
		}
	}()
	srv.RegisterPrompt(&noopPrompt{name: ""})
}

func TestRegisterPrompt_PanicsOnDuplicate(t *testing.T) {
	srv := New()
	srv.RegisterPrompt(&noopPrompt{name: "dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate prompt registration")
		}
	}()
	srv.RegisterPrompt(&noopPrompt{name: "dup"})
}

func TestRegistration_OrderPreserved(t *testing.T) {
	srv := New()
	names := []string{"z-tool", "a-tool", "m-tool"}
	for _, name := range names {
		srv.RegisterTool(&noopTool{name: name})
	}
	if len(srv.toolOrder) != len(names) {
		t.Fatalf("toolOrder length = %d, want %d", len(srv.toolOrder), len(names))
	}
	for i, want := range names {
		if srv.toolOrder[i] != want {
			t.Errorf("toolOrder[%d] = %q, want %q", i, srv.toolOrder[i], want)
		}
	}
}
