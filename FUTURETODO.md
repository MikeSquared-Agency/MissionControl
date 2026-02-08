# MissionControl — Roadmap

See [CHANGELOG.md](CHANGELOG.md) for completed work. See [TODO.md](TODO.md) for remaining v6 items.

## Version Status

| Version | Focus | Status |
|---------|-------|--------|
| v1 | Agent fundamentals (Python) | ✅ Done |
| v2 | Go orchestrator | ✅ Done |
| v3 | React dashboard | ✅ Done |
| v4 | Rust core | ✅ Done |
| v5 | King + mc CLI | ✅ Done |
| v5.1 | Quality of life | ✅ Done (UI items moved to darlington.dev) |
| v6 | State management + OpenClaw | ✅ Done (merged to main 2026-02-08) |
| v7 | Requirements & traceability | Next |
| v8 | Infrastructure & scale | Future |
| v9 | Advanced UI | Future |

---

## v7 — Requirements & Traceability

Full traceability from business needs to implementation.

- **7.1** Requirements directory structure (`requirements/` with levels)
- **7.2** Requirement CRUD: `mc req create/list/show/update`
- **7.3** Requirement hierarchy via `derivedFrom` field
- **7.4** Task-to-requirement refs: `mc task create --req REQ-CAP-001`
- **7.5** Task-to-spec refs: `mc task create --spec SPEC-auth.md --section FR-1`
- **7.6** Impact analysis: `mc req impact <id>` — blast radius
- **7.7** Coverage report: `mc req coverage` — implementation status rollup
- **7.8** Spec status: `mc spec status <file>` — completion per section
- **7.9** Spec orphans: `mc spec orphans` — sections without implementing tasks
- **7.10** Trace command: `mc trace <id>` — full lineage up and down
- **7.11** Requirements index cache for fast queries
- **7.12** Auto-generate tasks from spec: `mc task create --from-spec <file>`

---

## v8 — Infrastructure & Scale

Cost optimization and team features.

- **8.1** Multi-model routing (Haiku/Sonnet/Opus) based on task complexity
- **8.2** Cost tracking & budgets (token usage per task/worker/session)
- **8.3** Worker health monitoring (heartbeat, timeout, auto-restart)
- **8.4** Headless mode for CI/CD pipelines
- **8.5** Remote bridge deployment (Docker, auth, multi-tenant)

---

## v9 — Advanced UI (darlington.dev)

- **9.1** Requirements panel (tree view, coverage indicators)
- **9.2** Dependency graph visualization (D3/interactive)
- **9.3** Traceability view (lineage explorer)
- **9.4** Audit log viewer (timeline, filters, diffs)

---

## v10 — 3D Visualization (speculative)

- React Three Fiber agent avatars in 3D space
- Zone visualization, camera controls, animations
