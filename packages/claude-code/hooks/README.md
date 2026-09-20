# Hooks / wiring notes

Claude Code should treat:

- `.valid/skills/*.md` as slash/skill procedures (`plan`, `spec`, `build`, `audit`, `finish`, …)
- `.valid/agents/*.md` as subagent personas (`interviewer`, `developer`, `reviewer`)
- `valid` CLI as scaffold + gate/promote/merge/env plumbing (the LLM edits feature boards)

Point the plugin or project instructions at those paths after `valid init`.
