package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// Client connects to a remote MCP server and invokes tools.
type Client struct {
	endpoint string
	token    string
	client   *http.Client
	nextID   atomic.Int64
}

// ClientOption configures an MCP Client.
type ClientOption func(*Client)

// WithBearerToken sets the Bearer token for authentication.
func WithBearerToken(token string) ClientOption {
	return func(c *Client) { c.token = token }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.client = hc }
}

// NewClient creates an MCP client for the given HTTP Streamable endpoint.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 5 * time.Minute},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ToolInfo describes a tool available on the remote server.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CallTool invokes a tool on the remote MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	id := c.nextID.Add(1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      name,
			"arguments": args,
		},
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", fmt.Errorf("mcp client: call %q: %w", name, err)
	}

	if resp.Error != nil {
		return "", fmt.Errorf("mcp client: call %q: code %d: %s", name, resp.Error.Code, resp.Error.Message)
	}

	result, err := json.Marshal(resp.Result)
	if err != nil {
		return "", fmt.Errorf("mcp client: marshal result: %w", err)
	}
	return string(result), nil
}

// ListTools returns the tools available on the remote server.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	id := c.nextID.Add(1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/list",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("mcp client: list tools: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("mcp client: list tools: code %d: %s", resp.Error.Code, resp.Error.Message)
	}

	var result struct {
		Tools []ToolInfo `json:"tools"`
	}
	data, _ := json.Marshal(resp.Result)
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("mcp client: parse tools: %w", err)
	}
	return result.Tools, nil
}

func (c *Client) doRequest(ctx context.Context, rpcReq jsonRPCRequest) (*jsonRPCResponse, error) {
	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &rpcResp, nil
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
