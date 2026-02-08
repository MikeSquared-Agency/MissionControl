## Agent Context Management

CRITICAL: When running multi-agent/swarm tasks, proactively manage context windows:

- Run /compact when any agent reaches ~60% context usage - do NOT wait for the limit
- Before spawning sub-agents, instruct each one to self-compact at 60% capacity
- If a sub-agent's task is large, break it into smaller sequential steps rather than one massive operation
- After each sub-agent completes, summarize its results concisely before passing to the next task
- Never let more than 4 agents run concurrently to avoid simultaneous context blowouts
- When delegating to teammates, include in their instructions: "Compact your context proactively at 60% usage. Do not wait for context limit warnings."
