package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/banken7393/valid/internal/schema"
)

const (
	SkillsDir    = "skills"
	AgentsDir    = "agents"
	KnowledgeDir = "knowledge"
	FeaturesDir  = "features"
	WorkflowsDir = "workflows"
	ScriptsDir   = "scripts"
)

var gitignoreEntries = []string{
	"# VALID temporary directories",
	".valid/worktrees/",
	".valid/features/",
}

// InitOptions controls how Project is initialized.
type InitOptions struct {
	Force bool
}

// InitProject creates the full .valid/ hierarchy under root.
func InitProject(root string, opts InitOptions) error {
	if root == "" {
		return fmt.Errorf("root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}

	validRoot := filepath.Join(abs, schema.ValidDir)
	dirs := []string{
		validRoot,
		filepath.Join(validRoot, KnowledgeDir, "architecture"),
		filepath.Join(validRoot, KnowledgeDir, "conventions"),
		filepath.Join(validRoot, KnowledgeDir, "testing"),
		filepath.Join(validRoot, KnowledgeDir, "decisions"),
		filepath.Join(validRoot, KnowledgeDir, "glossary"),
		filepath.Join(validRoot, KnowledgeDir, "context"),
		filepath.Join(validRoot, KnowledgeDir, "features"),
		filepath.Join(validRoot, FeaturesDir),
		filepath.Join(validRoot, SkillsDir),
		filepath.Join(validRoot, AgentsDir),
		filepath.Join(validRoot, WorkflowsDir),
		filepath.Join(validRoot, ScriptsDir),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	// Never wipe an existing config on re-init / install refresh.
	if _, err := os.Stat(schema.ConfigPath(abs)); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat config.json: %w", err)
		}
		if err := schema.SaveConfig(abs, schema.DefaultConfig()); err != nil {
			return fmt.Errorf("write config.json: %w", err)
		}
	}

	// Remove legacy global data.json if present (v2 has no migrator).
	legacy := filepath.Join(validRoot, schema.DataFileName)
	_ = os.Remove(legacy)

	for name, body := range SkillFiles() {
		// Method files always refresh so upgrades land without --force.
		if err := writeFileMode(filepath.Join(validRoot, SkillsDir, name), body, true, 0o644); err != nil {
			return err
		}
	}
	for name, body := range AgentFiles() {
		if err := writeFileMode(filepath.Join(validRoot, AgentsDir, name), body, true, 0o644); err != nil {
			return err
		}
	}
	for rel, body := range KnowledgeFiles() {
		if err := writeFile(filepath.Join(validRoot, KnowledgeDir, rel), body, opts.Force); err != nil {
			return err
		}
	}
	for rel, body := range ScriptSeedFiles() {
		path := filepath.Join(validRoot, ScriptsDir, rel)
		mode := os.FileMode(0o644)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o755
		}
		// Seed example + README refresh; user scripts in other folders are untouched.
		forceSeed := opts.Force || rel == "README.md" || strings.HasPrefix(rel, "example_hello/")
		if err := writeFileMode(path, body, forceSeed, mode); err != nil {
			return err
		}
	}
	if err := writeFile(filepath.Join(validRoot, WorkflowsDir, "README.md"), workflowsREADME, true); err != nil {
		return err
	}

	if err := ensureGitignore(abs); err != nil {
		return err
	}
	return nil
}

func writeFile(path, body string, force bool) error {
	return writeFileMode(path, body, force, 0o644)
}

func writeFileMode(path, body string, force bool, mode os.FileMode) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func ensureGitignore(root string) error {
	path := filepath.Join(root, ".gitignore")
	var existing string
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read .gitignore: %w", err)
		}
	} else {
		existing = string(raw)
	}

	var toAdd []string
	for _, entry := range gitignoreEntries {
		if strings.HasPrefix(entry, "#") {
			if !strings.Contains(existing, entry) {
				toAdd = append(toAdd, entry)
			}
			continue
		}
		if !gitignoreHasEntry(existing, entry) {
			toAdd = append(toAdd, entry)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		b.WriteByte('\n')
	}
	if existing != "" {
		b.WriteByte('\n')
	}
	for _, line := range toAdd {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func gitignoreHasEntry(content, entry string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			return true
		}
	}
	return false
}

const workflowsREADME = `# Workflows (optional)

Multi-skill playbooks can live here. They do **not** replace ` + "`.valid/skills/`" + `.
Example: a short Markdown guide that says "for incident X, run /minipatch then /audit".
`
