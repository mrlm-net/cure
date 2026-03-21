package mcp

import (
	"errors"
	"testing"
)

func TestErrSentinels(t *testing.T) {
	t.Run("ErrToolNotFound", func(t *testing.T) {
		if ErrToolNotFound == nil {
			t.Fatal("ErrToolNotFound must not be nil")
		}
		if ErrToolNotFound.Error() == "" {
			t.Error("ErrToolNotFound.Error() must not be empty")
		}
	})
	t.Run("ErrResourceNotFound", func(t *testing.T) {
		if ErrResourceNotFound == nil {
			t.Fatal("ErrResourceNotFound must not be nil")
		}
	})
	t.Run("ErrPromptNotFound", func(t *testing.T) {
		if ErrPromptNotFound == nil {
			t.Fatal("ErrPromptNotFound must not be nil")
		}
	})
}

func TestToolCallError(t *testing.T) {
	inner := errors.New("something failed")

	tests := []struct {
		name    string
		tool    string
		err     error
		wantMsg string
	}{
		{
			name:    "basic error message",
			tool:    "my-tool",
			err:     inner,
			wantMsg: `mcp: tool "my-tool" call failed: something failed`,
		},
		{
			name:    "empty tool name",
			tool:    "",
			err:     inner,
			wantMsg: `mcp: tool "" call failed: something failed`,
		},
		{
			name:    "wrapped error",
			tool:    "calc",
			err:     errors.New("divide by zero"),
			wantMsg: `mcp: tool "calc" call failed: divide by zero`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tce := &ToolCallError{Tool: tt.tool, Err: tt.err}

			if got := tce.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}

			if !errors.Is(tce, tt.err) {
				t.Error("errors.Is(ToolCallError, inner) must be true via Unwrap")
			}

			unwrapped := tce.Unwrap()
			if unwrapped != tt.err {
				t.Errorf("Unwrap() = %v, want %v", unwrapped, tt.err)
			}
		})
	}
}

func TestToolCallError_ErrorsAs(t *testing.T) {
	inner := errors.New("inner error")
	tce := &ToolCallError{Tool: "mytool", Err: inner}

	var target *ToolCallError
	if !errors.As(tce, &target) {
		t.Fatal("errors.As must find *ToolCallError")
	}
	if target.Tool != "mytool" {
		t.Errorf("Tool = %q, want %q", target.Tool, "mytool")
	}
}
