package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ContentBlock is a sealed interface for message content blocks.
// The sealed isContentBlock method ensures only types in this package
// can satisfy the interface.
type ContentBlock interface{ isContentBlock() }

// TextBlock is a plain text content block.
type TextBlock struct{ Text string }

func (TextBlock) isContentBlock() {}

// ToolUseBlock represents a tool invocation requested by the model.
type ToolUseBlock struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

func (ToolUseBlock) isContentBlock() {}

// ToolResultBlock carries the result of a tool invocation back to the model.
type ToolResultBlock struct {
	ID       string `json:"id"`
	ToolName string `json:"tool_name"`
	Result   string `json:"result"`
	IsError  bool   `json:"is_error,omitempty"`
}

func (ToolResultBlock) isContentBlock() {}

// MessageContent holds the content of a message as a sequence of [ContentBlock]s.
//
// JSON codec compatibility:
//   - A single [TextBlock] marshals as a plain JSON string for backward compatibility
//     with pre-v0.10.x session files that stored content as a plain string.
//   - All other content (multi-block or non-text) marshals as a typed JSON array
//     where each element carries a "type" discriminator field.
//
// During unmarshaling, a plain JSON string is decoded back into a single [TextBlock],
// preserving round-trip compatibility with pre-v0.10.x sessions.
type MessageContent []ContentBlock

// MarshalJSON implements json.Marshaler.
// A single TextBlock marshals as a plain JSON string; all other content marshals
// as a typed JSON array with a "type" discriminator on each element.
func (mc MessageContent) MarshalJSON() ([]byte, error) {
	// Single TextBlock → plain string (backward compat with pre-v0.10.x sessions)
	if len(mc) == 1 {
		if tb, ok := mc[0].(TextBlock); ok {
			return json.Marshal(tb.Text)
		}
	}

	// Multiple blocks or non-text → typed array
	arr := make([]json.RawMessage, 0, len(mc))
	for _, b := range mc {
		var (
			raw json.RawMessage
			err error
		)
		switch v := b.(type) {
		case TextBlock:
			raw, err = json.Marshal(struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{"text", v.Text})
		case ToolUseBlock:
			raw, err = json.Marshal(struct {
				Type  string         `json:"type"`
				ID    string         `json:"id"`
				Name  string         `json:"name"`
				Input map[string]any `json:"input"`
			}{"tool_use", v.ID, v.Name, v.Input})
		case ToolResultBlock:
			raw, err = json.Marshal(struct {
				Type     string `json:"type"`
				ID       string `json:"id"`
				ToolName string `json:"tool_name"`
				Result   string `json:"result"`
				IsError  bool   `json:"is_error,omitempty"`
			}{"tool_result", v.ID, v.ToolName, v.Result, v.IsError})
		default:
			err = fmt.Errorf("agent: unknown ContentBlock type %T", b)
		}
		if err != nil {
			return nil, err
		}
		arr = append(arr, raw)
	}
	return json.Marshal(arr)
}

// UnmarshalJSON implements json.Unmarshaler.
// Accepts either a plain JSON string (pre-v0.10.x backward compat) or a typed
// JSON array produced by MarshalJSON.
func (mc *MessageContent) UnmarshalJSON(data []byte) error {
	// Try plain string first (backward compat with pre-v0.10.x sessions)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*mc = MessageContent{TextBlock{Text: s}}
		return nil
	}

	// Try typed array
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("agent: cannot unmarshal MessageContent: %w", err)
	}

	*mc = make(MessageContent, 0, len(arr))
	for _, raw := range arr {
		var t struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &t); err != nil {
			return err
		}
		switch t.Type {
		case "text":
			var b struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(raw, &b); err != nil {
				return err
			}
			*mc = append(*mc, TextBlock{b.Text})
		case "tool_use":
			var b ToolUseBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return err
			}
			*mc = append(*mc, b)
		case "tool_result":
			var b ToolResultBlock
			if err := json.Unmarshal(raw, &b); err != nil {
				return err
			}
			*mc = append(*mc, b)
		default:
			return fmt.Errorf("agent: unknown content block type %q", t.Type)
		}
	}
	return nil
}

// TextOf extracts all text from a MessageContent by joining any TextBlock text values.
// Used by provider adapters that need a plain string representation of the content
// (e.g., for providers that do not yet handle multi-block content natively).
// In v0.10.x PR C (#123), adapters will handle ToolUseBlock/ToolResultBlock directly.
func TextOf(mc MessageContent) string {
	if len(mc) == 1 {
		if tb, ok := mc[0].(TextBlock); ok {
			return tb.Text
		}
	}
	var parts []string
	for _, b := range mc {
		if tb, ok := b.(TextBlock); ok {
			parts = append(parts, tb.Text)
		}
	}
	return strings.Join(parts, "")
}
