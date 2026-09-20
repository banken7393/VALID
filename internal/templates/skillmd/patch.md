---
name: patch
description: >-
  Small change that still needs quarantine (worktree + feature Dev Container) and
  full gates. Shorter ceremony than a full plan/spec, same rigor on ACs and tests.
argument-hint: "[slug] what to change"
disable-model-invocation: true
user-invocable: true
---
# VALID: Patch

**Manual skill.** Invoke explicitly (e.g. `/patch`). Do not auto-start.

Patch is a **short VALID cycle with quarantine** — not a free-for-all.

Use when the change is small but still benefits from isolation (deps, risky edits, anything that might dirty the host or principal env).  
If there is **no precedent in code** and the shape is unclear → use **plan** / **spec** instead.  
If it is trivial and needs no quarantine → **minipatch**.

VALID cycle (compressed): scaffold → tiny WHAT → build/TDD → audit → finish.

## 1. Scaffold

```bash
valid feature new <slug> --mode patch --north-star "…"
```

Creates board + worktree + feature Dev Container (same quarantine as Plan). Soft-warn if DC up fails.

Review `.valid/worktrees/<slug>/.devcontainer/` and adjust `BASE_IMAGE` if needed. Log deltas in `deps-delta.md` / `pending_promotions[]`.

## 2. Agree a tiny WHAT

With the human, write a short `what[]` (and sketch `tasks[]` with `covers`) by **editing** `data.json`.  
Same AC quality bar as Spec — just fewer items. Reject vague “fix it” without a testable behaviour.

Set `lifecycle` appropriately (`spec` then `build` as you go).

## 3. Build with TDD

Follow skill **build** (developer agent, `test_command`, board evidence, in-the-loop by default). Prefer the feature worktree.

## 4. Audit → Finish

Same gates as a full feature:

```bash
valid gate <slug>
```

Then skill **finish** (promote knowledge/env fail-hard → merge → delete board → env down).

## Guardrails

- Full dual gate (AC coverage + green tests). **No shortcuts** because the change is “small”.
- Prefer quarantine; soft-warn if isolation fails.
- You own the board files; CLI is scaffold + gate/promote/merge plumbing.
- Do not open a full Plan ceremony for a one-line copy tweak — but never skip evidence either (use **minipatch** if no quarantine is needed).

