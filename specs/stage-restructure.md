# Spec: Stage Restructure — TDD, Review, E2E

## Problem

The current 10-stage workflow has overlapping responsibilities:
- **Implement** = write code (no testing discipline)
- **Verify** = "meets requirements" (vague)
- **Validate** = "works in real environment" (often skipped)

There's no code review stage, no TDD loop, and no clear separation between unit and integration testing.

## New Stage Definitions

```
discovery → goal → requirements → planning → design → implement → review → verify → document → release
```

Same 10 stages, same names for most. Three stages get redefined:

### Implement (TDD Loop)

The implement stage now follows a red/green TDD cycle with two personas working in sequence:

1. **unit-tester** writes failing unit tests based on the design (RED)
2. **developer** implements code to make tests pass (GREEN)
3. Loop until all tests pass
4. Developer refactors if needed (REFACTOR)

The unit-tester runs slightly ahead — writes tests for the next piece while the developer is implementing the current one. Both touch different files (`_test.go` vs `.go`), so they can run in parallel when the scope allows.

**Gate criteria:**
- All unit tests pass
- Code compiles
- No test stubs or TODOs left

### Review (was: Validate)

The validate stage is repurposed as a code review stage. A **reviewer** persona checks:

- Code quality and style consistency
- Architecture patterns (does it match ARCHITECTURE.md?)
- Error handling completeness
- Security concerns
- No unnecessary dependencies
- Commit hygiene

This is where CodeRabbit feedback gets addressed too.

**Gate criteria:**
- Reviewer persona has produced findings
- All review issues addressed or documented as accepted trade-offs
- Code ready for integration testing

### Verify (E2E Integration Testing)

Verify becomes dedicated E2E integration testing. An **integration-tester** persona:

- Spins up real services (`mc serve`, gateway)
- Runs end-to-end workflows (create mission → spawn workers → check dashboard)
- Playwright tests for frontend (dashboard, chat)
- Go integration tests for backend (real HTTP calls, real WebSocket)
- Tests cross-service communication (bridge → gateway → hub → dashboard)

**Gate criteria:**
- E2E test suite passes
- Real environment validated (services running, data flowing)
- No regressions in existing E2E tests

## Personas

| Persona | Stage | Role |
|---|---|---|
| researcher | discovery | Investigate, read code, map what exists |
| architect | design | Data structures, interfaces, system design |
| unit-tester | implement | Write unit tests (TDD red phase) |
| developer | implement | Write code to pass tests (TDD green phase) |
| reviewer | review | Code quality, patterns, security |
| integration-tester | verify | E2E tests, Playwright, real service testing |
| writer | document | Docs, specs, READMEs, ARCHITECTURE.md |

## TDD Flow Detail

```
┌─────────────────────────────────────────────┐
│                 IMPLEMENT                    │
│                                              │
│   unit-tester              developer         │
│   ┌──────────┐            ┌──────────┐      │
│   │ Write    │            │          │      │
│   │ failing  │───tests───>│ Read     │      │
│   │ tests    │            │ tests    │      │
│   │ (RED)    │            │          │      │
│   └──────────┘            │ Write    │      │
│                           │ code     │      │
│                           │ (GREEN)  │      │
│                           │          │      │
│                           │ Refactor │      │
│                           └──────────┘      │
│                                ↓             │
│                        All tests pass?       │
│                         yes ↓    no ↑        │
│                         GATE      LOOP       │
└─────────────────────────────────────────────┘
```

### Parallelism

The unit-tester and developer can work in parallel when:
- Tests are in `_test.go` files, implementation in `.go` files
- They're working on different packages/modules
- The tester is writing the NEXT set of tests while the developer finishes the current ones

They CANNOT work in parallel when:
- Implementation requires understanding test expectations (first iteration)
- Tests depend on interfaces the developer hasn't defined yet

**Default:** Run unit-tester first, then developer. Parallel only when explicitly scoped to different files.

## Implementation Plan

### Rust (`core/`)
- Update stage definitions in the workflow engine
- Update gate criteria for implement (all unit tests pass), review (reviewer findings addressed), verify (E2E tests pass)
- Rebuild `mc-core`

### Go (`cmd/mc/`)
- Update `stages` slice in `stage.go` (names stay the same, just redefine semantics)
- Update `checkGateViaCore` to handle new gate criteria
- No stage name changes needed — validate→review is a semantic change, not a rename

### Go (`orchestrator/`)
- No changes — the orchestrator doesn't enforce stage semantics

### Documentation
- Update ARCHITECTURE.md stage definitions
- Update MC skill file
- Update findings-format.md with persona expectations per stage

## Migration

No migration needed. Stage names don't change. The semantic shift is in:
1. Gate criteria (Rust)
2. Process documentation
3. How Kai uses the stages (skill file)

Existing missions continue to work — the gate criteria just become more specific.

## Not In Scope

- Changing stage names (too much churn for the same result)
- Automated TDD loop orchestration (Kai manages the loop manually for now)
- Playwright test framework setup (separate mission)
- CI integration for E2E tests (separate mission)
