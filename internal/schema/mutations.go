package schema

import (
	"fmt"
	"strings"
)

// AddDecision appends a decision entry.
func (b *Board) AddDecision(id, title, detail string, tags []string) error {
	if b == nil {
		return fmt.Errorf("board is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("decision id is required")
	}
	if tags == nil {
		tags = []string{}
	}
	for i := range b.Decisions {
		if b.Decisions[i].ID == id {
			b.Decisions[i].Title = title
			b.Decisions[i].Detail = detail
			b.Decisions[i].Tags = tags
			b.Touch()
			return b.Validate()
		}
	}
	b.Decisions = append(b.Decisions, Decision{ID: id, Title: title, Detail: detail, Tags: tags})
	b.Touch()
	return b.Validate()
}

// AddPromotion appends a pending promotion.
func (b *Board) AddPromotion(id, kind, description, target string) error {
	if b == nil {
		return fmt.Errorf("board is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("promotion id is required")
	}
	b.PendingPromotions = append(b.PendingPromotions, Promotion{
		ID: id, Kind: kind, Description: description, Target: target, Applied: false,
	})
	b.Touch()
	return b.Validate()
}
