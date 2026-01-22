# Agents Directory

This directory previously contained educational Python agents (v0-v3).
These have been archived to `docs/archive/agents/` for reference.

MissionControl now uses Claude Code for all agent operations:

- **Online mode:** Claude Code -> Anthropic API
- **Offline mode:** Claude Code -> Ollama (local models)

## Archived Agents

The following Python agent implementations are available in `docs/archive/agents/`:

- `v0_minimal.py` - Minimal agent with bash tool only
- `v1_basic.py` - Full agent with file tools (read, write, edit, bash)
- `v2_todo.py` - Agent with planning and todo tracking
- `v3_subagent.py` - Agent with delegation to subagents

These serve as educational examples of how to build agents from scratch.
