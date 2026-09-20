package templates

// KnowledgeFiles maps relative knowledge paths to seed Markdown documents.
func KnowledgeFiles() map[string]string {
	return map[string]string{
		"architecture/overview.md":   knowledgeArchitecture,
		"conventions/code-style.md":  knowledgeConventions,
		"testing/standards.md":       knowledgeTesting,
		"context/project.md":         knowledgeContext,
		"git/worktree-quarantine.md": knowledgeWorktree,
	}
}

const knowledgeArchitecture = `---
id: architecture-overview
title: Architecture Overview
category: architecture
tags:
  - overview
  - isolation
scope: project
---

# Architecture Overview

VALID isolates AI-driven development using:

1. Skills (Markdown): plan → spec → build → audit → finish (+ patch/minipatch/delegate). The LLM keeps a per-feature progress board under .valid/features/<slug>/.
2. Dashboard (` + "`valid dashboard`" + `) — visual check of AI implementation state (lifecycle, ACs, tasks, tests, Mermaid, gate).
3. CLI plumbing: scaffold feature/worktree/Dev Container, gate, promote, merge, env, install.
4. MCP: knowledge RAG under .valid/knowledge/ **plus** project scripts under .valid/scripts/ (never mutates boards/TDD).
`

const knowledgeConventions = `---
id: code-style
title: Code Style Conventions
category: conventions
tags:
  - style
  - board
scope: project
---

# Code Style

Prefer clear names, small diffs, and project ` + "`test_command`" + ` as the source of truth for how to run tests.

## Feature board JSON (` + "`.valid/features/<slug>/data.json`" + `)

Keys are English and **typed**. Wrong shapes make ` + "`valid dashboard`" + ` / gate fail to load the board.

` + "```" + `json
"what": [{ "id": "ac1", "description": "…" }]
"decisions": [{ "id": "D1", "title": "…", "detail": "…" }]
"assumptions": [{ "id": "A1", "detail": "…" }]
"tasks": [{ "id": "t1", "title": "…", "status": "pending|doing|done", "covers": ["ac1"] }]
"environment": { "isolation_warning": false }
` + "```" + `

Mermaid lives at ` + "`.valid/features/<slug>/how-it-works.mmd`" + ` (not under ` + "`workspace/`" + `).
`

const knowledgeTesting = `---
id: testing-standards
title: Testing Standards
category: testing
tags:
  - tdd
scope: project
---

# Testing Standards

1. Red → green → refactor.
2. Record results on the feature board ` + "`tdd`" + ` fields (edit data.json; optional helper: ` + "`valid report`" + `).
3. Gate requires AC↔task coverage and green executable tests via ` + "`test_command`" + `.
`

const knowledgeContext = `---
id: project-context
title: Project Context
category: context
tags:
  - context
scope: project
---

# Project Context

Describe the product and stack here. Agents should read this via MCP before inventing architecture.

Board prose language follows ` + "`.valid/config.json`" + ` field ` + "`language`" + ` (` + "`repo`" + ` = match the repository; or set en / es / …).
` + "`database.enabled`" + ` / ` + "`database.recipe`" + ` control feature DB isolation notes (default off).
`

const knowledgeWorktree = `---
id: worktree-quarantine
title: Worktree Quarantine
category: git
tags:
  - isolation
  - worktree
scope: project
---

# Worktree Quarantine

Prefer skill **plan** / **patch** (CLI ` + "`valid feature new`" + `) so work lands in ` + "`.valid/worktrees/<slug>/`" + ` on branch ` + "`feature/<slug>`" + ` until **finish** merges and tears down the env. Do not treat the principal Dev Container as a scratch pad for unproven deps.
`
