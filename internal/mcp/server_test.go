package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/banken7393/valid/internal/rag"
)

func TestToolsListIsRAGOnly(t *testing.T) {
	root := t.TempDir()
	idx := rag.NewIndex(root)
	if _, err := idx.Add("# TDD First\nWrite tests before code.\n", "testing", []string{"tdd"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	srv := NewServer(idx)

	listResp := srv.Handle(context.Background(), jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	})
	if listResp == nil || listResp.Error != nil {
		t.Fatalf("tools/list failed: %+v", listResp)
	}
	rawList, _ := json.Marshal(listResp.Result)
	for _, name := range []string{"search_knowledge", "list_documents", "get_document", "upsert_document"} {
		if !strings.Contains(string(rawList), name) {
			t.Fatalf("tools/list missing %s: %s", name, rawList)
		}
	}
	for _, banned := range []string{"report_tests", "bump_loop", "update_spec", "valid report"} {
		if strings.Contains(string(rawList), banned) {
			t.Fatalf("tools/list must not expose %s: %s", banned, rawList)
		}
	}

	params, _ := json.Marshal(map[string]interface{}{
		"name": "search_knowledge",
		"arguments": map[string]interface{}{
			"query":    "tdd tests",
			"category": "testing",
		},
	})
	callResp := srv.Handle(context.Background(), jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params:  params,
	})
	if callResp == nil || callResp.Error != nil {
		t.Fatalf("tools/call failed: %+v", callResp)
	}
	raw, _ := json.Marshal(callResp.Result)
	if !strings.Contains(string(raw), "TDD") && !strings.Contains(string(raw), "tdd") {
		t.Fatalf("unexpected result: %s", raw)
	}
}

func TestUnknownContractToolRejected(t *testing.T) {
	idx := rag.NewIndex(t.TempDir())
	_ = idx.Load()
	srv := NewServer(idx)

	params, _ := json.Marshal(map[string]interface{}{
		"name": "report_tests",
		"arguments": map[string]interface{}{
			"tests": []interface{}{},
		},
	})
	resp := srv.Handle(context.Background(), jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      99,
		Method:  "tools/call",
		Params:  params,
	})
	if resp == nil || resp.Error != nil {
		t.Fatalf("unexpected RPC error: %+v", resp)
	}
	raw, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(raw), "unknown tool") {
		t.Fatalf("expected rejection, got %s", raw)
	}
}

func TestProjectScriptExposedAndRunnable(t *testing.T) {
	repo := t.TempDir()
	scriptsRoot := filepath.Join(repo, ".valid", "scripts")
	dir := filepath.Join(scriptsRoot, "greet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "meta.json"), []byte(`{
  "description": "Greet via project script MCP node",
  "command": ["bash", "run.sh"],
  "timeout_sec": 10
}`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/usr/bin/env bash\necho greet:$1\n"), 0o755)

	idx := rag.NewIndex(t.TempDir())
	_ = idx.Load()
	srv := NewServer(idx, Options{RepoRoot: repo, ScriptsRoot: scriptsRoot})

	listResp := srv.Handle(context.Background(), jsonRPCRequest{
		JSONRPC: "2.0", ID: 1, Method: "tools/list",
	})
	rawList, _ := json.Marshal(listResp.Result)
	if !strings.Contains(string(rawList), "greet") {
		t.Fatalf("expected greet tool, got %s", rawList)
	}

	params, _ := json.Marshal(map[string]interface{}{
		"name": "greet",
		"arguments": map[string]interface{}{
			"args": []interface{}{"bob"},
		},
	})
	callResp := srv.Handle(context.Background(), jsonRPCRequest{
		JSONRPC: "2.0", ID: 2, Method: "tools/call", Params: params,
	})
	if callResp == nil || callResp.Error != nil {
		t.Fatalf("call failed: %+v", callResp)
	}
	raw, _ := json.Marshal(callResp.Result)
	if !strings.Contains(string(raw), "greet:bob") {
		t.Fatalf("unexpected: %s", raw)
	}
	if strings.Contains(string(raw), `"isError":true`) {
		t.Fatalf("expected success, got %s", raw)
	}
}

func TestHTTPRejectsNonLoopbackHost(t *testing.T) {
	idx := rag.NewIndex(t.TempDir())
	_ = idx.Load()
	srv := NewServer(idx, Options{HTTPToken: "secret", ScriptsDisabled: true})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ServeHTTP(ctx, "evil.example:17433")
	}()
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "localhost") {
			t.Fatalf("expected localhost rejection, got %v", err)
		}
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("ServeHTTP should reject non-loopback hostname immediately")
	}
}

func TestHTTPRequiresToken(t *testing.T) {
	idx := rag.NewIndex(t.TempDir())
	_ = idx.Load()
	srv := NewServer(idx, Options{ScriptsDisabled: true})
	err := srv.ServeHTTP(context.Background(), "127.0.0.1:17434")
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("expected token required, got %v", err)
	}
}

func TestScriptsDisabledOnOptions(t *testing.T) {
	root := t.TempDir()
	scriptDir := filepath.Join(root, ".valid", "scripts", "hello_tool")
	_ = os.MkdirAll(scriptDir, 0o755)
	_ = os.WriteFile(filepath.Join(scriptDir, "meta.json"), []byte(`{"description":"hi","command":["echo","hi"]}`+"\n"), 0o644)
	idx := rag.NewIndex(filepath.Join(root, "k"))
	_ = os.MkdirAll(filepath.Join(root, "k"), 0o755)
	_ = idx.Load()
	srv := NewServer(idx, Options{RepoRoot: root, ScriptsRoot: ".valid/scripts", ScriptsDisabled: true})
	listResp := srv.Handle(context.Background(), jsonRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	raw, _ := json.Marshal(listResp.Result)
	if strings.Contains(string(raw), "hello_tool") {
		t.Fatalf("scripts should be disabled: %s", raw)
	}
}
