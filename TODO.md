# TODO — MissionControl

*From retro 2026-02-09. Prioritised by impact.*

## Process Enforcement

- [x] **Reject tasks for future stages** — PR #33
- [x] **Gate criteria per stage** — `mc-core check-gate` returns per-stage criteria
- [x] **`mc stage` validates gate** — calls `mc-core check-gate`, blocks with clear error, `--force` to bypass
- [x] **Auto-mark tasks done on findings** — watcher `findings_ready` → task status update in serve.go
- [ ] **Auto-advance stage on gate met** — when all gate criteria satisfied, prompt for advancement (don't auto-advance — gate is a conversation)

## Worker Lifecycle

- [ ] **Label-based worker registration** — register by label before spawn, match lifecycle events by label instead of sessionKey (which is unknown until spawn returns)
- [ ] **Worker self-reporting** — workers write a structured status block in findings that the watcher can parse
- [ ] **Token/cost tracking** — wire OpenClaw session stats into tracker, surface on dashboard
- [ ] **Worker chain** — auto-include predecessor task findings in briefings based on task dependencies

## Developer Experience

- [x] **Objective.md required** — `mc task create` warns if objective.md is missing/empty
- [x] **`mc status` after transitions** — prints human-friendly status box to stderr after `mc stage` and `mc task update`
- [ ] **Structured findings format** — define minimal schema: status, summary, decisions, blockers, files_changed
- [ ] **Lean briefings** — reference predecessor findings by path instead of duplicating context
- [ ] **Checkpoint on compaction** — automatic or prompted checkpoint before context window fills

## Dashboard Integration

- [ ] **Verify dashboard shows live data after .mission reset** — buildState/watcher may cache stale tasks after reinit
- [ ] **Register workers with tracking API** — every `sessions_spawn` should call `POST /api/mc/worker/register` so workers appear on mission screen

## Worker Coordination

- [ ] **Parallel worker boundaries** — when running parallel workers, either assign one worker per file or give explicit function-level modification instructions to avoid overlaps
- [ ] **Integration step after parallel workers** — mandatory review of all parallel worker output together before marking implement as done

## Process

- [ ] **Hard stop at every gate** — no batching stage transitions, one approval per stage
- [ ] **Never skip stages** — every stage exists for a reason, even if the output seems obvious
- [ ] **Planning produces a plan, not code** — planning stage output is task breakdown + approach, not implementation
- [ ] **Retro after every project** — mandatory retrospective after each mission completes, write to `memory/YYYY-MM-DD-{project}-retro.md`, extract lessons into MEMORY.md and update TODO.md
