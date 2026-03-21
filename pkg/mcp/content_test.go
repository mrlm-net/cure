package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTextContent_contentType(t *testing.T) {
	tc := TextContent{Type: "text", Text: "hello"}
	if tc.contentType() != "text" {
		t.Errorf("contentType() = %q, want %q", tc.contentType(), "text")
	}
}

func TestImageContent_contentType(t *testing.T) {
	ic := ImageContent{Type: "image", Data: "abc", MIMEType: "image/png"}
	if ic.contentType() != "image" {
		t.Errorf("contentType() = %q, want %q", ic.contentType(), "image")
	}
}

func TestResourceContent_contentType(t *testing.T) {
	rc := ResourceContent{Type: "resource", URI: "file:///foo"}
	if rc.contentType() != "resource" {
		t.Errorf("contentType() = %q, want %q", rc.contentType(), "resource")
	}
}

func TestText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantTxt string
	}{
		{"empty string", "", 1, ""},
		{"simple text", "hello world", 1, "hello world"},
		{"unicode", "こんにちは", 1, "こんにちは"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Text(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("len(Text(%q)) = %d, want %d", tt.input, len(got), tt.wantLen)
			}
			tc, ok := got[0].(TextContent)
			if !ok {
				t.Fatalf("Text()[0] is %T, want TextContent", got[0])
			}
			if tc.Text != tt.wantTxt {
				t.Errorf("TextContent.Text = %q, want %q", tc.Text, tt.wantTxt)
			}
			if tc.Type != "text" {
				t.Errorf("TextContent.Type = %q, want %q", tc.Type, "text")
			}
		})
	}
}

func TestTextf(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		got := Textf("hello")
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].(TextContent).Text != "hello" {
			t.Errorf("Text = %q, want %q", got[0].(TextContent).Text, "hello")
		}
	})
	t.Run("one arg", func(t *testing.T) {
		got := Textf("hello %s", "world")
		if got[0].(TextContent).Text != "hello world" {
			t.Errorf("Text = %q, want %q", got[0].(TextContent).Text, "hello world")
		}
	})
	t.Run("multiple args", func(t *testing.T) {
		got := Textf("%d + %d = %d", 1, 2, 3)
		if got[0].(TextContent).Text != "1 + 2 = 3" {
			t.Errorf("Text = %q", got[0].(TextContent).Text)
		}
	})
	t.Run("float format", func(t *testing.T) {
		got := Textf("%.2f", 3.14159)
		if got[0].(TextContent).Text != "3.14" {
			t.Errorf("Text = %q, want %q", got[0].(TextContent).Text, "3.14")
		}
	})
	t.Run("returns TextContent type", func(t *testing.T) {
		got := Textf("test")
		tc, ok := got[0].(TextContent)
		if !ok {
			t.Fatalf("got[0] is %T, want TextContent", got[0])
		}
		if tc.Type != "text" {
			t.Errorf("Type = %q, want %q", tc.Type, "text")
		}
	})
}

// TestTextContent_JSON verifies that TextContent marshals with the correct type discriminator.
func TestTextContent_JSON(t *testing.T) {
	tc := TextContent{Type: "text", Text: "hello"}
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if m["type"] != "text" {
		t.Errorf(`"type" = %v, want "text"`, m["type"])
	}
	if m["text"] != "hello" {
		t.Errorf(`"text" = %v, want "hello"`, m["text"])
	}
}

// TestImageContent_JSON verifies that ImageContent marshals correctly.
func TestImageContent_JSON(t *testing.T) {
	ic := ImageContent{Type: "image", Data: "base64data", MIMEType: "image/png"}
	data, err := json.Marshal(ic)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if m["type"] != "image" {
		t.Errorf(`"type" = %v, want "image"`, m["type"])
	}
	if m["mimeType"] != "image/png" {
		t.Errorf(`"mimeType" = %v, want "image/png"`, m["mimeType"])
	}
}

// TestResourceContent_JSON verifies ResourceContent omits empty optional fields.
func TestResourceContent_JSON(t *testing.T) {
	rc := ResourceContent{Type: "resource", URI: "file:///foo", Text: "content"}
	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if m["type"] != "resource" {
		t.Errorf(`"type" = %v, want "resource"`, m["type"])
	}
	if _, ok := m["blob"]; ok {
		t.Error(`"blob" must be omitted when empty`)
	}
}

// TestContent_Sealed verifies that Content is a sealed interface — only package
// types can satisfy it. We test by confirming the known concrete types all pass
// through correctly.
func TestContent_Sealed(t *testing.T) {
	var _ Content = TextContent{}
	var _ Content = ImageContent{}
	var _ Content = ResourceContent{}
}

func TestTextf_equivalentToSprintf(t *testing.T) {
	want := fmt.Sprintf("result: %d items (%.1f%%)", 42, 95.5)
	got := Textf("result: %d items (%.1f%%)", 42, 95.5)
	if len(got) != 1 {
		t.Fatalf("expected 1 content, got %d", len(got))
	}
	tc := got[0].(TextContent)
	if tc.Text != want {
		t.Errorf("Textf text = %q, want %q", tc.Text, want)
	}
}
