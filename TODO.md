# TODO — MissionControl

*From retros 2026-02-09 through 2026-02-10. Prioritised by impact.*

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
- [x] **Document stage changes in ARCHITECTURE.md** — stage enforcement table + findings callback section — PR #41
- [x] **Update docs on every PR** — added to CLAUDE.md process discipline + SKILL.md rules — PR #41
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

- [x] **Don't skip validate** — code-enforced: zero-task block + velocity check on non-exempt stages — PR #40
- [x] **Don't rubber-stamp stages** — code-enforced: velocity check (<10s), zero-task block, mandatory reviewer/integrator — PR #40
- [x] **Role removal is a design decision** — documented in SKILL.md rule #11, CLAUDE.md — PR #41
- [x] **Small changes still need Verify** — documented in SKILL.md rule #12, CLAUDE.md — PR #41
- [x] **Scope paths need type awareness** — documented in SKILL.md rule #15, CLAUDE.md — PR #41
- [x] **Don't inflate retro scores** — documented in SKILL.md rule #13 — PR #41

## Gate UX

- [x] **`mc gate satisfy <criterion>`** — fuzzy match + `--all` flag, writes to gates.json — PR #38
- [x] **`mc gate status`** — shows ✓/✗ per criterion for current stage — PR #38
- [x] **`auto_mode` in config.json** — King can run without human gate approval — PR #38
- [x] **Document canonical status values** — "done" not "complete", added to ARCHITECTURE.md + CLAUDE.md — PR #40, #41
- [x] **Enforce structured findings format** — `extractSummary()` warns when findings lack Summary header — PR #41
- [x] **mc-core graceful fallback** — verified: mc-core already handles all stages + missing mission dirs gracefully. Original error was gates.json format mismatch, fixed in PR #40.

## Process

- [x] **Hard stop at every gate** — documented in SKILL.md rule #9, CLAUDE.md. Code-enforced via velocity check — PR #40, #41
- [x] **Never skip stages** — documented + code-enforced via zero-task block — PR #40, #41
- [x] **Planning produces a plan, not code** — documented in SKILL.md rule #10, CLAUDE.md — PR #41
- [x] **Retro after every project** — documented in SKILL.md rule #13 — PR #41
