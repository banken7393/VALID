package ide

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/banken7393/valid/internal/schema"
	"github.com/banken7393/valid/internal/templates"
)

func TestInstallCursorPublishesSkills(t *testing.T) {
	dir := t.TempDir()
	if err := templates.InitProject(dir, templates.InitOptions{}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	reports, err := Install(dir, "cursor", Options{})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(reports) != 1 || reports[0].Target != "cursor" {
		t.Fatalf("reports: %+v", reports)
	}

	checks := []string{
		filepath.Join(dir, ".cursor", "mcp.json"),
		filepath.Join(dir, ".cursor", "rules", "valid.mdc"),
		filepath.Join(dir, ".cursor", "skills", "plan", "SKILL.md"),
		filepath.Join(dir, ".cursor", "skills", "build", "SKILL.md"),
		filepath.Join(dir, ".cursor", "skills", "finish", "SKILL.md"),
		filepath.Join(dir, ".cursor", "skills", "delegate", "SKILL.md"),
		filepath.Join(dir, ".cursor", "agents", "developer.md"),
		filepath.Join(dir, ".cursor", "agents", "delegate.md"),
	}
	for _, p := range checks {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}

	body, err := os.ReadFile(filepath.Join(dir, ".cursor", "skills", "plan", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "disable-model-invocation") && !strings.Contains(string(body), "plan") {
		t.Fatalf("plan skill looks empty/wrong: %s", body[:min(80, len(body))])
	}
}

func TestInstallPreservesConfigAndKeepsMCPWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := templates.InitProject(dir, templates.InitOptions{}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	cfg, err := schema.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.TestCommand = "pytest -q"
	cfg.DashboardPort = 9001
	if err := schema.SaveConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	customMCP := []byte(`{"mcpServers":{"other":{"command":"echo"}}}` + "\n")
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".cursor", "mcp.json"), customMCP, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(dir, "cursor", Options{Force: false}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, err := schema.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.TestCommand != "pytest -q" || got.DashboardPort != 9001 {
		t.Fatalf("config overwritten: %+v", got)
	}

	raw, _ := os.ReadFile(filepath.Join(dir, ".cursor", "mcp.json"))
	if string(raw) != string(customMCP) {
		t.Fatalf("mcp.json should be kept without --force, got %s", raw)
	}
}

func TestInstallAllAndForceMergeMCP(t *testing.T) {
	dir := t.TempDir()
	if err := templates.InitProject(dir, templates.InitOptions{}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".cursor", "mcp.json"), []byte(`{"mcpServers":{"other":{"command":"echo"}}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	reports, err := Install(dir, "all", Options{Force: true})
	if err != nil {
		t.Fatalf("Install all: %v", err)
	}
	if len(reports) != 4 {
		t.Fatalf("want 4 reports, got %d", len(reports))
	}

	raw, err := os.ReadFile(filepath.Join(dir, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, `"other"`) || !strings.Contains(s, `"valid"`) {
		t.Fatalf("expected merged mcpServers, got %s", s)
	}

	for _, p := range []string{
		filepath.Join(dir, ".mcp.json"),
		filepath.Join(dir, ".claude", "skills", "plan", "SKILL.md"),
		filepath.Join(dir, ".opencode", "skills", "spec", "SKILL.md"),
		filepath.Join(dir, ".codex", "valid.mcp.toml"),
		filepath.Join(dir, "AGENTS.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
}

func TestInstallUnknownTarget(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "neovim", Options{}); err == nil {
		t.Fatal("expected error")
	}
}
