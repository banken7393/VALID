package rag

import (
	"sort"
	"strings"
	"unicode"
)

// Hit is one ranked search result.
type Hit struct {
	// Doc is the matched convention.
	Doc Convention
	// Score is the token-overlap relevance score.
	Score int
}

// SearchQuery holds optional filters for hierarchical knowledge search.
type SearchQuery struct {
	// Query is the free-text / keyword search string.
	Query string
	// Category filters by frontmatter category (or inferred folder).
	Category string
	// Scope filters by frontmatter scope.
	Scope string
	// Tag requires this tag to be present.
	Tag string
	// Limit caps the number of hits (0 = all).
	Limit int
}

// Search ranks conventions by token overlap with query, optionally filtered by domain/category.
// domain is accepted as a category alias for MCP tool compatibility.
func (idx *Index) Search(query, domain string, limit int) []Hit {
	return idx.SearchFiltered(SearchQuery{
		Query:    query,
		Category: domain,
		Limit:    limit,
	})
}

// SearchFiltered ranks conventions across the full hierarchical corpus.
func (idx *Index) SearchFiltered(q SearchQuery) []Hit {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	tokens := tokenize(q.Query)
	category := strings.TrimSpace(strings.ToLower(q.Category))
	scope := strings.TrimSpace(strings.ToLower(q.Scope))
	tag := strings.TrimSpace(strings.ToLower(q.Tag))

	type scored struct {
		doc   Convention
		score int
	}
	out := make([]scored, 0, len(idx.docs))
	for _, d := range idx.docs {
		if category != "" && strings.ToLower(d.Category) != category && strings.ToLower(d.Domain) != category {
			continue
		}
		if scope != "" && strings.ToLower(d.Scope) != scope {
			continue
		}
		if tag != "" && !hasTag(d.Tags, tag) {
			continue
		}

		s := 0
		if len(tokens) == 0 {
			// Metadata-only filter: include all matching docs with a base score.
			s = 1
		} else {
			for tok := range tokens {
				if tokenize(d.Title)[tok] || tokenize(d.ID)[tok] {
					s += 4
				}
				if tokenize(strings.Join(d.Tags, " "))[tok] {
					s += 3
				}
				if tokenize(d.Category)[tok] || tokenize(d.Domain)[tok] {
					s += 2
				}
				if tokenize(d.Scope)[tok] {
					s += 2
				}
				if tokenize(d.Path)[tok] {
					s += 2
				}
				if tokenize(d.Body)[tok] {
					s++
				}
			}
		}
		if s > 0 {
			out = append(out, scored{doc: d, score: s})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score == out[j].score {
			return out[i].doc.Path < out[j].doc.Path
		}
		return out[i].score > out[j].score
	})

	limit := q.Limit
	if limit <= 0 || limit > len(out) {
		limit = len(out)
	}
	hits := make([]Hit, 0, limit)
	for _, s := range out[:limit] {
		hits = append(hits, Hit{Doc: s.doc, Score: s.score})
	}
	return hits
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if strings.ToLower(t) == want {
			return true
		}
	}
	return false
}

// tokenize returns a set of lowercased alphanumeric tokens (length > 1).
func tokenize(s string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		if len(f) > 1 {
			out[f] = true
		}
	}
	return out
}
