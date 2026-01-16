# Agent Orchestra

A visual multi-agent orchestration system for spawning, monitoring, and coordinating AI agents working on your codebase.

## Stack

- **Agents**: Python (Anthropic SDK)
- **Orchestrator**: Go
- **Stream Parser**: Rust
- **Web UI**: React + Three.js

## Status

**Currently building:** v1 - Agent Fundamentals

See [SPEC.md](SPEC.md) for full project specification.
See [TODO.md](TODO.md) for current progress.

## Quick Start

```bash
# v1 agents (once built)
cd agents
pip install anthropic
python v0_minimal.py "your task here"
```

## Requirements

- Python 3.11+
- `ANTHROPIC_API_KEY` environment variable
