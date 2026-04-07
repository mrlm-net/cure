package agent_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

// TestSession_ToolsTransient verifies that the Tools field is marked json:"-"
// and is therefore excluded from JSON marshaling. This is a critical correctness
// property: tool implementations (closures, stateful objects) must never be
// written to the session file on disk.
func TestSession_ToolsTransient(t *testing.T) {
	minSchema := map[string]any{"type": "object", "properties": map[string]any{}}
	sess := agent.NewSession("claude", "claude-opus-4-6")
	sess.Tools = []agent.Tool{
		agent.FuncTool("my-tool", "desc", minSchema,
			func(_ context.Context, _ map[string]any) (string, error) { return "ok", nil },
		),
	}

	b, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, present := m["tools"]; present {
		t.Error("Session.Tools must not appear in JSON output (json:\"-\")")
	}
}

// TestSession_ToolsTransient_UnmarshalRestoresNil verifies that round-tripping a
// session through JSON leaves Tools as nil (since it is json:"-"). The caller is
// responsible for re-attaching tools after loading a persisted session.
func TestSession_ToolsTransient_UnmarshalRestoresNil(t *testing.T) {
	minSchema := map[string]any{"type": "object", "properties": map[string]any{}}
	orig := agent.NewSession("claude", "claude-opus-4-6")
	orig.Tools = []agent.Tool{
		agent.FuncTool("t", "d", minSchema,
			func(_ context.Context, _ map[string]any) (string, error) { return "", nil },
		),
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var restored agent.Session
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if restored.Tools != nil {
		t.Errorf("Tools after JSON round-trip = %v, want nil", restored.Tools)
	}
}

// TestSessionFork_ToolsSliceHeaderIndependent verifies that Fork gives the fork
// an independent slice header: truncating the fork's slice does not affect the
// original's length. Note that slice elements are shared (shallow copy) — mutating
// a Tool at a given index in the fork would affect the original at the same index.
func TestSessionFork_ToolsSliceHeaderIndependent(t *testing.T) {
	minSchema := map[string]any{"type": "object", "properties": map[string]any{}}
	orig := agent.NewSession("p", "m")
	orig.Tools = []agent.Tool{
		agent.FuncTool("t1", "tool 1", minSchema,
			func(_ context.Context, _ map[string]any) (string, error) { return "1", nil },
		),
		agent.FuncTool("t2", "tool 2", minSchema,
			func(_ context.Context, _ map[string]any) (string, error) { return "2", nil },
		),
	}

	fork := orig.Fork()

	if len(fork.Tools) != len(orig.Tools) {
		t.Fatalf("fork.Tools len = %d, want %d", len(fork.Tools), len(orig.Tools))
	}

	// Truncate the fork's slice header. The original must be unaffected because
	// the fork has an independent slice header (independent len/cap, same backing array).
	fork.Tools = fork.Tools[:1]
	if len(orig.Tools) != 2 {
		t.Errorf("orig.Tools len = %d after truncating fork, want 2", len(orig.Tools))
	}
}
