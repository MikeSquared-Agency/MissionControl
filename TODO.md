# TODO — MissionControl

*From retro 2026-02-09. Prioritised by impact.*

## Process Enforcement

- [x] **Reject tasks for future stages** — PR #33
- [x] **Gate criteria per stage** — `mc-core check-gate` returns per-stage criteria
- [x] **`mc stage` validates gate** — calls `mc-core check-gate`, blocks with clear error, `--force` to bypass
- [x] **Auto-mark tasks done on findings** — watcher `findings_ready` → task status update in serve.go
- [x] **Auto-advance stage on gate met** — `mc stage` checks gates.json, advances without --force when all criteria met — PR #38

## Worker Lifecycle

- [x] **Label-based worker registration** — two-step register/link flow, register by label then bind sessionKey after spawn
- [x] **Worker self-reporting** — structured findings header with Task ID, Status, Summary
- [x] **Token/cost tracking** — regex parse from announcement text, calls tracker.UpdateTokens()
- [x] **Worker chain** — `mc briefing generate` auto-includes predecessor findings paths + summaries — PR #37

## Developer Experience

- [x] **Objective.md required** — `mc task create` warns if objective.md is missing/empty
- [x] **`mc status` after transitions** — prints human-friendly status box to stderr after `mc stage` and `mc task update`
- [x] **Structured findings format** — docs/findings-format.md, markdown header with Task ID, Status, Summary
- [x] **Lean briefings** — `mc briefing generate` references findings by path, extracts summaries — PR #37
- [ ] **Document stage changes in ARCHITECTURE.md** — gate enforcement, status summary, findings callback not yet documented
- [ ] **Update docs on every PR** — add to process checklist: ARCHITECTURE.md must reflect any new features
- [ ] **Checkpoint on compaction** — automatic or prompted checkpoint before context window fills

## Dashboard Integration

- [ ] **Verify dashboard shows live data after .mission reset** — buildState/watcher may cache stale tasks after reinit
- [ ] **Register workers with tracking API** — every `sessions_spawn` should call `POST /api/mc/worker/register` so workers appear on mission screen

## Worker Coordination

- [x] **Design for parallelism** — split features across files/packages so workers don't collide. If everything lands in one file, use one worker.
- [x] **Parallel worker boundaries** — `--scope-paths` flag on `mc task create` assigns file-level boundaries — PR #37
- [x] **Integration step after parallel workers** — integrator gate check in Rust enforces done integrator task when >1 implement task — PR #37
- [x] **TDD flow** — documented in SKILL.md, used successfully in PR #37 (tester → developer via --depends-on)

## Stage Discipline

- [x] **Don't skip validate** — now code-enforced: zero-task block + velocity check on non-exempt stages — PR #40
- [x] **Don't rubber-stamp stages** — now code-enforced: velocity check (<10s), zero-task block, mandatory reviewer/integrator — PR #40
- [ ] **Role removal is a design decision** — removing roles (Security, QA, etc.) must be flagged at Design stage and approved before implementing
- [ ] **Small changes still need Verify** — a 3-file change that removes security roles is exactly when review catches mistakes
- [ ] **Scope paths need type awareness** — if a task modifies a function using types from another file, include both files in scope_paths
- [ ] **Don't inflate retro scores** — score honestly at time of retro, don't revise upward later. Kai scored retro 6 as 8 in memory but it was 4 at the time.

## Gate UX

- [x] **`mc gate satisfy <criterion>`** — fuzzy match + `--all` flag, writes to gates.json — PR #38
- [x] **`mc gate status`** — shows ✓/✗ per criterion for current stage — PR #38
- [x] **`auto_mode` in config.json** — King can run without human gate approval — PR #38
- [x] **Document canonical status values** — "done" not "complete", added to ARCHITECTURE.md — PR #40
- [ ] **Enforce structured findings format** — validate Summary header on write, or add a linter. Workers that skip it break the briefing chain.
- [ ] **mc-core graceful fallback** — check-gate should handle stages without specific criteria instead of erroring

## Process

- [ ] **Hard stop at every gate** — no batching stage transitions, one approval per stage
- [ ] **Never skip stages** — every stage exists for a reason, even if the output seems obvious
- [ ] **Planning produces a plan, not code** — planning stage output is task breakdown + approach, not implementation
- [ ] **Retro after every project** — mandatory retrospective after each mission completes, write to `memory/YYYY-MM-DD-{project}-retro.md`, extract lessons into MEMORY.md and update TODO.md
