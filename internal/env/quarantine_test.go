package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripJSONPreservesHTTPS(t *testing.T) {
	in := "{\n  // comment\n  \"image\": \"https://example.com/img\",\n  \"n\": 1\n}\n"
	out := stripJSONLineComments(in)
	if !strings.Contains(out, "https://example.com/img") {
		t.Fatalf("https stripped: %s", out)
	}
	if strings.Contains(out, "// comment") {
		t.Fatalf("full-line comment not removed: %s", out)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("parse: %v (%s)", err, out)
	}
}

func TestWriteDevcontainerJSONAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")
	m := map[string]interface{}{"name": "x", "image": "https://example.com/a"}
	if err := WriteDevcontainerJSON(path, m); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "https://example.com/a") {
		t.Fatalf("got %s", raw)
	}
}

func TestWriteQuarantineDevcontainer(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteQuarantineDevcontainer(dir, "demo", 7432, "debian:bookworm")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, `"dockerfile": "Dockerfile"`) {
		t.Fatalf("expected build.dockerfile, got %s", s)
	}
	if !strings.Contains(s, `"--name"`) || !strings.Contains(s, "valid-demo") {
		t.Fatalf("expected runArgs --name valid-demo, got %s", s)
	}
	if !strings.Contains(s, "debian:bookworm") {
		t.Fatalf("expected BASE_IMAGE arg, got %s", s)
	}
	if strings.Contains(s, "mcr.microsoft.com") {
		t.Fatal("must not use Microsoft base image URL")
	}
	df, err := os.ReadFile(filepath.Join(dir, ".devcontainer", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	dfs := string(df)
	if !strings.Contains(dfs, "useradd") || !strings.Contains(dfs, "1000") {
		t.Fatalf("expected UID 1000 user setup, got %s", dfs)
	}
	if !strings.Contains(dfs, "ARG BASE_IMAGE") {
		t.Fatal("expected ARG BASE_IMAGE")
	}
}

func TestDetectBaseImage(t *testing.T) {
	dir := t.TempDir()
	if got := DetectBaseImage(dir); got != "debian:bookworm" {
		t.Fatalf("empty repo: %s", got)
	}
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{}`), 0o644)
	if got := DetectBaseImage(dir); got != "node:22-bookworm" {
		t.Fatalf("node: %s", got)
	}
}

func TestInjectQuarantineWhenNoMain(t *testing.T) {
	dir := t.TempDir()
	wt := filepath.Join(dir, "wt")
	_ = os.MkdirAll(wt, 0o755)
	path, err := InjectDevcontainer(wt, "f1", 7532, "", dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "Dockerfile")); err != nil {
		t.Fatal(err)
	}
}

func TestCopyMainDoesNotStealPrincipalPorts(t *testing.T) {
	dir := t.TempDir()
	mainDir := filepath.Join(dir, ".devcontainer")
	_ = os.MkdirAll(mainDir, 0o755)
	mainJSON := filepath.Join(mainDir, "devcontainer.json")
	_ = os.WriteFile(mainJSON, []byte(`{
  "name": "principal",
  "image": "debian:bookworm",
  "forwardPorts": [3000, 5432, 7432, 7433],
  "portsAttributes": {"3000": {"label": "app"}}
}`), 0o644)
	_ = os.WriteFile(filepath.Join(mainDir, "Dockerfile"), []byte("FROM debian:bookworm\n"), 0o644)

	wt := filepath.Join(dir, "wt")
	_ = os.MkdirAll(wt, 0o755)
	path, err := CopyMainDevcontainerIntoWorktree(wt, "pay", mainJSON, 7601)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "3000") || strings.Contains(s, "5432") || strings.Contains(s, "7432") || strings.Contains(s, "7433") {
		t.Fatalf("must not forward principal ports, got %s", s)
	}
	if !strings.Contains(s, "7601") {
		t.Fatalf("expected feature port 7601, got %s", s)
	}
	if !strings.Contains(s, `"valid-pay"`) && !strings.Contains(s, "valid-pay") {
		t.Fatalf("expected unique name, got %s", s)
	}
	// Principal untouched
	mainRaw, _ := os.ReadFile(mainJSON)
	if !strings.Contains(string(mainRaw), "3000") {
		t.Fatal("principal DC must remain unchanged")
	}
	if _, err := os.Stat(filepath.Join(wt, ".devcontainer", "Dockerfile")); err != nil {
		t.Fatal("expected copied Dockerfile in worktree")
	}
}
