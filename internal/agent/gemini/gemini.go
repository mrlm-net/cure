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

	"github.com/mrlm-net/cure/pkg/agent"
)

const (
	defaultModel     = "gemini-2.5-pro"
	defaultMaxTokens = 8192
	defaultKeyEnv    = "GEMINI_API_KEY"
	defaultBaseURL   = "https://generativelanguage.googleapis.com"
	baseURLEnv       = "GEMINI_BASE_URL"
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
		client:    &http.Client{},
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

// geminiPart is a content part (text only for now).
type geminiPart struct {
	Text string `json:"text"`
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

// geminiRequest is the body for generateContent / streamGenerateContent.
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
}

// geminiStreamEvent is a single SSE data payload from streamGenerateContent.
type geminiStreamEvent struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
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

// --- Request building -------------------------------------------------------

// buildRequest constructs a geminiRequest from a session.
// RoleSystem messages are moved to SystemInstruction; they must not appear
// in the contents array.
func (a *geminiAdapter) buildRequest(sess *agent.Session, withGenConfig bool) geminiRequest {
	var contents []geminiContent
	for _, m := range sess.History {
		switch m.Role {
		case agent.RoleUser:
			// NOTE: m.Content will become MessageContent after PR #116-120 merges.
			// At that point, replace with: agent.TextOf(m.Content). See issue #123.
			contents = append(contents, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: m.Content}},
			})
		case agent.RoleAssistant:
			contents = append(contents, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: m.Content}},
			})
		// RoleSystem is handled via sess.SystemPrompt below; skip from history.
		}
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

	return req
}

// --- Streaming (Run) --------------------------------------------------------

// Run streams a response for the given session as an iter.Seq2[Event, error].
// The caller iterates events with: for ev, err := range a.Run(ctx, session) { ... }
// Cancelling ctx terminates the stream.
func (a *geminiAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s",
			a.baseURL, a.model, a.apiKey)

		body, err := json.Marshal(a.buildRequest(sess, true))
		if err != nil {
			yield(agent.Event{Kind: agent.EventKindError, Err: "gemini: failed to marshal request"}, fmt.Errorf("gemini: marshal request: %w", err))
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			msg := a.sanitiseString(fmt.Sprintf("gemini: failed to create request: %v", err))
			yield(agent.Event{Kind: agent.EventKindError, Err: msg}, fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(httpReq)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			msg := a.sanitiseString(fmt.Sprintf("gemini: HTTP request failed: %v", err))
			yield(agent.Event{Kind: agent.EventKindError, Err: msg}, fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			msg := a.sanitiseString(fmt.Sprintf("gemini: API error %d: %s", resp.StatusCode, strings.TrimSpace(string(raw))))
			yield(agent.Event{Kind: agent.EventKindError, Err: msg}, fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}

		// Emit start event before reading the stream.
		if !yield(agent.Event{Kind: agent.EventKindStart}, nil) {
			return
		}

		var (
			outputTokens int
			inputTokens  int
			stopReason   string
		)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return
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
						if !yield(agent.Event{Kind: agent.EventKindToken, Text: part.Text}, nil) {
							return
						}
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			if ctx.Err() != nil {
				return
			}
			msg := a.sanitiseString(fmt.Sprintf("gemini: stream read error: %v", err))
			yield(agent.Event{Kind: agent.EventKindError, Err: msg}, fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}

		yield(agent.Event{
			Kind:         agent.EventKindDone,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			StopReason:   stopReason,
		}, nil)
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

	raw, err := io.ReadAll(resp.Body)
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
