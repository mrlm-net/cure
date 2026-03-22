package claude

import (
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/mrlm-net/cure/pkg/agent"
)

// NewAdapterForTest creates a claudeAdapter with a pre-built client for testing.
// This bypasses the environment variable lookup used by NewClaudeAgent.
// It is only compiled into test binaries.
func NewAdapterForTest(client *anthropic.Client, model string, maxTokens int64, apiKey string) agent.Agent {
	return newTestAdapter(client, model, maxTokens, apiKey)
}
