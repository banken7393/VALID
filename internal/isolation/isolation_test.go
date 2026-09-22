package isolation_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/banken7393/valid/internal/isolation"
	"github.com/banken7393/valid/internal/schema"
)

func TestResolvePaths(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("demo", schema.ModeFeature)
	b.Paths.HowItWorks = ".valid/features/demo/how-it-works.mmd"
	b.Paths.WorkspaceDir = ".valid/features/demo/workspace"
	b.Paths.Worktree = ".valid/worktrees/demo"
	b.Environment.WorktreePath = filepath.Join(dir, ".valid", "worktrees", "demo")
	b.Environment.Branch = "feature/demo"
	b.Environment.BaseBranch = "main"

	p, err := isolation.Resolve(dir, b)
	if err != nil {
		t.Fatal(err)
	}
	if p.CodeRoot != b.Environment.WorktreePath {
		t.Fatalf("CODE_ROOT=%q", p.CodeRoot)
	}
	if !strings.HasSuffix(filepath.ToSlash(p.Board), ".valid/features/demo/data.json") {
		t.Fatalf("BOARD=%q", p.Board)
	}
	env := p.FormatEnv()
	if !strings.Contains(env, "CODE_ROOT=") || !strings.Contains(env, "slug=demo") {
		t.Fatalf("FormatEnv: %s", env)
	}
}

func TestDirtyPathsDetectsPrincipalContamination(t *testing.T) {
	dir := initGitRepo(t)
	// Allowed control-plane change
	_ = os.MkdirAll(filepath.Join(dir, ".valid", "features", "x"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, ".valid", "features", "x", "data.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Forbidden product change on principal
	if err := os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dirty, err := isolation.DirtyPaths(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range dirty {
		if p == "app.go" {
			found = true
		}
		if strings.HasPrefix(p, ".valid/") {
			t.Fatalf(".valid path should be allowed, got %q in %#v", p, dirty)
		}
	}
	if !found {
		t.Fatalf("expected app.go in dirty, got %#v", dirty)
	}
}

func TestInspectStrictElevates(t *testing.T) {
	dir := initGitRepo(t)
	wt := filepath.Join(dir, ".valid", "worktrees", "pay")
	_ = os.MkdirAll(wt, 0o755)
	if err := os.WriteFile(filepath.Join(dir, "svc.go"), []byte("package svc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := schema.NewBoard("pay", schema.ModeFeature)
	b.Environment.WorktreePath = wt
	cfg := schema.DefaultConfig()
	cfg.Isolation.Strict = true

	findings := isolation.Inspect(dir, b, cfg)
	var outside *schema.Finding
	for i := range findings {
		if findings[i].Code == isolation.CodeOutsideWorktree {
			outside = &findings[i]
		}
	}
	if outside == nil {
		t.Fatalf("expected code_outside_worktree, got %#v", findings)
	}
	if outside.Severity != schema.SevError {
		t.Fatalf("strict should elevate to error, got %s", outside.Severity)
	}
}

func TestInspectMinipatchSkips(t *testing.T) {
	dir := initGitRepo(t)
	_ = os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n"), 0o644)
	b := schema.NewBoard("tiny", schema.ModeMinipatch)
	cfg := schema.DefaultConfig()
	cfg.Isolation.Strict = true
	if findings := isolation.Inspect(dir, b, cfg); len(findings) != 0 {
		t.Fatalf("minipatch must skip, got %#v", findings)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "t@example.com")
	run("git", "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-m", "init")
	return dir
}
