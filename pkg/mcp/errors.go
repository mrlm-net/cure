package mcp

import (
	"errors"
	"fmt"
)

// ErrToolNotFound is returned when a requested tool name is not registered.
var ErrToolNotFound = errors.New("mcp: tool not found")

// ErrResourceNotFound is returned when a requested resource URI is not registered.
var ErrResourceNotFound = errors.New("mcp: resource not found")

// ErrPromptNotFound is returned when a requested prompt name is not registered.
var ErrPromptNotFound = errors.New("mcp: prompt not found")

// ToolCallError wraps an error returned by a Tool.Call handler. It preserves
// the tool name for structured error handling and supports errors.Is/As via Unwrap.
type ToolCallError struct {
	// Tool is the name of the tool whose handler returned the error.
	Tool string

	// Err is the underlying error from the tool handler.
	Err error
}

// Error returns a human-readable message including the tool name.
func (e *ToolCallError) Error() string {
	return fmt.Sprintf("mcp: tool %q call failed: %v", e.Tool, e.Err)
}

// Unwrap returns the underlying error, enabling errors.Is and errors.As traversal.
func (e *ToolCallError) Unwrap() error {
	return e.Err
}
