# VALID — Codex

Thin wiring for [OpenAI Codex](https://github.com/openai/codex) (CLI / IDE). No business logic.

## Contents

| File | Use |
|------|-----|
| `config.toml` | MCP fragment → `valid mcp` |
| `AGENTS.md` | Project instructions for the Codex agent |

## Install (automated)

```bash
valid init
valid install codex
```

That writes:

- `.codex/valid.mcp.toml` — merge into `~/.codex/config.toml` (or project Codex MCP config)
- `AGENTS.md` at the repo root (kept unless `--force`)
- `.codex/skills/<name>/SKILL.md`
- `.codex/agents/*.md`

Invoke skills from the method docs / AGENTS guidance; do not reimplement plan/build/finish in this package.
