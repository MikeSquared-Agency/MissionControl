// Swarm BFF response types

export interface SwarmOverview {
  warren?: WarrenData
  chronicle?: ChronicleData
  dispatch?: DispatchData
  promptforge?: PromptForgeData
  alexandria?: AlexandriaData
  errors: Record<string, string>
  fetched_at: string
}

export interface WarrenData {
  health?: WarrenHealth
  agents?: WarrenAgent[]
}

export interface WarrenHealth {
  status: string
  uptime?: number
  version?: string
  agents_connected?: number
}

export interface WarrenAgent {
  id: string
  name: string
  state: string // ready, sleeping, starting, stopping
  connections?: number
  policy?: string
  started_at?: string
}

export interface ChronicleData {
  metrics?: ChronicleMetrics
  dlq?: DLQStats
}

export interface ChronicleMetrics {
  total_events?: number
  events_per_minute?: number
  error_rate?: number
  [key: string]: unknown
}

export interface DLQStats {
  depth?: number
  oldest_age_seconds?: number
  processing_rate?: number
  [key: string]: unknown
}

export interface DispatchData {
  stats?: DispatchStats
  agents?: DispatchAgent[]
}

export interface DispatchStats {
  pending?: number
  in_progress?: number
  completed?: number
  failed?: number
  total?: number
  [key: string]: unknown
}

export interface DispatchAgent {
  id: string
  name?: string
  status: string
  current_task?: string
  tasks_completed?: number
}

export interface PromptForgeData {
  prompt_count?: number
  prompts?: unknown
}

export interface AlexandriaData {
  collection_count?: number
  collections?: unknown
}

// SSE event from Warren
export interface WarrenSSEEvent {
  id?: string
  type: string
  agent?: string
  data?: unknown
  timestamp: number
}

// Derived alert from cross-service data
export interface SwarmAlert {
  id: string
  level: 'info' | 'warning' | 'critical'
  service: string
  message: string
  timestamp: number
}
