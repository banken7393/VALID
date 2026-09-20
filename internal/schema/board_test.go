package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBoardValidate(t *testing.T) {
	b := NewBoard("auth", ModeFeature)
	if err := b.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if b.Version != Version || b.Lifecycle != LifePlan {
		t.Fatalf("unexpected board: %+v", b)
	}
	if b.Autonomy != AutonomyInTheLoop {
		t.Fatalf("default autonomy want in_the_loop, got %q", b.Autonomy)
	}
	if err := b.SetAutonomy(AutonomyAboveTheLoop); err != nil {
		t.Fatal(err)
	}
	if b.Autonomy != AutonomyAboveTheLoop {
		t.Fatal("set autonomy failed")
	}
	if err := b.SetAutonomy("nope"); err == nil {
		t.Fatal("expected invalid autonomy error")
	}
}

func TestAcceptanceCriterionAcceptsStringJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	body := `{
  "version": "2",
  "feature": "x",
  "mode": "feature",
  "lifecycle": "spec",
  "what": [
    "Applications hold generic dossiers",
    "ac2: Tipología vive en plantillas",
    {"id": "ac3", "description": "legal_procedures own the procedure"}
  ],
  "phases": [],
  "tasks": [],
  "decisions": [],
  "assumptions": [],
  "tdd": {"passed": 0, "failed": 0, "total": 0, "cases": []},
  "audit": {"passed": false, "findings": []},
  "pending_promotions": [],
  "environment": {"isolation_warning": false, "database_enabled": false}
}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBoard(path)
	if err != nil {
		t.Fatalf("LoadBoard: %v", err)
	}
	if len(b.What) != 3 {
		t.Fatalf("what len=%d", len(b.What))
	}
	if b.What[0].ID != "ac1" || b.What[0].Description == "" {
		t.Fatalf("string AC0: %+v", b.What[0])
	}
	if b.What[1].ID != "ac2" || !strings.Contains(b.What[1].Description, "Tipología") {
		t.Fatalf("labeled AC1: %+v", b.What[1])
	}
	if b.What[2].ID != "ac3" {
		t.Fatalf("object AC2: %+v", b.What[2])
	}
}

func TestUncoveredACsAndTestsGreen(t *testing.T) {
	b := NewBoard("x", ModeFeature)
	_ = b.SetWhat([]AcceptanceCriterion{
		{ID: "ac1", Description: "login"},
		{ID: "ac2", Description: "logout"},
	})
	_ = b.UpsertTask("t1", "Login form", "done", "", []string{"ac1"})
	missing := b.UncoveredACs()
	if len(missing) != 1 || missing[0] != "ac2" {
		t.Fatalf("missing=%v", missing)
	}
	if b.TestsGreen() {
		t.Fatal("expected not green")
	}
	_ = b.ApplyReport(TestPass, "TestLogin", "", "")
	if !b.TestsGreen() {
		t.Fatal("expected green after pass")
	}
}

func TestApplyReportPreservesRefactorPhase(t *testing.T) {
	b := NewBoard("x", ModeFeature)
	_ = b.ApplyReport(TestPass, "T1", "", TDDRefactor)
	if b.TDD.Phase != TDDRefactor {
		t.Fatalf("want refactor, got %q", b.TDD.Phase)
	}
	_ = b.ApplyReport(TestPass, "T2", "", "")
	if b.TDD.Phase != TDDRefactor {
		t.Fatalf("recalc should keep refactor when green, got %q", b.TDD.Phase)
	}
	_ = b.ApplyReport(TestFail, "T1", "boom", "")
	if b.TDD.Phase != TDDRed {
		t.Fatalf("want red after fail, got %q", b.TDD.Phase)
	}
}

func TestResolveUnderRootRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolveUnderRoot(dir, "../etc/passwd"); err == nil {
		t.Fatal("expected reject")
	}
	ok, err := ResolveUnderRoot(dir, ".valid/features/x/how-it-works.mmd")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(ok) {
		t.Fatalf("want abs path, got %s", ok)
	}
}

func TestAtomicWriteUniqueTemps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := AtomicWrite(path, []byte(`{"a":1}`+"\n")); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(path, []byte(`{"a":2}`+"\n")); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != `{"a":2}`+"\n" {
		t.Fatalf("got %s", raw)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Database.Enabled {
		t.Fatal("database should be off by default")
	}
	if cfg.Isolation.Strict {
		t.Fatal("isolation should be soft by default")
	}
	if cfg.DashboardPort != 7432 || cfg.MCPHTTPPort != 7433 {
		t.Fatalf("unexpected ports: %+v", cfg)
	}
	feat := cfg.ResolveFeatureDashboardPort("auth")
	if feat == cfg.DashboardPort || feat == cfg.MCPHTTPPort {
		t.Fatalf("feature port %d collides with principal ports", feat)
	}
	cfg.FeatureDashboardPort = 9001
	p1 := cfg.ResolveFeatureDashboardPort("auth")
	p2 := cfg.ResolveFeatureDashboardPort("billing")
	if p1 == 9001 {
		t.Fatal("feature_dashboard_port is a base; slug salt must still apply")
	}
	if p1 == p2 {
		t.Fatal("different slugs must resolve different ports")
	}
	if p1 < 9001 || p2 < 9001 {
		t.Fatalf("expected ports >= base, got %d %d", p1, p2)
	}
}
