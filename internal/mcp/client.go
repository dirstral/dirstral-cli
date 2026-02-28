package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const protocolVersion = "2025-11-25"

type Client struct {
	endpoint   string
	verbose    bool
	authToken  string
	sessionID  string
	httpClient *http.Client
	nextID     int
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResult struct {
	Content           []ContentItem     `json:"content"`
	StructuredContent map[string]any    `json:"structuredContent"`
	IsError           bool              `json:"isError"`
	Raw               map[string]any    `json:"-"`
	Elapsed           time.Duration     `json:"-"`
	HTTPStatus        int               `json:"-"`
	Headers           map[string]string `json:"-"`
}

type jsonRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      *int           `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func New(endpoint string, verbose bool) *Client {
	return &Client{
		endpoint:  endpoint,
		verbose:   verbose,
		authToken: strings.TrimSpace(os.Getenv("DIR2MCP_AUTH_TOKEN")),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		nextID: 1,
	}
}

func (c *Client) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"clientInfo":      map[string]any{"name": "dirstral", "version": "0.1.0"},
	}
	body, status, headers, err := c.call(ctx, "initialize", params, true)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("initialize failed with http status %d", status)
	}
	sessionID := headers.Get("MCP-Session-Id")
	if sessionID == "" {
		return fmt.Errorf("initialize response missing MCP-Session-Id")
	}
	c.sessionID = sessionID

	_ = body
	_, _, _, _ = c.call(ctx, "notifications/initialized", map[string]any{}, false)
	return nil
}

func (c *Client) SessionID() string {
	return c.sessionID
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	body, _, _, err := c.call(ctx, "tools/list", map[string]any{}, true)
	if err != nil {
		return nil, err
	}
	result, ok := body["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid tools/list result")
	}
	items, ok := result["tools"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid tools/list payload")
	}

	tools := make([]Tool, 0, len(items))
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		tools = append(tools, Tool{
			Name:        asString(m["name"]),
			Description: asString(m["description"]),
		})
	}
	return tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*ToolCallResult, error) {
	params := map[string]any{
		"name":      name,
		"arguments": args,
	}
	start := time.Now()
	body, status, headers, err := c.call(ctx, "tools/call", params, true)
	if err != nil {
		return nil, err
	}
	resultMap, ok := body["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid tools/call result")
	}
	content := []ContentItem{}
	if items, ok := resultMap["content"].([]any); ok {
		for _, it := range items {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			content = append(content, ContentItem{Type: asString(m["type"]), Text: asString(m["text"])})
		}
	}
	structured := map[string]any{}
	if sc, ok := resultMap["structuredContent"].(map[string]any); ok {
		structured = sc
	}
	out := &ToolCallResult{
		Content:           content,
		StructuredContent: structured,
		IsError:           asBool(resultMap["isError"]),
		Raw:               body,
		Elapsed:           time.Since(start),
		HTTPStatus:        status,
		Headers:           flattenHeaders(headers),
	}
	return out, nil
}

func (c *Client) call(ctx context.Context, method string, params map[string]any, withID bool) (map[string]any, int, http.Header, error) {
	var id *int
	if withID {
		n := c.nextID
		c.nextID++
		id = &n
	}
	message := jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	payload, err := json.Marshal(message)
	if err != nil {
		return nil, 0, nil, err
	}

	if c.verbose {
		fmt.Printf("\n[mcp] -> %s\n", method)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Protocol-Version", protocolVersion)
	req.Header.Set("Accept", "application/json, text/event-stream")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	if c.sessionID != "" {
		req.Header.Set("MCP-Session-Id", c.sessionID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}
	if len(bodyBytes) == 0 {
		return map[string]any{}, resp.StatusCode, resp.Header, nil
	}

	var envelope jsonRPCResponse
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}
	if envelope.Error != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("json-rpc error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	var raw map[string]any
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}
	if c.verbose {
		fmt.Printf("[mcp] <- %s (%d)\n", method, resp.StatusCode)
	}
	return raw, resp.StatusCode, resp.Header, nil
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = strings.Join(v, ",")
	}
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}
