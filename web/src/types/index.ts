// Re-export v4 types
export * from './v4'

// Agent types
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
  tools: string[]
  skills: string[]
  systemPrompt: string
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

// API types
export interface SpawnRequest {
  type: AgentType
  name: string
  task: string
  persona?: string
  zone: string
  workingDir?: string
  agent?: string // for python agent version
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

// Default personas
export const DEFAULT_PERSONAS: Persona[] = [
  {
    id: 'code-reviewer',
    name: 'Code Reviewer',
    description: 'Reviews code for bugs, security, and best practices',
    color: '#22c55e',
    tools: ['read', 'grep', 'bash_readonly'],
    skills: ['code-review', 'security-audit'],
    systemPrompt: 'You are a code reviewer. Focus on bugs, security, and best practices. Do not modify files.'
  },
  {
    id: 'full-developer',
    name: 'Full Developer',
    description: 'Senior developer with full access to all tools',
    color: '#3b82f6',
    tools: ['read', 'write', 'edit', 'bash', 'grep', 'tree'],
    skills: ['code-review', 'implementation', 'refactoring', 'testing'],
    systemPrompt: 'You are a senior developer. Write clean, tested, documented code.'
  },
  {
    id: 'test-writer',
    name: 'Test Writer',
    description: 'QA engineer focused on comprehensive testing',
    color: '#eab308',
    tools: ['read', 'write_tests', 'bash', 'grep'],
    skills: ['testing', 'test-coverage'],
    systemPrompt: 'You are a QA engineer. Write comprehensive tests. Do not modify source files, only test files.'
  },
  {
    id: 'documentation',
    name: 'Documentation',
    description: 'Technical writer for clear documentation',
    color: '#a855f7',
    tools: ['read', 'write_docs', 'grep'],
    skills: ['documentation', 'api-docs'],
    systemPrompt: 'You are a technical writer. Create clear, comprehensive documentation.'
  }
]

// Default zone
export const DEFAULT_ZONE: Zone = {
  id: 'default',
  name: 'Default',
  color: '#6b7280',
  workingDir: ''
}
