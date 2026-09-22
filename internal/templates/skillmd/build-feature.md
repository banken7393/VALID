---
name: build-feature
description: >-
  VALID method: decide the HOW and implement it with TDD in the quarantined
  worktree/Dev Container. Keep the feature board current (decisions, tasks with
  covers, test evidence). Invoke explicitly after /spec — never auto-run.
argument-hint: "[feature-slug]"
disable-model-invocation: true
user-invocable: true
---

# VALID: Build

**Manual skill.** Invoke explicitly as **`/build-feature`** (not `/build` — that name conflicts with Cursor UI). Do not auto-start.

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
2. **Path contract (non-negotiable for feature/patch):** stay in the **principal** repo folder (multi-agent hub). Print and obey:

```bash
valid paths <slug>
```

| Path | Write here |
|------|------------|
| `CODE_ROOT` (`.valid/worktrees/<slug>/`) | **All feature product code** |
| `BOARD` / Mermaid / `WORKSPACE` under `.valid/features/<slug>/` | Board only |
| Principal tree outside `.valid/` | **Forbidden** during build (contamination) |

Do **not** “open the worktree as the IDE root”. Resolve file paths under `CODE_ROOT`. If you already dirtied the principal: stop, move changes into `CODE_ROOT`, clean the principal, run `valid doctor <slug>`.
3. Soft env isolation (`devcontainer up` failed) → set `environment.isolation_warning` and continue; still write code under `CODE_ROOT`.
4. Invoke agent **developer** (`.valid/agents/developer.md`) for implementation focus; you still own the board file.
## 1. Read

Read `data.json` (the WHAT), Mermaid how-it-works, and MCP knowledge (conventions/decisions) read-only. The drawing is the map for decisions and every task.

Read `test_command` from `.valid/config.json`. That command is language-agnostic truth for this repo — VALID does not own your runner.

## 2. Decide the approach, out loud

Phases come from Spec (`id` / `name` / `outcome` / `status`); take the first whose `status` is not `done`. No `phases[]` → the whole feature is one phase. Never disappear and reappear with a finished plan.

1. Agree the phase’s key **`decisions`**: the few interdependent choices that must precede code, each with its why. One at a time; the human reshapes. Persist each by editing the board.
2. **Sketch** the phase’s tasks lightly — enough to see shape and coverage, no more. Detail is decided right before each task is built; the list may grow as the build teaches you.

Canonical task shape (dashboard groups by `phase`, colours status):

```json
{
  "id": "t1",
  "title": "Seed A1–C3 on city create",
  "status": "pending",
  "phase": "p1",
  "covers": ["ac1", "ac2"],
  "description": "Optional short note"
}
```

| Field | Required | Notes |
|-------|----------|--------|
| `id` | yes | `t1`, `t2`, … |
| `title` | yes | Short work label |
| `status` | yes | `pending` (orange) \| `doing` (teal) \| `done` (green/blue accent) \| `failed` (red) |
| `phase` | when `phases[]` exists | Must match `phases[].id` (`p1`, …) |
| `covers` | yes | ≥1 `what[].id` this task satisfies |
| `description` | no | Extra context |

3. **Coverage**: every AC in the phase covered by ≥1 task. Close every gap in the sketch, then report `Coverage: N/N, 0 gaps`.
4. Set the active phase `status` to `doing` while working; `done` only when its `outcome` is honestly true and covered tasks are `done`. Mark a task `failed` if blocked / red for real — do not leave it fake-`done`.

Example decision / assumption / phase shapes:

```json
{ "id": "D1", "title": "short question or name", "detail": "chosen option + rationale" }
{ "id": "A1", "detail": "assumption. Breaks if wrong: …" }
{ "id": "p1", "name": "…", "outcome": "…", "status": "doing" }
```

Do **not** use `question` / `choice` / `text` / `decided_at` on decisions — map those into `title` + `detail` (put the date inside `detail` if needed).  
Do **not** use `title` / `order` on phases — use `name` + `outcome` + `status`; sequence is array order.  
Do **not** omit `phase` or `covers` on tasks when phases/ACs exist.

## 3. Build with TDD, task by task

### Board-first (non-negotiable)

The dashboard only shows what is in `data.json`. **Status moves on the board before the work, not after.**

For **every** task, in this order:

1. **Claim it on the board first** — edit `data.json` and set that task `"status": "doing"` (and the active phase to `"doing"` if it is not already). Do this **before** any test, code, or tool call for that task. If you skip this, the board lies (still `pending` while you are mid-work).
2. Then decide detail with the human when there is a real choice (always in `in_the_loop`; on substantial work in `above_the_loop`).
3. **Red** — write a failing test; set `tdd.phase` to `red`; record the failing case under `tdd.cases`.
4. Run `test_command` **inside `CODE_ROOT`** (feature worktree / Dev Environment when available). Soft warn if the DC is down — still do not write product files on the principal tree.5. **Green** — implement until green; update cases to `pass`; `tdd.phase` = `green`.
6. **Refactor** — clean up; keep green; `tdd.phase` = `refactor` then back to green.
7. **Close it on the board immediately** — `"status": "done"` only when covered ACs are honestly satisfied; `"failed"` if blocked or the suite stays red for that work. Never leave a task `doing` while you start the next one. Never leave a lie as `done`.
8. Keep `phase` and `covers` accurate as you edit the board.
9. Report briefly: which file(s), what changed, what it covers (task / AC / phase), why that way. Enumerate files — never “I created the data layer”.

**One task in `doing` at a time** (unless the human explicitly batches). The board is the live signal; chat is not.

### Loop mode (board `autonomy`)

| Mode | When | Behaviour |
|------|------|-----------|
| **`in_the_loop`** (default) | Everyday pairing | Human present on **every** task. Do not silently go autonomous. |
| **`above_the_loop`** | Human said “stream the trivial ones” / skill **delegate** | Stream low-risk tasks that follow agreed decisions; **stop** on substantial tasks, phase boundaries, friction, or risk. Still **board-first**: flip each task to `doing` before starting it. |

Set explicitly (human or skill **delegate**):

```bash
valid board set <slug> --autonomy above_the_loop   # or in_the_loop
```

For a bounded phase/chunk with a stop report, prefer skill **`/delegate`** (agent **delegate**).

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

- **Board-first:** never start code or tests for a task while it is still `pending` — flip to `doing` in `data.json` first; flip to `done`/`failed` before picking up the next task.
- **Path contract:** product code only under `CODE_ROOT` from `valid paths <slug>`. Principal stays the control plane. Run `valid doctor <slug>` if unsure; `isolation.strict` makes contamination fail the gate.
- Prefer feature worktree + DC. Soft isolation warning if DC is down — say so on the board (`isolation_warning`).
- Never mark tests green without running `test_command`.
- MCP is knowledge RAG **plus** project scripts; do not use it to fake TDD/ACs.
- No silent scope expansion: new behaviour needs a new AC (edit `what[]` with the human).
- Do not modify other features’ boards. Cross-cutting discoveries wait for **finish** graduation.
