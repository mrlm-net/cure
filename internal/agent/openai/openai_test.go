package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

// ---- Factory / config tests -----------------------------------------------

func TestNewOpenAIAgent_MissingAPIKey(t *testing.T) {
	// Use a custom env key that is definitely not set.
	cfg := map[string]any{"api_key_env": "TEST_OPENAI_KEY_NOTSET_XYZ"}
	_, err := NewOpenAIAgent(cfg)
	if err == nil {
		t.Fatal("expected error when API key not set, got nil")
	}
	if !strings.Contains(err.Error(), "TEST_OPENAI_KEY_NOTSET_XYZ") {
		t.Errorf("error should mention env var name, got: %v", err)
	}
}

func TestNewOpenAIAgent_Defaults(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-default")

	a, err := NewOpenAIAgent(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oa := a.(*openaiAdapter)

	if oa.model != defaultModel {
		t.Errorf("model = %q, want %q", oa.model, defaultModel)
	}
	if oa.maxTokens != defaultMaxTokens {
		t.Errorf("maxTokens = %d, want %d", oa.maxTokens, defaultMaxTokens)
	}
	if oa.apiKey != "sk-test-default" {
		t.Errorf("apiKey = %q, want %q", oa.apiKey, "sk-test-default")
	}
}

func TestNewOpenAIAgent_OverrideModel(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-override")

	cfg := map[string]any{
		"model":      "gpt-3.5-turbo",
		"max_tokens": 1024,
	}
	a, err := NewOpenAIAgent(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oa := a.(*openaiAdapter)

	if oa.model != "gpt-3.5-turbo" {
		t.Errorf("model = %q, want %q", oa.model, "gpt-3.5-turbo")
	}
	if oa.maxTokens != 1024 {
		t.Errorf("maxTokens = %d, want 1024", oa.maxTokens)
	}
}

func TestNewOpenAIAgent_MaxTokensTypes(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")

	tests := []struct {
		name      string
		maxTokens any
		want      int
	}{
		{"int", 512, 512},
		{"int64", int64(1024), 1024},
		{"float64", float64(2048), 2048},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := map[string]any{"max_tokens": tt.maxTokens}
			a, err := NewOpenAIAgent(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			oa := a.(*openaiAdapter)
			if oa.maxTokens != tt.want {
				t.Errorf("maxTokens = %d, want %d", oa.maxTokens, tt.want)
			}
		})
	}
}

func TestNewOpenAIAgent_CustomKeyEnv(t *testing.T) {
	t.Setenv("MY_CUSTOM_OPENAI_KEY", "sk-custom")

	cfg := map[string]any{"api_key_env": "MY_CUSTOM_OPENAI_KEY"}
	a, err := NewOpenAIAgent(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oa := a.(*openaiAdapter)
	if oa.apiKey != "sk-custom" {
		t.Errorf("apiKey = %q, want %q", oa.apiKey, "sk-custom")
	}
}

// ---- Provider ---------------------------------------------------------------

func TestProvider(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	a, err := NewOpenAIAgent(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := a.Provider(); got != "openai" {
		t.Errorf("Provider() = %q, want %q", got, "openai")
	}
}

// ---- CountTokens ------------------------------------------------------------

func TestCountTokens_ReturnsNotSupported(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	a, err := NewOpenAIAgent(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess := agent.NewSession("openai", "gpt-4o")
	n, err := a.CountTokens(context.Background(), sess)

	if n != 0 {
		t.Errorf("CountTokens() count = %d, want 0", n)
	}
	if !errors.Is(err, agent.ErrCountNotSupported) {
		t.Errorf("CountTokens() error = %v, want ErrCountNotSupported", err)
	}
}

// ---- Role mapping -----------------------------------------------------------

func TestMapRole(t *testing.T) {
	tests := []struct {
		role agent.Role
		want string
	}{
		{agent.RoleUser, "user"},
		{agent.RoleAssistant, "assistant"},
		{agent.RoleSystem, "system"},
		{agent.Role("custom"), "custom"},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := mapRole(tt.role); got != tt.want {
				t.Errorf("mapRole(%q) = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}

// ---- sanitiseError ----------------------------------------------------------

func TestSanitiseError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		apiKey string
		want   string
	}{
		{
			name:   "redacts API key in error message",
			err:    fmt.Errorf("request failed with key sk-secret123"),
			apiKey: "sk-secret123",
			want:   "request failed with key ***",
		},
		{
			name:   "no key in error — unchanged",
			err:    fmt.Errorf("some other error"),
			apiKey: "sk-secret123",
			want:   "some other error",
		},
		{
			name:   "empty API key — unchanged",
			err:    fmt.Errorf("error with sk-secret456"),
			apiKey: "",
			want:   "error with sk-secret456",
		},
		{
			name:   "nil error returns nil",
			err:    nil,
			apiKey: "sk-secret",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitiseError(tt.err, tt.apiKey)
			if tt.err == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil error, got nil")
			}
			if result.Error() != tt.want {
				t.Errorf("sanitised error = %q, want %q", result.Error(), tt.want)
			}
		})
	}
}

// ---- buildMessages ----------------------------------------------------------

func TestBuildMessages(t *testing.T) {
	t.Run("system prompt is prepended", func(t *testing.T) {
		sess := agent.NewSession("openai", "gpt-4o")
		sess.SystemPrompt = "You are helpful."
		sess.AppendUserMessage("hello")
		sess.AppendAssistantMessage("hi")

		msgs := buildMessages(sess)
		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}
		if msgs[0].Role != "system" || msgs[0].Content != "You are helpful." {
			t.Errorf("msgs[0] = %+v, want system prompt", msgs[0])
		}
		if msgs[1].Role != "user" || msgs[1].Content != "hello" {
			t.Errorf("msgs[1] = %+v, want user message", msgs[1])
		}
		if msgs[2].Role != "assistant" || msgs[2].Content != "hi" {
			t.Errorf("msgs[2] = %+v, want assistant message", msgs[2])
		}
	})

	t.Run("no system prompt — only history messages", func(t *testing.T) {
		sess := agent.NewSession("openai", "gpt-4o")
		sess.AppendUserMessage("ping")

		msgs := buildMessages(sess)
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		if msgs[0].Role != "user" {
			t.Errorf("msgs[0].Role = %q, want %q", msgs[0].Role, "user")
		}
	})
}

// ---- TestBuildMessages_ToolHistory ------------------------------------------

func TestBuildMessages_ToolHistory(t *testing.T) {
	sess := agent.NewSession("openai", "gpt-4o")
	sess.AppendUserMessage("what is the weather?")

	// Simulate the assistant asking for a tool call.
	sess.AppendAssistantBlocks([]agent.ContentBlock{
		agent.ToolUseBlock{
			ID:    "call_abc",
			Name:  "get_weather",
			Input: map[string]any{"location": "Paris"},
		},
	})

	// Simulate the tool result being appended.
	sess.AppendToolResult("call_abc", "get_weather", "72°F and sunny", false)

	msgs := buildMessages(sess)

	// Expected: [user, assistant(tool_calls), tool]
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(msgs), msgs)
	}

	// msgs[0]: user
	if msgs[0].Role != "user" || msgs[0].Content != "what is the weather?" {
		t.Errorf("msgs[0] = %+v, want user with weather question", msgs[0])
	}

	// msgs[1]: assistant with tool_calls
	if msgs[1].Role != "assistant" {
		t.Errorf("msgs[1].Role = %q, want %q", msgs[1].Role, "assistant")
	}
	if len(msgs[1].ToolCalls) != 1 {
		t.Fatalf("msgs[1].ToolCalls len = %d, want 1", len(msgs[1].ToolCalls))
	}
	tc := msgs[1].ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("tool_call.id = %q, want %q", tc.ID, "call_abc")
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("tool_call.function.name = %q, want %q", tc.Function.Name, "get_weather")
	}
	// Verify the arguments JSON is parseable and contains the location.
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		t.Errorf("tool_call.function.arguments is not valid JSON: %v", err)
	}
	if loc, _ := args["location"].(string); loc != "Paris" {
		t.Errorf("tool_call.function.arguments[location] = %q, want %q", loc, "Paris")
	}

	// msgs[2]: tool result
	if msgs[2].Role != "tool" {
		t.Errorf("msgs[2].Role = %q, want %q", msgs[2].Role, "tool")
	}
	if msgs[2].ToolCallID != "call_abc" {
		t.Errorf("msgs[2].ToolCallID = %q, want %q", msgs[2].ToolCallID, "call_abc")
	}
	if msgs[2].Content != "72°F and sunny" {
		t.Errorf("msgs[2].Content = %q, want %q", msgs[2].Content, "72°F and sunny")
	}
}

// ---- E2E test with httptest.Server ------------------------------------------

// buildSSEResponse constructs a minimal SSE response with the given token texts.
func buildSSEResponse(tokens []string) string {
	var sb strings.Builder
	for _, tok := range tokens {
		// Each line is a valid SSE data event with OpenAI delta structure.
		payload := map[string]any{
			"choices": []map[string]any{
				{
					"delta": map[string]any{
						"content": tok,
					},
				},
			},
		}
		data, _ := json.Marshal(payload)
		sb.WriteString("data: ")
		sb.Write(data)
		sb.WriteString("\n")
	}
	sb.WriteString("data: [DONE]\n")
	return sb.String()
}

// buildSSEToolCallResponse constructs an SSE response representing a tool call request.
// The arguments are chunked to simulate real streaming fragmentation.
func buildSSEToolCallResponse(callID, toolName, argsJSON string) string {
	var sb strings.Builder

	// First delta: tool_call with id, type, and function name (no arguments yet)
	firstDelta := map[string]any{
		"choices": []map[string]any{
			{
				"delta": map[string]any{
					"tool_calls": []map[string]any{
						{
							"index":    0,
							"id":       callID,
							"type":     "function",
							"function": map[string]any{"name": toolName, "arguments": ""},
						},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(firstDelta)
	sb.WriteString("data: ")
	sb.Write(data)
	sb.WriteString("\n")

	// Second delta: arguments fragment
	argsDelta := map[string]any{
		"choices": []map[string]any{
			{
				"delta": map[string]any{
					"tool_calls": []map[string]any{
						{
							"index":    0,
							"function": map[string]any{"arguments": argsJSON},
						},
					},
				},
			},
		},
	}
	data, _ = json.Marshal(argsDelta)
	sb.WriteString("data: ")
	sb.Write(data)
	sb.WriteString("\n")

	sb.WriteString("data: [DONE]\n")
	return sb.String()
}

func TestRun_E2E_StreamsTokensAndDone(t *testing.T) {
	expectedTokens := []string{"Hello", ", ", "world", "!"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer sk-test-e2e" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildSSEResponse(expectedTokens))
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "sk-test-e2e")
	t.Setenv("OPENAI_BASE_URL", srv.URL)

	a, err := NewOpenAIAgent(map[string]any{})
	if err != nil {
		t.Fatalf("NewOpenAIAgent error: %v", err)
	}

	sess := agent.NewSession("openai", "gpt-4o")
	sess.AppendUserMessage("Say hello")

	var receivedTokens []string
	var gotStart, gotDone bool

	for ev, err := range a.Run(context.Background(), sess) {
		if err != nil {
			t.Fatalf("unexpected error in stream: %v", err)
		}
		switch ev.Kind {
		case agent.EventKindStart:
			gotStart = true
		case agent.EventKindToken:
			receivedTokens = append(receivedTokens, ev.Text)
		case agent.EventKindDone:
			gotDone = true
		case agent.EventKindError:
			t.Fatalf("unexpected error event: %s", ev.Err)
		}
	}

	if !gotStart {
		t.Error("expected EventKindStart, not received")
	}
	if !gotDone {
		t.Error("expected EventKindDone, not received")
	}

	if len(receivedTokens) != len(expectedTokens) {
		t.Fatalf("got %d tokens, want %d\nreceived: %v", len(receivedTokens), len(expectedTokens), receivedTokens)
	}
	for i, want := range expectedTokens {
		if receivedTokens[i] != want {
			t.Errorf("token[%d] = %q, want %q", i, receivedTokens[i], want)
		}
	}
}

func TestRun_E2E_HTTPErrorReturnsErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"model not found"}}`, http.StatusNotFound)
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "sk-test-err")
	t.Setenv("OPENAI_BASE_URL", srv.URL)

	a, err := NewOpenAIAgent(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess := agent.NewSession("openai", "gpt-4o")
	sess.AppendUserMessage("test")

	var gotError bool
	for ev, err := range a.Run(context.Background(), sess) {
		if err != nil || ev.Kind == agent.EventKindError {
			gotError = true
			// Verify the API key is not exposed in the error.
			errMsg := ev.Err
			if errMsg == "" && err != nil {
				errMsg = err.Error()
			}
			if strings.Contains(errMsg, "sk-test-err") {
				t.Errorf("API key leaked in error: %s", errMsg)
			}
		}
	}
	if !gotError {
		t.Error("expected error event for 404 response, got none")
	}
}

func TestRun_E2E_ContextCancellation(t *testing.T) {
	// Server that blocks until the client disconnects.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Send one token and then block.
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "sk-test-cancel")
	t.Setenv("OPENAI_BASE_URL", srv.URL)

	a, err := NewOpenAIAgent(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sess := agent.NewSession("openai", "gpt-4o")
	sess.AppendUserMessage("stream test")

	// Cancel after receiving the first token.
	count := 0
	for ev, _ := range a.Run(ctx, sess) {
		if ev.Kind == agent.EventKindToken {
			count++
			cancel()
		}
	}
	// We just verify the range loop terminates — no panic or hang.
	_ = count
}

// ---- Tool loop tests --------------------------------------------------------

// echoTool returns a Tool that echoes its "text" argument as the result.
func echoTool() agent.Tool {
	return agent.FuncTool(
		"echo",
		"Echoes the input text back",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string"},
			},
			"required": []string{"text"},
		},
		func(_ context.Context, args map[string]any) (string, error) {
			if v, ok := args["text"].(string); ok {
				return v, nil
			}
			return "", nil
		},
	)
}

func TestRun_ToolLoop_SingleCallThenText(t *testing.T) {
	var callCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		switch n {
		case 1:
			// First turn: model requests echo("hello")
			fmt.Fprint(w, buildSSEToolCallResponse("call_echo1", "echo", `{"text":"hello"}`))
		case 2:
			// Second turn: model returns plain text after seeing the tool result
			fmt.Fprint(w, buildSSEResponse([]string{"done"}))
		default:
			// Should not be called more than twice
			http.Error(w, "too many calls", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	a := NewAdapterForTest("sk-tool-test", srv.URL, "gpt-4o")

	sess := agent.NewSession("openai", "gpt-4o")
	sess.Tools = []agent.Tool{echoTool()}
	sess.AppendUserMessage("please echo hello")

	var (
		gotStart      bool
		gotDone       bool
		gotToolCall   *agent.ToolCallEvent
		gotToolResult *agent.ToolResultEvent
		tokens        []string
	)

	for ev, err := range a.Run(context.Background(), sess) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		switch ev.Kind {
		case agent.EventKindStart:
			gotStart = true
		case agent.EventKindDone:
			gotDone = true
		case agent.EventKindToolCall:
			gotToolCall = ev.ToolCall
		case agent.EventKindToolResult:
			gotToolResult = ev.ToolResult
		case agent.EventKindToken:
			tokens = append(tokens, ev.Text)
		case agent.EventKindError:
			t.Fatalf("unexpected error event: %s", ev.Err)
		}
	}

	if !gotStart {
		t.Error("expected EventKindStart")
	}
	if !gotDone {
		t.Error("expected EventKindDone")
	}
	if gotToolCall == nil {
		t.Fatal("expected EventKindToolCall, got none")
	}
	if gotToolCall.ToolName != "echo" {
		t.Errorf("ToolCall.ToolName = %q, want %q", gotToolCall.ToolName, "echo")
	}
	if gotToolResult == nil {
		t.Fatal("expected EventKindToolResult, got none")
	}
	if gotToolResult.Result != "hello" {
		t.Errorf("ToolResult.Result = %q, want %q", gotToolResult.Result, "hello")
	}
	if gotToolResult.IsError {
		t.Error("ToolResult.IsError should be false")
	}
	if len(tokens) == 0 || strings.Join(tokens, "") != "done" {
		t.Errorf("tokens = %v, want [done]", tokens)
	}

	// Verify server was called exactly twice.
	if n := atomic.LoadInt32(&callCount); n != 2 {
		t.Errorf("HTTP handler called %d times, want 2", n)
	}
}

func TestRun_ToolLoop_MaxTurnsExceeded(t *testing.T) {
	var callCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Always return a tool call — never terminates naturally.
		fmt.Fprint(w, buildSSEToolCallResponse("call_loop", "echo", `{"text":"loop"}`))
	}))
	defer srv.Close()

	a := NewAdapterForTest("sk-loop-test", srv.URL, "gpt-4o")

	sess := agent.NewSession("openai", "gpt-4o")
	sess.Tools = []agent.Tool{echoTool()}
	sess.AppendUserMessage("loop forever")

	var gotError bool
	for ev, err := range a.Run(context.Background(), sess) {
		if ev.Kind == agent.EventKindError || err != nil {
			gotError = true
		}
	}

	if !gotError {
		t.Error("expected EventKindError after maxToolTurns exceeded, got none")
	}

	// Verify server was called exactly maxToolTurns times.
	if n := atomic.LoadInt32(&callCount); n != maxToolTurns {
		t.Errorf("HTTP handler called %d times, want %d", n, maxToolTurns)
	}
}

func TestRun_ToolLoop_ToolNotFound(t *testing.T) {
	var callCount int32

	// Multi-turn server: first turn returns a call to an unknown tool;
	// second turn returns plain text after receiving the "tool not found" result.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		switch n {
		case 1:
			fmt.Fprint(w, buildSSEToolCallResponse("call_missing", "nonexistent_tool", `{"x":"y"}`))
		default:
			fmt.Fprint(w, buildSSEResponse([]string{"ok"}))
		}
	}))
	defer srv.Close()

	a := NewAdapterForTest("sk-notfound-test", srv.URL, "gpt-4o")

	sess := agent.NewSession("openai", "gpt-4o")
	// Register an echo tool but NOT nonexistent_tool.
	sess.Tools = []agent.Tool{echoTool()}
	sess.AppendUserMessage("call a missing tool")

	var gotToolResult *agent.ToolResultEvent
	for ev, err := range a.Run(context.Background(), sess) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Kind == agent.EventKindToolResult {
			gotToolResult = ev.ToolResult
		}
	}

	if gotToolResult == nil {
		t.Fatal("expected EventKindToolResult for missing tool, got none")
	}
	if !gotToolResult.IsError {
		t.Error("ToolResult.IsError should be true for missing tool")
	}
	if !strings.Contains(gotToolResult.Result, "nonexistent_tool") {
		t.Errorf("ToolResult.Result = %q, want it to mention the missing tool name", gotToolResult.Result)
	}
}
