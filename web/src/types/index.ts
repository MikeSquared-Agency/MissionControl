// Re-export workflow types
export * from './workflow'
import type { Stage } from './workflow'

// Agent types
// Note: 'python' is deprecated, all agents now use 'claude-code'
export type AgentType = 'python' | 'claude-code'

export type AgentStatus = 'starting' | 'working' | 'idle' | 'error' | 'waiting' | 'stopped'

export interface Agent {
  id: string
  name: string
  type: AgentType
  persona: string | null
  status: AgentStatus
  tokens: number
  cost: number
  task: string
  zone: string
  workingDir: string
  attention: AttentionRequest | null
  findings: string[]
  conversation: ConversationMessage[]
  created_at: string
  error?: string
  pid?: number
  offlineMode?: boolean
  model?: string
}

// Attention types
export interface AttentionRequest {
  type: 'question' | 'permission' | 'error' | 'complete'
  message: string
  since: number
  retryable?: boolean
  retryIn?: number
}

// Conversation types
export interface ConversationMessage {
  role: 'user' | 'assistant' | 'error'
  content: string
  toolCalls?: ToolCall[]
  isQuestion?: boolean
  isPermission?: boolean
  timestamp: number
}

export interface ToolCall {
  id: string
  tool: string
  args: Record<string, unknown>
  result?: string
  collapsed: boolean
}

// Zone types
export interface Zone {
  id: string
  name: string
  color: string
  workingDir: string
}

// Persona types
export interface Persona {
  id: string
  name: string
  description: string
  color: string
  stage: Stage // which workflow stage this persona belongs to
  enabled: boolean // can be disabled per-project
  tools: string[] // available tools for this persona
  skills: string[] // skills/capabilities
  systemPrompt: string
  isBuiltin: boolean // true for the workflow personas
}

// King Mode types
export interface KingMessage {
  role: 'user' | 'assistant'
  content: string
  thinking?: string
  actions?: KingAction[]
  timestamp: number
}

export interface KingAction {
  type: 'spawn' | 'kill' | 'message' | 'create_zone' | 'move_agent'
  agent?: string
  persona?: string
  zone?: string
  task?: string
  content?: string
}

export interface KingQuestion {
  question: string
  options: string[]
  selected: number
}

// API types
export interface SpawnRequest {
  type?: AgentType
  name: string
  task: string
  persona?: string
  zone: string
  workingDir?: string
  agent?: string // for python agent version
  offlineMode?: boolean
  ollamaModel?: string
}

// Connection status
export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected'

// Settings
export interface Settings {
  apiKey: string
  defaultWorkingDir: string
}

// Legacy Event type for backward compatibility with current WebSocket
export interface Event {
  type: string
  agent_id?: string
  content?: string
  tool?: string
  args?: Record<string, unknown>
  result?: string
  turn?: number
  tokens?: number
  status?: string
  error?: string
  data?: unknown
}

// The workflow personas
export const DEFAULT_PERSONAS: Persona[] = [
  {
    id: 'researcher',
    name: 'Researcher',
    description: 'Research prior art, assess feasibility, analyze competitors',
    color: '#6366f1', // indigo
    stage: 'discovery',
    enabled: true,
    tools: ['read', 'grep', 'web_search', 'bash_readonly'],
    skills: ['research', 'analysis', 'feasibility-assessment'],
    systemPrompt: 'You are a Researcher. Focus on research, prior art, and feasibility analysis. Do not modify files.',
    isBuiltin: true
  },
  {
    id: 'analyst',
    name: 'Analyst',
    description: 'Defines goals, success metrics, and project scope',
    color: '#f59e0b', // amber
    stage: 'goal',
    enabled: true,
    tools: ['read', 'write', 'bash'],
    skills: ['goal-definition', 'metrics', 'scope-analysis'],
    systemPrompt: 'You are an Analyst. Define clear goals, success metrics, and project scope. Focus on measurable outcomes.',
    isBuiltin: true
  },
  {
    id: 'requirements-engineer',
    name: 'Requirements Engineer',
    description: 'Documents requirements and acceptance criteria',
    color: '#8b5cf6', // violet
    stage: 'requirements',
    enabled: true,
    tools: ['read', 'write', 'bash'],
    skills: ['requirements-gathering', 'acceptance-criteria', 'specifications'],
    systemPrompt: 'You are a Requirements Engineer. Document requirements and acceptance criteria. Be precise and testable.',
    isBuiltin: true
  },
  {
    id: 'architect',
    name: 'Architect',
    description: 'API contracts, data models, system design',
    color: '#06b6d4', // cyan
    stage: 'planning',
    enabled: true,
    tools: ['read', 'write', 'grep', 'bash_readonly'],
    skills: ['api-design', 'data-modeling', 'system-architecture', 'technical-specs'],
    systemPrompt: 'You are an Architect. Design API contracts, data models, and system architecture. Write specs, not code.',
    isBuiltin: true
  },
  {
    id: 'designer',
    name: 'Designer',
    description: 'UI mockups, wireframes, user flows',
    color: '#ec4899', // pink
    stage: 'design',
    enabled: true,
    tools: ['read', 'write', 'grep'],
    skills: ['ui-design', 'wireframing', 'user-flows', 'mockups'],
    systemPrompt: 'You are a Designer. Create UI mockups, wireframes, and user flows. Focus on design artifacts.',
    isBuiltin: true
  },
  {
    id: 'developer',
    name: 'Developer',
    description: 'Production code, tests, feature implementation',
    color: '#3b82f6', // blue
    stage: 'implement',
    enabled: true,
    tools: ['read', 'write', 'edit', 'bash', 'grep', 'tree'],
    skills: ['implementation', 'testing', 'refactoring', 'debugging'],
    systemPrompt: 'You are a Developer. Write clean, tested, documented code. Follow existing patterns.',
    isBuiltin: true
  },
  {
    id: 'debugger',
    name: 'Debugger',
    description: 'Bug investigation, root cause analysis, fixes',
    color: '#f97316', // orange
    stage: 'implement',
    enabled: true,
    tools: ['read', 'write', 'edit', 'bash', 'grep'],
    skills: ['debugging', 'root-cause-analysis', 'bug-fixing', 'log-analysis'],
    systemPrompt: 'You are a Debugger. Investigate bugs, identify root causes, and implement fixes. Document findings.',
    isBuiltin: true
  },
  {
    id: 'reviewer',
    name: 'Reviewer',
    description: 'Code quality review, best practices',
    color: '#22c55e', // green
    stage: 'verify',
    enabled: true,
    tools: ['read', 'grep', 'bash_readonly'],
    skills: ['code-review', 'best-practices', 'quality-assurance'],
    systemPrompt: 'You are a Reviewer. Review code for bugs, quality, and best practices. Do not modify files.',
    isBuiltin: true
  },
  {
    id: 'security',
    name: 'Security',
    description: 'Vulnerability checks, OWASP compliance',
    color: '#ef4444', // red
    stage: 'verify',
    enabled: false, // disabled by default for personal projects
    tools: ['read', 'grep', 'bash_readonly'],
    skills: ['security-audit', 'vulnerability-assessment', 'owasp', 'penetration-testing'],
    systemPrompt: 'You are a Security auditor. Check for vulnerabilities (OWASP Top 10), review auth, and assess input validation. Do not modify files.',
    isBuiltin: true
  },
  {
    id: 'tester',
    name: 'Tester',
    description: 'Unit and integration tests, coverage',
    color: '#eab308', // yellow
    stage: 'verify',
    enabled: true,
    tools: ['read', 'write', 'bash', 'grep'],
    skills: ['unit-testing', 'integration-testing', 'test-coverage', 'test-design'],
    systemPrompt: 'You are a Tester. Write comprehensive unit and integration tests. Only modify test files.',
    isBuiltin: true
  },
  {
    id: 'qa',
    name: 'QA',
    description: 'E2E validation, user flows, UX testing',
    color: '#a855f7', // purple
    stage: 'validate',
    enabled: false, // disabled by default for personal projects
    tools: ['read', 'write', 'bash', 'grep'],
    skills: ['e2e-testing', 'user-acceptance-testing', 'ux-validation', 'manual-testing'],
    systemPrompt: 'You are a QA engineer. Validate user flows end-to-end, test edge cases, and check UX. Write E2E test scripts.',
    isBuiltin: true
  },
  {
    id: 'docs',
    name: 'Docs',
    description: 'Documentation, guides, API docs',
    color: '#64748b', // slate
    stage: 'document',
    enabled: true,
    tools: ['read', 'write', 'grep'],
    skills: ['documentation', 'technical-writing', 'api-docs', 'tutorials'],
    systemPrompt: 'You are a Documentation writer. Create clear, comprehensive documentation. Write markdown files only.',
    isBuiltin: true
  },
  {
    id: 'devops',
    name: 'DevOps',
    description: 'CI/CD, deployments, versioning',
    color: '#10b981', // emerald
    stage: 'release',
    enabled: false, // disabled by default for personal projects
    tools: ['read', 'write', 'edit', 'bash', 'grep'],
    skills: ['ci-cd', 'deployment', 'infrastructure', 'monitoring', 'versioning'],
    systemPrompt: 'You are a DevOps engineer. Configure CI/CD pipelines, manage deployments, and handle versioning.',
    isBuiltin: true
  }
]

// Default zone
export const DEFAULT_ZONE: Zone = {
  id: 'default',
  name: 'Default',
  color: '#6b7280',
  workingDir: ''
}
