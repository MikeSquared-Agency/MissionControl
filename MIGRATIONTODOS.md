# DutyBound — Hibernation Migration TODOs

*Tracking all work to get DutyBound shipped and safely hibernated.*
*See the full plan in the hibernation plan document.*

---

## Stream 0: Branding

- [x] ~~0.1 Rename GitHub repo to `dutybound`~~ — **SKIPPED** (doing later or manually)
- [x] 0.2 Update README header, tagline, description — DutyBound name, one-liner, quick brand pass
- [x] 0.3 Update `darlington.dev` route to `/dutybound` — demo URL

## Stream 1: Stabilise & Document

- [x] 1.1 Fix CI pipeline — fixed: `--validate-only` exits cleanly when no `.mission/` dir
- [x] 1.2 Full README rewrite — reframed as DutyBound, "Why DutyBound?" section, updated architecture diagram, process enforcement, quick start, stack, naming hierarchy
- [x] ~~1.3 Update agent personas~~ — **SKIPPED** (low priority for now)
- [x] 1.4 Remove `--force` from Kai — added explicit ban to openClawPrompt and .mission/CLAUDE.md
- [x] 1.5 Fix broken dashboard views — fixed: "done" vs "complete" status mismatch across all components, token WS handler now uses adaptTokens() instead of shallow merge, audit filters derived from actual data

## Stream 2: On-Demand Container

- [x] 2.1 Wake-on-request endpoint — `mc launcher` with `/api/wake` + auto-wake on any request; `mc wake` CLI command
- [x] 2.2 Auto-shutdown on idle — launcher stops both services after `--idle-timeout` (default 30m)
- [x] 2.3 Health/status page — `dutybound-client.tsx` polls `/api/health`, shows loading state, renders KaiClient when ready

## Stream 3: Kai Setup Wizard

- [ ] 3.1 Conversational onboarding flow — Kai greets user, guides through setup naturally
- [ ] 3.2 Repo cloning capability — Kai can `git clone` repos under user's GitHub
- [ ] 3.3 Auto-gate option in onboarding — surface existing `auto_mode` as a choice during setup
- [ ] 3.4 UI transition animation — chat slides left, full MC dashboard reveals

## Stream 4: Guided Demo

- [ ] 4.1 Create demo repository — small self-contained project, pre-configured tasks and scope paths
- [ ] 4.2 Demo mode restrictions — read-only for visitors, Kai runs autonomously with auto-gating
- [ ] 4.3 Demo reset — reset demo repo to initial state (`mc demo reset` or similar)

## Stream 5: Article

- [ ] 5.1 Write and publish blog post — provenance story, architectural journey, "King should BE Claude Code" insight

---

## Known Bugs & Issues (document as found)

- [ ] Dashboard may cache stale tasks after `.mission` reset
- [ ] Workers not registered with tracking API on spawn
- [ ] Checkpoint on compaction not implemented
- [ ] Frontend agent views partially broken
- [ ] Frontend token usage display broken
- [ ] Frontend audit filters broken

---

## Out of Scope (backlog)

- Orchestrator full rebuild (spec exists, only implement what demo needs)
- v7 Requirements & Traceability (done per plan but listed as "next" in FUTURETODO)
- v8 Infrastructure (multi-model routing, remote bridge, Docker multi-tenant)
- v9 Advanced UI (dependency graph, audit viewer — done per plan)
- v10 3D Visualization
- Multi-user sandbox (auth, isolation, rate limiting)
- WhatsApp integration for Kai
- C# / work integration
