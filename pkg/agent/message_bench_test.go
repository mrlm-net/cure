package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

// build10BlockMessageContent returns a MessageContent with 10 mixed blocks:
// 5 TextBlocks, 3 ToolUseBlocks, and 2 ToolResultBlocks.
func build10BlockMessageContent() agent.MessageContent {
	return agent.MessageContent{
		agent.TextBlock{Text: "block one text content"},
		agent.ToolUseBlock{ID: "tu_1", Name: "search", Input: map[string]any{"q": "golang"}},
		agent.TextBlock{Text: "block three text content"},
		agent.ToolResultBlock{ID: "tu_1", ToolName: "search", Result: "10 results", IsError: false},
		agent.TextBlock{Text: "block five text content"},
		agent.ToolUseBlock{ID: "tu_2", Name: "calculate", Input: map[string]any{"expr": "1+1"}},
		agent.TextBlock{Text: "block seven text content"},
		agent.ToolResultBlock{ID: "tu_2", ToolName: "calculate", Result: "2", IsError: false},
		agent.ToolUseBlock{ID: "tu_3", Name: "fetch", Input: map[string]any{"url": "https://example.com"}},
		agent.TextBlock{Text: "block ten text content"},
	}
}

// BenchmarkMessageContent_Marshal_10Blocks benchmarks marshaling a 10-block
// MessageContent to JSON. This exercises the multi-block typed array path.
func BenchmarkMessageContent_Marshal_10Blocks(b *testing.B) {
	mc := build10BlockMessageContent()
	// Measure the serialised size so go test -bench reports MB/s throughput.
	data, err := json.Marshal(mc)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(mc)
	}
}

// BenchmarkMessageContent_Unmarshal_10Blocks benchmarks unmarshaling a 10-block
// typed JSON array into MessageContent.
func BenchmarkMessageContent_Unmarshal_10Blocks(b *testing.B) {
	mc := build10BlockMessageContent()
	data, err := json.Marshal(mc)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var got agent.MessageContent
		_ = json.Unmarshal(data, &got)
	}
}

// BenchmarkMessageContent_RoundTrip_10Blocks benchmarks a full marshal/unmarshal
// round-trip of a 10-block MessageContent.
func BenchmarkMessageContent_RoundTrip_10Blocks(b *testing.B) {
	mc := build10BlockMessageContent()
	// Measure the serialised size so go test -bench reports MB/s throughput.
	setupData, err := json.Marshal(mc)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}
	b.SetBytes(int64(len(setupData)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(mc)
		var got agent.MessageContent
		_ = json.Unmarshal(data, &got)
	}
}
