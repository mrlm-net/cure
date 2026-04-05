package agent_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestFuncTool(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string"},
		},
	}

	tests := []struct {
		name     string
		toolName string
		desc     string
		schema   map[string]any
		fn       func(context.Context, map[string]any) (string, error)
		args     map[string]any
		wantOut  string
		wantErr  bool
	}{
		{
			name:     "happy path returns correct metadata and result",
			toolName: "echo",
			desc:     "Echo the input",
			schema:   schema,
			fn: func(_ context.Context, args map[string]any) (string, error) {
				v, _ := args["input"].(string)
				return v, nil
			},
			args:    map[string]any{"input": "hello"},
			wantOut: "hello",
		},
		{
			name:     "error propagates from fn",
			toolName: "failing",
			desc:     "Always fails",
			schema:   map[string]any{"type": "object"},
			fn: func(_ context.Context, _ map[string]any) (string, error) {
				return "", errors.New("tool error")
			},
			args:    nil,
			wantErr: true,
		},
		{
			name:     "nil context does not panic",
			toolName: "noop",
			desc:     "Does nothing",
			schema:   map[string]any{"type": "object"},
			fn: func(_ context.Context, _ map[string]any) (string, error) {
				return "ok", nil
			},
			args:    nil,
			wantOut: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := agent.FuncTool(tt.toolName, tt.desc, tt.schema, tt.fn)

			if tool.Name() != tt.toolName {
				t.Errorf("Name() = %q, want %q", tool.Name(), tt.toolName)
			}
			if tool.Description() != tt.desc {
				t.Errorf("Description() = %q, want %q", tool.Description(), tt.desc)
			}
			if got := tool.Schema(); len(got) != len(tt.schema) {
				t.Errorf("Schema() = %v, want %v", got, tt.schema)
			}

			//nolint:staticcheck // intentionally passing nil context for test coverage
			out, err := tool.Call(nil, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Call() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Call() unexpected error: %v", err)
			}
			if out != tt.wantOut {
				t.Errorf("Call() = %q, want %q", out, tt.wantOut)
			}
		})
	}
}

func BenchmarkFuncTool_Call(b *testing.B) {
	tool := agent.FuncTool(
		"bench",
		"Benchmark tool",
		map[string]any{"type": "object"},
		func(_ context.Context, args map[string]any) (string, error) {
			v, _ := args["x"].(string)
			return v, nil
		},
	)
	args := map[string]any{"x": "value"}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Call(ctx, args)
	}
}

func TestFuncTool_NilSchemaPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil schema, got none")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "nil schema") {
			t.Errorf("panic message %q does not mention nil schema", msg)
		}
		if !strings.Contains(msg, "myTool") {
			t.Errorf("panic message %q does not include tool name", msg)
		}
	}()
	agent.FuncTool("myTool", "desc", nil, func(_ context.Context, _ map[string]any) (string, error) {
		return "", nil
	})
}

func TestFuncTool_WrongTypePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for wrong schema type, got none")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, `"array"`) {
			t.Errorf("panic message %q does not include actual type value", msg)
		}
		if !strings.Contains(msg, "myTool") {
			t.Errorf("panic message %q does not include tool name", msg)
		}
	}()
	agent.FuncTool("myTool", "desc", map[string]any{"type": "array"}, func(_ context.Context, _ map[string]any) (string, error) {
		return "", nil
	})
}

func TestFuncTool_ValidObjectSchemaPasses(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "string"},
		},
	}
	tool := agent.FuncTool("myTool", "desc", schema, func(_ context.Context, _ map[string]any) (string, error) {
		return "ok", nil
	})
	if tool.Name() != "myTool" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "myTool")
	}
}
