---
name: spec
description: >-
  Define WHAT to build on the feature progress board. Collaborative Q&A; write north star,
  acceptance criteria, and phases so the WHAT appears on the board as you define it.
  Decisions and tasks come later in build.
argument-hint: feature-slug
disable-model-invocation: true
user-invocable: true
---
# VALID: Spec

**Manual skill.** Invoke explicitly (e.g. `/spec`). Do not auto-start.

VALID cycle: Plan → **Spec** → Build → Audit → Finish.

Spec answers **what counts as done**. Never how.

Board: `.valid/features/<slug>/data.json`. **You edit the board** with your file tools. CLI is not required to write ACs.

## Produces

- `north_star` — the usable outcome of the whole feature
- `phases[]` — vertical slices with `name` + `outcome` (only if more than one slice)
- `what[]` — acceptance criteria: indivisible, testable, each with stable `id`

Nothing else. Decisions, architecture, file paths, and tasks belong to **build**. At Spec time those columns stay empty **by design**.

## 1. Bootstrap

1. Resolve `slug` from `$ARGUMENTS` or ask. Prefer working inside the feature worktree Plan created (`.valid/worktrees/<slug>/`).
2. Read the existing board. **Never overwrite** Plan outputs (`assumptions`, Mermaid, early decisions) without consent.
3. Read MCP knowledge (`search_knowledge`) + `.valid/knowledge/context/project.md` before asking.
4. If there is no board yet, stop and run skill **plan** first (or scaffold with `valid feature new` only if the human insists on skipping Plan — still require Mermaid or an explicit “no mechanism” note).

## 2. Ask only what Plan did not answer

Synthesize from Plan’s `assumptions[]` / `decisions[]` / Mermaid first. What they do not cover is yours to ask.  
A blocking fork that reopens the mechanism → go back to **plan**. One missing answer does not.

Discussion proportional to complexity. Challenge assumptions (edge cases, implicit requirements, conflicts with existing knowledge). Detect gaps (error and empty states, permissions, boundaries). Offer the simpler or reusable shape. **Reject vague words** (“basic”, “simple”, “standard”) until behaviour is concrete.  
One focused round at a time, by impact.

## 3. Agree the spine

- `north_star`: what “done” feels like — the usable outcome of the whole feature.
- `phases[]` only if more than one usable slice. Each phase ships **one** usable, testable outcome; phase 1 proves the core assumption. Keep count small (~4 max) or split into features.

Canonical phase shape (array order = sequence — **do not** write `order` or `title`):

```json
{
  "id": "p1",
  "name": "Defaults al crear ciudad",
  "outcome": "Al crear una ciudad, … measurable done-state for this slice.",
  "status": "agreed"
}
```

| Field | Required | Notes |
|-------|----------|--------|
| `id` | yes | Stable `p1`, `p2`, … |
| `name` | yes | Short label |
| `outcome` | yes | What “this slice is done” means (testable) |
| `status` | yes | `agreed` at Spec; Build may move to `doing` / `done` |

Wrong: `{ "id": "p1", "title": "…", "order": 1 }`.

Persist by editing `data.json` as you go. Set `lifecycle` to `spec`.

## 4. Derive ACs one at a time

With Mermaid how-it-works: every labeled edge/boundary is a candidate AC (behaviour, error path, empty state).  
Without a drawing: derive from conversation only.

Shape each AC as:

```json
{ "id": "ac1", "description": "…" }
```

Never write `what` as a bare string array. Wrong: `"what": ["…"]`. Right: objects with `id` + `description`.

Rules:

- Minimal, indivisible, testable.
- Confirm with the human and **write it to `what[]` before the next**.
- Tiny batches only if they ask.
- **Never** dump a finished set for one-shot approval — that is the failure mode this step guards against.

## 5. Audit the WHAT

Walk ACs against: multiplicity, lifecycle (CRUD), ownership, empty state, failure modes, boundaries, temporal triggers, conflicts with knowledge. Resolve every gap with the human; update `data.json`.

Optionally self-check claims against code/docs (evidence-based): an AC that is not testable, two that contradict, or a claim about the code that is false must be fixed before presenting.

## 6. Leave

Show the board / `valid dashboard --feature <slug>` so the human can visually check progress. Iterate until they approve. Then skill **build-feature** (`/build-feature`).  
Do not invent `tasks[]` here unless the human insists on a tiny preview — and even then, Build owns HOW.

## Guardrails

- **WHAT, not HOW.** No implementation decisions, types, file paths, or task lists as the Spec deliverable.
- Spec never implements production code.
- Do not call `valid board set` as the primary writer — **edit the JSON**.
- The human decides what the thing is. There is no mode that hands that over.
- Keys English; descriptions in the repo language.

