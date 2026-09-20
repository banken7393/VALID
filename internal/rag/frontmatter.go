// Package rag indexes Markdown convention files with YAML frontmatter.
package rag

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontMatter is the machine-readable header of a convention Markdown file.
type FrontMatter struct {
	// ID is an optional stable slug identifier.
	ID string `yaml:"id"`
	// Title is the human-readable title.
	Title string `yaml:"title"`
	// Category groups conventions by folder theme (architecture, testing, …).
	Category string `yaml:"category"`
	// Domain is a legacy alias for Category (still accepted when reading).
	Domain string `yaml:"domain"`
	// Tags are free-form labels for retrieval.
	Tags []string `yaml:"tags"`
	// Scope describes applicability (project, procedure, persona, …).
	Scope string `yaml:"scope"`
}

// Convention is one indexed knowledge document.
type Convention struct {
	// ID is a stable slug identifier.
	ID string
	// Title is the human-readable title.
	Title string
	// Category is the primary grouping label.
	Category string
	// Domain mirrors Category for backward-compatible search filters.
	Domain string
	// Tags are free-form labels.
	Tags []string
	// Scope describes applicability.
	Scope string
	// Body is the Markdown body without frontmatter.
	Body string
	// Path is the relative path under the knowledge root.
	Path string
}

// EffectiveCategory returns category, falling back to domain.
func (fm FrontMatter) EffectiveCategory() string {
	if strings.TrimSpace(fm.Category) != "" {
		return fm.Category
	}
	return fm.Domain
}

// ParseMarkdown splits YAML frontmatter from the Markdown body.
func ParseMarkdown(raw string) (FrontMatter, string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return FrontMatter{}, trimmed, nil
	}

	rest := strings.TrimPrefix(trimmed, "---")
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return FrontMatter{}, "", fmt.Errorf("unclosed YAML frontmatter")
	}

	yamlBlock := rest[:end]
	body := strings.TrimSpace(rest[end+4:])

	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return FrontMatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, body, nil
}

// RenderMarkdown builds a convention file from frontmatter and body.
func RenderMarkdown(fm FrontMatter, body string) (string, error) {
	// Normalize: prefer category; keep domain in sync for older readers.
	if fm.Category == "" && fm.Domain != "" {
		fm.Category = fm.Domain
	}
	if fm.Domain == "" {
		fm.Domain = fm.Category
	}
	raw, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(raw)
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n")
	return b.String(), nil
}
