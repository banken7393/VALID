---
name: build
description: >-
  Decide the HOW and implement it with TDD inside the quarantined worktree/Dev
  Keep the feature progress board current: decisions, tasks with covers, and
  executable test evidence. Invoke explicitly after /spec to implement with TDD — never auto-run.
argument-hint: feature-slug
disable-model-invocation: true
user-invocable: true
---
# VALID: Build

**Manual skill.** Invoke explicitly (e.g. `/build`). Do not auto-start.

VALID cycle: Plan → Spec → **Build** → Audit → Finish.

Build answers **how we make the WHAT true**, then does it, with evidence.

Board: `.valid/features/<slug>/data.json` — single source of truth. **You edit** tasks, covers, decisions, assumptions, TDD fields, and notes as you go. Optional CLI helpers (`valid task`, `valid report`) exist for power users; prefer editing the board yourself so the LLM stays in control.

## Produces

- Working code on branch `feature/<slug>` inside `.valid/worktrees/<slug>/`
- `decisions[]` — the few architectural choices that must precede code
- `tasks[]` with `covers` → AC ids (`what[].id`)
- `tdd` evidence (cases + pass/fail) aligned with `.valid/config.json` → `test_command`
- Updated `workspace/deps-delta.md` / `pending_promotions[]` when the feature env gains deps
- `lifecycle`: `build`

## 0. Resolve

1. Arg given → that slug. None → the unfinished feature board under `.valid/features/*/`; several → ask. None anywhere → “Run **spec** first” and stop.
2. If `.valid/worktrees/<slug>/` exists and the session is outside it: prefer entering it before writing code. Soft-warn and continue if you must work outside (`environment.isolation_warning`).
3. Invoke agent **developer** (`.valid/agents/developer.md`) for implementation focus; you still own the board file.

## 1. Read

Read `data.json` (the WHAT), Mermaid how-it-works, and MCP knowledge (conventions/decisions) read-only. The drawing is the map for decisions and every task.

Read `test_command` from `.valid/config.json`. That command is language-agnostic truth for this repo — VALID does not own your runner.

## 2. Decide the approach, out loud

Phases come from Spec; take the first not done. No `phases[]` → the whole feature is one phase. Never disappear and reappear with a finished plan.

1. Agree the phase’s key **`decisions`**: the few interdependent choices that must precede code, each with its why. One at a time; the human reshapes. Persist each by editing the board.
2. **Sketch** the phase’s tasks lightly (`id`, `title`, `status`, `covers`): enough to see shape and coverage, no more. Detail is decided right before each task is built; the list may grow as the build teaches you.
3. **Coverage**: every AC in the phase covered by ≥1 task. Close every gap in the sketch, then report `Coverage: N/N, 0 gaps`.

Example task / decision shapes:

```json
{ "id": "t1", "title": "…", "status": "pending", "covers": ["ac1"] }
{ "id": "D1", "title": "short name", "detail": "rationale" }
{ "id": "A1", "detail": "assumption. Breaks if wrong: …" }
```

Do not use a `text` field on decisions/assumptions — use `title`/`detail`.

## 3. Build with TDD, task by task

### Loop mode (board `autonomy`)

| Mode | When | Behaviour |
|------|------|-----------|
| **`in_the_loop`** (default) | Everyday pairing | Human present on **every** task. Do not silently go autonomous. |
| **`above_the_loop`** | Human said “stream the trivial ones” / skill **delegate** | Stream low-risk tasks that follow agreed decisions; **stop** on substantial tasks, phase boundaries, friction, or risk. |

Set explicitly (human or skill **delegate**):

```bash
valid board set <slug> --autonomy above_the_loop   # or in_the_loop
```

For a bounded phase/chunk with a stop report, prefer skill **`/delegate`** (agent **delegate**).

For each task:

1. Decide detail with the human when there is a real choice (always in `in_the_loop`; on substantial work in `above_the_loop`).
2. **Red** — write a failing test; set `tdd.phase` to `red`; record the failing case under `tdd.cases`.
3. Run `test_command` **inside the feature Dev Environment** when available (soft warn if not).
4. **Green** — implement until green; update cases to `pass`; `tdd.phase` = `green`.
5. **Refactor** — clean up; keep green; `tdd.phase` = `refactor` then back to green.
6. Mark task `done` only when covered ACs are honestly satisfied by the code.
7. Report briefly: which file(s), what changed, what it covers (task / AC), why that way. Enumerate files — never “I created the data layer”.

Update aggregates (`passed` / `failed` / `total`) when you change cases.

**Friction:** a gotcha, surprise, plan that turned out wrong, or code that no longer fits the drawing → record on the board (decision/assumption/note) and **stop** to surface it. Never bury it.

**Deps:** install/try packages inside the feature DC only. Log durable deltas in `deps-delta.md` + `pending_promotions[]`.

## 4. Phase boundary

Before leaving a phase (and before Audit):

1. Coverage again: every AC in the phase covered by a `done` task.
2. Re-run `test_command`; board TDD must be green.
3. Diff sanity: nothing unrelated sneaked in.
4. Present residual risks in a short paragraph; wait for “next” / approval.

## 5. Leave

All tasks done and the WHAT looks implemented with green tests: tell the story in a few lines (what shipped, shaping decisions, what stayed open). Suggest skill **audit**. Do not merge yourself. Commits/merge are **finish**’s (or the human’s explicit call).

## Guardrails

- Prefer feature worktree + DC. Soft isolation warning if outside — say so on the board.
- Never mark tests green without running `test_command`.
- MCP is knowledge RAG **plus** project scripts; do not use it to fake TDD/ACs.
- No silent scope expansion: new behaviour needs a new AC (edit `what[]` with the human).
- Do not modify other features’ boards. Cross-cutting discoveries wait for **finish** graduation.

