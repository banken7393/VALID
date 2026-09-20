---
name: delegate
description: >-
  Hand a phase or task chunk to the delegate agent with higher autonomy
  (above the loop). Stream trivial work; stop on real decisions. Invoke
  explicitly during /build — never auto-run.
argument-hint: "[feature-slug] [phase-id|task-range]"
disable-model-invocation: true
user-invocable: true
---

# VALID: Delegate

**Manual skill.** Invoke explicitly (e.g. `/delegate`). Do not auto-start.

Use this when you want the agent to **own a bounded chunk** of implementation
instead of stopping on every trivial task. Same gates as **build**; more
autonomy, not less rigor.

Board: `.valid/features/<slug>/data.json`.

## Produces

- Board `autonomy` set to `above_the_loop` for the duration of the chunk
  (restore `in_the_loop` when the human wants full presence again)
- Progress on the delegated phase / tasks with honest `tdd` evidence
- A clear stop report: done / blocked / next decision

## 0. Resolve

1. Arg slug → that feature. None → unfinished board under `.valid/features/*/`.
2. Optional second arg: phase id, task id, or “phase N trivial tasks”.
3. Confirm with the human: **what is delegated**, **what must still stop**.
4. Set board `autonomy` to `above_the_loop` (edit JSON or
   `valid board set <slug> --autonomy above_the_loop`).
5. Invoke agent **delegate** (`.valid/agents/delegate.md`). You still own the board.

## 1. Bound the chunk

Write (or confirm) on the board:

- Which `phases[].id` / `tasks[]` are in scope
- Decisions already agreed (must not be reopened silently)
- Hard stops: security, schema changes, new AC, deps that need promote

## 2. Execute

Follow skill **build** TDD mechanics inside the feature worktree/DC:

- Stream **trivial** tasks without pausing for each one
- Pause on substantial tasks, phase boundaries, and hard stops
- Keep `tasks[]`, `covers`, and `tdd` current by editing the board

## 3. Leave the chunk

1. Re-run `test_command`; board must reflect green or an honest fail.
2. Summarize: files, tasks done, ACs covered, risks.
3. Ask: stay `above_the_loop`, return to `in_the_loop`, or hand off to **audit**.

Optional restore:

```bash
valid board set <slug> --autonomy in_the_loop
```

## Guardrails

- Delegate ≠ skip audit/finish. Same AC↔task + test gates.
- Never merge or promote from this skill.
- Never invent ACs to keep streaming.
- MCP is knowledge RAG + project scripts — not a fake TDD channel.
