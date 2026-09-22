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

func TestLoadBoardDogfoodLLMShapes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	body := `{
  "version": "2",
  "feature": "generic-dossiers",
  "mode": "feature",
  "lifecycle": "plan",
  "autonomy": "in_the_loop",
  "north_star": "Kernel de dossier genérico",
  "how_it_works": "prose note only",
  "what": [
    "Evolucionar applications → kernel dossier",
    "Introducir catálogo seedado"
  ],
  "phases": [],
  "tasks": [],
  "decisions": [
    { "id": "D1", "text": "Concepto e identidad = dossier." },
    {
      "id": "D2",
      "question": "¿Corregir mapping?",
      "choice": "Sí. Biodiversidad→1.10",
      "decided_at": "2026-09-22"
    }
  ],
  "assumptions": [
    {
      "id": "A1",
      "text": "Hoy Application = dossier package.",
      "breaks_if_wrong": "Si solo NZ, overkill."
    }
  ],
  "tdd": { "passed": 0, "failed": 0, "total": 0 },
  "audit": { "passed": false, "findings": [], "at": "0001-01-01T00:00:00Z" },
  "environment": {
    "worktree_path": "/workspace/.valid/worktrees/generic-dossiers",
    "isolation_warning": "Soft isolation: feature Dev Environment no levantó",
    "database_enabled": false
  },
  "pending_promotions": [],
  "paths": {
    "board_dir": ".valid/features/generic-dossiers",
    "how_it_works": ".valid/features/generic-dossiers/how-it-works.mmd"
  },
  "updated_at": "2026-09-20T15:44:00Z"
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBoard(path)
	if err != nil {
		t.Fatalf("LoadBoard dogfood shapes: %v", err)
	}
	if len(b.What) != 2 || b.What[0].ID != "ac1" {
		t.Fatalf("what: %+v", b.What)
	}
	if len(b.Decisions) != 2 {
		t.Fatalf("decisions: %+v", b.Decisions)
	}
	if b.Decisions[0].Title == "" {
		t.Fatalf("decisions[0] text→title: %+v", b.Decisions[0])
	}
	if b.Decisions[1].Title == "" || !strings.Contains(b.Decisions[1].Detail, "Biodiversidad") {
		t.Fatalf("decisions[1] question/choice: %+v", b.Decisions[1])
	}
	if !strings.Contains(b.Decisions[1].Detail, "Decided:") {
		t.Fatalf("decided_at should fold into detail: %+v", b.Decisions[1])
	}
	if len(b.Assumptions) != 1 || !strings.Contains(b.Assumptions[0].Detail, "Breaks if wrong") {
		t.Fatalf("assumptions: %+v", b.Assumptions)
	}
	if !b.Environment.IsolationWarning {
		t.Fatal("expected isolation_warning string → true")
	}
}

func TestLoadBoardPhaseNameOutcomeStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	body := `{
  "version": "2",
  "feature": "city-defaults",
  "mode": "feature",
  "lifecycle": "spec",
  "autonomy": "in_the_loop",
  "north_star": "City defaults",
  "what": [{ "id": "ac1", "description": "defaults on create" }],
  "phases": [
    {
      "id": "p1",
      "name": "Defaults al crear ciudad",
      "outcome": "Al crear una ciudad, A1–C3 nacen con mapping correcto.",
      "status": "agreed"
    },
    {
      "id": "p2",
      "name": "Sync global catalog-aware",
      "outcome": "El sync aplica reglas D8.",
      "status": "agreed"
    }
  ],
  "tasks": [],
  "decisions": [],
  "assumptions": [],
  "tdd": { "passed": 0, "failed": 0, "total": 0, "cases": [] },
  "audit": { "passed": false, "findings": [] },
  "pending_promotions": [],
  "environment": { "isolation_warning": false, "database_enabled": false }
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBoard(path)
	if err != nil {
		t.Fatalf("LoadBoard: %v", err)
	}
	if len(b.Phases) != 2 {
		t.Fatalf("phases: %+v", b.Phases)
	}
	if b.Phases[0].Name != "Defaults al crear ciudad" || b.Phases[0].Outcome == "" || b.Phases[0].Status != "agreed" {
		t.Fatalf("phase0: %+v", b.Phases[0])
	}
	if b.Phases[0].Order != 1 || b.Phases[1].Order != 2 {
		t.Fatalf("auto order want 1,2 got %d,%d", b.Phases[0].Order, b.Phases[1].Order)
	}
	if b.Phases[0].Title != b.Phases[0].Name {
		t.Fatalf("title should mirror name: %+v", b.Phases[0])
	}
	if b.Phases[0].Label() != b.Phases[0].Name {
		t.Fatalf("Label: %q", b.Phases[0].Label())
	}
}

func TestLoadBoardPhaseLegacyTitleOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	body := `{
  "version": "2",
  "feature": "legacy",
  "mode": "feature",
  "lifecycle": "spec",
  "what": [],
  "phases": [{ "id": "p1", "title": "Old shape", "order": 3 }],
  "tasks": [],
  "decisions": [],
  "assumptions": [],
  "tdd": { "passed": 0, "failed": 0, "total": 0, "cases": [] },
  "audit": { "passed": false, "findings": [] },
  "pending_promotions": [],
  "environment": { "isolation_warning": false }
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBoard(path)
	if err != nil {
		t.Fatalf("LoadBoard: %v", err)
	}
	if b.Phases[0].Name != "Old shape" || b.Phases[0].Order != 3 {
		t.Fatalf("legacy title/order: %+v", b.Phases[0])
	}
}

func TestLoadBoardTaskPhaseAndFailed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	body := `{
  "version": "2",
  "feature": "city-defaults",
  "mode": "feature",
  "lifecycle": "build",
  "what": [
    { "id": "ac1", "description": "defaults on create" },
    { "id": "ac2", "description": "i18n texts" }
  ],
  "phases": [
    { "id": "p1", "name": "Defaults al crear ciudad", "outcome": "…", "status": "doing" }
  ],
  "tasks": [
    {
      "id": "t1",
      "title": "Seed A1–C3",
      "status": "pending",
      "phase": "p1",
      "covers": ["ac1", "ac2"]
    },
    {
      "id": "t2",
      "title": "Attach catalog markdown",
      "status": "failed",
      "phase_id": "p1",
      "covers": ["ac1"]
    }
  ],
  "decisions": [],
  "assumptions": [],
  "tdd": { "passed": 0, "failed": 0, "total": 0, "cases": [] },
  "audit": { "passed": false, "findings": [] },
  "pending_promotions": [],
  "environment": { "isolation_warning": false }
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBoard(path)
	if err != nil {
		t.Fatalf("LoadBoard: %v", err)
	}
	if b.Tasks[0].Phase != "p1" || b.Tasks[0].Status != "pending" {
		t.Fatalf("task0: %+v", b.Tasks[0])
	}
	if b.Tasks[1].Phase != "p1" || b.Tasks[1].Status != "failed" {
		t.Fatalf("task1 phase_id alias / failed: %+v", b.Tasks[1])
	}
	if err := b.UpsertTaskFull("t3", "Retry attach", "doing", "", "p1", []string{"ac1"}); err != nil {
		t.Fatal(err)
	}
	if b.Tasks[2].Phase != "p1" || b.Tasks[2].Status != "doing" {
		t.Fatalf("upsert full: %+v", b.Tasks[2])
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
	if cfg.Theme != ThemeDark {
		t.Fatalf("default theme want dark, got %q", cfg.Theme)
	}
	if NormalizeTheme("") != ThemeDark || NormalizeTheme("LIGHT") != ThemeLight {
		t.Fatal("NormalizeTheme failed")
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
