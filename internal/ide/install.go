package ide

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/banken7393/valid/internal/templates"
)

//go:embed assets/cursor/mcp.json assets/cursor/rules/valid.mdc assets/claude-code/mcp.json assets/claude-code/CLAUDE.md assets/claude-code/plugin.json assets/opencode/opencode.valid.json assets/codex/config.toml assets/codex/AGENTS.md
var assets embed.FS

// Options controls IDE wiring install.
type Options struct {
	Force bool // overwrite existing adapter files (mcp/rules/CLAUDE.md/…)
	Link  bool // symlink skills/agents into IDE dirs (else copy)
}

// Report summarizes what was written for one target.
type Report struct {
	Target  string
	Actions []string
}

// Install wires one or all IDE adapters into repoRoot.
// target: cursor | claude | opencode | codex | all
func Install(repoRoot, target string, opts Options) ([]Report, error) {
	if repoRoot == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}

	validCfg := filepath.Join(abs, ".valid", "config.json")
	if _, err := os.Stat(validCfg); err != nil {
		if err := templates.InitProject(abs, templates.InitOptions{Force: false}); err != nil {
			return nil, fmt.Errorf("valid init required first: %w", err)
		}
	} else {
		// Refresh method skills/agents without touching user config.
		if err := templates.InitProject(abs, templates.InitOptions{Force: false}); err != nil {
			return nil, fmt.Errorf("refresh .valid skills/agents: %w", err)
		}
	}

	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = "all"
	}
	if target == "claude-code" {
		target = "claude"
	}

	var targets []string
	switch target {
	case "all":
		targets = []string{"cursor", "claude", "opencode", "codex"}
	case "cursor", "claude", "opencode", "codex":
		targets = []string{target}
	default:
		return nil, fmt.Errorf("unknown target %q (want cursor|claude|opencode|codex|all)", target)
	}

	var reports []Report
	for _, t := range targets {
		var r *Report
		var err error
		switch t {
		case "cursor":
			r, err = installCursor(abs, opts)
		case "claude":
			r, err = installClaude(abs, opts)
		case "opencode":
			r, err = installOpenCode(abs, opts)
		case "codex":
			r, err = installCodex(abs, opts)
		}
		if err != nil {
			return reports, fmt.Errorf("%s: %w", t, err)
		}
		reports = append(reports, *r)
	}
	return reports, nil
}

func installCursor(repo string, opts Options) (*Report, error) {
	r := &Report{Target: "cursor"}
	if err := writeAssetFile(repo, "assets/cursor/mcp.json", ".cursor/mcp.json", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := writeAssetFile(repo, "assets/cursor/rules/valid.mdc", ".cursor/rules/valid.mdc", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := publishSkills(repo, filepath.Join(repo, ".cursor", "skills"), opts, &r.Actions); err != nil {
		return r, err
	}
	if err := publishAgents(repo, filepath.Join(repo, ".cursor", "agents"), opts, &r.Actions); err != nil {
		return r, err
	}
	r.Actions = append(r.Actions, "reload Cursor MCP (Settings → MCP) if tools do not appear")
	return r, nil
}

func installClaude(repo string, opts Options) (*Report, error) {
	r := &Report{Target: "claude"}
	if err := writeAssetFile(repo, "assets/claude-code/mcp.json", ".mcp.json", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := writeAssetFile(repo, "assets/claude-code/plugin.json", ".claude/plugins/valid/plugin.json", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := writeAssetFile(repo, "assets/claude-code/CLAUDE.md", ".claude/CLAUDE.md", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := publishSkills(repo, filepath.Join(repo, ".claude", "skills"), opts, &r.Actions); err != nil {
		return r, err
	}
	if err := publishAgents(repo, filepath.Join(repo, ".claude", "agents"), opts, &r.Actions); err != nil {
		return r, err
	}
	r.Actions = append(r.Actions, "restart Claude Code / reconnect MCP after install")
	return r, nil
}

func installOpenCode(repo string, opts Options) (*Report, error) {
	r := &Report{Target: "opencode"}
	if err := writeAssetFile(repo, "assets/opencode/opencode.valid.json", ".opencode.valid.json", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := publishSkills(repo, filepath.Join(repo, ".opencode", "skills"), opts, &r.Actions); err != nil {
		return r, err
	}
	if err := publishAgents(repo, filepath.Join(repo, ".opencode", "agents"), opts, &r.Actions); err != nil {
		return r, err
	}
	r.Actions = append(r.Actions, "merge .opencode.valid.json into your OpenCode config (mcp + instructions)")
	return r, nil
}

func installCodex(repo string, opts Options) (*Report, error) {
	r := &Report{Target: "codex"}
	if err := writeAssetFile(repo, "assets/codex/config.toml", ".codex/valid.mcp.toml", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := writeAssetFile(repo, "assets/codex/AGENTS.md", "AGENTS.md", opts.Force, &r.Actions); err != nil {
		return r, err
	}
	if err := publishSkills(repo, filepath.Join(repo, ".codex", "skills"), opts, &r.Actions); err != nil {
		return r, err
	}
	if err := publishAgents(repo, filepath.Join(repo, ".codex", "agents"), opts, &r.Actions); err != nil {
		return r, err
	}
	r.Actions = append(r.Actions, "merge .codex/valid.mcp.toml into ~/.codex/config.toml (or project Codex MCP config)")
	return r, nil
}

// publishSkills copies/links each .valid/skills/<name>.md → <dest>/<name>/SKILL.md
// Always refreshes so IDE skill trees stay in sync with method upgrades.
func publishSkills(repo, destRoot string, opts Options, actions *[]string) error {
	srcDir := filepath.Join(repo, ".valid", "skills")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read skills: %w", err)
	}
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		src := filepath.Join(srcDir, e.Name())
		dstDir := filepath.Join(destRoot, name)
		dst := filepath.Join(dstDir, "SKILL.md")
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return err
		}
		if err := placeFile(src, dst, opts.Link, true); err != nil {
			return err
		}
		*actions = append(*actions, fmt.Sprintf("skill %s → %s", name, rel(repo, dst)))
	}
	return nil
}

func publishAgents(repo, destRoot string, opts Options, actions *[]string) error {
	srcDir := filepath.Join(repo, ".valid", "agents")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read agents: %w", err)
	}
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(destRoot, e.Name())
		if err := placeFile(src, dst, opts.Link, true); err != nil {
			return err
		}
		*actions = append(*actions, fmt.Sprintf("agent %s → %s", e.Name(), rel(repo, dst)))
	}
	return nil
}

func placeFile(src, dst string, link, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return nil
		}
	}
	_ = os.RemoveAll(dst)
	if link {
		absSrc, err := filepath.Abs(src)
		if err != nil {
			return err
		}
		return os.Symlink(absSrc, dst)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writeAssetFile(repo, assetPath, relOut string, force bool, actions *[]string) error {
	raw, err := assets.ReadFile(assetPath)
	if err != nil {
		return fmt.Errorf("read asset %s: %w", assetPath, err)
	}
	dst := filepath.Join(repo, relOut)
	if _, err := os.Stat(dst); err == nil && !force {
		*actions = append(*actions, "keep existing "+relOut+" (use --force to overwrite)")
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if force && strings.HasSuffix(relOut, "mcp.json") {
		if err := mergeMCPJSON(dst, raw); err == nil {
			*actions = append(*actions, "merged MCP into "+relOut)
			return nil
		}
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		return err
	}
	*actions = append(*actions, "wrote "+relOut)
	return nil
}

func mergeMCPJSON(dst string, fragment []byte) error {
	existing := map[string]interface{}{}
	if raw, err := os.ReadFile(dst); err == nil {
		_ = json.Unmarshal(raw, &existing)
	}
	frag := map[string]interface{}{}
	if err := json.Unmarshal(fragment, &frag); err != nil {
		return err
	}
	for _, key := range []string{"mcpServers", "mcp"} {
		if add, ok := frag[key].(map[string]interface{}); ok {
			cur, _ := existing[key].(map[string]interface{})
			if cur == nil {
				cur = map[string]interface{}{}
			}
			for k, v := range add {
				cur[k] = v
			}
			existing[key] = cur
		}
	}
	for k, v := range frag {
		if k == "mcpServers" || k == "mcp" {
			continue
		}
		if _, ok := existing[k]; !ok {
			existing[k] = v
		}
	}
	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(dst, out, 0o644)
}

func rel(repo, path string) string {
	r, err := filepath.Rel(repo, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}
