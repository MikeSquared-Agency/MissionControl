# Token Efficiency & Context Passing — Research Summary

## The Core Problem

As agents run longer, context accumulates (history, tool outputs, reasoning, findings). This creates:

1. **Cost explosion** — Quadratic scaling with context length
2. **Context rot** — Performance degrades as tokens increase (lost in the middle effect)
3. **Attention budget depletion** — Models lose focus on current task
4. **Handoff losses** — Context gets lost between agent transitions

---

## Key Insights from Research

### 1. Context Rot is Real

From Chroma Research and JetBrains studies:
- LLMs recall information at the **beginning and end** of context better than the middle
- Real applications, such as agent tasks or summarization, demand significantly more processing and reasoning over broader, often more ambiguous information than simple retrieval benchmarks test
- Performance actually **degrades** with more context after a point
- Agent-generated context quickly becomes noise rather than useful information

**Implication:** More context ≠ better results. Curated, minimal context wins.

---

### 2. Two Main Compression Approaches

From JetBrains NeurIPS 2025 research:

| Approach | How it works | Pros | Cons |
|----------|--------------|------|------|
| **Observation Masking** | Hide older, less important observations | Simple, preserves chain-of-thought | Loses detail |
| **LLM Summarization** | AI generates compressed summaries | Preserves meaning | Expensive, adds latency |

**Finding:** Simple observation masking—omitting all but the M most recent observations—can deliver solve-rate and cost performance equal or superior to complex LLM summarization for code-centric agents.

**Implication:** Don't over-engineer. Simple masking often beats fancy summarization.

---

### 3. Tiered Context Architecture (Google ADK)

Google's Agent Development Kit separates storage from presentation:

```
┌─────────────────────────────────────────────────────────────┐
│  STORAGE LAYER (persistent, complete)                       │
│  - Full conversation history                                │
│  - All tool outputs                                         │
│  - Complete reasoning traces                                │
└─────────────────────────────────────────────────────────────┘
                      │
                      ▼ compiled view
┌─────────────────────────────────────────────────────────────┐
│  WORKING CONTEXT (what model sees)                          │
│  - System instructions + agent identity                     │
│  - Selected history (relevance-filtered)                    │
│  - Current tool outputs                                     │
│  - Memory results (if needed)                               │
│  - Artifact references                                      │
└─────────────────────────────────────────────────────────────┘
```

Handoff behavior is controlled by knobs like include_contents on the callee, which determine how much context flows from the root agent to a sub-agent. In the default mode, ADK passes the full contents of the caller's working context—useful when the sub-agent genuinely benefits from the entire history. In none mode, the sub-agent sees no prior history

**Implication:** Separate what you store from what you show. Compile context per-task.

---

### 4. Gastown's Approach: External State + GUPP

Gastown (Steve Yegge) takes a radical stance:

**Work state lives OUTSIDE agent memory:**
- Work state lost in agent memory → Work state stored in Beads ledger
- Hooks are git-backed persistent storage
- Agents read state from files, not conversation history

**GUPP Principle:** "If there is work on your Hook, YOU MUST RUN IT"
- When context gets full or an agent needs a fresh start, handoff transfers work state to a new session
- Agents self-destruct and restart fresh rather than accumulating context
- Use gt handoff liberally after every task, in every worker. Only let sessions go long if they need to accumulate important context for a big design or decision

**Seance:** Query previous sessions for context
- Communicating with previous sessions via gt seance. Allows agents to query their predecessors for context and decisions from earlier work.

**Implication:** Files are memory. Fresh agents beat bloated ones. Let agents die and resurrect.

---

### 5. AgentDiet: Trajectory Reduction

From research on inference-time optimization:

AgentDiet, an inference-time trajectory reduction framework that removes useless, redundant, and expired information from the serialized agent trajectory. This approach yields input-token savings of 40–60% and cuts final computational cost by 21–36%

Three pruning strategies:
1. **Useless content** — Semantic similarity + rule-based filters
2. **Redundant content** — Detect duplicate text spans
3. **Expired content** — Track updates to referenced files/resources

**Implication:** Actively prune context. Remove what's outdated, duplicated, or irrelevant.

---

### 6. AgentFold: Sublinear Context Growth

Results show substantially reduced context growth (sublinear <7k tokens after 100 turns vs. >91k for ReAct) while preserving 98–99% survival probability of key details

Uses "granular and deep folding directives" — learned compression strategies that maintain critical information while dramatically reducing token count.

**Implication:** Context can grow sublinearly if you compress intelligently.

---

### 7. Git-Based Memory (GCC)

Git-Context-Controller (GCC) formalizes agent memory as a versioned hierarchy akin to a Git repository: memory is manipulated by COMMIT, BRANCH, MERGE, and CONTEXT operations

Operations:
- **COMMIT** — Save milestone state
- **BRANCH** — Explore alternative paths
- **MERGE** — Combine findings
- **CONTEXT** — Load relevant state

**Implication:** Version control for agent memory. Branch for exploration, merge for synthesis.

---

### 8. Multi-Agent Handoff Patterns

From Anthropic's Research system:

We implemented patterns where agents summarize completed work phases and store essential information in external memory before proceeding to new tasks. When context limits approach, agents can spawn fresh subagents with clean contexts while maintaining continuity through careful handoffs.

**Key insight:** splitting the work among subagents (each with their own 100k window, effectively) and then compressing their findings back to the lead agent was far more effective

**Implication:** Parallel agents with independent contexts > one bloated agent. Compress findings on return.

---

### 9. Structured Handoffs (Not Free-Text)

From best practices research:

Free-text handoffs are the main source of context loss. Treat inter-agent transfer like a public API

Requirements:
- Schema-constrained outputs (JSON)
- Versioned payloads
- Explicit contracts between agents

**Implication:** Handoffs need structure. Define schemas, not prose.

---

### 10. Cascaded Model Orchestration

Cascaded LLM orchestration schemes such as BudgetMLAgent demonstrate that using a low-cost model for most agentic calls, escalating only on failure or explicit "ask-the-expert" inflection, drives down average run cost by over 94%

- Use cheap models for simple tasks
- Escalate to powerful models only when needed
- Different models for different worker types

**Implication:** Not every agent needs the best model. Match capability to task.

---

## Synthesis: MissionControl Token Strategy

Based on this research, here's a multi-layered approach:

### Layer 1: File-Based Truth (Gastown-inspired)

```
.mission/
├── specs/       ← Full specifications (persistent)
├── progress/    ← TODO state (persistent)
├── findings/    ← Worker discoveries (persistent)
└── briefings/   ← Compiled handoffs (ephemeral)
```

- All truth lives in files, not conversation
- Agents read state, do work, write findings, die
- Context = reading relevant files, not accumulating history

### Layer 2: Briefing Compilation (ADK-inspired)

King compiles **briefings** from source files:

```
Full spec (2000 tokens)
        │
        ▼ King distills
Briefing (300 tokens)
  ├── Task description
  ├── Key requirements (3-5 bullets)
  ├── Relevant decisions
  └── File paths for deep-dive
```

- Workers receive briefings, not full specs
- On-demand expansion: worker can request full doc if stuck
- Zone-scoped: Frontend Developer doesn't see backend spec

### Layer 3: Handoff Schema (Structured)

```typescript
interface WorkerHandoff {
  task_id: string;
  status: 'complete' | 'blocked' | 'partial';
  findings: Finding[];
  artifacts: string[];        // file paths created
  open_questions: string[];
  next_steps?: string[];      // if partial
  context_for_successor?: {   // if context needed
    key_decisions: string[];
    gotchas: string[];
  };
}

interface Finding {
  type: 'discovery' | 'blocker' | 'decision' | 'concern';
  summary: string;           // 1-2 sentences
  details_path?: string;     // file with full details
}
```

- Workers report back in structured format
- King compresses findings before storing
- Successor workers get distilled handoff, not raw output

### Layer 4: Context Lifecycle

```
SPAWN
  │
  ▼
Worker receives briefing (~300 tokens)
  │
  ▼
Worker reads relevant files on-demand
  │
  ▼
Worker does work, context grows
  │
  ├─── Context < 50% window → Continue
  │
  ├─── Context > 50% window → Consider handoff
  │
  └─── Context > 75% window → Force handoff
          │
          ▼
      Worker summarizes findings
          │
          ▼
      King spawns fresh worker with handoff
          │
          ▼
      Old worker dies
```

### Layer 5: Pruning Strategies (AgentDiet-inspired)

During long-running work, King prunes context by:

1. **Stale tool outputs** — File read from 20 turns ago? Drop it.
2. **Superseded decisions** — "Changed approach" invalidates old reasoning
3. **Duplicate information** — Same file read twice? Keep latest.
4. **Completed subtasks** — Task done? Collapse to finding summary.

### Layer 6: Parallel Worker Pattern (Anthropic-inspired)

For large tasks:

```
King spawns:
├── Developer-Frontend (own context)
├── Developer-Backend (own context)  
└── Developer-Database (own context)

Each works in parallel with full context budget.

On completion:
├── Each submits structured findings
├── King compresses and merges
└── Combined findings < sum of individual contexts
```

### Layer 7: Model Cascading (Future)

| Worker Type | Model | Rationale |
|-------------|-------|-----------|
| King | Claude Opus | Coordination, synthesis, judgment |
| Designer | Claude Sonnet | Creative, iterative |
| Developer | Claude Sonnet | Implementation |
| Reviewer | Claude Haiku | Pattern matching, checklist |
| Docs | Claude Haiku | Templated writing |

Match model to task complexity. Save expensive models for judgment calls.

---

## Implementation Priority

### Phase 1: File-Based State
- `.mission/` structure
- Briefing format
- Handoff schema

### Phase 2: King Compilation
- Spec → briefing distillation
- Zone-scoped context
- On-demand expansion

### Phase 3: Context Monitoring
- Track worker context size
- Auto-handoff triggers
- Fresh worker spawning

### Phase 4: Pruning
- Stale content detection
- Compression on findings
- Duplicate removal

### Phase 5: Model Cascading
- Worker-type model mapping
- Cost tracking per worker
- Dynamic model selection

---

## Key Takeaways

1. **Files beat memory** — Store truth externally, compile context per-task
2. **Fresh beats bloated** — Spawn new workers rather than accumulate
3. **Structure beats prose** — Schema-constrained handoffs, not free text
4. **Minimal beats maximal** — Briefings > full specs
5. **Parallel beats serial** — Independent contexts that merge findings
6. **Pruning is active** — Remove stale, duplicate, expired content
7. **Match model to task** — Expensive models for judgment, cheap for execution

---

## Open Questions

1. **Handoff trigger heuristics** — How do we detect "context getting bloated" before it hurts performance?
2. **Briefing quality** — How do we validate that briefings preserve critical information?
3. **Finding compression** — How much can we compress findings without losing actionable detail?
4. **Cross-worker context** — When does Worker B need to know what Worker A discovered?
5. **Seance mechanism** — How do we let new workers query old worker sessions?