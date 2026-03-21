package mcp

import "fmt"

// Content is the sealed interface for tool result and prompt message content.
// Only types defined in this package satisfy Content — the unexported contentType
// method prevents external implementations.
type Content interface {
	contentType() string
}

// TextContent represents plain-text content returned by a tool or included in a
// prompt message. The Type field is always "text" and is set automatically by [Text]
// and [Textf].
type TextContent struct {
	Type string `json:"type"` // always "text"
	Text string `json:"text"`
}

// contentType implements Content. Returns "text".
func (t TextContent) contentType() string { return "text" }

// ImageContent represents base64-encoded image content. The Type field is always
// "image".
type ImageContent struct {
	Type     string `json:"type"`     // always "image"
	Data     string `json:"data"`     // base64-encoded image bytes
	MIMEType string `json:"mimeType"` // e.g. "image/png"
}

// contentType implements Content. Returns "image".
func (i ImageContent) contentType() string { return "image" }

// ResourceContent represents an embedded resource reference in tool results or
// prompt messages. The Type field is always "resource".
type ResourceContent struct {
	Type     string `json:"type"`               // always "resource"
	URI      string `json:"uri"`                // resource URI
	MIMEType string `json:"mimeType,omitempty"` // optional MIME type
	Text     string `json:"text,omitempty"`     // optional text representation
	Blob     string `json:"blob,omitempty"`     // optional base64-encoded binary
}

// contentType implements Content. Returns "resource".
func (r ResourceContent) contentType() string { return "resource" }

// Text creates a single-element []Content slice containing a TextContent with
// the given string. Use this as a convenience when returning a simple text result
// from a tool handler.
func Text(s string) []Content {
	return []Content{TextContent{Type: "text", Text: s}}
}

// Textf creates a single-element []Content slice containing a TextContent with a
// formatted string. The format string and arguments follow fmt.Sprintf conventions.
func Textf(format string, args ...any) []Content {
	return []Content{TextContent{Type: "text", Text: fmt.Sprintf(format, args...)}}
}
