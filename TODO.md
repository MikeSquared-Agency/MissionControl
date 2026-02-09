# TODO — MissionControl

*From retro 2026-02-09. Prioritised by impact.*

## Process Enforcement

- [ ] **Reject tasks for future stages** — `mc task create` should error if task.stage > current stage
- [ ] **Gate criteria per stage** — define what must be true before advancing (all tasks done, findings reviewed, etc.)
- [ ] **`mc stage` validates gate** — wire `mc-core check-gate` into stage advancement, block if criteria unmet
- [ ] **Auto-mark tasks done on findings** — watcher detects `findings_ready`, matches task by ID, sets status=done
- [ ] **Auto-advance stage on gate met** — when all gate criteria satisfied, prompt for advancement (don't auto-advance — gate is a conversation)

## Worker Lifecycle

- [ ] **Label-based worker registration** — register by label before spawn, match lifecycle events by label instead of sessionKey (which is unknown until spawn returns)
- [ ] **Worker self-reporting** — workers write a structured status block in findings that the watcher can parse
- [ ] **Token/cost tracking** — wire OpenClaw session stats into tracker, surface on dashboard
- [ ] **Worker chain** — auto-include predecessor task findings in briefings based on task dependencies

## Developer Experience

- [ ] **Objective.md required** — `mc task create` should warn/error if `.mission/state/objective.md` is empty
- [ ] **`mc status` after transitions** — CLI should print status summary after `mc stage` and `mc task update`
- [ ] **Structured findings format** — define minimal schema: status, summary, decisions, blockers, files_changed
- [ ] **Lean briefings** — reference predecessor findings by path instead of duplicating context
- [ ] **Checkpoint on compaction** — automatic or prompted checkpoint before context window fills

## Process

- [ ] **Retro after every project** — mandatory retrospective after each mission completes, write to `memory/YYYY-MM-DD-{project}-retro.md`, extract lessons into MEMORY.md and update TODO.md
