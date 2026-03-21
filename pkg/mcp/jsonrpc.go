package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// jsonrpcRequest is the wire format for a JSON-RPC 2.0 request or notification.
// Notifications have a nil ID.
type jsonrpcRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

// jsonrpcResponse is the wire format for a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string           `json:"jsonrpc"` // always "2.0"
	ID      *json.RawMessage `json:"id"`
	Result  any              `json:"result,omitempty"`
	Error   *jsonrpcError    `json:"error,omitempty"`
}

// jsonrpcError represents the error object in a JSON-RPC 2.0 error response.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// MCP-specific application error codes (in the reserved range).
const (
	codeToolNotFound     = -32000
	codeResourceNotFound = -32001
	codePromptNotFound   = -32002
	codeToolCallFailed   = -32003
)

// errResponse constructs a JSON-RPC 2.0 error response with the given ID, code,
// and message.
func errResponse(id *json.RawMessage, code int, msg string) *jsonrpcResponse {
	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonrpcError{Code: code, Message: msg},
	}
}

// okResponse constructs a JSON-RPC 2.0 success response.
func okResponse(id *json.RawMessage, result any) *jsonrpcResponse {
	return &jsonrpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

// handleRequest dispatches a parsed JSON-RPC 2.0 request to the appropriate
// handler method. Returns nil for notifications (requests without an ID that
// require no response).
func (s *Server) handleRequest(ctx context.Context, req jsonrpcRequest) *jsonrpcResponse {
	if s.logger != nil {
		s.logger.DebugContext(ctx, "mcp: dispatching", "method", req.Method)
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(ctx, req)
	case "notifications/initialized":
		return nil // notification — no response
	case "ping":
		return okResponse(req.ID, struct{}{})
	case "tools/list":
		return s.handleToolsList(ctx, req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "resources/list":
		return s.handleResourcesList(ctx, req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
	case "prompts/list":
		return s.handlePromptsList(ctx, req)
	case "prompts/get":
		return s.handlePromptsGet(ctx, req)
	default:
		if req.ID == nil {
			return nil // unknown notification — ignore per spec
		}
		return errResponse(req.ID, codeMethodNotFound, "method not found: "+req.Method)
	}
}

// ---- MCP capability and response types ----

// initializeResult is the response payload for the initialize method.
type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      serverInfo         `json:"serverInfo"`
	Capabilities    serverCapabilities `json:"capabilities"`
}

// serverInfo identifies the server in the initialize response.
type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// serverCapabilities declares which MCP capability groups the server supports.
type serverCapabilities struct {
	Tools     *capabilityEntry `json:"tools,omitempty"`
	Resources *capabilityEntry `json:"resources,omitempty"`
	Prompts   *capabilityEntry `json:"prompts,omitempty"`
}

// capabilityEntry is the value for each capability key.
type capabilityEntry struct {
	ListChanged bool `json:"listChanged"`
}

// ---- Handler implementations ----

// handleInitialize responds with the server's protocol version, identity, and
// capabilities. Capability sections are only included if at least one item of
// that type is registered.
func (s *Server) handleInitialize(_ context.Context, req jsonrpcRequest) *jsonrpcResponse {
	s.mu.RLock()
	nTools := len(s.tools)
	nResources := len(s.resources)
	nPrompts := len(s.prompts)
	s.mu.RUnlock()

	caps := serverCapabilities{}
	if nTools > 0 {
		caps.Tools = &capabilityEntry{}
	}
	if nResources > 0 {
		caps.Resources = &capabilityEntry{}
	}
	if nPrompts > 0 {
		caps.Prompts = &capabilityEntry{}
	}

	result := initializeResult{
		ProtocolVersion: "2025-03-26",
		ServerInfo:      serverInfo{Name: s.name, Version: s.version},
		Capabilities:    caps,
	}
	return okResponse(req.ID, result)
}

// toolListEntry is the per-tool payload in the tools/list response.
type toolListEntry struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// handleToolsList returns all registered tools in registration order.
func (s *Server) handleToolsList(_ context.Context, req jsonrpcRequest) *jsonrpcResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]toolListEntry, 0, len(s.toolOrder))
	for _, name := range s.toolOrder {
		t := s.tools[name]
		tools = append(tools, toolListEntry{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Schema(),
		})
	}
	return okResponse(req.ID, map[string]any{"tools": tools})
}

// toolCallParams is the decoded parameter block for tools/call.
type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// toolCallResult is the response payload for a successful or failed tool call.
// MCP spec: on tool error return isError:true with text content, not a JSON-RPC error.
type toolCallResult struct {
	Content []any `json:"content"`
	IsError bool  `json:"isError"`
}

// handleToolsCall dispatches a tools/call request to the named tool.
func (s *Server) handleToolsCall(ctx context.Context, req jsonrpcRequest) *jsonrpcResponse {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errResponse(req.ID, codeInvalidParams, "invalid params: "+err.Error())
	}
	if params.Name == "" {
		return errResponse(req.ID, codeInvalidParams, "missing tool name")
	}

	s.mu.RLock()
	tool, ok := s.tools[params.Name]
	s.mu.RUnlock()

	if !ok {
		return errResponse(req.ID, codeToolNotFound,
			fmt.Sprintf("tool not found: %s", params.Name))
	}

	args := params.Arguments
	if args == nil {
		args = make(map[string]any)
	}

	contents, err := tool.Call(ctx, args)
	if err != nil {
		// Per MCP spec: tool errors are represented in content, not as JSON-RPC errors.
		errText := (&ToolCallError{Tool: params.Name, Err: err}).Error()
		return okResponse(req.ID, toolCallResult{
			Content: contentSliceToAny(Text(errText)),
			IsError: true,
		})
	}

	return okResponse(req.ID, toolCallResult{
		Content: contentSliceToAny(contents),
		IsError: false,
	})
}

// resourceListEntry is the per-resource payload in the resources/list response.
type resourceListEntry struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

// handleResourcesList returns all registered resources in registration order.
func (s *Server) handleResourcesList(_ context.Context, req jsonrpcRequest) *jsonrpcResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]resourceListEntry, 0, len(s.resOrder))
	for _, uri := range s.resOrder {
		r := s.resources[uri]
		resources = append(resources, resourceListEntry{
			URI:         r.URI(),
			Name:        r.Name(),
			Description: r.Description(),
			MIMEType:    r.MIMEType(),
		})
	}
	return okResponse(req.ID, map[string]any{"resources": resources})
}

// resourceReadParams is the decoded parameter block for resources/read.
type resourceReadParams struct {
	URI string `json:"uri"`
}

// handleResourcesRead reads a single resource by URI.
func (s *Server) handleResourcesRead(ctx context.Context, req jsonrpcRequest) *jsonrpcResponse {
	var params resourceReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errResponse(req.ID, codeInvalidParams, "invalid params: "+err.Error())
	}
	if params.URI == "" {
		return errResponse(req.ID, codeInvalidParams, "missing resource URI")
	}

	s.mu.RLock()
	r, ok := s.resources[params.URI]
	s.mu.RUnlock()

	if !ok {
		return errResponse(req.ID, codeResourceNotFound,
			fmt.Sprintf("resource not found: %s", params.URI))
	}

	contents, err := r.Read(ctx)
	if err != nil {
		return errResponse(req.ID, codeInternalError,
			fmt.Sprintf("resource read error: %v", err))
	}
	return okResponse(req.ID, map[string]any{"contents": contents})
}

// promptListEntry is the per-prompt payload in the prompts/list response.
type promptListEntry struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// handlePromptsList returns all registered prompts in registration order.
func (s *Server) handlePromptsList(_ context.Context, req jsonrpcRequest) *jsonrpcResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := make([]promptListEntry, 0, len(s.prmOrder))
	for _, name := range s.prmOrder {
		p := s.prompts[name]
		prompts = append(prompts, promptListEntry{
			Name:        p.Name(),
			Description: p.Description(),
			Arguments:   p.Arguments(),
		})
	}
	return okResponse(req.ID, map[string]any{"prompts": prompts})
}

// promptGetParams is the decoded parameter block for prompts/get.
type promptGetParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
}

// handlePromptsGet retrieves a rendered prompt by name.
func (s *Server) handlePromptsGet(ctx context.Context, req jsonrpcRequest) *jsonrpcResponse {
	var params promptGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errResponse(req.ID, codeInvalidParams, "invalid params: "+err.Error())
	}
	if params.Name == "" {
		return errResponse(req.ID, codeInvalidParams, "missing prompt name")
	}

	s.mu.RLock()
	p, ok := s.prompts[params.Name]
	s.mu.RUnlock()

	if !ok {
		return errResponse(req.ID, codePromptNotFound,
			fmt.Sprintf("prompt not found: %s", params.Name))
	}

	args := params.Arguments
	if args == nil {
		args = make(map[string]string)
	}

	messages, err := p.Get(ctx, args)
	if err != nil {
		return errResponse(req.ID, codeInternalError,
			fmt.Sprintf("prompt get error: %v", err))
	}

	// Serialize messages with proper content marshalling.
	result := map[string]any{
		"description": p.Description(),
		"messages":    marshalMessages(messages),
	}
	return okResponse(req.ID, result)
}

// ---- Content marshalling helpers ----

// contentSliceToAny converts []Content to []any for JSON marshalling.
// Each concrete Content type (TextContent, ImageContent, ResourceContent) has
// exported fields with proper json tags, so encoding/json handles them correctly
// as long as the Type field is pre-set (which Text/Textf/constructors ensure).
func contentSliceToAny(cs []Content) []any {
	out := make([]any, len(cs))
	for i, c := range cs {
		out[i] = c
	}
	return out
}

// marshalMessages converts []Message to a JSON-serialisable form. The Content
// field of each message must be serialised via contentSliceToAny so that
// concrete types are preserved through the interface boundary.
func marshalMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, len(msgs))
	for i, m := range msgs {
		out[i] = map[string]any{
			"role":    m.Role,
			"content": contentSliceToAny(m.Content),
		}
	}
	return out
}
