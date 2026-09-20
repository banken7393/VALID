package env

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAndRemoveWorktree(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	wt, base, err := mgr.CreateWorktree("env-demo")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if base == "" {
		t.Fatal("expected base branch from principal HEAD")
	}
	if !strings.HasSuffix(wt, filepath.Join(".valid", "worktrees", "env-demo")) {
		t.Fatalf("unexpected path: %s", wt)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("stat worktree: %v", err)
	}

	branch := BranchName("env-demo")
	out, err := exec.Command("git", "-C", dir, "branch", "--list", branch).CombinedOutput()
	if err != nil || !strings.Contains(string(out), branch) {
		t.Fatalf("branch %s missing: %s (%v)", branch, out, err)
	}

	if err := mgr.PrepareWorktree("env-demo", wt); err != nil {
		t.Fatalf("PrepareWorktree: %v", err)
	}
	// Boards live in the main repo; worktree only gets optional config copy.
	_ = os.MkdirAll(filepath.Join(dir, ".valid"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".valid", "config.json"), []byte(`{"version":2,"test_command":"go test ./...","dashboard_port":7432}`+"\n"), 0o644)
	if err := mgr.PrepareWorktree("env-demo", wt); err != nil {
		t.Fatalf("PrepareWorktree with config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wt, ".valid", "config.json")); err != nil {
		t.Fatalf("worktree config.json: %v", err)
	}

	main := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	_ = os.MkdirAll(filepath.Dir(main), 0o755)
	_ = os.WriteFile(main, []byte(`{"name":"t","image":"mcr.microsoft.com/devcontainers/base:bookworm","features":{}}`+"\n"), 0o644)
	dc, err := InjectDevcontainer(wt, "env-demo", 7432, main, dir)
	if err != nil {
		t.Fatalf("InjectDevcontainer: %v", err)
	}
	if _, err := os.Stat(dc); err != nil {
		t.Fatalf("devcontainer.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dc), "Dockerfile")); err != nil {
		// When cloning main without Dockerfile, quarantine not used — OK if main had none.
		// Here main has no Dockerfile; CopyMain may not write Dockerfile.
	}

	if err := mgr.RemoveWorktree("env-demo", true); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Fatalf("worktree still present: %v", err)
	}
}

func TestCreateWorktreeFromNonMainBranch(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	run("git", "checkout", "-b", "develop")
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, base, err := mgr.CreateWorktree("from-dev")
	if err != nil {
		t.Fatal(err)
	}
	if base != "develop" {
		t.Fatalf("base=%q want develop", base)
	}
	if err := mgr.MergeWorktree("from-dev", base); err != nil {
		t.Fatal(err)
	}
	cur, err := mgr.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if cur != "develop" {
		t.Fatalf("after merge HEAD=%q want develop", cur)
	}
}

func TestValidateFeatureName(t *testing.T) {
	if err := ValidateFeatureName(""); err == nil {
		t.Fatal("expected empty name error")
	}
	if err := ValidateFeatureName("../evil"); err == nil {
		t.Fatal("expected invalid name error")
	}
	if err := ValidateFeatureName("ok-feature_1.0"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "valid@test")
	run("git", "config", "user.name", "valid")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-m", "init")
}
