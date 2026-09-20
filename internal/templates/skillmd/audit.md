---
name: audit
description: >-
  Evidence-based review of the diff and the progress board before finish. Invoke the
  reviewer agent. Require AC↔task coverage and green executable tests. Soft
  isolation warnings only.
argument-hint: feature-slug
disable-model-invocation: true
user-invocable: true
---
# VALID: Audit

**Manual skill.** Invoke explicitly (e.g. `/audit`). Do not auto-start.

VALID cycle: Plan → Spec → Build → **Audit** → Finish.

Audit answers **is this honest and mergeable** — grade the **result**, not the story told while building.

## Produces

- Human-visible review findings (and optional notes on `audit` in the board)
- Pass / fail from mechanical gate `valid gate <slug>`
- Go / no-go for skill **finish**

## 1. Resolve and invoke reviewer

1. Resolve `slug`. Prefer the feature worktree for `git diff` / `git status`.
2. Load and follow `.valid/agents/reviewer.md`.
3. Read the board end-to-end: `what`, `tasks`, `covers`, `tdd`, `assumptions`, Mermaid, `pending_promotions`.

## 2. Reconcile board vs reality

Walk the board against the working tree:

| Check | Hard? |
|-------|-------|
| Each task `status` matches reality (`done` only if truly done) | Yes — fix board or code |
| Each AC is satisfied by code, or explicitly deferred by the human | Yes |
| Every `what[].id` appears in some `tasks[].covers` | Yes (gate) |
| Executable tests green via `valid gate` (`test_command`) | Yes (gate) |
| Mermaid still draws the mechanism that shipped | Soft → redraw or note stale |
| Open assumptions resolved, invalidated (with cost), or explicitly deferred | Soft → surface |
| Behaviour built beyond the WHAT | Soft → add AC or strip code |
| Isolation / env warnings only | Soft — never sole hard fail |

## 3. Evidence rules (reviewer lens)

For each load-bearing claim on the board (“AC ac3 is done”, “tests prove X”):

- Locate primary evidence: code, test output, config — not memory, not the author’s justification alone.
- Verdicts: verified / partially correct / unverified / incorrect / conflicting.
- Cite file paths (and commands run). “I believe” is not evidence.

Do **not** soft-pedal. If it is wrong, say so.

## 4. Hard gates

Run:

```bash
valid gate <slug>
```

Hard fail when any AC is uncovered, or when there are no executable tests / tests are not green.  
Soft: `isolation_warning` alone does not fail the gate.

You may perform the same checks by reading the board, but the CLI gate is the shared mechanical record — prefer running it.

## 5. Leave

- Gate red or review blocked → return to **build**, fix, re-audit.
- Gate green and human accepts residual risk → skill **finish**.
- Write a short residual-risk paragraph on the board or in chat before handing off.

## Guardrails

- No merge, no promote, no board delete in Audit.
- No fake `reviewed` flag — gate + human judgment are the record.
- Do not weaken gates for **patch** / **minipatch** (same rigor).
- Audit does not “optimize” or redesign; it verifies. Fixes happen under Build.

