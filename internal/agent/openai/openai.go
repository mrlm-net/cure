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
	"io"
	"iter"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mrlm-net/cure/internal/agent/sseutil"
	"github.com/mrlm-net/cure/pkg/agent"
)

const (
	defaultModel     = "gpt-4o"
	defaultMaxTokens = 4096
	defaultKeyEnv    = "OPENAI_API_KEY"
	defaultBaseURL   = "https://api.openai.com/v1"

	// maxToolTurns is the hard cap on tool-call iterations within a single Run.
	// If the model keeps requesting tools after this many turns, executeToolLoop
	// returns an error rather than looping indefinitely.
	maxToolTurns = 32
)

func init() {
	agent.Register("openai", NewOpenAIAgent)
}

// openaiAdapter implements agent.Agent for the OpenAI Chat Completions API.
type openaiAdapter struct {
	apiKey     string // held only for sanitiseError — never emitted in events
	baseURL    string
	model      string
	maxTokens  int
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
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}, nil
}

// newTestAdapter creates an openaiAdapter with the given credentials and baseURL.
// It is used in tests to bypass the environment variable lookup.
func newTestAdapter(apiKey, baseURL, model string) *openaiAdapter {
	return &openaiAdapter{
		apiKey:    apiKey,
		baseURL:   baseURL,
		model:     model,
		maxTokens: defaultMaxTokens,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Provider returns the provider name "openai".
func (a *openaiAdapter) Provider() string { return "openai" }

// CountTokens returns ErrCountNotSupported — the OpenAI Chat Completions API
// does not expose a dedicated token counting endpoint.
func (a *openaiAdapter) CountTokens(_ context.Context, _ *agent.Session) (int, error) {
	return 0, agent.ErrCountNotSupported
}

// result is a (event, error) pair sent over the internal channel in Run.
type result struct {
	ev  agent.Event
	err error
}

// send delivers a result to ch, aborting if ctx is cancelled.
// Returns false if the send was cancelled (goroutine should stop).
func send(ctx context.Context, ch chan<- result, r result) bool {
	select {
	case ch <- r:
		return true
	case <-ctx.Done():
		return false
	}
}

// Run streams a response for the given session as an iter.Seq2[Event, error].
// When the session has Tools registered and the model requests tool calls,
// Run automatically executes the tool loop: it calls each requested tool,
// emits EventKindToolCall and EventKindToolResult events, appends the results
// to the session history, and re-invokes the model. The loop repeats until
// the model returns without requesting tools or until maxToolTurns is reached.
func (a *openaiAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		ch := make(chan result)
		go func() {
			defer close(ch)
			a.executeToolLoop(ctx, sess, ch)
		}()
		for r := range ch {
			if !yield(r.ev, r.err) {
				return
			}
		}
	}
}

// ---- JSON request/response types --------------------------------------------

// openaiFunction describes a function tool in the OpenAI request format.
type openaiFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

// openaiTool wraps a function tool for the OpenAI request format.
type openaiTool struct {
	Type     string         `json:"type"` // always "function"
	Function openaiFunction `json:"function"`
}

// chatRequest is the JSON body sent to the OpenAI Chat Completions API.
type chatRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
	Messages  []chatMessage `json:"messages"`
	Tools     []openaiTool  `json:"tools,omitempty"`
}

// chatToolCallFunction holds the function name and accumulated arguments JSON.
type chatToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// chatToolCall is a tool call entry in an assistant message or SSE delta.
type chatToolCall struct {
	Index    int                  `json:"index"`
	ID       string               `json:"id,omitempty"`
	Type     string               `json:"type,omitempty"`
	Function chatToolCallFunction `json:"function"`
}

// chatMessage is a single message in the OpenAI messages array.
type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// streamDelta is a subset of the SSE delta event JSON for extracting token text
// and tool call fragments.
type streamDelta struct {
	Choices []struct {
		Delta struct {
			Content   string         `json:"content"`
			ToolCalls []chatToolCall `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

// ---- Tool loop --------------------------------------------------------------

// executeToolLoop orchestrates the multi-turn tool loop.
//
// On each iteration it:
//  1. Calls streamInto to stream the model response and collect content blocks.
//  2. Inspects the returned blocks for ToolUseBlock values.
//  3. If tool-use blocks are present AND sess.Tools is non-empty:
//     a. Appends the assistant turn (with tool_use blocks) to session history.
//     b. Emits EventKindToolCall for each tool request.
//     c. Calls the matching tool.
//     d. Emits EventKindToolResult for each outcome.
//     e. Appends tool results to the session history.
//     f. Loops back to step 1 with the updated session.
//  4. If no tool-use blocks (or no tools registered) the assistant message is
//     appended to history and the loop exits normally.
//
// An error is sent (via result on ch) when maxToolTurns is exceeded.
func (a *openaiAdapter) executeToolLoop(ctx context.Context, sess *agent.Session, ch chan<- result) {
	// Build a lookup map for fast tool resolution once — sess.Tools is immutable
	// for the lifetime of a Run call.
	toolByName := make(map[string]agent.Tool, len(sess.Tools))
	for _, t := range sess.Tools {
		toolByName[t.Name()] = t
	}

	for turn := 0; turn < maxToolTurns; turn++ {
		// Stream one model response turn; collect all content blocks.
		blocks, ok := a.streamInto(ctx, sess, ch)
		if !ok {
			// Context cancelled or streaming error — streamInto already sent the
			// error event; stop the loop.
			return
		}

		// Separate tool-use requests from the rest of the content.
		toolUseBlocks := collectToolUseBlocks(blocks)

		// If no tools are registered or no tool-use blocks in the response,
		// this is a terminal text response. Append and exit.
		if len(toolUseBlocks) == 0 || len(sess.Tools) == 0 {
			sess.AppendAssistantBlocks(blocks)
			return
		}

		// Append the assistant turn (which includes tool_use blocks) so that
		// subsequent requests include the full conversation history.
		sess.AppendAssistantBlocks(blocks)

		// Execute each tool and collect results.
		for _, tub := range toolUseBlocks {
			// Marshal Input map to JSON string for the event payload.
			inputBytes, err := json.Marshal(tub.Input)
			if err != nil {
				msg := sanitiseError(
					fmt.Errorf("openai: failed to marshal tool input for %s: %v", tub.Name, err),
					a.apiKey,
				).Error()
				send(ctx, ch, result{
					ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
					err: fmt.Errorf("%s", msg), //nolint:goerr113
				})
				return
			}
			inputJSON := string(inputBytes)

			// Emit tool call event.
			if !send(ctx, ch, result{ev: agent.Event{
				Kind: agent.EventKindToolCall,
				ToolCall: &agent.ToolCallEvent{
					ID:        tub.ID,
					ToolName:  tub.Name,
					InputJSON: inputJSON,
				},
			}}) {
				return
			}

			tool, found := toolByName[tub.Name]
			var (
				toolResult string
				isError    bool
			)
			if !found {
				toolResult = fmt.Sprintf("tool not found: %s", tub.Name)
				isError = true
			} else {
				var callErr error
				toolResult, callErr = tool.Call(ctx, tub.Input)
				if callErr != nil {
					// Treat any context error (Canceled or DeadlineExceeded) as a
					// clean shutdown — don't emit an error event.
					if ctx.Err() != nil {
						return
					}
					toolResult = callErr.Error()
					isError = true
				}
			}

			// Emit tool result event.
			if !send(ctx, ch, result{ev: agent.Event{
				Kind: agent.EventKindToolResult,
				ToolResult: &agent.ToolResultEvent{
					ID:       tub.ID,
					ToolName: tub.Name,
					Result:   toolResult,
					IsError:  isError,
				},
			}}) {
				return
			}

			// Append the tool result to session history so the model sees it.
			sess.AppendToolResult(tub.ID, tub.Name, toolResult, isError)
		}
		// Loop back: re-invoke the model with updated history.
	}

	// Hard cap exceeded.
	msg := fmt.Sprintf("openai: tool loop exceeded %d turns", maxToolTurns)
	send(ctx, ch, result{
		ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
		err: fmt.Errorf("%s", msg), //nolint:goerr113 // intentional string-only error
	})
}

// streamInto performs the HTTP request, drives the SSE event loop, sends events
// on ch, and returns accumulated content blocks on success.
//
// Returns the accumulated content blocks and true on success.
// On context cancellation or stream error it sends an error result and returns nil, false.
func (a *openaiAdapter) streamInto(
	ctx context.Context,
	sess *agent.Session,
	ch chan<- result,
) ([]agent.ContentBlock, bool) {
	msgs := buildMessages(sess)

	reqBody := chatRequest{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		Stream:    true,
		Messages:  msgs,
	}

	// Attach tools when the session has any registered.
	if len(sess.Tools) > 0 {
		tools := make([]openaiTool, 0, len(sess.Tools))
		for _, t := range sess.Tools {
			tools = append(tools, openaiTool{
				Type: "function",
				Function: openaiFunction{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  schemaToFunctionParameters(t.Schema()),
				},
			})
		}
		reqBody.Tools = tools
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		errMsg := sanitiseError(fmt.Errorf("openai: marshal request: %w", err), a.apiKey).Error()
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: errMsg},
			err: fmt.Errorf("%s", errMsg), //nolint:goerr113
		})
		return nil, false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		if ctx.Err() != nil {
			return nil, false
		}
		errMsg := sanitiseError(fmt.Errorf("openai: create request: %w", err), a.apiKey).Error()
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: errMsg},
			err: fmt.Errorf("%s", errMsg), //nolint:goerr113
		})
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false
		}
		errMsg := sanitiseError(fmt.Errorf("openai: do request: %w", err), a.apiKey).Error()
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: errMsg},
			err: fmt.Errorf("%s", errMsg), //nolint:goerr113
		})
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		_, _ = io.Copy(&errBody, io.LimitReader(resp.Body, 64*1024))
		errMsg := sanitiseError(
			fmt.Errorf("openai: unexpected status %d: %s", resp.StatusCode, errBody.String()),
			a.apiKey,
		).Error()
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: errMsg},
			err: fmt.Errorf("%s", errMsg), //nolint:goerr113
		})
		return nil, false
	}

	// Emit start event.
	if !send(ctx, ch, result{ev: agent.Event{Kind: agent.EventKindStart}}) {
		return nil, false
	}

	// Accumulate text and tool call fragments across SSE deltas.
	var textBuf strings.Builder

	// toolCallAccum maps tool_calls[i].index → accumulated state.
	type toolCallAccum struct {
		id   string
		name string
		args strings.Builder
	}
	toolCalls := make(map[int]*toolCallAccum)

	parseErr := sseutil.Parse(ctx, resp.Body, func(data []byte) bool {
		var delta streamDelta
		if err := json.Unmarshal(data, &delta); err != nil {
			// Skip malformed lines.
			return true
		}
		for _, choice := range delta.Choices {
			// Accumulate text tokens.
			if choice.Delta.Content != "" {
				textBuf.WriteString(choice.Delta.Content)
				if !send(ctx, ch, result{ev: agent.Event{
					Kind: agent.EventKindToken,
					Text: choice.Delta.Content,
				}}) {
					return false
				}
			}

			// Accumulate tool call fragments by index.
			for _, tc := range choice.Delta.ToolCalls {
				acc, exists := toolCalls[tc.Index]
				if !exists {
					acc = &toolCallAccum{}
					toolCalls[tc.Index] = acc
				}
				if tc.ID != "" {
					acc.id = tc.ID
				}
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					acc.args.WriteString(tc.Function.Arguments)
				}
			}
		}
		return true
	})

	if parseErr != nil {
		if ctx.Err() != nil {
			return nil, false
		}
		errMsg := sanitiseError(fmt.Errorf("openai: stream parse: %w", parseErr), a.apiKey).Error()
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: errMsg},
			err: fmt.Errorf("%s", errMsg), //nolint:goerr113
		})
		return nil, false
	}

	// Reconstruct content blocks from accumulated state.
	var blocks []agent.ContentBlock

	if text := textBuf.String(); text != "" {
		blocks = append(blocks, agent.TextBlock{Text: text})
	}

	// Reconstruct tool use blocks in index order.
	for i := 0; i < len(toolCalls); i++ {
		acc, ok := toolCalls[i]
		if !ok {
			continue
		}
		// Parse the accumulated JSON arguments string into map[string]any.
		var input map[string]any
		if argsStr := acc.args.String(); argsStr != "" {
			if err := json.Unmarshal([]byte(argsStr), &input); err != nil {
				// Malformed arguments — use empty map.
				input = make(map[string]any)
			}
		}
		if input == nil {
			input = make(map[string]any)
		}
		blocks = append(blocks, agent.ToolUseBlock{
			ID:    acc.id,
			Name:  acc.name,
			Input: input,
		})
	}

	// Emit done event (OpenAI streaming doesn't provide token counts).
	if !send(ctx, ch, result{ev: agent.Event{Kind: agent.EventKindDone}}) {
		return nil, false
	}

	return blocks, true
}

// buildMessages converts a session's history into OpenAI chat messages.
// The system prompt is prepended as a system role message when present.
//
// History message handling:
//   - RoleAssistant messages with ToolUseBlock content: emit as assistant message with tool_calls.
//   - RoleUser messages with ToolResultBlock content: emit as role "tool" messages with tool_call_id.
//   - All other messages: use TextOf to extract plain text.
func buildMessages(sess *agent.Session) []chatMessage {
	msgs := make([]chatMessage, 0, len(sess.History)+1)
	if sess.SystemPrompt != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: sess.SystemPrompt})
	}
	for _, m := range sess.History {
		role := mapRole(m.Role)

		// Check if this is an assistant message with tool_use blocks.
		if m.Role == agent.RoleAssistant {
			var toolCalls []chatToolCall
			for _, b := range m.Content {
				if tub, ok := b.(agent.ToolUseBlock); ok {
					toolCalls = append(toolCalls, chatToolCall{
						ID:   tub.ID,
						Type: "function",
						Function: chatToolCallFunction{
							Name:      tub.Name,
							Arguments: marshalInputToJSON(tub.Input),
						},
					})
				}
			}
			if len(toolCalls) > 0 {
				msgs = append(msgs, chatMessage{
					Role:      role,
					ToolCalls: toolCalls,
				})
				continue
			}
		}

		// Check if this is a user message containing tool result blocks.
		// Each ToolResultBlock emits as a separate "tool" role message.
		if m.Role == agent.RoleUser {
			var hasToolResult bool
			for _, b := range m.Content {
				if trb, ok := b.(agent.ToolResultBlock); ok {
					hasToolResult = true
					msgs = append(msgs, chatMessage{
						Role:       "tool",
						Content:    trb.Result,
						ToolCallID: trb.ID,
					})
				}
			}
			if hasToolResult {
				continue
			}
		}

		// Plain text message.
		msgs = append(msgs, chatMessage{Role: role, Content: agent.TextOf(m.Content)})
	}
	return msgs
}

// marshalInputToJSON marshals a map[string]any to a JSON string.
// Returns "{}" on error.
func marshalInputToJSON(input map[string]any) string {
	if input == nil {
		return "{}"
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// schemaToFunctionParameters extracts the properties and required fields from
// a full JSON Schema map and returns a map[string]any suitable for the OpenAI
// function parameters field.
//
// agent.Tool.Schema() returns a full JSON Schema:
//
//	{"type":"object","properties":{...},"required":[...]}
//
// The full schema map is passed directly as the parameters value — OpenAI
// accepts the full schema object including "type", "properties", and "required".
func schemaToFunctionParameters(schema map[string]any) map[string]any {
	if schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return schema
}

// collectToolUseBlocks filters a content-block slice and returns only the
// agent.ToolUseBlock values.
func collectToolUseBlocks(blocks []agent.ContentBlock) []agent.ToolUseBlock {
	var out []agent.ToolUseBlock
	for _, b := range blocks {
		if tub, ok := b.(agent.ToolUseBlock); ok {
			out = append(out, tub)
		}
	}
	return out
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
