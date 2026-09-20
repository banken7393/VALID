# VALID — Claude Code

Thin wiring: plugin manifest + MCP. No business logic.

## Contents

| File | Use |
|------|-----|
| `plugin.json` | Adapter manifest |
| `mcp.json` | MCP server → `valid mcp` |
| `hooks/README.md` | Notes for hooks (optional) |

## Install (automated)

```bash
valid init
valid install claude          # add --force to overwrite adapters
```

That writes:

- `.mcp.json` (project MCP)
- `.claude/CLAUDE.md`
- `.claude/plugins/valid/plugin.json`
- `.claude/skills/<name>/SKILL.md`
- `.claude/agents/*.md`

Restart Claude Code / reconnect MCP. Invoke skills **manually** (`/plan`, …).
