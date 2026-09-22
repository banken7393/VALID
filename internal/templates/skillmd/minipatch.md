---
name: minipatch
description: >-
  Minimal change with an ephemeral board but no worktree/Dev Container. Same AC
  and test gates. Commit on the current branch; no feature-branch merge.
argument-hint: "[slug] what to change"
disable-model-invocation: true
user-invocable: true
---
# VALID: Minipatch

**Manual skill.** Invoke explicitly (e.g. `/minipatch`). Do not auto-start.

Minipatch answers **tiny change, still honest** — no quarantine ceremony.

Use when the change is minimal, has clear precedent in code, and does not need an isolated env.  
If shape is unclear → **plan** / **spec**. If isolation matters → **patch**.

## 1. Scaffold board only

```bash
valid feature new <slug> --mode minipatch
```

Creates `.valid/features/<slug>/` **without** worktree/DC. Expect `environment.isolation_warning` — by design, not a bug.

## 2. Agree done-list as ACs

Edit the board yourself: small `what[]` + `tasks[]` like `{ "id":"t1", "title":"…", "status":"pending", "covers":["ac1"] }` (`phase` if you added phases). Max ~5 testable lines.  
If you cannot point at a precedent in code, stop — this is **plan** / **spec**.

## 3. Change code on the current branch

Implement + run `.valid/config.json` → `test_command`. Update `tdd` on the board yourself (red → green). Soft isolation is expected; say so if you touch global toolchains anyway.

## 4. Gate

```bash
valid gate <slug>
```

Same hard rules as a feature: every AC covered, tests green.

## 5. Close

- `valid promote knowledge` / `valid promote env` if there is anything to graduate (env promote still **fail hard** on doubt).
- **Do not** `valid merge` (no feature branch / worktree).
- Human/agent commits on the **current** branch with explicit approval.
- Delete `.valid/features/<slug>/` when done:

```bash
valid feature delete <slug>
```

(or remove the directory after promote succeeds).

## Guardrails

- No worktree/DC by design.
- No relaxed gates.
- Never open a full board ceremony for a one-line copy change — but never skip evidence either.
- You own board edits; CLI scaffolds and gates.

