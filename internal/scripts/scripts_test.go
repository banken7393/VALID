package scripts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndRunScript(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, ".valid", "scripts", "hello")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{
  "description": "Print a hello line for MCP smoke tests",
  "command": ["bash", "run.sh"],
  "timeout_sec": 10
}`
	if err := os.WriteFile(filepath.Join(root, "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	run := "#!/usr/bin/env bash\nset -euo pipefail\necho \"hello:$1\"\n"
	if err := os.WriteFile(filepath.Join(root, "run.sh"), []byte(run), 0o755); err != nil {
		t.Fatal(err)
	}

	all, err := LoadAll(repo, filepath.Join(repo, ".valid", "scripts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "hello" {
		t.Fatalf("got %+v", all)
	}

	res, err := all[0].Run(context.Background(), []string{"world"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "hello:world") {
		t.Fatalf("result=%+v", res)
	}
}

func TestDisabledAndReservedSkipped(t *testing.T) {
	repo := t.TempDir()
	scriptsRoot := filepath.Join(repo, ".valid", "scripts")

	write := func(id, meta string) {
		dir := filepath.Join(scriptsRoot, id)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644)
	}
	write("search_knowledge", `{"description":"bad","command":["true"]}`)
	enabled := `{"description":"off","command":["true"],"enabled":false}`
	write("offline", enabled)
	write("ok_tool", `{"description":"ready","command":["true"]}`)

	all, err := LoadAll(repo, scriptsRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "ok_tool" {
		t.Fatalf("got %+v", all)
	}
}
