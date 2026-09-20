# VALID — OpenCode

Thin wiring: config fragment. No business logic.

## Contents

| File | Use |
|------|-----|
| `opencode.valid.json` | MCP + instructions to merge into OpenCode config |

## Install (automated)

```bash
valid init
valid install opencode
```

That writes:

- `.opencode.valid.json` — merge this into your OpenCode config
- `.opencode/skills/<name>/SKILL.md`
- `.opencode/agents/*.md`

Use `--force` to overwrite the fragment. Skills always refresh from `.valid/skills/`.
