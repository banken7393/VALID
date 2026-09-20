# VALID (Codex)

This repository uses VALID for isolated feature work with AI agents.

## Method

1. Follow skills in `.valid/skills/` (`plan`, `spec`, `build`, `audit`, `finish`, `patch`, `minipatch`, `delegate`). Those Markdown files **are** the method.
2. When a skill says so, use agents in `.valid/agents/`:
   - `interviewer` — scope and acceptance criteria
   - `developer` — implement with TDD (`autonomy`: `in_the_loop` default, or `above_the_loop`)
   - `delegate` — bounded autonomous chunk (skill `/delegate`)
   - `reviewer` — code/contract review before finish
3. **Edit** `.valid/features/<slug>/data.json`, Mermaid, and workspace files yourself. The `valid` CLI is minimal scaffolding and plumbing (`feature`, `gate`, `promote`, `merge`, `env`, optional `board`/`task`/`report` helpers).
4. MCP tools: knowledge RAG **and** project scripts under `.valid/scripts/<id>/` (each `meta.json` is an MCP tool the agent may run). Never use MCP to mutate TDD or ACs.

## Quick CLI (plumbing)

```bash
valid feature new <slug>
valid gate <slug>
valid promote knowledge|env <slug>
valid merge <slug>
valid env up|down <slug>
```
