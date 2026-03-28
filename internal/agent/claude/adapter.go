// Package claude provides a Claude AI provider adapter for pkg/agent.
// Import this package with a blank import to register the "claude" provider:
//
//	import _ "github.com/mrlm-net/cure/internal/agent/claude"
package claude

import (
	"fmt"
	"os"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/mrlm-net/cure/pkg/agent"
)

const (
	defaultModel     = "claude-opus-4-6"
	defaultMaxTokens = int64(8192)
	defaultKeyEnv    = "ANTHROPIC_API_KEY"
)

func init() {
	agent.Register("claude", NewClaudeAgent)
}

// claudeAdapter implements agent.Agent for the Anthropic Claude API.
type claudeAdapter struct {
	client    *anthropic.Client
	model     string
	maxTokens int64
	apiKey    string // held only for sanitiseError — never emitted in events
}

// NewClaudeAgent is the AgentFactory for the "claude" provider.
// cfg keys: "api_key_env" (default "ANTHROPIC_API_KEY"), "model" (default "claude-opus-4-6"),
// "max_tokens" (default 8192).
func NewClaudeAgent(cfg map[string]any) (agent.Agent, error) {
	keyEnv := defaultKeyEnv
	if v, ok := cfg["api_key_env"].(string); ok && v != "" {
		keyEnv = v
	}
	apiKey := os.Getenv(keyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("claude: API key not set — populate the %s environment variable", keyEnv)
	}

	model := defaultModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}

	maxTokens := defaultMaxTokens
	switch v := cfg["max_tokens"].(type) {
	case int:
		maxTokens = int64(v)
	case int64:
		maxTokens = v
	case float64:
		maxTokens = int64(v)
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &claudeAdapter{
		client:    &client,
		model:     model,
		maxTokens: maxTokens,
		apiKey:    apiKey,
	}, nil
}

// newTestAdapter creates a claudeAdapter with a custom client, used in tests.
func newTestAdapter(client *anthropic.Client, model string, maxTokens int64, apiKey string) *claudeAdapter {
	return &claudeAdapter{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
		apiKey:    apiKey,
	}
}

// Provider returns the provider name "claude".
func (a *claudeAdapter) Provider() string { return "claude" }

// buildParams constructs SDK message params from a session.
// RoleSystem messages are moved to the System field; they must not appear
// in the messages array (the Anthropic API rejects system role in messages).
//
// NOTE: In v0.10.x PR C (#123), this method will be updated to handle
// ToolUseBlock and ToolResultBlock natively. For now it extracts plain text
// from MessageContent so the project compiles after the Message.Content type
// change from string to MessageContent.
func (a *claudeAdapter) buildParams(sess *agent.Session) anthropic.MessageNewParams {
	msgs := make([]anthropic.MessageParam, 0, len(sess.History))
	for _, m := range sess.History {
		switch m.Role {
		case agent.RoleUser:
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(agent.TextOf(m.Content))))
		case agent.RoleAssistant:
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(agent.TextOf(m.Content))))
		// RoleSystem is handled via sess.SystemPrompt below; skip from history
		}
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: a.maxTokens,
		Messages:  msgs,
	}
	if sess.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: sess.SystemPrompt},
		}
	}
	return params
}

// sanitiseError replaces the API key value with "[REDACTED]" in error strings.
func (a *claudeAdapter) sanitiseError(err error) string {
	s := err.Error()
	if a.apiKey != "" {
		s = strings.ReplaceAll(s, a.apiKey, "[REDACTED]")
	}
	return s
}
