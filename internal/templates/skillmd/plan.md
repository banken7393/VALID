---
name: plan
description: >-
  Manually invoke to give the feature its shape before anything is specified.
  Interview until nothing blocking is open, look facts up in the code, settle
  assumptions, and draw how it works (Mermaid). Scaffolds quarantined worktree +
  feature Dev Container. Invoke before /spec — never auto-run.
argument-hint: "[feature-slug]"
disable-model-invocation: true
user-invocable: true
---

# VALID: Plan

**Manual skill.** Invoke explicitly (e.g. `/plan` or “run plan”). Do not auto-start.

VALID cycle: **Plan** → Spec → Build → Audit → Finish.

Plan answers **what are we building, and does the mechanism hold up**.

Board: `.valid/features/<slug>/data.json` (schema v2). JSON **keys** stay English; board **prose** follows the repository language.  
How-it-works: `.valid/features/<slug>/how-it-works.mmd` (Mermaid — never SVG).

**You own the board.** Edit `data.json`, Mermaid, and workspace markdown with your file tools.  
The CLI is scaffolding only (`valid feature new` creates empty structure + worktree + quarantine Dev Container). Do not treat CLI `board`/`task`/`report` as the primary writers.

## Board JSON contract (strict keys)

The dashboard and CLI **fail to load** the board if shapes are wrong. Always use these shapes:

```json
"what": [{ "id": "ac1", "description": "…" }]
"decisions": [{ "id": "D1", "title": "short name", "detail": "full rationale / choice" }]
"assumptions": [{ "id": "A1", "detail": "… Breaks if wrong: …" }]
"tasks": [{ "id": "t1", "title": "…", "status": "pending", "phase": "p1", "covers": ["ac1"] }]
"phases": [{ "id": "p1", "name": "…", "outcome": "…", "status": "agreed" }]
"environment.isolation_warning": false
```

Forbidden / broken shapes (do **not** write these):

- `what` as string array
- `decisions[].question` / `decisions[].choice` / `decisions[].text` — use **`title`** + **`detail`** only
- `assumptions[].text` instead of `detail`
- `phases[].title` / `phases[].order` — use **`name`** + **`outcome`** + **`status`**; sequence = array order (no `order` field)
- `tasks` without `covers` / `phase` when phases exist — every task needs `covers` (AC ids) and `phase` (`p1`…) when `phases[]` is non-empty
- `isolation_warning` as a prose string (must be boolean)
- Mermaid under `workspace/how-it-works.mmd` — correct path is `.valid/features/<slug>/how-it-works.mmd`
- Putting the full Mermaid only in `data.json` `how_it_works` — keep Mermaid in the `.mmd` file; JSON field may be empty or a one-line note

Decision example (canonical — dashboard reads these keys):

```json
{
  "id": "D1",
  "title": "¿Corregir mapping definitions ↔ docs?",
  "detail": "Sí. Biodiversidad→1.10; …"
}
```

Wrong (dashboard shows empty / "—"):

```json
{ "id": "D1", "question": "…", "choice": "…", "decided_at": "2026-09-22" }
```

Phase example (canonical — Spec writes these; Build consumes them):

```json
{
  "id": "p1",
  "name": "Defaults al crear ciudad",
  "outcome": "Al crear una ciudad, A1–C3 nacen con mapping correcto…",
  "status": "agreed"
}
```

Wrong: `{ "id": "p1", "title": "…", "order": 1 }` — no `title`/`order`; use `name`/`outcome`/`status`.

Task example (canonical — Build writes these; dashboard groups by `phase`):

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
| `id` | yes | Stable `t1`, `t2`, … |
| `title` | yes | Short work label |
| `status` | yes | `pending` \| `doing` \| `done` \| `failed` |
| `phase` | when phases exist | Must be a `phases[].id` |
| `covers` | yes | One or more `what[].id` |
| `description` | no | Extra context |

Wrong: flat `"covers=ac1"` prose, missing `phase`, or inventing status values outside the four above.## Produces

- `north_star`, early `assumptions[]`, `decisions[]` candidates
- `how_it_works` + `how-it-works.mmd` — the mechanism drawing
- Feature worktree under `.valid/worktrees/<slug>/` + ephemeral Dev Container (quarantine)
- Minimal board ready for **spec**

Nothing else. Acceptance criteria and tasks belong to later skills.

## 0. Route

1. Agree a kebab-case `slug` with the human (or use `$ARGUMENTS`).
2. Read `.valid/config.json` and search `.valid/knowledge/**` via MCP (`search_knowledge`) **before** asking anything you can look up.
3. Scaffold once (creates board dir, worktree, quarantine DC):

```bash
valid feature new <slug> --mode feature --north-star "…"
```

   - Board already exists → **read and resume**. Never overwrite Plan outputs without consent.
   - Soft isolation: if Dev Container up fails, warn on the board (`environment.isolation_warning`) and continue; do not hard-stop.

4. Keep open: `.valid/features/<slug>/data.json` and the worktree `.valid/worktrees/<slug>/`.
5. Optional live view: `valid dashboard --feature <slug>` (port from `dashboard_port` in config).

## 1. Interview until nothing blocking is open

Ask **every question that can be answered now, in one numbered round**, each with your **recommended answer**. Then wait. A question that depends on an answer you have not heard yet belongs to a later round.

- **Facts are your job, decisions are theirs.** If the code answers it, read the code. Never ask what you can look up.
- Explain before asking when context is needed — and never explain a component from memory. Read it first.
- Challenge. Offer the simpler shape when you see one.
- Persist answers as you go by **editing** `data.json` (`assumptions`, `decisions`, refinements to `north_star`). Never batch-write only at the end.
- An answer you cannot get today is not a blocker. Write an `assumption` with what breaks if it is wrong, and move on.
- Done when nothing **blocking** is open and the human confirms — not when you run out of questions.

## 2. Prove what you cannot settle by reading

Only for technology that docs alone cannot settle. Guide a hands-on probe inside the **feature** Dev Container when it is up: install, hello-world, the capabilities this feature needs, then push until it breaks. Mark on the board what was tested vs assumed. Never claim “proven” from documentation alone — that is an assumption.

## 3. Draw how it works

Hand-author `.valid/features/<slug>/how-it-works.mmd` (Mermaid flowchart or sequence). Point `data.json` at it (`how_it_works` and/or `paths.how_it_works`).

**If you cannot draw it, it is not planned.** Exception: there is no mechanism (no flow, boundary, or state change) — write that sentence on the board and say the change may be too small for a full feature (consider **minipatch** / **patch**).

## 4. Quarantine env check (no conflict with principal)

Review `.valid/worktrees/<slug>/.devcontainer/` only — **never** edit the project principal `.devcontainer/`.

- Cloned from the principal DC if one existed (Dockerfile siblings copied); otherwise VALID quarantine (`Dockerfile` with UID 1000 + `ARG BASE_IMAGE` + docker-outside-of-docker).
- Container name is `valid-<slug>` (unique `runArgs --name`) so it does not collide with the principal container.
- Host ports: feature DC forwards **only** the feature dashboard port from config (`feature_dashboard_port`, or auto `dashboard_port + feature_port_offset + slug salt`). It does **not** re-bind principal `forwardPorts` (app/DB/dashboard/MCP). Adjust `BASE_IMAGE` / features inside the feature env if needed.
- Record durable deltas in `workspace/deps-delta.md` and `pending_promotions[]` for later `promote env` (fail-hard; principal is never invented).

## 5. Leave

- Set `lifecycle` to `plan` (edit the board).
- Glossary/convention candidates: note them; durable knowledge writes happen at **finish** (or a careful MCP `upsert_document` if truly cross-cutting and the human agrees now).
- Next: skill **spec** (manual invoke).

## Guardrails

- No production feature code in Plan (throwaway probes only).
- Never present `valid plan` as a CLI command — this Markdown **is** the method.
- MCP is knowledge RAG **plus** project scripts under `.valid/scripts/`; never use it to fake board/TDD state.
- Soft isolation only; do not block the human for a missing DC.
- Never write the principal Dev Container from this skill.
