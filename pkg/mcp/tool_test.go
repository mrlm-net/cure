package mcp

import (
	"context"
	"errors"
	"testing"
)

// staticTool is a test implementation of Tool using a struct.
type staticTool struct {
	name   string
	desc   string
	schema InputSchema
	result []Content
	err    error
}

func (t *staticTool) Name() string        { return t.name }
func (t *staticTool) Description() string { return t.desc }
func (t *staticTool) Schema() InputSchema { return t.schema }
func (t *staticTool) Call(_ context.Context, _ map[string]any) ([]Content, error) {
	return t.result, t.err
}

func TestFuncTool_Metadata(t *testing.T) {
	schema := Schema().String("msg", "message", Required()).Build()
	tool := FuncTool("echo", "Echo input", schema, func(_ context.Context, _ map[string]any) ([]Content, error) {
		return nil, nil
	})

	if tool.Name() != "echo" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "echo")
	}
	if tool.Description() != "Echo input" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "Echo input")
	}
	got := tool.Schema()
	if got.Type != "object" {
		t.Errorf("Schema().Type = %q, want %q", got.Type, "object")
	}
	if len(got.Required) != 1 || got.Required[0] != "msg" {
		t.Errorf("Schema().Required = %v, want [msg]", got.Required)
	}
}

func TestFuncTool_Call_Success(t *testing.T) {
	tool := FuncTool(
		"greet",
		"Greet the user",
		Schema().String("name", "Name", Required()).Build(),
		func(_ context.Context, args map[string]any) ([]Content, error) {
			name, _ := args["name"].(string)
			return Text("Hello, " + name + "!"), nil
		},
	)

	ctx := context.Background()
	result, err := tool.Call(ctx, map[string]any{"name": "Alice"})
	if err != nil {
		t.Fatalf("Call() returned error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Call() returned %d content items, want 1", len(result))
	}
	tc, ok := result[0].(TextContent)
	if !ok {
		t.Fatalf("result[0] is %T, want TextContent", result[0])
	}
	if tc.Text != "Hello, Alice!" {
		t.Errorf("Text = %q, want %q", tc.Text, "Hello, Alice!")
	}
}

func TestFuncTool_Call_Error(t *testing.T) {
	wantErr := errors.New("tool failed")
	tool := FuncTool(
		"fail",
		"Always fails",
		Schema().Build(),
		func(_ context.Context, _ map[string]any) ([]Content, error) {
			return nil, wantErr
		},
	)

	_, err := tool.Call(context.Background(), nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("Call() error = %v, want %v", err, wantErr)
	}
}

func TestFuncTool_Call_NilArgs(t *testing.T) {
	// Tool should handle nil args gracefully.
	tool := FuncTool(
		"noargs",
		"No args needed",
		Schema().Build(),
		func(_ context.Context, args map[string]any) ([]Content, error) {
			// should not panic on nil args
			_ = args
			return Text("ok"), nil
		},
	)
	result, err := tool.Call(context.Background(), nil)
	if err != nil {
		t.Fatalf("Call() with nil args returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 content, got %d", len(result))
	}
}

func TestFuncTool_ImplementsToolInterface(t *testing.T) {
	var _ Tool = FuncTool("t", "d", Schema().Build(), func(_ context.Context, _ map[string]any) ([]Content, error) {
		return nil, nil
	})
}

func TestStaticTool_ImplementsToolInterface(t *testing.T) {
	var _ Tool = &staticTool{}
}

func TestTool_Call_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	tool := FuncTool(
		"ctx-check",
		"Checks context",
		Schema().Build(),
		func(ctx context.Context, _ map[string]any) ([]Content, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return Text("ok"), nil
		},
	)

	_, err := tool.Call(ctx, nil)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}
