# VALID

This project uses **VALID** (Visual Agentic Loop for Isolated Development).

## Method

1. Invoke skills under `.claude/skills/` (or `.valid/skills/`) **manually**: `/plan`, `/spec`, `/build`, `/audit`, `/finish`, `/patch`, `/minipatch`, `/delegate`.
2. Agents (roles) live in `.claude/agents/` and `.valid/agents/`: `interviewer`, `developer`, `reviewer`, `delegate`.
3. Board `autonomy`: `in_the_loop` (default) or `above_the_loop`. Use `/delegate` for bounded autonomous chunks.
4. Edit `.valid/features/<slug>/data.json` yourself. CLI is scaffolding (`valid feature`, `gate`, `promote`, `merge`, `env`).
5. MCP server `valid` is knowledge RAG + project scripts under `.valid/scripts/`.

Do not invent `valid plan` / `valid build` as CLI product commands — those names are skills.
