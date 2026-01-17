# MissionControl — Personas, Workflow & Agents Spec

## Overview

This spec defines the agent personas, workflow phases, zone structure, and token-efficient architecture for MissionControl. Designed for solo developers who want coordinated multi-agent workflows without losing context.

---

## Core Principles

1. **King coordinates, never implements** — King spawns workers, manages context, gates phases. Never writes code.
2. **Workers are ephemeral** — Spawn → task → die. Fresh workers beat bloated context.
3. **Files are truth, briefings are context** — Full specs persist in `.mission/`. Workers receive distilled briefings.
4. **Phases have gates** — No rushing to implementation. Explicit approval between phases.
5. **Audience level determines rigor** — Personal vs External changes how thorough agents need to be.

---

## Audience Levels

| Level | Review | Security | Tester | QA | Docs |
|-------|--------|----------|--------|-----|------|
| **Personal** | Yes | Basic | Basic | Skip | README + setup |
| **External** | Full | Full | Full | Full | Complete |

Personal still has guardrails — everything deploys live.

---

## Zone Hierarchy

```
System (root)
├── Frontend    — UI, components, client logic, styling
├── Backend     — API, services, business logic
├── Database    — Schema, migrations, queries
├── Infra       — Docker, CI/CD, deployment
└── Shared      — Types, utils, config
```

**System zone** holds cross-cutting artifacts: specs, project config, the meta-layer. Workers in System zone affect the whole project.

Zones are *where* in the codebase. Phases are *when* in the workflow.

---

## Phases & Gates

```
┌─────────────────────────────────────────────────────────────┐
│  IDEA PHASE                                                 │
│  "Is this worth building?"                                  │
│                                                             │
│  Workers: Researcher                                        │
│  Activities:                                                │
│    - Research prior art, existing solutions                 │
│    - Feasibility assessment                                 │
│    - Effort/value estimation                                │
│                                                             │
│  Output: .mission/ideas/IDEA-{name}.md                      │
│  Gate: "Worth pursuing?" (you decide)                       │
└─────────────────────┬───────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  DESIGN PHASE                                               │
│  "What are we building?"                                    │
│                                                             │
│  Workers: Designer (Frontend), Architect (Backend)          │
│  Activities:                                                │
│    - UI mockups and iteration                               │
│    - API contracts, data models                             │
│    - Technical decisions                                    │
│    - Spec crystallization                                   │
│                                                             │
│  Output: .mission/specs/SPEC-{name}.md + mockups/ + api.md  │
│  Gate: "Spec ready?" (explicit confirmation)                │
└─────────────────────┬───────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  IMPLEMENT PHASE                                            │
│  "Build it"                                                 │
│                                                             │
│  Workers: Developer (per zone)                              │
│  Activities:                                                │
│    - Write code per spec                                    │
│    - Track progress in TODO                                 │
│    - Document findings/blockers                             │
│    - Fresh workers spawned as context bloats                │
│                                                             │
│  Output: CODE + .mission/progress/TODO-{name}.md            │
│  Gate: "Implementation complete?" (TODO clear)              │
└─────────────────────┬───────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  VERIFY PHASE (conditional on audience)                     │
│  "Does it work? Is it good?"                                │
│                                                             │
│  Workers: Reviewer, Security, Tester, QA                    │
│  Activities:                                                │
│    - Code review (quality, patterns)                        │
│    - Security audit (vulnerabilities)                       │
│    - Unit/integration tests                                 │
│    - E2E user flow validation (External only)               │
│                                                             │
│  Output: .mission/reviews/REVIEW-{name}.md + tests/         │
│  Gate: "All findings addressed?"                            │
└─────────────────────┬───────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  DOCUMENT PHASE                                             │
│  "Can someone else understand this?"                        │
│                                                             │
│  Workers: Docs                                              │
│  Activities:                                                │
│    - README.md (always)                                     │
│    - Setup/install docs                                     │
│    - Architecture docs (External)                           │
│    - API documentation (External)                           │
│                                                             │
│  Output: README.md + docs/                                  │
│  Gate: "Docs sufficient for audience?"                      │
└─────────────────────┬───────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  RELEASE PHASE                                              │
│  "Ship it"                                                  │
│                                                             │
│  Workers: DevOps                                            │
│  Activities:                                                │
│    - Version bump                                           │
│    - Changelog finalization                                 │
│    - Deploy to environment                                  │
│    - Smoke test                                             │
│                                                             │
│  Output: CHANGELOG.md + deployed artifact                   │
│  Gate: "Live and working?"                                  │
└─────────────────────────────────────────────────────────────┘
```

---

## Agents

### Persistent Agent

| Agent | Hands-on | Role |
|-------|----------|------|
| **King** | ❌ Never | Coordinates workflow. Spawns workers. Gates phases. Distills context into briefings. Manages token efficiency. Never writes code. |

### Worker Personas (Ephemeral)

| Persona | Phase | Zone | Hands-on | Role |
|---------|-------|------|----------|------|
| **Researcher** | Idea | System | ❌ Read-only | Prior art, feasibility, competitor analysis |
| **Designer** | Design | Frontend | ✅ Mockup files | UI exploration, visual iteration |
| **Architect** | Design | Backend, Database | ✅ Spec files | API contracts, data models, system design |
| **Developer** | Implement | Any | ✅ Full access | Writes production code |
| **Reviewer** | Verify | System | ❌ Read-only | Code quality, patterns, best practices |
| **Security** | Verify | System | ❌ Read-only | Vulnerabilities, OWASP, secrets, auth |
| **Tester** | Verify | Any | ✅ Test files only | Unit tests, integration tests, coverage |
| **QA** | Verify | Frontend, Backend | ❌ Read-only* | E2E tests, user flow validation |
| **Docs** | Document | System | ✅ Markdown only | README, setup guides, API docs |
| **DevOps** | Release | Infra | ✅ Config files | CI/CD, deployment, versioning |
| **Debugger** | Any | Any | ✅ Full access | Bug hunting specialist, log analysis |

*QA may need limited write for test automation scripts

---

## Persona Tool & MCP Restrictions

Personas restrict capabilities at **medium granularity**: read-only vs full access per MCP.

### Tool Access by Persona

| Persona | read | write | edit | bash | grep | tree | git |
|---------|------|-------|------|------|------|------|-----|
| Researcher | ✅ | ❌ | ❌ | 🔒 | ✅ | ✅ | 🔒 |
| Designer | ✅ | ✅* | ✅* | ❌ | ✅ | ✅ | ✅ |
| Architect | ✅ | ✅* | ✅* | ❌ | ✅ | ✅ | ✅ |
| Developer | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Reviewer | ✅ | ❌ | ❌ | 🔒 | ✅ | ✅ | 🔒 |
| Security | ✅ | ❌ | ❌ | 🔒 | ✅ | ✅ | 🔒 |
| Tester | ✅ | ✅* | ✅* | ✅ | ✅ | ✅ | ✅ |
| QA | ✅ | ✅* | ✅* | 🔒 | ✅ | ✅ | 🔒 |
| Docs | ✅ | ✅* | ✅* | ❌ | ✅ | ✅ | ✅ |
| DevOps | ✅ | ✅* | ✅* | ✅ | ✅ | ✅ | ✅ |
| Debugger | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

**Legend:**
- ✅ = Full access
- ❌ = No access
- 🔒 = Read-only / non-destructive only
- ✅* = Restricted to specific file patterns (see below)

### File Pattern Restrictions

| Persona | Can write to |
|---------|--------------|
| Designer | `mockups/**`, `.mission/specs/**` |
| Architect | `.mission/specs/**`, `docs/**` |
| Tester | `tests/**`, `**/*.test.*`, `**/*.spec.*` |
| QA | `e2e/**`, `tests/e2e/**` |
| Docs | `*.md`, `docs/**` |
| DevOps | `Dockerfile`, `.github/**`, `docker-compose.*`, `*.yml`, `*.yaml`, `.mission/releases/**` |

### MCP Access by Persona

| Persona | Filesystem | Git | GitHub | Supabase | Web |
|---------|------------|-----|--------|----------|-----|
| Researcher | 🔒 | 🔒 | 🔒 | ❌ | ✅ |
| Designer | ✅* | ✅ | 🔒 | ❌ | ✅ |
| Architect | ✅* | ✅ | 🔒 | 🔒 | ✅ |
| Developer | ✅ | ✅ | ✅ | ✅ | ✅ |
| Reviewer | 🔒 | 🔒 | 🔒 | 🔒 | ✅ |
| Security | 🔒 | 🔒 | 🔒 | 🔒 | ✅ |
| Tester | ✅* | ✅ | ✅ | 🔒 | ❌ |
| QA | ✅* | 🔒 | 🔒 | 🔒 | ✅ |
| Docs | ✅* | ✅ | ✅ | ❌ | ✅ |
| DevOps | ✅* | ✅ | ✅ | ✅ | ❌ |
| Debugger | ✅ | ✅ | ✅ | ✅ | ✅ |

**Legend:**
- ✅ = Full access
- ❌ = No access  
- 🔒 = Read-only
- ✅* = Write restricted to file patterns

---

## File Structure

```
project/
├── .mission/
│   ├── config.md                    # Project settings
│   │   ├── Audience level (Personal/External)
│   │   ├── Active zones
│   │   └── Custom persona overrides
│   │
│   ├── ideas/
│   │   └── IDEA-{name}.md
│   │       ├── Problem statement
│   │       ├── Prior art / research
│   │       ├── Feasibility assessment
│   │       ├── Effort estimate
│   │       ├── Open questions
│   │       └── Decision: proceed / park / kill
│   │
│   ├── specs/
│   │   └── {name}/
│   │       ├── SPEC.md
│   │       │   ├── Overview
│   │       │   ├── Requirements
│   │       │   ├── Non-requirements
│   │       │   ├── Technical decisions
│   │       │   └── Open questions
│   │       ├── api.md               # API contracts
│   │       └── models.md            # Data models
│   │
│   ├── mockups/
│   │   └── {name}/
│   │       ├── v1.html
│   │       └── v2.jsx
│   │
│   ├── progress/
│   │   └── TODO-{name}.md
│   │       ├── Tasks: pending / in_progress / done
│   │       ├── Findings
│   │       └── Blockers
│   │
│   ├── reviews/
│   │   └── REVIEW-{name}.md
│   │       ├── Code review findings
│   │       ├── Security findings
│   │       ├── Test coverage
│   │       └── QA results
│   │
│   └── releases/
│       └── RELEASE-{version}.md
│
├── src/                             # Actual code
├── tests/                           # Test files
├── docs/                            # Documentation
└── README.md
```

---

## Token Efficiency

### The Problem

Naive approach: Every worker reads full spec (2000 tokens) + full context = expensive.

### The Solution: Briefings

```
┌─────────────────────────────────────────────────────────────┐
│  SOURCE OF TRUTH (files in .mission/)                       │
│  Full specs, complete history, git-tracked                  │
│  2000+ tokens                                               │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼ King distills
┌─────────────────────────────────────────────────────────────┐
│  BRIEFING (what worker receives)                            │
│  - Task: specific assignment                                │
│  - Key requirements (bullets)                               │
│  - Relevant decisions (bullets)                             │
│  - File paths to reference if stuck                         │
│  ~300 tokens                                                │
└─────────────────────────────────────────────────────────────┘
```

### King's Context Management

1. **Distills specs** — Creates task-specific briefings for each worker
2. **Scopes by zone** — Frontend Developer only gets frontend-relevant context
3. **On-demand expansion** — Worker can request full doc if stuck
4. **Findings compression** — Distills worker output before storing
5. **Fresh spawns** — New worker with clean context beats bloated worker

### Lossless Principles

1. **File > Memory** — Decisions, findings, progress live in files, not conversation
2. **Links, not copies** — Specs reference ideas, reviews reference specs
3. **Incremental updates** — Files are updated, not replaced (git tracks history)
4. **Explicit state** — TODO.md is source of truth for "where are we?"

---

## Gate Approvals

| Mode | How to approve |
|------|----------------|
| King mode (conversational) | "Looks good, proceed" / "Hold on, change X" |
| Dashboard | "Finalize" button on phase card |

Both update the same state. King sees button clicks, dashboard sees conversation.

---

## UI Integration

### Settings

- Persona management lives in Settings
- Add/edit/remove personas
- Configure tool and MCP restrictions per persona
- Set project defaults

### Spawn Dialog

- Select persona from dropdown
- Small "Manage personas →" link to settings
- Zone assignment
- Task description

### Org View

Visual hierarchy of agents:

```
┌─────────────────────────────────────────┐
│  👑 King                                │
│  └─ coordinating 4 workers              │
│                                         │
│  ┌─ Zone: Backend ──────────────────┐   │
│  │  🔧 dev-auth (Developer)         │   │
│  │     └─ working: "implement JWT"  │   │
│  │  🧪 test-auth (Tester)           │   │
│  │     └─ idle: waiting on dev      │   │
│  └──────────────────────────────────┘   │
│                                         │
│  ┌─ Zone: Frontend ─────────────────┐   │
│  │  🎨 design-login (Designer)      │   │
│  │     └─ working: "mockup v2"      │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

---

## Phase × Zone Matrix

```
              │ System   │ Frontend │ Backend  │ Database │ Infra   │
──────────────┼──────────┼──────────┼──────────┼──────────┼─────────│
Idea          │ Researcher                                          │
──────────────┼──────────┼──────────┼──────────┼──────────┼─────────│
Design        │          │ Designer │ Architect│ Architect│         │
──────────────┼──────────┼──────────┼──────────┼──────────┼─────────│
Implement     │          │ Developer│ Developer│ Developer│         │
──────────────┼──────────┼──────────┼──────────┼──────────┼─────────│
Verify        │ Reviewer, Security  │ Tester   │ Tester   │         │
              │ QA                  │ QA       │          │         │
──────────────┼──────────┼──────────┼──────────┼──────────┼─────────│
Document      │ Docs                                                │
──────────────┼──────────┼──────────┼──────────┼──────────┼─────────│
Release       │                                           │ DevOps  │
```

---

## Skills (To Be Researched)

Each persona should have associated skills that shape their behavior. Skills are Claude Code skills (`.claude/skills/`).

### Placeholder Skill Assignments

| Persona | Skills (TBD) |
|---------|--------------|
| Researcher | research, analysis |
| Designer | design-principles, ui-patterns |
| Architect | system-design, api-design |
| Developer | implementation, refactoring |
| Reviewer | code-review, best-practices |
| Security | security-audit, owasp |
| Tester | testing, coverage |
| QA | e2e-testing, user-flows |
| Docs | documentation, technical-writing |
| DevOps | deployment, ci-cd |
| Debugger | debugging, log-analysis |

**Research needed:**
- Existing Claude Code skills ecosystem
- Custom skills to create
- Skill content and structure

---

## MCPs (To Be Researched)

### Known MCPs to Evaluate

- Filesystem (read/write/edit)
- Git
- GitHub
- Supabase
- Web search

### Research Needed

- Full list of available MCPs
- Read vs write operation breakdown for each
- Which MCPs map to which personas
- Integration with Personal OS vision

---

## Open Questions

1. **Designer output workflow** — How to make mockup iteration smoother than "create file, open in browser"?

2. **Worker handoff** — Exact mechanism for King to spawn fresh worker with distilled context from dying worker?

3. **Skill content** — What goes in each skill file?

4. **MCP granularity** — Final list of MCPs and their read/write operation split?

---

## Summary

| Concept | Decision |
|---------|----------|
| Coordinator | King (never implements, just coordinates) |
| Workers | Ephemeral, persona-based |
| Default personas | Researcher, Designer, Architect, Developer, Reviewer, Security, Tester, QA, Docs, DevOps, Debugger |
| Audience levels | Personal, External |
| Zones | System > Frontend, Backend, Database, Infra, Shared |
| Phases | Idea → Design → Implement → Verify → Document → Release |
| Gates | Conversational (King mode) or Finalize button (Dashboard) |
| Persistence | `.mission/` directory, files are truth |
| Token efficiency | King distills specs into briefings |
| MCP restrictions | Medium granularity (read-only vs full per MCP) |
| Persona management | Settings panel, linked from spawn dialog |