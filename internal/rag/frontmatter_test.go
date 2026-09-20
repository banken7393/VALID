package rag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMarkdownFrontmatter(t *testing.T) {
	raw := "---\ntitle: Go Tests\ncategory: testing\nscope: project\ntags:\n  - tdd\n---\n\n# Go Tests\nAlways write tests first.\n"
	fm, body, err := ParseMarkdown(raw)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if fm.Title != "Go Tests" {
		t.Fatalf("Title = %q", fm.Title)
	}
	if fm.EffectiveCategory() != "testing" {
		t.Fatalf("Category = %q", fm.EffectiveCategory())
	}
	if fm.Scope != "project" {
		t.Fatalf("Scope = %q", fm.Scope)
	}
	if !strings.Contains(body, "Always write tests first") {
		t.Fatalf("body missing content: %q", body)
	}
}

func TestScanKnowledgeRecursiveNested(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "architecture", "deep", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntitle: Deep Doc\ncategory: architecture\ntags:\n  - deep\nscope: project\n---\n\nNested knowledge.\n"
	if err := os.WriteFile(filepath.Join(nested, "deep.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Sibling at top level
	if err := os.MkdirAll(filepath.Join(root, "testing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "testing", "standards.md"), []byte("---\ntitle: Standards\ncategory: testing\n---\n\nTDD.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	docs, err := ScanKnowledge(root)
	if err != nil {
		t.Fatalf("ScanKnowledge: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("docs = %d, want 2", len(docs))
	}
	foundDeep := false
	for _, d := range docs {
		if strings.Contains(d.Path, "deep/nested/deep.md") {
			foundDeep = true
			if d.Category != "architecture" {
				t.Fatalf("category = %q", d.Category)
			}
		}
	}
	if !foundDeep {
		t.Fatalf("missing nested doc: %+v", docs)
	}
}

func TestIndexAddAndSearch(t *testing.T) {
	root := t.TempDir()
	idx := NewIndex(root)
	if err := idx.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	c, err := idx.Add("# Worktree Quarantine\nIsolate features in git worktrees.\n", "git", []string{"isolation"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if c.Category != "git" {
		t.Fatalf("Category = %q", c.Category)
	}
	if _, err := os.Stat(filepath.Join(root, "git", c.ID+".md")); err != nil {
		t.Fatalf("expected file: %v", err)
	}

	hits := idx.Search("worktree isolation", "git", 5)
	if len(hits) == 0 {
		t.Fatal("expected at least one hit")
	}
	if hits[0].Doc.ID != c.ID {
		t.Fatalf("hit ID = %q, want %q", hits[0].Doc.ID, c.ID)
	}

	// Metadata-only filter
	meta := idx.SearchFiltered(SearchQuery{Category: "git", Limit: 5})
	if len(meta) == 0 {
		t.Fatal("expected category filter hits")
	}
}
