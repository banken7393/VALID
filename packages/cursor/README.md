# VALID — Cursor

Thin wiring: MCP + rules. No business logic.

## Contents

| File | Use |
|------|-----|
| `mcp.json` | Cursor MCP snippet (`valid mcp`) |
| `rules/valid.mdc` | Project rule: skills/agents + LLM-owned boards |

## Install (automated)

```bash
# once: build VALID and put `valid` on PATH
valid init
valid install cursor          # add --force to overwrite mcp/rules
```

That writes:

- `.cursor/mcp.json`
- `.cursor/rules/valid.mdc`
- `.cursor/skills/<name>/SKILL.md` (from `.valid/skills/`)
- `.cursor/agents/*.md` (from `.valid/agents/`)

Then reload Cursor MCP if tools do not appear (Settings → MCP). Invoke skills **manually** (`/plan`, `/spec`, …) — they are `disable-model-invocation`.

## Manual (not recommended)

Same files live in this folder if you need to merge by hand.
