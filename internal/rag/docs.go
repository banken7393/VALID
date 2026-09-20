package rag

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeSegment = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

// SanitizeSegment rejects path traversal and separators for knowledge category/id.
func SanitizeSegment(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty path segment")
	}
	if strings.Contains(s, "..") || strings.ContainsAny(s, `/\:`) {
		return "", fmt.Errorf("invalid path segment %q", s)
	}
	if !safeSegment.MatchString(s) {
		return "", fmt.Errorf("invalid path segment %q", s)
	}
	return s, nil
}

// Get returns a document by id or relative path.
func (idx *Index) Get(idOrPath string) (Convention, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	idOrPath = strings.TrimSpace(idOrPath)
	for _, d := range idx.docs {
		if d.ID == idOrPath || d.Path == idOrPath {
			return d, true
		}
	}
	return Convention{}, false
}

// List returns all documents, optionally filtered by category.
func (idx *Index) List(category string) []Convention {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := make([]Convention, 0, len(idx.docs))
	for _, d := range idx.docs {
		if category != "" && d.Category != category {
			continue
		}
		out = append(out, d)
	}
	return out
}

// UpsertDocument writes or updates a knowledge fragment with an explicit id.
func (idx *Index) UpsertDocument(id, title, category, body string, tags []string) (Convention, error) {
	id, err := SanitizeSegment(id)
	if err != nil {
		return Convention{}, fmt.Errorf("id: %w", err)
	}
	if category == "" {
		category = "general"
	}
	category, err = SanitizeSegment(category)
	if err != nil {
		return Convention{}, fmt.Errorf("category: %w", err)
	}
	if title == "" {
		title = id
	}
	if tags == nil {
		tags = []string{}
	}
	content := strings.TrimSpace(body)
	fm := FrontMatter{
		ID:       id,
		Title:    title,
		Category: category,
		Domain:   category,
		Tags:     tags,
		Scope:    "project",
	}
	rendered, err := RenderMarkdown(fm, content)
	if err != nil {
		return Convention{}, err
	}
	dir := filepath.Join(idx.root, category)
	// Ensure resolved path stays under knowledge root.
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return Convention{}, err
	}
	absRoot, err := filepath.Abs(idx.root)
	if err != nil {
		return Convention{}, err
	}
	if !strings.HasPrefix(absDir, absRoot+string(os.PathSeparator)) && absDir != absRoot {
		return Convention{}, fmt.Errorf("refusing to write outside knowledge root")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Convention{}, fmt.Errorf("create category dir: %w", err)
	}
	path := filepath.Join(dir, id+".md")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		return Convention{}, fmt.Errorf("write document: %w", err)
	}
	if err := idx.Load(); err != nil {
		return Convention{}, err
	}
	rel := filepath.ToSlash(filepath.Join(category, id+".md"))
	return Convention{
		ID: id, Title: title, Category: category, Domain: category,
		Tags: tags, Scope: "project", Body: content, Path: rel,
	}, nil
}
