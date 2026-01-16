export type AgentType = 'python' | 'claude'

export type AgentStatus = 'starting' | 'running' | 'idle' | 'error' | 'stopped'

export interface Agent {
  id: string
  type: AgentType
  task: string
  workdir: string
  status: AgentStatus
  pid: number
  tokens: number
  created_at: string
  error?: string
}

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
}

export interface SpawnRequest {
  type: AgentType
  task: string
  workdir?: string
  agent?: string // for python type
}
