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

	// maxToolTurns is the hard cap on tool-call iterations within a single Run.
	// If the model keeps requesting tools after this many turns, executeToolLoop
	// returns an error rather than looping indefinitely.
	maxToolTurns = 32
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
// Handles all ContentBlock types:
//   - TextBlock       → anthropic.TextBlockParam (user or assistant role)
//   - ToolUseBlock    → anthropic.ToolUseBlockParam (assistant role only)
//   - ToolResultBlock → anthropic.ToolResultBlockParam (user role only)
//
// When sess.Tools is non-empty the tools are attached to the params so that
// the model can invoke them.
func (a *claudeAdapter) buildParams(sess *agent.Session) anthropic.MessageNewParams {
	msgs := make([]anthropic.MessageParam, 0, len(sess.History))
	for _, m := range sess.History {
		switch m.Role {
		case agent.RoleUser:
			blocks := contentBlocksToUserParam(m.Content)
			if len(blocks) > 0 {
				msgs = append(msgs, anthropic.NewUserMessage(blocks...))
			}
		case agent.RoleAssistant:
			blocks := contentBlocksToAssistantParam(m.Content)
			if len(blocks) > 0 {
				msgs = append(msgs, anthropic.NewAssistantMessage(blocks...))
			}
		// RoleSystem is handled via sess.SystemPrompt below; skip from history.
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

	// Attach tools when the session has any registered.
	if len(sess.Tools) > 0 {
		toolUnions := make([]anthropic.ToolUnionParam, 0, len(sess.Tools))
		for _, t := range sess.Tools {
			tp := anthropic.ToolParam{
				Name:        t.Name(),
				Description: anthropic.String(t.Description()),
				InputSchema: schemaToToolInputParam(t.Schema()),
			}
			toolUnions = append(toolUnions, anthropic.ToolUnionParam{OfTool: &tp})
		}
		params.Tools = toolUnions
	}

	return params
}

// schemaToToolInputParam converts an agent.Tool JSON Schema map into the
// Anthropic SDK's ToolInputSchemaParam.
//
// agent.Tool.Schema() returns a full JSON Schema:
//
//	{"type":"object","properties":{...},"required":[...]}
//
// ToolInputSchemaParam mirrors the API wire shape — Properties holds the inner
// properties sub-object and Required holds the required array. Passing the full
// schema to Properties would nest it one level too deep and silently drop
// Required from the serialised request.
func schemaToToolInputParam(schema map[string]any) anthropic.ToolInputSchemaParam {
	param := anthropic.ToolInputSchemaParam{}
	if props, ok := schema["properties"]; ok {
		param.Properties = props
	}
	if reqRaw, ok := schema["required"]; ok {
		switch v := reqRaw.(type) {
		case []string:
			param.Required = v
		case []any:
			strs := make([]string, 0, len(v))
			for _, s := range v {
				if str, ok := s.(string); ok {
					strs = append(strs, str)
				}
			}
			param.Required = strs
		}
	}
	return param
}

// contentBlocksToUserParam converts a MessageContent into Anthropic user-role
// ContentBlockParamUnion values. ToolResultBlock is the primary non-text type
// expected in user messages during tool loops.
func contentBlocksToUserParam(mc agent.MessageContent) []anthropic.ContentBlockParamUnion {
	out := make([]anthropic.ContentBlockParamUnion, 0, len(mc))
	for _, b := range mc {
		switch v := b.(type) {
		case agent.TextBlock:
			out = append(out, anthropic.NewTextBlock(v.Text))
		case agent.ToolResultBlock:
			out = append(out, anthropic.NewToolResultBlock(v.ID, v.Result, v.IsError))
		}
		// ToolUseBlock must not appear in user messages; skip silently.
	}
	return out
}

// contentBlocksToAssistantParam converts a MessageContent into Anthropic
// assistant-role ContentBlockParamUnion values. ToolUseBlock is the primary
// non-text type expected in assistant messages during tool loops.
func contentBlocksToAssistantParam(mc agent.MessageContent) []anthropic.ContentBlockParamUnion {
	out := make([]anthropic.ContentBlockParamUnion, 0, len(mc))
	for _, b := range mc {
		switch v := b.(type) {
		case agent.TextBlock:
			out = append(out, anthropic.NewTextBlock(v.Text))
		case agent.ToolUseBlock:
			out = append(out, anthropic.NewToolUseBlock(v.ID, v.Input, v.Name))
		}
		// ToolResultBlock must not appear in assistant messages; skip silently.
	}
	return out
}

// sanitiseError replaces the API key value with "[REDACTED]" in error strings.
func (a *claudeAdapter) sanitiseError(err error) string {
	s := err.Error()
	if a.apiKey != "" {
		s = strings.ReplaceAll(s, a.apiKey, "[REDACTED]")
	}
	return s
}
