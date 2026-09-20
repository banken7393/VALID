package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/banken7393/valid/internal/rag"
	"github.com/banken7393/valid/internal/scripts"
)

func ragToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "search_knowledge",
			"description": "Search .valid/knowledge/ by query and optional category/scope/tag filters.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":    map[string]interface{}{"type": "string"},
					"category": map[string]interface{}{"type": "string"},
					"scope":    map[string]interface{}{"type": "string"},
					"tag":      map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			"name":        "list_documents",
			"description": "List knowledge documents, optionally filtered by category.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			"name":        "get_document",
			"description": "Get one knowledge document by id or relative path.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Document id or path"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "upsert_document",
			"description": "Create or update a Markdown fragment under .valid/knowledge/<category>/.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":       map[string]interface{}{"type": "string"},
					"title":    map[string]interface{}{"type": "string"},
					"category": map[string]interface{}{"type": "string"},
					"content":  map[string]interface{}{"type": "string"},
					"tags":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
				"required": []string{"content"},
			},
		},
	}
}

func (s *Server) allToolDefinitions() []map[string]interface{} {
	tools := ragToolDefinitions()
	if !s.allowScripts {
		return tools
	}
	loaded, err := scripts.LoadAll(s.repoRoot, s.scriptsRoot)
	if err != nil || len(loaded) == 0 {
		return tools
	}
	for _, sc := range loaded {
		tools = append(tools, sc.ToolDefinition())
	}
	return tools
}

func (s *Server) runTool(ctx context.Context, name string, args map[string]interface{}) (text string, isErr bool, err error) {
	switch name {
	case "search_knowledge":
		s.mu.Lock()
		defer s.mu.Unlock()
		query, _ := args["query"].(string)
		category, _ := args["category"].(string)
		scope, _ := args["scope"].(string)
		tag, _ := args["tag"].(string)
		if strings.TrimSpace(query) == "" && category == "" && scope == "" && tag == "" {
			return "provide query and/or filters", true, nil
		}
		if err := s.index.Load(); err != nil {
			return "", true, fmt.Errorf("reload knowledge: %w", err)
		}
		hits := s.index.SearchFiltered(rag.SearchQuery{
			Query: query, Category: category, Scope: scope, Tag: tag, Limit: 10,
		})
		if len(hits) == 0 {
			return "No documents matched.", false, nil
		}
		var b strings.Builder
		for i, h := range hits {
			fmt.Fprintf(&b, "%d. [%s] %s (category=%s score=%d path=%s)\n%s\n\n",
				i+1, h.Doc.ID, h.Doc.Title, h.Doc.Category, h.Score, h.Doc.Path, truncate(h.Doc.Body, 400))
		}
		return strings.TrimSpace(b.String()), false, nil

	case "list_documents":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.index.Load(); err != nil {
			return "", true, err
		}
		category, _ := args["category"].(string)
		docs := s.index.List(category)
		raw, _ := json.MarshalIndent(docs, "", "  ")
		return string(raw), false, nil

	case "get_document":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.index.Load(); err != nil {
			return "", true, err
		}
		id, _ := args["id"].(string)
		doc, ok := s.index.Get(id)
		if !ok {
			return "document not found", true, nil
		}
		raw, _ := json.MarshalIndent(doc, "", "  ")
		return string(raw), false, nil

	case "upsert_document":
		s.mu.Lock()
		defer s.mu.Unlock()
		content, _ := args["content"].(string)
		category, _ := args["category"].(string)
		id, _ := args["id"].(string)
		title, _ := args["title"].(string)
		tags := stringSliceArg(args["tags"])
		if strings.TrimSpace(content) == "" {
			return "content is required", true, nil
		}
		var c rag.Convention
		var upsertErr error
		if id != "" {
			c, upsertErr = s.index.UpsertDocument(id, title, category, content, tags)
		} else {
			c, upsertErr = s.index.Add(content, category, tags)
		}
		if upsertErr != nil {
			return upsertErr.Error(), true, nil
		}
		raw, _ := json.MarshalIndent(c, "", "  ")
		return string(raw), false, nil

	default:
		if !s.allowScripts {
			return fmt.Sprintf("unknown tool: %s (scripts disabled on this MCP transport)", name), true, nil
		}
		// Do not hold the RAG mutex across long-running script execution.
		sc, findErr := scripts.Find(s.repoRoot, s.scriptsRoot, name)
		if findErr != nil {
			return fmt.Sprintf("unknown tool: %s (not a RAG tool and no project script with that id)", name), true, nil
		}
		extra := stringSliceArg(args["args"])
		res, runErr := sc.Run(ctx, extra)
		if runErr != nil {
			return runErr.Error(), true, nil
		}
		text := scripts.FormatResult(sc, res)
		isErr = res.ExitCode != 0 || res.TimedOut
		return text, isErr, nil
	}
}

func stringSliceArg(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
