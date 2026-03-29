// Package openai provides an OpenAI Chat Completions provider adapter for pkg/agent.
// Import this package with a blank import to register the "openai" provider:
//
//	import _ "github.com/mrlm-net/cure/internal/agent/openai"
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"os"
	"strings"

	"github.com/mrlm-net/cure/internal/agent/sseutil"
	"github.com/mrlm-net/cure/pkg/agent"
)

const (
	defaultModel     = "gpt-4o"
	defaultMaxTokens = 4096
	defaultKeyEnv    = "OPENAI_API_KEY"
	defaultBaseURL   = "https://api.openai.com/v1"
)

func init() {
	agent.Register("openai", NewOpenAIAgent)
}

// openaiAdapter implements agent.Agent for the OpenAI Chat Completions API.
type openaiAdapter struct {
	apiKey    string // held only for sanitiseError — never emitted in events
	baseURL   string
	model     string
	maxTokens int
	httpClient *http.Client
}

// NewOpenAIAgent is the AgentFactory for the "openai" provider.
// cfg keys: "api_key_env" (default "OPENAI_API_KEY"), "model" (default "gpt-4o"),
// "max_tokens" (default 4096).
// Returns an error if the API key environment variable is not set.
func NewOpenAIAgent(cfg map[string]any) (agent.Agent, error) {
	keyEnv := defaultKeyEnv
	if v, ok := cfg["api_key_env"].(string); ok && v != "" {
		keyEnv = v
	}
	apiKey := os.Getenv(keyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("openai: API key not set — populate the %s environment variable", keyEnv)
	}

	model := defaultModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}

	maxTokens := defaultMaxTokens
	switch v := cfg["max_tokens"].(type) {
	case int:
		maxTokens = v
	case int64:
		maxTokens = int(v)
	case float64:
		maxTokens = int(v)
	}

	// Allow tests to override the base URL via OPENAI_BASE_URL env variable.
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &openaiAdapter{
		apiKey:    apiKey,
		baseURL:   baseURL,
		model:     model,
		maxTokens: maxTokens,
		httpClient: &http.Client{},
	}, nil
}

// Provider returns the provider name "openai".
func (a *openaiAdapter) Provider() string { return "openai" }

// CountTokens returns ErrCountNotSupported — the OpenAI Chat Completions API
// does not expose a dedicated token counting endpoint.
func (a *openaiAdapter) CountTokens(_ context.Context, _ *agent.Session) (int, error) {
	return 0, agent.ErrCountNotSupported
}

// Run streams a response for the given session as an iter.Seq2[Event, error].
// It calls the OpenAI Chat Completions endpoint with stream=true and parses
// the SSE response using sseutil.Parse.
func (a *openaiAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		if err := a.stream(ctx, sess, yield); err != nil {
			yield(agent.Event{Kind: agent.EventKindError, Err: err.Error()}, err)
		}
	}
}

// chatRequest is the JSON body sent to the OpenAI Chat Completions API.
type chatRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
	Messages  []chatMessage `json:"messages"`
}

// chatMessage is a single message in the OpenAI messages array.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// streamDelta is a subset of the SSE delta event JSON for extracting token text.
type streamDelta struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// stream performs the HTTP request and drives the SSE event loop,
// yielding events to the caller. Returns a non-nil error on failure.
func (a *openaiAdapter) stream(ctx context.Context, sess *agent.Session, yield func(agent.Event, error) bool) error {
	msgs := buildMessages(sess)

	reqBody := chatRequest{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		Stream:    true,
		Messages:  msgs,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return sanitiseError(fmt.Errorf("openai: marshal request: %w", err), a.apiKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return sanitiseError(fmt.Errorf("openai: create request: %w", err), a.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return sanitiseError(fmt.Errorf("openai: do request: %w", err), a.apiKey)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		_, _ = errBody.ReadFrom(resp.Body)
		return sanitiseError(fmt.Errorf("openai: unexpected status %d: %s", resp.StatusCode, errBody.String()), a.apiKey)
	}

	// Emit start event.
	if !yield(agent.Event{Kind: agent.EventKindStart}, nil) {
		return nil
	}

	// Parse the SSE stream.
	parseErr := sseutil.Parse(ctx, resp.Body, func(data []byte) bool {
		var delta streamDelta
		if err := json.Unmarshal(data, &delta); err != nil {
			// Skip malformed lines.
			return true
		}
		for _, choice := range delta.Choices {
			text := choice.Delta.Content
			if text == "" {
				continue
			}
			if !yield(agent.Event{Kind: agent.EventKindToken, Text: text}, nil) {
				return false
			}
		}
		return true
	})

	if parseErr != nil {
		if ctx.Err() != nil {
			return nil
		}
		return sanitiseError(fmt.Errorf("openai: stream parse: %w", parseErr), a.apiKey)
	}

	// Emit done event.
	yield(agent.Event{Kind: agent.EventKindDone}, nil)
	return nil
}

// buildMessages converts a session's history into OpenAI chat messages.
// The system prompt is prepended as a system role message when present.
func buildMessages(sess *agent.Session) []chatMessage {
	msgs := make([]chatMessage, 0, len(sess.History)+1)
	if sess.SystemPrompt != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: sess.SystemPrompt})
	}
	for _, m := range sess.History {
		role := mapRole(m.Role)
		// NOTE: m.Content will become MessageContent after PR #116-120 merges.
		// At that point, replace with: agent.TextOf(m.Content). See issue #123.
		msgs = append(msgs, chatMessage{Role: role, Content: m.Content})
	}
	return msgs
}

// mapRole translates agent.Role values to OpenAI role strings.
func mapRole(r agent.Role) string {
	switch r {
	case agent.RoleUser:
		return "user"
	case agent.RoleAssistant:
		return "assistant"
	case agent.RoleSystem:
		return "system"
	default:
		return string(r)
	}
}

// sanitiseError replaces the API key value with "***" in error strings to
// prevent accidental key exposure in logs and error messages.
func sanitiseError(err error, apiKey string) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	if apiKey != "" {
		s = strings.ReplaceAll(s, apiKey, "***")
	}
	return fmt.Errorf("%s", s) //nolint:goerr113 // intentional: error string sanitisation
}
