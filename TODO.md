# TODO — Remaining v6 Items

Items not yet completed from the v6 integration. See CHANGELOG.md for what shipped.

## OpenClaw Integration (B4 — partial)

The OpenClaw bridge code exists but the orchestrator binary hasn't been rebuilt with it. These items need the orchestrator service restarted with the bridge wired up:

- [ ] **B4.1–B4.3** REST endpoints + WS bridge — code merged, needs orchestrator rebuild
- [ ] **B4.4** Message relay — code merged, needs orchestrator rebuild
- [ ] **B4.7** `--openclaw-gateway` flag for `mc serve`
- [ ] **B5.4** Tests for OpenClaw endpoint handlers

## OpenClaw Skill & Config (D)

- [x] **D1** MissionControl skill file (`~/.openclaw/workspace/skills/missioncontrol/SKILL.md`)
- [x] **D2.1** OpenClaw agent config (model, sub-agents, compaction)
- [x] **D2.2** Pre-compaction memory flush referencing stages
- [x] **D2.3** Channel connectivity (WhatsApp, WebChat, Cloudflare Tunnel)
- [x] **D2.4** Project symlinks
- [ ] **D3** Agent Teams — test spawning with MC worker personas

## Integration Testing (F)

- [x] **F1** End-to-end: Kai ran full 10-stage pipeline (2026-02-07)
- [x] **F2** Multi-channel gate approval (integration test)
- [x] **F3** State persistence (integration test)
- [x] **F5** v5→v6 migration (integration test)
- [x] **F6** Full 10-stage walkthrough (integration test)
- [ ] **F7** Checkpoint round-trip: create → restart → verify briefing → verify state
- [ ] **F8** Auto-checkpoint: approve gate → verify checkpoint created + git committed
- [x] **F9** All tests green

## Checkpoint Skill (G6)

- [ ] **G6.1** Skill reads `current.json` on startup for briefing
- [ ] **G6.2** Pre-compaction calls `mc checkpoint` before memory flush
- [x] **G6.3** Skill documents checkpoint commands

## v5.1 Leftovers

These v5.1 items were superseded by the move to darlington.dev but some are still relevant:

- [ ] Homebrew tap distribution
- [ ] Dynamic project switching without orchestrator restart
- [ ] Sort sidebar by `lastOpened` descending
