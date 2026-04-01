// Package gemini provides a Google Gemini AI provider adapter for pkg/agent.
// Import this package with a blank import to register the "gemini" provider:
//
//	import _ "github.com/mrlm-net/cure/internal/agent/gemini"
package gemini

import (
	"bufio"
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

	"github.com/mrlm-net/cure/pkg/agent"
)

const (
	defaultModel     = "gemini-2.5-pro"
	defaultMaxTokens = 8192
	defaultKeyEnv    = "GEMINI_API_KEY"
	defaultBaseURL   = "https://generativelanguage.googleapis.com"
	baseURLEnv       = "GEMINI_BASE_URL"

	// maxToolTurns is the hard cap on tool-call iterations within a single Run.
	// If the model keeps requesting tools after this many turns, executeToolLoop
	// returns an error rather than looping indefinitely.
	maxToolTurns = 32
)

func init() {
	agent.Register("gemini", NewGeminiAgent)
}

// geminiAdapter implements agent.Agent for the Google Gemini API.
type geminiAdapter struct {
	apiKey    string // held only for sanitiseError — never emitted in events
	model     string
	maxTokens int
	baseURL   string
	client    *http.Client
}

// NewGeminiAgent is the AgentFactory for the "gemini" provider.
// cfg keys: "api_key_env" (default "GEMINI_API_KEY"), "model" (default "gemini-2.5-pro"),
// "max_tokens" (default 8192).
func NewGeminiAgent(cfg map[string]any) (agent.Agent, error) {
	keyEnv := defaultKeyEnv
	if v, ok := cfg["api_key_env"].(string); ok && v != "" {
		keyEnv = v
	}
	apiKey := os.Getenv(keyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: API key not set — populate the %s environment variable", keyEnv)
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

	// Allow base URL override via environment for testing.
	baseURL := os.Getenv(baseURLEnv)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &geminiAdapter{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		baseURL:   baseURL,
		client: &http.Client{
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

// Provider returns the provider name "gemini".
func (a *geminiAdapter) Provider() string { return "gemini" }

// sanitiseError replaces the API key value with "***" in error strings.
func (a *geminiAdapter) sanitiseError(err error) string {
	s := err.Error()
	if a.apiKey != "" {
		s = strings.ReplaceAll(s, a.apiKey, "***")
	}
	return s
}

// sanitiseString replaces the API key value with "***" in arbitrary strings.
func (a *geminiAdapter) sanitiseString(s string) string {
	if a.apiKey != "" {
		s = strings.ReplaceAll(s, a.apiKey, "***")
	}
	return s
}

// --- Request/Response types -----------------------------------------------

// geminiPart is a content part: text, function call request, or function response.
type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

// geminiFunctionCall is a function invocation request from the model.
type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// geminiFunctionResponse is the tool result fed back to the model.
type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// geminiContent is a conversation turn.
type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

// geminiGenerationConfig holds sampling/output configuration.
type geminiGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens"`
}

// geminiToolDeclaration describes a single function available to the model.
type geminiToolDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// geminiTool groups function declarations in the Gemini request format.
type geminiTool struct {
	FunctionDeclarations []geminiToolDeclaration `json:"functionDeclarations"`
}

// geminiRequest is the body for generateContent / streamGenerateContent.
type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Tools             []geminiTool           `json:"tools,omitempty"`
}

// geminiStreamEvent is a single SSE data payload from streamGenerateContent.
type geminiStreamEvent struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string              `json:"text"`
				FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

// geminiCountTokensResponse is the response from the countTokens endpoint.
type geminiCountTokensResponse struct {
	TotalTokens int `json:"totalTokens"`
}

// --- Internal channel type --------------------------------------------------

// geminiResult is a (event, error) pair sent over the internal channel in Run.
type geminiResult struct {
	ev  agent.Event
	err error
}

// geminiSend delivers a result to ch, aborting if ctx is cancelled.
// Returns false if the send was cancelled (goroutine should stop).
func geminiSend(ctx context.Context, ch chan<- geminiResult, r geminiResult) bool {
	select {
	case ch <- r:
		return true
	case <-ctx.Done():
		return false
	}
}

// --- Request building -------------------------------------------------------

// buildRequest constructs a geminiRequest from a session.
// RoleSystem messages are moved to SystemInstruction; they must not appear
// in the contents array. Consecutive ToolResultBlock-only user messages are
// grouped into a single user content with multiple functionResponse parts,
// matching the Gemini API requirement.
func (a *geminiAdapter) buildRequest(sess *agent.Session, withGenConfig bool) geminiRequest {
	var contents []geminiContent

	i := 0
	for i < len(sess.History) {
		m := sess.History[i]
		switch m.Role {
		case agent.RoleAssistant:
			// Check for ToolUseBlocks — emit as functionCall parts.
			var funcCallParts []geminiPart
			for _, b := range m.Content {
				if tub, ok := b.(agent.ToolUseBlock); ok {
					args := tub.Input
					if args == nil {
						args = make(map[string]any)
					}
					funcCallParts = append(funcCallParts, geminiPart{
						FunctionCall: &geminiFunctionCall{
							Name: tub.Name,
							Args: args,
						},
					})
				}
			}
			if len(funcCallParts) > 0 {
				contents = append(contents, geminiContent{Role: "model", Parts: funcCallParts})
				i++
				continue
			}
			contents = append(contents, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: agent.TextOf(m.Content)}},
			})
		case agent.RoleUser:
			// Scan forward to group consecutive tool-result-only user messages into
			// a single user content with multiple functionResponse parts.
			var funcRespParts []geminiPart
			j := i
			for j < len(sess.History) {
				mj := sess.History[j]
				if mj.Role != agent.RoleUser {
					break
				}
				var trbs []agent.ToolResultBlock
				allToolResults := true
				for _, b := range mj.Content {
					if trb, ok := b.(agent.ToolResultBlock); ok {
						trbs = append(trbs, trb)
					} else {
						allToolResults = false
						break
					}
				}
				if !allToolResults || len(trbs) == 0 {
					break
				}
				for _, trb := range trbs {
					funcRespParts = append(funcRespParts, geminiPart{
						FunctionResponse: &geminiFunctionResponse{
							Name:     trb.ToolName,
							Response: map[string]any{"output": trb.Result},
						},
					})
				}
				j++
			}
			if len(funcRespParts) > 0 {
				contents = append(contents, geminiContent{Role: "user", Parts: funcRespParts})
				i = j
				continue
			}
			// Regular user text message.
			contents = append(contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: agent.TextOf(m.Content)}},
			})
		// RoleSystem is handled via sess.SystemPrompt below; skip from history.
		}
		i++
	}

	req := geminiRequest{Contents: contents}

	if withGenConfig {
		req.GenerationConfig = &geminiGenerationConfig{
			MaxOutputTokens: a.maxTokens,
		}
	}

	if sess.SystemPrompt != "" {
		req.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: sess.SystemPrompt}},
		}
	}

	// Attach tools when the session has any registered.
	if len(sess.Tools) > 0 {
		decls := make([]geminiToolDeclaration, 0, len(sess.Tools))
		for _, t := range sess.Tools {
			decls = append(decls, geminiToolDeclaration{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			})
		}
		req.Tools = []geminiTool{{FunctionDeclarations: decls}}
	}

	return req
}

// --- Tool loop --------------------------------------------------------------

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
// An error is sent (via ch) when maxToolTurns is exceeded.
func (a *geminiAdapter) executeToolLoop(ctx context.Context, sess *agent.Session, ch chan<- geminiResult) {
	// Build a lookup map for fast tool resolution once — sess.Tools is immutable
	// for the lifetime of a Run call.
	toolByName := make(map[string]agent.Tool, len(sess.Tools))
	for _, t := range sess.Tools {
		toolByName[t.Name()] = t
	}

	for turn := 0; turn < maxToolTurns; turn++ {
		// Stream one model response turn; collect all content blocks.
		blocks, ok := a.streamInto(ctx, sess, turn, ch)
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
				msg := a.sanitiseString(fmt.Sprintf("gemini: failed to marshal tool input for %s: %v", tub.Name, err))
				geminiSend(ctx, ch, geminiResult{
					ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
					err: fmt.Errorf("%s", msg), //nolint:goerr113
				})
				return
			}
			inputJSON := string(inputBytes)

			// Emit tool call event.
			if !geminiSend(ctx, ch, geminiResult{ev: agent.Event{
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
			if !geminiSend(ctx, ch, geminiResult{ev: agent.Event{
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

			// Append the tool result to session history so the model sees it on the
			// next turn. The ID is the locally generated "fc_{turn}_{idx}" value.
			sess.AppendToolResult(tub.ID, tub.Name, toolResult, isError)
		}
		// Loop back: re-invoke the model with updated history.
	}

	// Hard cap exceeded.
	msg := fmt.Sprintf("gemini: tool loop exceeded %d turns", maxToolTurns)
	geminiSend(ctx, ch, geminiResult{
		ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
		err: fmt.Errorf("%s", msg), //nolint:goerr113
	})
}

// streamInto performs the HTTP request, drives the SSE scanner, sends events on
// ch, and returns accumulated content blocks on success.
//
// Returns the accumulated content blocks and true on success.
// On context cancellation or stream error it sends an error result and returns nil, false.
func (a *geminiAdapter) streamInto(
	ctx context.Context,
	sess *agent.Session,
	turn int,
	ch chan<- geminiResult,
) ([]agent.ContentBlock, bool) {
	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s",
		a.baseURL, a.model, a.apiKey)

	body, err := json.Marshal(a.buildRequest(sess, true))
	if err != nil {
		msg := a.sanitiseString(fmt.Sprintf("gemini: failed to marshal request: %v", err))
		geminiSend(ctx, ch, geminiResult{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
		return nil, false
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		if ctx.Err() != nil {
			return nil, false
		}
		msg := a.sanitiseString(fmt.Sprintf("gemini: failed to create request: %v", err))
		geminiSend(ctx, ch, geminiResult{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
		return nil, false
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false
		}
		msg := a.sanitiseString(fmt.Sprintf("gemini: HTTP request failed: %v", err))
		geminiSend(ctx, ch, geminiResult{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		msg := a.sanitiseString(fmt.Sprintf("gemini: API error %d: %s", resp.StatusCode, strings.TrimSpace(string(raw))))
		geminiSend(ctx, ch, geminiResult{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
		return nil, false
	}

	// Emit start event before reading the stream.
	if !geminiSend(ctx, ch, geminiResult{ev: agent.Event{Kind: agent.EventKindStart}}) {
		return nil, false
	}

	var (
		outputTokens  int
		inputTokens   int
		stopReason    string
		textBuf       strings.Builder
		funcCallParts []geminiFunctionCall
	)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil, false
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "" || data == "[DONE]" {
			continue
		}

		var ev geminiStreamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			// Skip malformed events; Gemini sometimes sends partial lines.
			continue
		}

		// Accumulate token usage from usageMetadata (present on last event).
		if ev.UsageMetadata.PromptTokenCount > 0 {
			inputTokens = ev.UsageMetadata.PromptTokenCount
		}
		if ev.UsageMetadata.CandidatesTokenCount > 0 {
			outputTokens = ev.UsageMetadata.CandidatesTokenCount
		}

		for _, candidate := range ev.Candidates {
			if candidate.FinishReason != "" {
				stopReason = candidate.FinishReason
			}
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					textBuf.WriteString(part.Text)
					if !geminiSend(ctx, ch, geminiResult{ev: agent.Event{
						Kind: agent.EventKindToken,
						Text: part.Text,
					}}) {
						return nil, false
					}
				}
				if part.FunctionCall != nil {
					funcCallParts = append(funcCallParts, *part.FunctionCall)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil, false
		}
		msg := a.sanitiseString(fmt.Sprintf("gemini: stream read error: %v", err))
		geminiSend(ctx, ch, geminiResult{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
		return nil, false
	}

	// Reconstruct content blocks from accumulated state.
	var blocks []agent.ContentBlock

	if text := textBuf.String(); text != "" {
		blocks = append(blocks, agent.TextBlock{Text: text})
	}

	// Convert function call parts to ToolUseBlocks, assigning locally generated
	// IDs that encode both turn and index for cross-turn uniqueness.
	for i, fc := range funcCallParts {
		args := fc.Args
		if args == nil {
			args = make(map[string]any)
		}
		blocks = append(blocks, agent.ToolUseBlock{
			ID:    fmt.Sprintf("fc_%d_%d", turn, i),
			Name:  fc.Name,
			Input: args,
		})
	}

	if !geminiSend(ctx, ch, geminiResult{ev: agent.Event{
		Kind:         agent.EventKindDone,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		StopReason:   stopReason,
	}}) {
		return nil, false
	}

	return blocks, true
}

// --- Streaming (Run) --------------------------------------------------------

// Run streams a response for the given session as an iter.Seq2[Event, error].
// When the session has Tools registered and the model requests function calls,
// Run automatically executes the tool loop: it calls each requested tool,
// emits EventKindToolCall and EventKindToolResult events, appends the results
// to the session history, and re-invokes the model. The loop repeats until
// the model returns without requesting tools or until maxToolTurns is reached.
// Cancelling ctx terminates the stream.
func (a *geminiAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		ch := make(chan geminiResult)
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

// --- Token counting ---------------------------------------------------------

// CountTokens returns the token count for the session using the Gemini API.
func (a *geminiAdapter) CountTokens(ctx context.Context, sess *agent.Session) (int, error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:countTokens?key=%s",
		a.baseURL, a.model, a.apiKey)

	body, err := json.Marshal(a.buildRequest(sess, false))
	if err != nil {
		return 0, fmt.Errorf("gemini: marshal countTokens request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("gemini: countTokens create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("gemini: countTokens HTTP: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return 0, fmt.Errorf("gemini: countTokens read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := a.sanitiseString(fmt.Sprintf("gemini: countTokens API error %d: %s", resp.StatusCode, strings.TrimSpace(string(raw))))
		return 0, fmt.Errorf("%s", msg) //nolint:goerr113
	}

	var result geminiCountTokensResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return 0, fmt.Errorf("gemini: countTokens decode response: %w", err)
	}

	return result.TotalTokens, nil
}

// --- Helpers ----------------------------------------------------------------

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
