package promote

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/banken7393/valid/internal/env"
	"github.com/banken7393/valid/internal/schema"
)

func writeMainDC(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	m := env.DefaultMainDevcontainer("test", 7432)
	if err := env.WriteDevcontainerJSON(path, m); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPromoteEnvAutoMergeFeatures(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeMainDC(t, dir)

	b := schema.NewBoard("feat", schema.ModeFeature)
	b.Paths.WorkspaceDir = filepath.ToSlash(filepath.Join(".valid", "features", "feat", "workspace"))
	_ = os.MkdirAll(filepath.Join(dir, b.Paths.WorkspaceDir), 0o755)
	_ = b.AddPromotion("p1", KindDCFeature, `{}`, "ghcr.io/devcontainers/features/node:1")
	_ = b.AddPromotion("p2", KindForwardPort, "8080", "8080")
	_ = b.AddPromotion("p3", KindExtension, "golang.go", "golang.go")

	cfg := schema.DefaultConfig()
	res, err := PromoteEnv(dir, b, cfg, false)
	if err != nil {
		t.Fatalf("PromoteEnv: %v\n%s", err, res.Diff)
	}
	if !res.Applied {
		t.Fatal("expected applied")
	}

	m, err := env.LoadDevcontainerJSON(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	feats, _ := m["features"].(map[string]interface{})
	if _, ok := feats["ghcr.io/devcontainers/features/node:1"]; !ok {
		t.Fatalf("feature not merged: %#v", feats)
	}
}

func TestPromoteEnvFailHardWritesTodo(t *testing.T) {
	dir := t.TempDir()
	_ = writeMainDC(t, dir)

	b := schema.NewBoard("feat", schema.ModeFeature)
	b.Paths.WorkspaceDir = filepath.ToSlash(filepath.Join(".valid", "features", "feat", "workspace"))
	ws := filepath.Join(dir, b.Paths.WorkspaceDir)
	_ = os.MkdirAll(ws, 0o755)
	_ = b.AddPromotion("p1", KindDCFeature, `{}`, "ghcr.io/devcontainers/features/node:1")
	_ = b.AddPromotion("opaque", "dockerfile_line", "RUN apt-get install -y foo", "")

	cfg := schema.DefaultConfig()
	res, err := PromoteEnv(dir, b, cfg, false)
	if err == nil {
		t.Fatal("expected fail hard")
	}
	if !res.Conflict {
		t.Fatal("expected conflict")
	}
	todo := filepath.Join(ws, "env-promote-todo.md")
	if _, err := os.Stat(todo); err != nil {
		t.Fatalf("todo missing: %v", err)
	}
	m, _ := env.LoadDevcontainerJSON(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
	feats, _ := m["features"].(map[string]interface{})
	if len(feats) != 0 {
		t.Fatalf("partial apply not allowed, got %#v", feats)
	}
}

func TestPromoteEnvMissingMainFailHard(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("feat", schema.ModeFeature)
	b.Paths.WorkspaceDir = filepath.ToSlash(filepath.Join(".valid", "features", "feat", "workspace"))
	ws := filepath.Join(dir, b.Paths.WorkspaceDir)
	_ = os.MkdirAll(ws, 0o755)
	_ = b.AddPromotion("p1", KindForwardPort, "8080", "8080")

	cfg := schema.DefaultConfig()
	res, err := PromoteEnv(dir, b, cfg, false)
	if err == nil {
		t.Fatal("expected fail when main DC missing")
	}
	if res == nil || !res.Conflict {
		t.Fatal("expected conflict result")
	}
	if _, err := os.Stat(filepath.Join(ws, "env-promote-todo.md")); err != nil {
		t.Fatalf("todo missing: %v", err)
	}
}

func TestPromoteEnvNothingToDoWithoutMain(t *testing.T) {
	dir := t.TempDir()
	b := schema.NewBoard("x", schema.ModeMinipatch)
	b.Paths.WorkspaceDir = filepath.ToSlash(filepath.Join(".valid", "features", "x", "workspace"))
	_ = os.MkdirAll(filepath.Join(dir, b.Paths.WorkspaceDir), 0o755)
	cfg := schema.DefaultConfig()
	res, err := PromoteEnv(dir, b, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied {
		t.Fatal("empty promote should succeed without main DC")
	}
}
