package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// buildRawID creates a *json.RawMessage from a JSON literal.
func buildRawID(v any) *json.RawMessage {
	data, _ := json.Marshal(v)
	raw := json.RawMessage(data)
	return &raw
}

// parseResponse unmarshals a JSON-RPC response into a map for assertions.
func parseResponse(t *testing.T, resp *jsonrpcResponse) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal response: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal response: %v", err)
	}
	return m
}

func TestHandleRequest_Ping(t *testing.T) {
	srv := New()
	id := buildRawID(1)
	req := jsonrpcRequest{JSONRPC: "2.0", ID: id, Method: "ping"}
	resp := srv.handleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("ping must return a response")
	}
	if resp.Error != nil {
		t.Errorf("ping returned error: %v", resp.Error)
	}
}

func TestHandleRequest_UnknownMethod(t *testing.T) {
	srv := New()
	id := buildRawID(42)
	req := jsonrpcRequest{JSONRPC: "2.0", ID: id, Method: "unknown/method"}
	resp := srv.handleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("unknown method with ID must return error response")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != codeMethodNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, codeMethodNotFound)
	}
}

func TestHandleRequest_UnknownNotification(t *testing.T) {
	srv := New()
	// No ID = notification; unknown notifications must be silently ignored.
	req := jsonrpcRequest{JSONRPC: "2.0", Method: "unknown/notification"}
	resp := srv.handleRequest(context.Background(), req)
	if resp != nil {
		t.Errorf("unknown notification must return nil, got %+v", resp)
	}
}

func TestHandleRequest_NotificationsInitialized(t *testing.T) {
	srv := New()
	req := jsonrpcRequest{JSONRPC: "2.0", Method: "notifications/initialized"}
	resp := srv.handleRequest(context.Background(), req)
	if resp != nil {
		t.Errorf("notifications/initialized must return nil, got %+v", resp)
	}
}

func TestHandleInitialize(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(s *Server)
		wantTools     bool
		wantResources bool
		wantPrompts   bool
	}{
		{
			name:  "empty server — no capabilities",
			setup: func(s *Server) {},
		},
		{
			name: "with tool",
			setup: func(s *Server) {
				s.RegisterTool(&noopTool{name: "t"})
			},
			wantTools: true,
		},
		{
			name: "with resource",
			setup: func(s *Server) {
				s.RegisterResource(&noopResource{uri: "file:///r"})
			},
			wantResources: true,
		},
		{
			name: "with prompt",
			setup: func(s *Server) {
				s.RegisterPrompt(&noopPrompt{name: "p"})
			},
			wantPrompts: true,
		},
		{
			name: "all capabilities",
			setup: func(s *Server) {
				s.RegisterTool(&noopTool{name: "t"})
				s.RegisterResource(&noopResource{uri: "file:///r"})
				s.RegisterPrompt(&noopPrompt{name: "p"})
			},
			wantTools:     true,
			wantResources: true,
			wantPrompts:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(WithName("srv"), WithVersion("2.0.0"))
			tt.setup(srv)

			req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "initialize"}
			resp := srv.handleRequest(context.Background(), req)
			if resp == nil {
				t.Fatal("initialize must return a response")
			}
			if resp.Error != nil {
				t.Fatalf("initialize returned error: %v", resp.Error)
			}

			m := parseResponse(t, resp)
			result, ok := m["result"].(map[string]any)
			if !ok {
				t.Fatalf("result is %T, want map", m["result"])
			}

			if result["protocolVersion"] != "2025-03-26" {
				t.Errorf("protocolVersion = %v, want %q", result["protocolVersion"], "2025-03-26")
			}

			info, ok := result["serverInfo"].(map[string]any)
			if !ok {
				t.Fatal("serverInfo must be a map")
			}
			if info["name"] != "srv" {
				t.Errorf("serverInfo.name = %v, want %q", info["name"], "srv")
			}
			if info["version"] != "2.0.0" {
				t.Errorf("serverInfo.version = %v, want %q", info["version"], "2.0.0")
			}

			caps, ok := result["capabilities"].(map[string]any)
			if !ok {
				t.Fatal("capabilities must be a map")
			}
			_, hasTools := caps["tools"]
			_, hasResources := caps["resources"]
			_, hasPrompts := caps["prompts"]

			if hasTools != tt.wantTools {
				t.Errorf("capabilities.tools present = %v, want %v", hasTools, tt.wantTools)
			}
			if hasResources != tt.wantResources {
				t.Errorf("capabilities.resources present = %v, want %v", hasResources, tt.wantResources)
			}
			if hasPrompts != tt.wantPrompts {
				t.Errorf("capabilities.prompts present = %v, want %v", hasPrompts, tt.wantPrompts)
			}
		})
	}
}

func TestHandleToolsList(t *testing.T) {
	srv := New()
	srv.RegisterTool(FuncTool(
		"add", "Add two numbers",
		Schema().Number("a", "First", Required()).Number("b", "Second", Required()).Build(),
		func(_ context.Context, _ map[string]any) ([]Content, error) { return nil, nil },
	))
	srv.RegisterTool(&noopTool{name: "echo"})

	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "tools/list"}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %v", resp.Error)
	}

	m := parseResponse(t, resp)
	result := m["result"].(map[string]any)
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools is %T, want []any", result["tools"])
	}
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}

	// Order must be preserved: "add" first, "echo" second.
	first := tools[0].(map[string]any)
	if first["name"] != "add" {
		t.Errorf("tools[0].name = %v, want %q", first["name"], "add")
	}
	second := tools[1].(map[string]any)
	if second["name"] != "echo" {
		t.Errorf("tools[1].name = %v, want %q", second["name"], "echo")
	}
}

func TestHandleToolsCall_Success(t *testing.T) {
	srv := New()
	srv.RegisterTool(FuncTool(
		"greet", "Greet",
		Schema().String("name", "Name", Required()).Build(),
		func(_ context.Context, args map[string]any) ([]Content, error) {
			name, _ := args["name"].(string)
			return Text("Hello, " + name + "!"), nil
		},
	))

	params, _ := json.Marshal(map[string]any{
		"name":      "greet",
		"arguments": map[string]any{"name": "World"},
	})
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      buildRawID(2),
		Method:  "tools/call",
		Params:  params,
	}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("tools/call error: %v", resp.Error)
	}

	m := parseResponse(t, resp)
	result := m["result"].(map[string]any)
	if result["isError"].(bool) {
		t.Error("isError must be false on success")
	}
	contents := result["content"].([]any)
	if len(contents) != 1 {
		t.Fatalf("content len = %d, want 1", len(contents))
	}
	tc := contents[0].(map[string]any)
	if tc["text"] != "Hello, World!" {
		t.Errorf("content[0].text = %v, want %q", tc["text"], "Hello, World!")
	}
}

func TestHandleToolsCall_ToolError(t *testing.T) {
	srv := New()
	srv.RegisterTool(FuncTool(
		"failing", "Always fails",
		Schema().Build(),
		func(_ context.Context, _ map[string]any) ([]Content, error) {
			return nil, errors.New("boom")
		},
	))

	params, _ := json.Marshal(map[string]any{"name": "failing", "arguments": map[string]any{}})
	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(3), Method: "tools/call", Params: params}
	resp := srv.handleRequest(context.Background(), req)

	// Per MCP spec: tool errors use isError:true in content, not JSON-RPC error.
	if resp.Error != nil {
		t.Fatalf("expected no JSON-RPC error, got: %v", resp.Error)
	}

	m := parseResponse(t, resp)
	result := m["result"].(map[string]any)
	if !result["isError"].(bool) {
		t.Error("isError must be true on tool failure")
	}
	contents := result["content"].([]any)
	if len(contents) == 0 {
		t.Fatal("content must not be empty on error")
	}
}

func TestHandleToolsCall_ToolNotFound(t *testing.T) {
	srv := New()
	params, _ := json.Marshal(map[string]any{"name": "nonexistent", "arguments": map[string]any{}})
	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(4), Method: "tools/call", Params: params}
	resp := srv.handleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error for unknown tool")
	}
	if resp.Error.Code != codeToolNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, codeToolNotFound)
	}
}

func TestHandleToolsCall_InvalidParams(t *testing.T) {
	srv := New()
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      buildRawID(5),
		Method:  "tools/call",
		Params:  json.RawMessage(`{invalid json`),
	}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for invalid JSON params")
	}
	if resp.Error.Code != codeInvalidParams {
		t.Errorf("code = %d, want %d", resp.Error.Code, codeInvalidParams)
	}
}

func TestHandleResourcesList(t *testing.T) {
	srv := New()
	srv.RegisterResource(&noopResource{uri: "file:///a"})
	srv.RegisterResource(&noopResource{uri: "file:///b"})

	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "resources/list"}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("resources/list error: %v", resp.Error)
	}

	m := parseResponse(t, resp)
	result := m["result"].(map[string]any)
	resources := result["resources"].([]any)
	if len(resources) != 2 {
		t.Fatalf("len(resources) = %d, want 2", len(resources))
	}
	// Order preserved.
	if resources[0].(map[string]any)["uri"] != "file:///a" {
		t.Errorf("resources[0].uri = %v, want file:///a", resources[0].(map[string]any)["uri"])
	}
}

func TestHandleResourcesRead_Success(t *testing.T) {
	srv := New()
	srv.RegisterResource(&noopResource{uri: "file:///test"})

	params, _ := json.Marshal(map[string]any{"uri": "file:///test"})
	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "resources/read", Params: params}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}
}

func TestHandleResourcesRead_NotFound(t *testing.T) {
	srv := New()
	params, _ := json.Marshal(map[string]any{"uri": "file:///missing"})
	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "resources/read", Params: params}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for missing resource")
	}
	if resp.Error.Code != codeResourceNotFound {
		t.Errorf("code = %d, want %d", resp.Error.Code, codeResourceNotFound)
	}
}

func TestHandlePromptsList(t *testing.T) {
	srv := New()
	srv.RegisterPrompt(&noopPrompt{name: "p1"})
	srv.RegisterPrompt(&noopPrompt{name: "p2"})

	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "prompts/list"}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("prompts/list error: %v", resp.Error)
	}

	m := parseResponse(t, resp)
	result := m["result"].(map[string]any)
	prompts := result["prompts"].([]any)
	if len(prompts) != 2 {
		t.Fatalf("len(prompts) = %d, want 2", len(prompts))
	}
}

func TestHandlePromptsGet_Success(t *testing.T) {
	srv := New()
	srv.RegisterPrompt(&noopPrompt{name: "greet"})

	params, _ := json.Marshal(map[string]any{"name": "greet", "arguments": map[string]string{}})
	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "prompts/get", Params: params}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("prompts/get error: %v", resp.Error)
	}

	m := parseResponse(t, resp)
	result := m["result"].(map[string]any)
	if _, ok := result["messages"]; !ok {
		t.Error("prompts/get result must have 'messages' key")
	}
}

func TestHandlePromptsGet_NotFound(t *testing.T) {
	srv := New()
	params, _ := json.Marshal(map[string]any{"name": "missing"})
	req := jsonrpcRequest{JSONRPC: "2.0", ID: buildRawID(1), Method: "prompts/get", Params: params}
	resp := srv.handleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for missing prompt")
	}
	if resp.Error.Code != codePromptNotFound {
		t.Errorf("code = %d, want %d", resp.Error.Code, codePromptNotFound)
	}
}
