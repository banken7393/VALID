// Package mcp implements a minimal MCP server (JSON-RPC) with stdio and HTTP transports.
// Tools: RAG over .valid/knowledge/ + project scripts under .valid/scripts/ (dynamic nodes).
// Never mutates feature boards / TDD.
//
// HTTP transport is localhost-oriented and requires a bearer token; scripts are disabled
// on HTTP unless explicitly allowed (stdio is the trusted path for script execution).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/banken7393/valid/internal/rag"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "valid"
	serverVersion   = "0.4.0"
)

// Options configures optional MCP surfaces beyond RAG.
type Options struct {
	RepoRoot         string // absolute repo root (required for scripts)
	ScriptsRoot      string // absolute or repo-relative scripts directory
	ScriptsDisabled  bool   // when true, project scripts are not listed/runnable (HTTP default)
	HTTPToken        string // required for HTTP; compared to Authorization: Bearer
}

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Server is the VALID MCP tool host (RAG + optional project scripts).
type Server struct {
	index        *rag.Index
	repoRoot     string
	scriptsRoot  string
	allowScripts bool
	httpToken    string
	mu           sync.Mutex
}

// NewServer creates an MCP server. Pass Options to enable project script tools.
func NewServer(index *rag.Index, opts ...Options) *Server {
	s := &Server{index: index, allowScripts: true}
	if len(opts) > 0 {
		s.repoRoot = opts[0].RepoRoot
		s.scriptsRoot = opts[0].ScriptsRoot
		s.allowScripts = !opts[0].ScriptsDisabled
		s.httpToken = opts[0].HTTPToken
		if s.scriptsRoot != "" && s.repoRoot != "" && !filepath.IsAbs(s.scriptsRoot) {
			s.scriptsRoot = filepath.Join(s.repoRoot, s.scriptsRoot)
		}
	}
	return s
}

// Handle processes one JSON-RPC request and returns a response (nil for notifications).
func (s *Server) Handle(ctx context.Context, req jsonRPCRequest) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return s.respond(req.ID, map[string]interface{}{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    serverName,
				"version": serverVersion,
			},
		})
	case "notifications/initialized", "initialized":
		return nil
	case "ping":
		return s.respond(req.ID, map[string]interface{}{})
	case "tools/list":
		return s.respond(req.ID, map[string]interface{}{
			"tools": s.allToolDefinitions(),
		})
	case "tools/call":
		return s.callTool(ctx, req)
	default:
		if req.ID == nil {
			return nil
		}
		return s.fail(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) callTool(ctx context.Context, req jsonRPCRequest) *jsonRPCResponse {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.fail(req.ID, -32602, "invalid tools/call params")
	}
	if params.Arguments == nil {
		params.Arguments = map[string]interface{}{}
	}

	text, isErr, err := s.runTool(ctx, params.Name, params.Arguments)
	if err != nil {
		return s.fail(req.ID, -32000, err.Error())
	}
	return s.respond(req.ID, map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
		"isError": isErr,
	})
}

func (s *Server) respond(id interface{}, result interface{}) *jsonRPCResponse {
	return &jsonRPCResponse{JSONRPC: "2.0", Result: result, ID: id}
}

func (s *Server) fail(id interface{}, code int, message string) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
}

// HandleBytes unmarshals raw JSON and handles the request.
func (s *Server) HandleBytes(ctx context.Context, raw []byte) ([]byte, error) {
	var req jsonRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		errResp := jsonRPCResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32700, Message: "parse error"},
		}
		return json.Marshal(errResp)
	}
	resp := s.Handle(ctx, req)
	if resp == nil {
		return nil, nil
	}
	return json.Marshal(resp)
}
