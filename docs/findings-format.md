# Findings File Format

Workers write findings to `.mission/findings/<task_id>.md`. The file must start with a structured header for machine parsing, followed by free-form markdown.

## Required Header

```markdown
# Findings: <title>

- **Task ID:** <hash>
- **Status:** complete | blocked | partial
- **Summary:** <one-line description of what was done>
```

## Optional Fields

```markdown
- **Files Changed:** file1.go, file2.go
- **Blockers:** <description of what's blocking>
- **Decisions:** <key decisions made>
```

## Example

```markdown
# Findings: Implement register/link flow

- **Task ID:** 66f27427ba
- **Status:** complete
- **Summary:** Added two-step worker registration with label-based lookup and token parsing

## Changes
- handler.go: register/link endpoints, token regex, label registry
- handler_test.go: 8 new tests

## Decisions
- Kept old combined /register endpoint for backward compat
- Cost estimation deferred (tokens tracked, USD = 0 for now)
```

## Parsing

The watcher uses these fields to:
- Match findings to tasks via **Task ID**
- Auto-mark tasks done when **Status** is `complete`
- Display summary in dashboard and status output
