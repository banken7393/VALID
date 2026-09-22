package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/banken7393/valid/internal/schema"
)

func trustRun(dir string, b *schema.Board, cfg schema.Config) Result {
	return RunWithOptions(dir, b, cfg, Options{TrustBoard: true})
}

func TestGateFailsOnStubHowItWorks(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("x", schema.ModeFeature)
	b.Lifecycle = schema.LifeBuild
	b.HowItWorks = "flowchart LR\n  start[Start] --> goal[x]\n"
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "a"}})
	_ = b.UpsertTask("t1", "do", "done", "", []string{"ac1"})
	_ = b.ApplyReport(schema.TestPass, "TestA", "", "")
	cfg := schema.DefaultConfig()
	res := trustRun(dir, b, cfg)
	if res.Passed {
		t.Fatal("stub how-it-works must fail gate")
	}
	found := false
	for _, f := range res.Findings {
		if f.Code == "how_it_works_stub" || f.Code == "how_it_works_missing" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected how_it_works finding, got %#v", res.Findings)
	}
}

func TestGatePassesWithRealMermaid(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("pay", schema.ModeFeature)
	b.Lifecycle = schema.LifeAudit
	b.HowItWorks = "flowchart TD\n  client[Client] -->|POST /pay| api[API]\n  api --> db[(DB)]\n  api -->|webhook| provider[Provider]\n"
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "charge card"}})
	_ = b.UpsertTask("t1", "charge", "done", "", []string{"ac1"})
	_ = b.ApplyReport(schema.TestPass, "TestCharge", "", "")
	cfg := schema.DefaultConfig()
	res := trustRun(dir, b, cfg)
	if !res.Passed {
		t.Fatalf("expected pass, findings=%#v", res.Findings)
	}
}

func TestGateFailsUncoveredAC(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("x", schema.ModeMinipatch)
	b.Lifecycle = schema.LifeBuild
	_ = b.SetWhat([]schema.AcceptanceCriterion{
		{ID: "ac1", Description: "a"},
		{ID: "ac2", Description: "b"},
	})
	_ = b.UpsertTask("t1", "do", "done", "", []string{"ac1"})
	_ = b.ApplyReport(schema.TestPass, "T", "", "")
	res := trustRun(dir, b, schema.DefaultConfig())
	if res.Passed {
		t.Fatal("expected fail for uncovered ac2")
	}
}

func TestGateReadsHowFromFile(t *testing.T) {
	dir := t.TempDir()
	rel := filepath.Join(".valid", "features", "x", "how-it-works.mmd")
	abs := filepath.Join(dir, rel)
	_ = os.MkdirAll(filepath.Dir(abs), 0o755)
	body := "sequenceDiagram\n  participant U as User\n  participant S as Service\n  U->>S: request\n  S-->>U: response\n"
	_ = os.WriteFile(abs, []byte(body), 0o644)
	b := schema.NewBoard("x", schema.ModeFeature)
	b.Lifecycle = schema.LifeBuild
	b.Paths.HowItWorks = filepath.ToSlash(rel)
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "a"}})
	_ = b.UpsertTask("t1", "do", "done", "", []string{"ac1"})
	_ = b.ApplyReport(schema.TestPass, "T", "", "")
	res := trustRun(dir, b, schema.DefaultConfig())
	if !res.Passed {
		t.Fatalf("expected pass via file, %#v", res.Findings)
	}
}

func TestGateRunsTestCommand(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("x", schema.ModeMinipatch)
	b.Lifecycle = schema.LifeAudit
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "a"}})
	_ = b.UpsertTask("t1", "do", "done", "", []string{"ac1"})
	cfg := schema.DefaultConfig()
	cfg.TestCommand = "exit 0"
	res := Run(dir, b, cfg)
	if !res.Passed {
		t.Fatalf("expected pass after exit 0, %#v", res.Findings)
	}
	if b.TDD.Total == 0 || !b.TestsGreen() {
		t.Fatalf("expected board TDD updated, got %+v", b.TDD)
	}

	cfg.TestCommand = "exit 1"
	b2 := schema.NewBoard("y", schema.ModeMinipatch)
	b2.Lifecycle = schema.LifeAudit
	_ = b2.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "a"}})
	_ = b2.UpsertTask("t1", "do", "done", "", []string{"ac1"})
	res2 := Run(dir, b2, cfg)
	if res2.Passed {
		t.Fatal("expected fail after exit 1")
	}
}

func TestGateMissingWorktreeDoesNotFallback(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("pay", schema.ModeFeature)
	b.Lifecycle = schema.LifeAudit
	b.HowItWorks = "flowchart TD\n  client[Client] -->|POST /pay| api[API]\n  api --> db[(DB)]\n  api -->|webhook| provider[Provider]\n"
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "charge card"}})
	_ = b.UpsertTask("t1", "charge", "done", "", []string{"ac1"})
	b.Environment.WorktreePath = filepath.Join(dir, "gone-worktree")
	cfg := schema.DefaultConfig()
	cfg.TestCommand = "exit 0"
	res := Run(dir, b, cfg)
	if res.Passed {
		t.Fatal("expected fail when worktree is missing")
	}
	found := false
	for _, f := range res.Findings {
		if f.Code == "worktree_missing" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected worktree_missing, got %#v", res.Findings)
	}
}

func TestGateLiveRunClearsStaleTDD(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("x", schema.ModeMinipatch)
	b.Lifecycle = schema.LifeAudit
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "a"}})
	_ = b.UpsertTask("t1", "do", "done", "", []string{"ac1"})
	_ = b.ApplyReport(schema.TestFail, "OldFail", "stale", "")
	_ = b.ApplyReport(schema.TestPending, "OldPending", "", "")
	cfg := schema.DefaultConfig()
	cfg.TestCommand = "exit 0"
	res := Run(dir, b, cfg)
	if !res.Passed {
		t.Fatalf("expected pass; stale cases must not veto live green, %#v", res.Findings)
	}
	if !b.TestsGreen() || b.TDD.Failed != 0 {
		t.Fatalf("expected only live suite evidence, got %+v", b.TDD)
	}
}

func TestGateCodeOutsideWorktreeStrict(t *testing.T) {
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
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644)
	run("git", "add", "README.md")
	run("git", "commit", "-m", "init")

	wt := filepath.Join(dir, ".valid", "worktrees", "pay")
	_ = os.MkdirAll(wt, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "leak.go"), []byte("package leak\n"), 0o644)

	b := schema.NewBoard("pay", schema.ModeFeature)
	b.Lifecycle = schema.LifeAudit
	b.HowItWorks = "flowchart TD\n  client[Client] -->|POST /pay| api[API]\n  api --> db[(DB)]\n  api -->|webhook| provider[Provider]\n"
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "charge card"}})
	_ = b.UpsertTask("t1", "charge", "done", "", []string{"ac1"})
	_ = b.ApplyReport(schema.TestPass, "TestCharge", "", "")
	b.Environment.WorktreePath = wt

	cfg := schema.DefaultConfig()
	cfg.Isolation.Strict = true
	res := trustRun(dir, b, cfg)
	if res.Passed {
		t.Fatal("expected fail under isolation.strict with principal contamination")
	}
	found := false
	for _, f := range res.Findings {
		if f.Code == "code_outside_worktree" && f.Severity == schema.SevError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected code_outside_worktree error, %#v", res.Findings)
	}
	if !b.Environment.IsolationWarning {
		t.Fatal("expected isolation_warning set on board when contaminated")
	}
}

func TestGateMinipatchIgnoresIsolationStrict(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("x", schema.ModeMinipatch)
	b.Lifecycle = schema.LifeAudit
	b.Environment.IsolationWarning = true
	_ = b.SetWhat([]schema.AcceptanceCriterion{{ID: "ac1", Description: "a"}})
	_ = b.UpsertTask("t1", "do", "done", "", []string{"ac1"})
	_ = b.ApplyReport(schema.TestPass, "T", "", "")
	cfg := schema.DefaultConfig()
	cfg.Isolation.Strict = true
	res := trustRun(dir, b, cfg)
	if !res.Passed {
		t.Fatalf("minipatch soft isolation must not fail under isolation.strict, %#v", res.Findings)
	}
}
