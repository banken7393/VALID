package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/schema"
	"github.com/banken7393/valid/internal/templates"
)

func TestInitCreatesHierarchy(t *testing.T) {
	dir := t.TempDir()

	cmd := NewRoot()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--repo", dir, "init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	checks := []string{
		filepath.Join(dir, ".valid", "config.json"),
		filepath.Join(dir, ".valid", "skills", "plan.md"),
		filepath.Join(dir, ".valid", "skills", "build-feature.md"),
		filepath.Join(dir, ".valid", "skills", "finish.md"),
		filepath.Join(dir, ".valid", "skills", "delegate.md"),
		filepath.Join(dir, ".valid", "skills", "minipatch.md"),
		filepath.Join(dir, ".valid", "agents", "interviewer.md"),
		filepath.Join(dir, ".valid", "agents", "delegate.md"),
		filepath.Join(dir, ".valid", "knowledge", "architecture", "overview.md"),
		filepath.Join(dir, ".valid", "scripts", "README.md"),
		filepath.Join(dir, ".valid", "scripts", "example_hello", "meta.json"),
		filepath.Join(dir, ".gitignore"),
	}
	for _, p := range checks {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}

	gi, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(gi), ".valid/worktrees/") || !strings.Contains(string(gi), ".valid/features/") {
		t.Fatalf("gitignore missing VALID entries: %s", gi)
	}

	cfg, err := schema.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Database.Enabled {
		t.Fatal("database should default off")
	}
	if _, err := os.Stat(filepath.Join(dir, ".valid", "data.json")); err == nil {
		t.Fatal("legacy global data.json should not exist")
	}
}

func TestMinipatchBoardGateFlow(t *testing.T) {
	dir := t.TempDir()
	if err := templates.InitProject(dir, templates.InitOptions{}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		cmd := NewRoot()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(append([]string{"--repo", dir}, args...))
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	run("feature", "new", "fix-typo", "--mode", "minipatch", "--north-star", "fix typo")
	run("board", "set", "fix-typo", "--what", `[{"id":"ac1","description":"typo fixed"}]`, "--lifecycle", "spec")
	run("task", "fix-typo", "t1", "--title", "Fix string", "--covers", "ac1", "--status", "done")
	run("report", "fix-typo", "--test-name", "TestTypo", "--status", "pass")
	// Trust board evidence in unit test (no real suite in temp dir).
	cmd := NewRoot()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--repo", dir, "gate", "fix-typo", "--trust-board"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("gate: %v", err)
	}

	b, err := feature.Load(dir, "fix-typo")
	if err != nil {
		t.Fatal(err)
	}
	if !b.Audit.Passed {
		t.Fatalf("expected gate pass, findings=%v", b.Audit.Findings)
	}
}

func TestGateFailsWithoutCoverage(t *testing.T) {
	dir := t.TempDir()
	if err := templates.InitProject(dir, templates.InitOptions{}); err != nil {
		t.Fatal(err)
	}
	runOK := func(args ...string) {
		t.Helper()
		cmd := NewRoot()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs(append([]string{"--repo", dir}, args...))
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	runOK("feature", "new", "gap", "--mode", "minipatch")
	runOK("board", "set", "gap", "--what", `[{"id":"ac1","description":"a"},{"id":"ac2","description":"b"}]`)
	runOK("task", "gap", "t1", "--covers", "ac1", "--status", "done")
	runOK("report", "gap", "--test-name", "T", "--status", "pass")

	cmd := NewRoot()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--repo", dir, "gate", "gap", "--trust-board"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected gate failure for uncovered AC")
	}
}

func TestInstallCursorCLI(t *testing.T) {
	dir := t.TempDir()
	cmd := NewRoot()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--repo", dir, "install", "cursor"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".cursor", "skills", "plan", "SKILL.md")); err != nil {
		t.Fatalf("skill not published: %v", err)
	}
	if !strings.Contains(buf.String(), "[cursor]") {
		t.Fatalf("expected report output, got %q", buf.String())
	}
}
