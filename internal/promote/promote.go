// Package promote merges feature knowledge into the MCP corpus and applies env deltas.
package promote

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/banken7393/valid/internal/env"
	"github.com/banken7393/valid/internal/rag"
	"github.com/banken7393/valid/internal/schema"
)

// KnowledgeResult describes a knowledge promote run.
type KnowledgeResult struct {
	DryRun      bool
	FichaPath   string
	UpdatedDocs []string
	CreatedDocs []string
	Message     string
}

// PromoteKnowledge merges board decisions/assumptions into knowledge fragments + writes the short ficha.
func PromoteKnowledge(repoRoot string, b *schema.Board, cfg schema.Config, dryRun bool) (*KnowledgeResult, error) {
	if b == nil {
		return nil, fmt.Errorf("board is nil")
	}
	knowRoot := schema.KnowledgePath(repoRoot, cfg)
	idx := rag.NewIndex(knowRoot)
	if err := idx.Load(); err != nil {
		return nil, err
	}

	res := &KnowledgeResult{DryRun: dryRun}
	fichaRel := filepath.ToSlash(filepath.Join("features", b.Feature+".md"))
	fichaAbs := filepath.Join(knowRoot, "features", b.Feature+".md")
	res.FichaPath = fichaAbs

	if !dryRun {
		if err := os.MkdirAll(filepath.Dir(fichaAbs), 0o755); err != nil {
			return nil, err
		}
	}

	for _, d := range b.Decisions {
		category := "decisions"
		body := fmt.Sprintf("# %s\n\n%s\n\n_Promoted from feature `%s`._\n", d.Title, d.Detail, b.Feature)
		docID := d.ID
		if docID == "" {
			docID = rag.Slugify(d.Title)
		}
		pathHint := category + "/" + docID + ".md"
		if dryRun {
			res.CreatedDocs = append(res.CreatedDocs, pathHint)
			continue
		}
		c, err := idx.UpsertDocument(docID, d.Title, category, body, d.Tags)
		if err != nil {
			return nil, fmt.Errorf("promote decision %s: %w", d.ID, err)
		}
		res.UpdatedDocs = append(res.UpdatedDocs, c.Path)
	}

	for _, a := range b.Assumptions {
		detail := strings.TrimSpace(a.Detail)
		if detail == "" {
			continue
		}
		category := "context"
		title := "Assumption " + a.ID
		if a.ID == "" {
			title = "Assumption"
		}
		body := fmt.Sprintf("# %s\n\n%s\n\n_Assumption promoted from feature `%s`._\n", title, detail, b.Feature)
		docID := a.ID
		if docID == "" {
			docID = rag.Slugify(detail)
		}
		if docID == "" {
			continue
		}
		docID = "assumption-" + docID
		pathHint := category + "/" + docID + ".md"
		if dryRun {
			res.CreatedDocs = append(res.CreatedDocs, pathHint)
			continue
		}
		c, err := idx.UpsertDocument(docID, title, category, body, []string{"assumption"})
		if err != nil {
			return nil, fmt.Errorf("promote assumption %s: %w", a.ID, err)
		}
		res.UpdatedDocs = append(res.UpdatedDocs, c.Path)
	}

	// Optional workspace gotchas file.
	if b.Paths.WorkspaceDir != "" {
		gotchasPath, pathErr := schema.ResolveUnderRoot(repoRoot, filepath.ToSlash(filepath.Join(b.Paths.WorkspaceDir, "gotchas.md")))
		if pathErr == nil {
			if raw, err := os.ReadFile(gotchasPath); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
				docID := "gotchas-" + rag.Slugify(b.Feature)
				pathHint := "conventions/" + docID + ".md"
				if dryRun {
					res.CreatedDocs = append(res.CreatedDocs, pathHint)
				} else {
					c, err := idx.UpsertDocument(docID, "Gotchas: "+b.Feature, "conventions", string(raw), []string{"gotcha"})
					if err != nil {
						return nil, fmt.Errorf("promote gotchas: %w", err)
					}
					res.UpdatedDocs = append(res.UpdatedDocs, c.Path)
				}
			}
		}
	}

	promoted := append(append([]string{}, res.UpdatedDocs...), res.CreatedDocs...)
	fichaBody := fmt.Sprintf(`---
id: feature-%s
title: Feature %s
category: features
tags: [history]
---

# Feature: %s

- North star: %s
- Mode: %s
- Closed: %s
- Promoted docs: %s

`, b.Feature, b.Feature, b.Feature, b.NorthStar, b.Mode, time.Now().UTC().Format(time.RFC3339), strings.Join(promoted, ", "))

	if !dryRun {
		if err := os.WriteFile(fichaAbs, []byte(fichaBody), 0o644); err != nil {
			return nil, fmt.Errorf("write ficha: %w", err)
		}
		res.CreatedDocs = append(res.CreatedDocs, fichaRel)
		b.PromotionSummary = fmt.Sprintf("knowledge promote for %s → %s", b.Feature, fichaRel)
	} else {
		res.CreatedDocs = append(res.CreatedDocs, fichaRel+" (dry-run)")
	}
	res.Message = fmt.Sprintf("knowledge promote for %s → %s (dry_run=%v)", b.Feature, fichaRel, dryRun)
	return res, nil
}

// EnvResult describes an env promote run.
type EnvResult struct {
	DryRun       bool
	Diff         string
	Applied      bool
	Conflict     bool
	TodoPath     string
	AppliedIDs   []string
	DeferredIDs  []string
	Message      string
}

// Auto-mergeable promotion kinds (Target holds the value to merge).
const (
	KindDCFeature   = "dc_feature"   // Target = feature id URI; Description may hold JSON options
	KindForwardPort = "forward_port" // Target = port number as string
	KindExtension   = "extension"    // Target = VS Code extension id
)

// PromoteEnv tries automatic merges into the main Devcontainer.
// Safe merges: add missing features / ports / extensions.
// Any doubt or conflict → fail hard, write HUMAN todo list, leave pending_promotions untouched for deferred items.
func PromoteEnv(repoRoot string, b *schema.Board, cfg schema.Config, dryRun bool) (*EnvResult, error) {
	if b == nil {
		return nil, fmt.Errorf("board is nil")
	}

	// Resolve main DC path — NEVER create it (consumer owns the principal).
	mainPath := cfg.MainDevcontainer
	if mainPath == "" {
		mainPath = ".devcontainer/devcontainer.json"
	}
	if !filepath.IsAbs(mainPath) {
		mainPath = filepath.Join(repoRoot, mainPath)
	}

	res := &EnvResult{DryRun: dryRun}
	todoRel := filepath.ToSlash(filepath.Join(schema.ValidDir, "features", b.Feature, "workspace", "env-promote-todo.md"))
	if b.Paths.WorkspaceDir != "" {
		todoRel = filepath.ToSlash(filepath.Join(b.Paths.WorkspaceDir, "env-promote-todo.md"))
	}
	todoPath, err := schema.ResolveUnderRoot(repoRoot, todoRel)
	if err != nil {
		return nil, fmt.Errorf("todo path: %w", err)
	}
	res.TodoPath = todoPath

	var pending []schema.Promotion
	for _, p := range b.PendingPromotions {
		if !p.Applied {
			pending = append(pending, p)
		}
	}

	depsNote := ""
	if b.Paths.DepsDelta != "" {
		if abs, err := schema.ResolveUnderRoot(repoRoot, b.Paths.DepsDelta); err == nil {
			if raw, err := os.ReadFile(abs); err == nil {
				depsNote = string(raw)
			}
		}
	}

	if len(pending) == 0 && !depsDeltaHasActionable(depsNote) {
		res.Message = "nothing to promote for env"
		res.Applied = true
		return res, nil
	}

	if _, err := os.Stat(mainPath); err != nil {
		if os.IsNotExist(err) {
			return writeTodoAndFail(res, todoPath, b, pending, depsNote,
				fmt.Sprintf("main Devcontainer missing at %s — VALID does not create the project principal DC; add one or clear pending env promotions", mainPath))
		}
		return nil, fmt.Errorf("stat main devcontainer: %w", err)
	}

	main, err := env.LoadDevcontainerJSON(mainPath)
	if err != nil {
		return writeTodoAndFail(res, todoPath, b, pending, depsNote, fmt.Sprintf("cannot parse main DC: %v", err))
	}

	var diff strings.Builder
	fmt.Fprintf(&diff, "Main DC: %s\n", mainPath)
	var humanOnly []schema.Promotion
	var autoOK []schema.Promotion
	conflictMsgs := []string{}

	for _, p := range pending {
		fmt.Fprintf(&diff, "- [%s] %s → %s (target=%s)\n", p.Kind, p.ID, p.Description, p.Target)
		switch strings.ToLower(strings.TrimSpace(p.Kind)) {
		case KindDCFeature, "feature":
			ok, msg := tryMergeFeature(main, p)
			if !ok {
				conflictMsgs = append(conflictMsgs, msg)
				humanOnly = append(humanOnly, p)
			} else {
				autoOK = append(autoOK, p)
				fmt.Fprintf(&diff, "  AUTO: %s\n", msg)
			}
		case KindForwardPort, "port":
			ok, msg := tryMergePort(main, p)
			if !ok {
				conflictMsgs = append(conflictMsgs, msg)
				humanOnly = append(humanOnly, p)
			} else {
				autoOK = append(autoOK, p)
				fmt.Fprintf(&diff, "  AUTO: %s\n", msg)
			}
		case KindExtension, "vscode_extension":
			ok, msg := tryMergeExtension(main, p)
			if !ok {
				conflictMsgs = append(conflictMsgs, msg)
				humanOnly = append(humanOnly, p)
			} else {
				autoOK = append(autoOK, p)
				fmt.Fprintf(&diff, "  AUTO: %s\n", msg)
			}
		default:
			// Dockerfile lines, go.mod, apt, opaque deps → human.
			humanOnly = append(humanOnly, p)
			fmt.Fprintf(&diff, "  DEFER: kind %q requires human merge\n", p.Kind)
		}
	}

	if strings.TrimSpace(depsNote) != "" && depsDeltaHasActionable(depsNote) {
		fmt.Fprintf(&diff, "\n--- deps-delta.md (human review) ---\n%s\n", depsNote)
		// Actionable deps-delta always requires human judgment (fail hard).
		humanOnly = append(humanOnly, schema.Promotion{
			ID:          "deps-delta",
			Kind:        "deps_delta",
			Description: "workspace/deps-delta.md has actionable content",
		})
	}
	res.Diff = diff.String()

	// Fail hard if anything needs human judgment — do not write partial main DC.
	if len(humanOnly) > 0 || len(conflictMsgs) > 0 {
		res.Conflict = true
		res.DeferredIDs = promotionIDs(humanOnly)
		msg := "promote env fail hard: automatic merge incomplete; see " + todoPath
		if len(conflictMsgs) > 0 {
			msg = msg + " — " + strings.Join(conflictMsgs, "; ")
		}
		_ = writeEnvTodo(todoPath, b, mainPath, autoOK, humanOnly, conflictMsgs, depsNote, res.Diff)
		res.Message = msg
		return res, fmt.Errorf("%s", msg)
	}

	if dryRun {
		res.Message = "dry-run: automatic merges would apply cleanly"
		res.AppliedIDs = promotionIDs(autoOK)
		return res, nil
	}

	if err := env.WriteDevcontainerJSON(mainPath, main); err != nil {
		return nil, fmt.Errorf("write main DC: %w", err)
	}

	for i := range b.PendingPromotions {
		for _, ok := range autoOK {
			if b.PendingPromotions[i].ID == ok.ID {
				b.PendingPromotions[i].Applied = true
			}
		}
	}
	res.Applied = true
	res.AppliedIDs = promotionIDs(autoOK)
	res.Message = fmt.Sprintf("env promote applied to %s (%d changes)", mainPath, len(autoOK))

	// Success audit trail (not a human blocker).
	auditPath := filepath.Join(filepath.Dir(mainPath), "valid-promotions.md")
	var audit strings.Builder
	if prev, err := os.ReadFile(auditPath); err == nil {
		audit.Write(prev)
		audit.WriteString("\n")
	}
	fmt.Fprintf(&audit, "## Promote env OK — feature `%s` — %s\n\n%s\n", b.Feature, time.Now().UTC().Format(time.RFC3339), res.Diff)
	_ = os.WriteFile(auditPath, []byte(audit.String()), 0o644)
	_ = os.Remove(todoPath) // clear previous failure todo if any
	return res, nil
}

func promotionIDs(ps []schema.Promotion) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.ID)
	}
	return out
}

func depsDeltaHasActionable(note string) bool {
	n := strings.TrimSpace(note)
	if n == "" {
		return false
	}
	// Ignore the seed stub.
	if strings.Contains(n, "Record host/deps changes") && len(strings.Split(n, "\n")) <= 4 {
		return false
	}
	return true
}

func tryMergeFeature(main map[string]interface{}, p schema.Promotion) (bool, string) {
	featID := strings.TrimSpace(p.Target)
	if featID == "" {
		featID = strings.TrimSpace(p.Description)
	}
	if featID == "" {
		return false, fmt.Sprintf("%s: missing feature id in target", p.ID)
	}
	features, _ := main["features"].(map[string]interface{})
	if features == nil {
		features = map[string]interface{}{}
		main["features"] = features
	}
	var opts interface{} = map[string]interface{}{}
	if strings.TrimSpace(p.Description) != "" && json.Valid([]byte(p.Description)) {
		_ = json.Unmarshal([]byte(p.Description), &opts)
	}
	if existing, ok := features[featID]; ok {
		exRaw, _ := json.Marshal(existing)
		newRaw, _ := json.Marshal(opts)
		if string(exRaw) != string(newRaw) && string(newRaw) != "{}" {
			return false, fmt.Sprintf("%s: feature %q already exists with different options", p.ID, featID)
		}
		return true, fmt.Sprintf("feature %q already present (noop)", featID)
	}
	features[featID] = opts
	return true, fmt.Sprintf("added feature %q", featID)
}

func tryMergePort(main map[string]interface{}, p schema.Promotion) (bool, string) {
	portStr := strings.TrimSpace(p.Target)
	if portStr == "" {
		portStr = strings.TrimSpace(p.Description)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil || port <= 0 {
		return false, fmt.Sprintf("%s: invalid port %q", p.ID, portStr)
	}
	ports := []interface{}{}
	seen := map[int]bool{}
	if existing, ok := main["forwardPorts"].([]interface{}); ok {
		for _, item := range existing {
			n, ok := coercePort(item)
			if !ok {
				ports = append(ports, item)
				continue
			}
			ports = append(ports, n)
			seen[n] = true
		}
	}
	if seen[port] {
		return true, fmt.Sprintf("port %d already present (noop)", port)
	}
	ports = append(ports, port)
	main["forwardPorts"] = ports
	return true, fmt.Sprintf("added forwardPort %d", port)
}

func coercePort(item interface{}) (int, bool) {
	switch n := item.(type) {
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		v, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(v), true
	case string:
		var p int
		if _, err := fmt.Sscanf(strings.TrimSpace(n), "%d", &p); err == nil && p > 0 {
			return p, true
		}
	}
	return 0, false
}

func tryMergeExtension(main map[string]interface{}, p schema.Promotion) (bool, string) {
	ext := strings.TrimSpace(p.Target)
	if ext == "" {
		ext = strings.TrimSpace(p.Description)
	}
	if ext == "" {
		return false, fmt.Sprintf("%s: missing extension id", p.ID)
	}
	cust, _ := main["customizations"].(map[string]interface{})
	if cust == nil {
		cust = map[string]interface{}{}
		main["customizations"] = cust
	}
	vscode, _ := cust["vscode"].(map[string]interface{})
	if vscode == nil {
		vscode = map[string]interface{}{}
		cust["vscode"] = vscode
	}
	exts := []interface{}{}
	seen := map[string]bool{}
	if existing, ok := vscode["extensions"].([]interface{}); ok {
		for _, item := range existing {
			if s, ok := item.(string); ok {
				exts = append(exts, s)
				seen[s] = true
			}
		}
	}
	if seen[ext] {
		return true, fmt.Sprintf("extension %q already present (noop)", ext)
	}
	exts = append(exts, ext)
	vscode["extensions"] = exts
	return true, fmt.Sprintf("added extension %q", ext)
}

func writeTodoAndFail(res *EnvResult, todoPath string, b *schema.Board, pending []schema.Promotion, depsNote, reason string) (*EnvResult, error) {
	res.Conflict = true
	res.DeferredIDs = promotionIDs(pending)
	if err := writeEnvTodo(todoPath, b, "", nil, pending, []string{reason}, depsNote, reason); err != nil {
		res.Message = reason + " — also failed to write todo: " + err.Error()
		res.TodoPath = todoPath
		return res, fmt.Errorf("%s", res.Message)
	}
	res.Message = reason + " — see " + todoPath
	res.TodoPath = todoPath
	return res, fmt.Errorf("%s", res.Message)
}

func writeEnvTodo(path string, b *schema.Board, mainPath string, autoOK, human []schema.Promotion, conflicts []string, depsNote, diff string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var bld strings.Builder
	fmt.Fprintf(&bld, "# Env promote TODO — feature `%s`\n\n", b.Feature)
	fmt.Fprintf(&bld, "Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&bld, "VALID attempted automatic merges into the main Devcontainer and **stopped (fail hard)**.\n")
	fmt.Fprintf(&bld, "Nothing partial was written to the main DC for deferred/conflicting items.\n\n")
	if mainPath != "" {
		fmt.Fprintf(&bld, "Main DC path: `%s`\n\n", mainPath)
	}
	if len(conflicts) > 0 {
		bld.WriteString("## Conflicts / doubts\n\n")
		for _, c := range conflicts {
			fmt.Fprintf(&bld, "- %s\n", c)
		}
		bld.WriteString("\n")
	}
	if len(autoOK) > 0 {
		bld.WriteString("## Would auto-apply (blocked because other items need humans)\n\n")
		for _, p := range autoOK {
			fmt.Fprintf(&bld, "- [%s] `%s` — %s (target=%s)\n", p.Kind, p.ID, p.Description, p.Target)
		}
		bld.WriteString("\n")
	}
	if len(human) > 0 {
		bld.WriteString("## Human must apply\n\n")
		for _, p := range human {
			fmt.Fprintf(&bld, "- [%s] `%s` — %s (target=%s)\n", p.Kind, p.ID, p.Description, p.Target)
		}
		bld.WriteString("\n")
	}
	if strings.TrimSpace(depsNote) != "" {
		bld.WriteString("## deps-delta.md\n\n")
		bld.WriteString(depsNote)
		bld.WriteString("\n")
	}
	if strings.TrimSpace(diff) != "" {
		bld.WriteString("## Diff dump\n\n```\n")
		bld.WriteString(diff)
		bld.WriteString("\n```\n")
	}
	bld.WriteString("\nAfter fixing, re-run `valid promote env <slug>` (or skill finish).\n")
	return os.WriteFile(path, []byte(bld.String()), 0o644)
}
