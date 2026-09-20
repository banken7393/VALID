package rag

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Index loads and searches Markdown conventions under a knowledge root.
type Index struct {
	root string
	mu   sync.RWMutex
	docs []Convention
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

// NewIndex creates an empty index rooted at knowledgeDir.
func NewIndex(knowledgeDir string) *Index {
	return &Index{root: knowledgeDir, docs: []Convention{}}
}

// Root returns the knowledge directory path.
func (idx *Index) Root() string {
	return idx.root
}

// Load walks the knowledge tree recursively (unlimited depth) and indexes all .md files.
func (idx *Index) Load() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	docs, err := ScanKnowledge(idx.root)
	if err != nil {
		return err
	}
	idx.docs = docs
	return nil
}

// ScanKnowledge recursively walks knowledgeRoot with filepath.WalkDir and parses every .md file.
func ScanKnowledge(knowledgeRoot string) ([]Convention, error) {
	docs := make([]Convention, 0)
	if err := os.MkdirAll(knowledgeRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create knowledge dir: %w", err)
	}

	err := filepath.WalkDir(knowledgeRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		fm, body, parseErr := ParseMarkdown(string(raw))
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		rel, relErr := filepath.Rel(knowledgeRoot, path)
		if relErr != nil {
			rel = path
		}
		id := fm.ID
		if id == "" {
			id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		title := fm.Title
		if title == "" {
			title = id
		}
		category := fm.EffectiveCategory()
		if category == "" {
			// Infer category from the first path segment under knowledge/.
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) > 1 {
				category = parts[0]
			}
		}
		docs = append(docs, Convention{
			ID:       id,
			Title:    title,
			Category: category,
			Domain:   category,
			Tags:     fm.Tags,
			Scope:    fm.Scope,
			Body:     body,
			Path:     filepath.ToSlash(rel),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// Docs returns a copy of the indexed conventions.
func (idx *Index) Docs() []Convention {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := make([]Convention, len(idx.docs))
	copy(out, idx.docs)
	return out
}

// Add writes or updates a convention Markdown file under the matching category folder.
func (idx *Index) Add(content, category string, tags []string) (Convention, error) {
	if strings.TrimSpace(content) == "" {
		return Convention{}, fmt.Errorf("content is required")
	}
	if category == "" {
		category = "general"
	}
	var err error
	category, err = SanitizeSegment(category)
	if err != nil {
		return Convention{}, fmt.Errorf("category: %w", err)
	}
	if tags == nil {
		tags = []string{}
	}

	title := firstLineTitle(content)
	id := Slugify(title)
	if id == "" {
		id = slugifyFallback(category)
	}
	id, err = SanitizeSegment(id)
	if err != nil {
		return Convention{}, fmt.Errorf("id: %w", err)
	}

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
		return Convention{}, fmt.Errorf("write convention: %w", err)
	}
	if err := idx.Load(); err != nil {
		return Convention{}, err
	}

	rel := filepath.ToSlash(filepath.Join(category, id+".md"))
	return Convention{
		ID:       id,
		Title:    title,
		Category: category,
		Domain:   category,
		Tags:     tags,
		Scope:    "project",
		Body:     strings.TrimSpace(content),
		Path:     rel,
	}, nil
}

func slugifyFallback(category string) string {
	id := Slugify(category + "-convention")
	if id == "" {
		return "convention"
	}
	return id
}

// Slugify turns a title into a filesystem-safe id.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugPattern.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = s[:80]
		s = strings.Trim(s, "-")
	}
	return s
}

// firstLineTitle extracts a title from the first non-empty Markdown line.
func firstLineTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "untitled"
}
