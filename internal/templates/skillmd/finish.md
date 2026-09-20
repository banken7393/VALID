---
name: finish
description: >-
  Close the feature: truth-up the board, graduate knowledge sparingly, promote env
  deltas with fail-hard on doubt, merge the worktree, delete the temporary board,
  tear down the feature environment. Manually invoke when the feature is audited and ready.
argument-hint: feature-slug
disable-model-invocation: true
user-invocable: true
---
# VALID: Finish

**Manual skill.** Invoke explicitly (e.g. `/finish`). Do not auto-start.

VALID cycle: Plan → Spec → Build → Audit → **Finish**.

Finish answers **what actually shipped, and what must outlive the feature**.

Because Build kept the board current, this is a truth-up — brief, one sentence where one does.  
You still **edit** knowledge fragments and the board when needed; the CLI runs deterministic promote / merge / teardown.

## Sequence (abort on first failure)

1. **Gate** — ensure Audit passed:

```bash
valid gate <slug>
```

STOP if not green.

2. **Truth-up the board** (edit `data.json` / Mermaid yourself):

   - Task statuses match reality.
   - Assumptions: resolved / invalidated (with cost) / deferred with a forward check — never dropped in silence.
   - Mermaid matches what shipped, or note why it is stale.
   - Behaviour beyond the WHAT → add an AC or strip the code (human decides).

3. **Graduate knowledge sparingly** — prepare fragments, then:

```bash
valid promote knowledge <slug> [--dry-run]
```

   Or write carefully via MCP `upsert_document` with human agreement. Prefer almost nothing (see below).

4. **Promote environment** into the **existing** project principal Dev Container:

```bash
valid promote env <slug> [--dry-run]
```

   - Auto-merge only safe `devcontainer.json` keys (`dc_feature`, `forward_port`, `extension`).
   - Any doubt / opaque Dockerfile/manifest / actionable `deps-delta.md` / missing principal DC → **fail hard**, write `.valid/features/<slug>/workspace/env-promote-todo.md`, **no partial write**. STOP. Human fixes; re-run. Do **not** merge or delete the board after this failure.

5. **Merge** feature branch (**skip for minipatch** — commit on the current branch instead). For feature/patch this also tears down the feature container and removes the worktree:

```bash
valid merge <slug>
```

   Never commit/merge without explicit human approval when git asks for it. A merge that refuses or conflicts stops here and removes nothing.

6. **Delete** the temporary board (and any leftover env) via CLI:

```bash
valid feature delete <slug>
```

   For minipatch: after gate + optional knowledge promote, skip merge/promote-env unless you have pending env promotions; then `valid feature delete <slug>`.

7. Lifecycle conceptually `done` (board may already be gone — that is fine).

## Graduate sparingly

Promote **almost nothing** into `.valid/knowledge/**`:

| Kind | Destination | Bar |
|------|-------------|-----|
| Cross-cutting gotchas | `knowledge/conventions/**` | Frontier model could not infer from code |
| Stable cross-feature decisions | `knowledge/decisions/**` | Important, stable, cross-feature, non-obvious |
| Terms coined | `knowledge/glossary/**` | With a short avoid-line if useful |
| Historical index only | `knowledge/features/<slug>.md` | Short card — not the corpus |

If a frontier model could infer it from code alone, do not graduate it. Feature-local friction dies with the board unless it clears the bar.

## Guardrails

- Abort-on-failure: promote env failure means **no** merge and **no** board delete.
- Never remove a worktree while merge failed.
- Never leave durable lessons only on a deleted feature board.
- Never invent the project principal Dev Container — fail hard + todo if missing and deltas exist.
- This skill is the normal close path; raw `valid merge` alone does not replace promote.

