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
			schema:   schema,
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
			schema:   schema,
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

func TestFuncTool_Panics(t *testing.T) {
	t.Run("nil schema panics", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for nil schema, got none")
			}
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("expected string panic value, got %T: %v", r, r)
			}
			if !strings.Contains(msg, "nil-schema-tool") {
				t.Errorf("panic message %q does not contain tool name %q", msg, "nil-schema-tool")
			}
			if !strings.Contains(msg, "nil") {
				t.Errorf("panic message %q does not contain %q", msg, "nil")
			}
		}()
		agent.FuncTool(
			"nil-schema-tool", "desc",
			nil,
			func(_ context.Context, _ map[string]any) (string, error) { return "", nil },
		)
	})

	t.Run("wrong type panics", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for wrong schema type, got none")
			}
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("expected string panic value, got %T: %v", r, r)
			}
			if !strings.Contains(msg, "wrong-type-tool") {
				t.Errorf("panic message %q does not contain tool name %q", msg, "wrong-type-tool")
			}
			if !strings.Contains(msg, "string") {
				t.Errorf("panic message %q does not contain %q", msg, "string")
			}
		}()
		agent.FuncTool(
			"wrong-type-tool", "desc",
			map[string]any{"type": "string"},
			func(_ context.Context, _ map[string]any) (string, error) { return "", nil },
		)
	})

	t.Run("valid schema accepted", func(t *testing.T) {
		var result agent.Tool
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("unexpected panic for valid schema: %v", r)
				}
			}()
			result = agent.FuncTool(
				"valid-tool", "desc",
				map[string]any{"type": "object", "properties": map[string]any{}},
				func(_ context.Context, _ map[string]any) (string, error) { return "", nil },
			)
		}()
		if result == nil {
			t.Error("FuncTool returned nil for valid schema")
		}
	})
}

func BenchmarkFuncTool_Call(b *testing.B) {
	benchSchema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"x": map[string]any{"type": "string"}},
	}
	tool := agent.FuncTool(
		"bench",
		"Benchmark tool",
		benchSchema,
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
