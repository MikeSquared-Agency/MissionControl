# Briefing Format

Worker briefings are JSON files stored in `.mission/handoffs/<task-id>-briefing.json`. They provide ephemeral workers with everything needed to complete a task without access to full project context.

## Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task_id` | string | ✅ | Short hex ID of the task (from `mc task create`) |
| `task_name` | string | ✅ | Human-readable task name |
| `stage` | string | ✅ | Current workflow stage (e.g. `implement`, `verify`) |
| `zone` | string | ✅ | Codebase zone (`frontend`, `backend`, `shared`, etc.) |
| `persona` | string | ✅ | Worker persona (`developer`, `unit-tester`, `docs`, etc.) |
| `objective` | string | ✅ | Clear description of what the worker must accomplish |
| `scope_paths` | string[] | ✅ | Files/directories the worker is allowed to modify |
| `predecessor_findings_paths` | string[] | ❌ | Relative paths to findings from dependency tasks |
| `predecessor_summaries` | object | ❌ | Map of task ID → summary string, parsed from predecessor findings |
| `instructions` | string[] | ✅ | Step-by-step instructions for the worker |
| `output` | string | ✅ | Path where the worker must write its findings file |

## Example

```json
{
  "task_id": "6bc408564b",
  "task_name": "Docs: briefing-format.md + SKILL.md TDD and integration patterns",
  "stage": "implement",
  "zone": "shared",
  "persona": "docs",
  "objective": "Document the briefing schema and update SKILL.md with TDD and integration step patterns.",
  "scope_paths": ["docs/briefing-format.md", "skills/missioncontrol/SKILL.md"],
  "predecessor_findings_paths": [
    ".mission/findings/3e79894550.md",
    ".mission/findings/2f01d6e3ff.md"
  ],
  "predecessor_summaries": {
    "3e79894550": "Created 5 failing TDD tests for generateBriefing()",
    "2f01d6e3ff": "Added ScopePaths field to Task struct and --scope-paths flag"
  },
  "instructions": [
    "Create docs/briefing-format.md documenting the briefing JSON schema",
    "Update SKILL.md to add TDD pattern section",
    "Do NOT modify any Go or Rust files"
  ],
  "output": ".mission/findings/6bc408564b.md"
}
```

## Generating Briefings

After creating a task with `mc task create`, generate its briefing automatically:

```bash
mc briefing generate <task-id>
```

This reads task metadata (name, stage, zone, persona, scope_paths) and predecessor findings to produce the briefing JSON. Predecessor summaries are parsed from the `Summary:` header in each findings file.

## Findings Format

Workers write structured findings to their `output` path:

```markdown
## Task ID: <task-id>
## Status: complete
## Summary: <one-line summary>

<detailed findings, changes made, notes>
```

The `Summary` header is extracted by `mc briefing generate` when building `predecessor_summaries` for downstream tasks.
