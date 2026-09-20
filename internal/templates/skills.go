package templates

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed skillmd/*.md
var skillFS embed.FS

//go:embed agentmd/*.md
var agentFS embed.FS

// SkillFiles returns Markdown skill scripts (method UX). Not CLI commands.
func SkillFiles() map[string]string {
	return readEmbeddedMD(skillFS, "skillmd")
}

// AgentFiles returns role prompts seeded under .valid/agents/.
func AgentFiles() map[string]string {
	return readEmbeddedMD(agentFS, "agentmd")
}

func readEmbeddedMD(efs embed.FS, dir string) map[string]string {
	out := map[string]string{}
	entries, err := fs.ReadDir(efs, dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw, err := efs.ReadFile(dir + "/" + e.Name())
		if err != nil {
			continue
		}
		out[e.Name()] = string(raw)
	}
	return out
}
