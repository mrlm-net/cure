package gemini_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	gemini "github.com/mrlm-net/cure/internal/agent/gemini"
	"github.com/mrlm-net/cure/pkg/agent"
)

// newTestSession returns a minimal session for testing.
func newTestSession() *agent.Session {
	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.AppendUserMessage("Hello")
	return sess
}

// newTestSessionWithSystem returns a session with a system prompt for testing.
func newTestSessionWithSystem() *agent.Session {
	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.SystemPrompt = "You are a helpful assistant."
	sess.AppendUserMessage("Hello")
	return sess
}

// sseData formats a data line for SSE.
func sseData(data string) string {
	return fmt.Sprintf("data: %s\n\n", data)
}

// validStreamBody returns a complete SSE stream simulating a Gemini response.
func validStreamBody() string {
	first := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{{"text": "Hello"}},
					"role":  "model",
				},
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     10,
			"candidatesTokenCount": 0,
		},
	}
	second := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{{"text": ", world!"}},
					"role":  "model",
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     10,
			"candidatesTokenCount": 5,
		},
	}
	b1, _ := json.Marshal(first)
	b2, _ := json.Marshal(second)
	return sseData(string(b1)) + sseData(string(b2))
}

// TestRun_Success verifies the happy path: EventKindStart first, ≥1 EventKindToken, EventKindDone last.
func TestRun_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, validStreamBody())
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := newTestSession()

	var events []agent.Event
	for ev, err := range a.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, ev)
	}

	if len(events) == 0 {
		t.Fatal("no events received")
	}

	// First event must be EventKindStart.
	if events[0].Kind != agent.EventKindStart {
		t.Errorf("first event kind = %q, want %q", events[0].Kind, agent.EventKindStart)
	}

	// At least one EventKindToken.
	tokenCount := 0
	for _, ev := range events {
		if ev.Kind == agent.EventKindToken {
			tokenCount++
		}
	}
	if tokenCount == 0 {
		t.Error("expected at least 1 EventKindToken, got 0")
	}

	// Last event must be EventKindDone.
	last := events[len(events)-1]
	if last.Kind != agent.EventKindDone {
		t.Errorf("last event kind = %q, want %q", last.Kind, agent.EventKindDone)
	}
}

// TestRun_ErrorStatus verifies that a non-200 response produces EventKindError
// and that the API key does NOT appear in Event.Err.
func TestRun_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error":{"code":401,"message":"API key not valid. key=test-key"}}`)
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := newTestSession()

	var gotError bool
	for ev, err := range a.Run(ctx, sess) {
		if ev.Kind == agent.EventKindError || err != nil {
			gotError = true
			// API key must NOT appear in the error string.
			if strings.Contains(ev.Err, "test-key") {
				t.Errorf("API key leaked in Event.Err: %q", ev.Err)
			}
			if err != nil && strings.Contains(err.Error(), "test-key") {
				t.Errorf("API key leaked in err: %v", err)
			}
		}
	}
	if !gotError {
		t.Error("expected EventKindError for 401, got none")
	}
}

// TestRun_ContextCancel verifies that cancelling the context terminates the
// iterator cleanly.
func TestRun_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("ResponseWriter does not implement http.Flusher")
			return
		}
		// Send one token event.
		ev := map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]any{{"text": "token"}}, "role": "model"}},
			},
		}
		b, _ := json.Marshal(ev)
		fmt.Fprint(w, sseData(string(b)))
		flusher.Flush()

		// Block until client disconnects.
		<-r.Context().Done()
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx, cancel := context.WithCancel(context.Background())
	sess := newTestSession()

	var received int
	for ev, _ := range a.Run(ctx, sess) {
		received++
		if ev.Kind == agent.EventKindStart || ev.Kind == agent.EventKindToken {
			cancel()
			break
		}
	}
	cancel() // ensure cancel is always called

	// We received at least one event (start or token) before cancel.
	if received == 0 {
		t.Error("expected at least one event before cancel")
	}
}

// TestCountTokens_Success verifies that CountTokens returns the correct token count.
func TestCountTokens_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST to the countTokens endpoint.
		if !strings.Contains(r.URL.Path, "countTokens") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"totalTokens":42}`)
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := newTestSession()

	count, err := a.CountTokens(ctx, sess)
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, want 42", count)
	}
}

// TestCountTokens_Error verifies that a non-200 response returns an error.
func TestCountTokens_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"code":500,"message":"internal error"}}`)
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := newTestSession()

	_, err := a.CountTokens(ctx, sess)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

// TestNewGeminiAgent_MissingKey verifies that the factory returns an error
// when the API key environment variable is not set.
func TestNewGeminiAgent_MissingKey(t *testing.T) {
	const envKey = "GEMINI_API_KEY_TEST_MISSING_12345"
	t.Setenv(envKey, "")

	a, err := agent.New("gemini", map[string]any{
		"api_key_env": envKey,
	})
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
	if a != nil {
		t.Error("expected nil agent on error")
	}
	if !strings.Contains(err.Error(), envKey) {
		t.Errorf("error %q does not contain env var name %q", err.Error(), envKey)
	}
}

// TestRoleMapping_AssistantToModel verifies that RoleAssistant is mapped to "model"
// in the request body sent to the Gemini API.
func TestRoleMapping_AssistantToModel(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		capturedBody, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read body: %v", readErr)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"totalTokens":10}`)
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.AppendUserMessage("Hi")
	sess.AppendAssistantMessage("Hello there")
	sess.AppendUserMessage("How are you?")

	_, _ = a.CountTokens(ctx, sess)

	// Inspect captured body for role mapping.
	var reqBody struct {
		Contents []struct {
			Role string `json:"role"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(capturedBody, &reqBody); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}

	roles := make([]string, 0, len(reqBody.Contents))
	for _, c := range reqBody.Contents {
		roles = append(roles, c.Role)
	}

	// Expected roles: user, model, user — assistant must map to "model".
	expected := []string{"user", "model", "user"}
	if len(roles) != len(expected) {
		t.Fatalf("role count = %d, want %d; roles: %v", len(roles), len(expected), roles)
	}
	for i, want := range expected {
		if roles[i] != want {
			t.Errorf("roles[%d] = %q, want %q", i, roles[i], want)
		}
	}
}

// TestSanitiseError verifies that the API key is redacted from error messages.
func TestSanitiseError(t *testing.T) {
	err := errors.New("request failed: key=super-secret-key unauthorized")
	sanitised := gemini.SanitiseError("super-secret-key", err)

	if strings.Contains(sanitised, "super-secret-key") {
		t.Errorf("API key not redacted: %q", sanitised)
	}
	if !strings.Contains(sanitised, "***") {
		t.Errorf("expected *** in sanitised string, got: %q", sanitised)
	}
}

// TestRun_SystemPrompt verifies that a session with SystemPrompt produces
// a systemInstruction field in the request body.
func TestRun_SystemPrompt(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		capturedBody, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read body: %v", readErr)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		ev := map[string]any{
			"candidates": []map[string]any{
				{
					"content":      map[string]any{"parts": []map[string]any{{"text": "hi"}}, "role": "model"},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{"promptTokenCount": 5, "candidatesTokenCount": 2},
		}
		b, _ := json.Marshal(ev)
		fmt.Fprint(w, sseData(string(b)))
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := newTestSessionWithSystem()

	// Drain the iterator.
	for ev, err := range a.Run(ctx, sess) {
		_, _ = ev, err
	}

	var reqBody struct {
		SystemInstruction *struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"systemInstruction"`
	}
	if err := json.Unmarshal(capturedBody, &reqBody); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}
	if reqBody.SystemInstruction == nil {
		t.Fatal("systemInstruction missing from request body")
	}
	if len(reqBody.SystemInstruction.Parts) == 0 {
		t.Fatal("systemInstruction.parts is empty")
	}
	if reqBody.SystemInstruction.Parts[0].Text != "You are a helpful assistant." {
		t.Errorf("systemInstruction text = %q, want %q",
			reqBody.SystemInstruction.Parts[0].Text, "You are a helpful assistant.")
	}
}

// ---- Tool loop tests --------------------------------------------------------

// geminiEchoTool returns a FuncTool that echoes its "text" argument.
func geminiEchoTool() agent.Tool {
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

// buildFunctionCallStream returns an SSE stream containing a single functionCall part.
// Unlike OpenAI, Gemini delivers the complete function call in one event.
func buildFunctionCallStream(toolName string, args map[string]any) string {
	ev := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"functionCall": map[string]any{"name": toolName, "args": args}},
					},
					"role": "model",
				},
				"finishReason": "FUNCTION_CALL",
			},
		},
	}
	b, _ := json.Marshal(ev)
	return sseData(string(b))
}

// TestRun_ToolLoop_SingleCallThenText verifies that a single function call
// followed by a plain-text response completes the tool loop correctly.
func TestRun_ToolLoop_SingleCallThenText(t *testing.T) {
	var callCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		switch n {
		case 1:
			fmt.Fprint(w, buildFunctionCallStream("echo", map[string]any{"text": "hello"}))
		case 2:
			fmt.Fprint(w, validStreamBody())
		default:
			http.Error(w, "too many calls", http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.Tools = []agent.Tool{geminiEchoTool()}
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
	if len(tokens) == 0 {
		t.Error("expected at least one token after tool loop")
	}
	if n := atomic.LoadInt32(&callCount); n != 2 {
		t.Errorf("HTTP handler called %d times, want 2", n)
	}
}

// TestRun_ToolLoop_MaxTurnsExceeded verifies that an EventKindError is emitted
// when the model exceeds the tool turn hard cap.
func TestRun_ToolLoop_MaxTurnsExceeded(t *testing.T) {
	const localMaxTurns = 32 // must match maxToolTurns in gemini.go

	var callCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, buildFunctionCallStream("echo", map[string]any{"text": "loop"}))
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.Tools = []agent.Tool{geminiEchoTool()}
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
	if n := atomic.LoadInt32(&callCount); n != int32(localMaxTurns) {
		t.Errorf("HTTP handler called %d times, want %d", n, localMaxTurns)
	}
}

// TestRun_ToolLoop_ToolNotFound verifies that an unknown tool call results in
// an IsError=true ToolResultEvent (not a fatal error), and the loop continues.
func TestRun_ToolLoop_ToolNotFound(t *testing.T) {
	var callCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		switch n {
		case 1:
			fmt.Fprint(w, buildFunctionCallStream("nonexistent_tool", map[string]any{"x": "y"}))
		default:
			fmt.Fprint(w, validStreamBody())
		}
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.Tools = []agent.Tool{geminiEchoTool()}
	sess.AppendUserMessage("call unknown tool")

	var (
		gotToolResult *agent.ToolResultEvent
		gotFatalErr   bool
	)
	for ev, err := range a.Run(context.Background(), sess) {
		if err != nil || ev.Kind == agent.EventKindError {
			gotFatalErr = true
		}
		if ev.Kind == agent.EventKindToolResult {
			gotToolResult = ev.ToolResult
		}
	}

	if gotFatalErr {
		t.Error("expected no fatal error for unknown tool — should yield IsError=true result instead")
	}
	if gotToolResult == nil {
		t.Fatal("expected EventKindToolResult, got none")
	}
	if !gotToolResult.IsError {
		t.Error("ToolResult.IsError should be true for unknown tool")
	}
	if !strings.Contains(gotToolResult.Result, "nonexistent_tool") {
		t.Errorf("ToolResult.Result = %q, want it to mention the tool name", gotToolResult.Result)
	}
}

// TestRun_ToolLoop_ToolsIncludedInRequest verifies that registered tools are
// serialised into the Gemini request as functionDeclarations.
func TestRun_ToolLoop_ToolsIncludedInRequest(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, validStreamBody())
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.Tools = []agent.Tool{geminiEchoTool()}
	sess.AppendUserMessage("test tools in request")

	for _, _ = range a.Run(context.Background(), sess) {
	}

	var reqBody struct {
		Tools []struct {
			FunctionDeclarations []struct {
				Name string `json:"name"`
			} `json:"functionDeclarations"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(capturedBody, &reqBody); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}
	if len(reqBody.Tools) == 0 {
		t.Fatal("tools array missing from request body")
	}
	if len(reqBody.Tools[0].FunctionDeclarations) == 0 {
		t.Fatal("functionDeclarations missing from tools[0]")
	}
	if reqBody.Tools[0].FunctionDeclarations[0].Name != "echo" {
		t.Errorf("functionDeclarations[0].name = %q, want %q",
			reqBody.Tools[0].FunctionDeclarations[0].Name, "echo")
	}
}
